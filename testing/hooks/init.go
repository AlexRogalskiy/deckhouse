package hooks

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	// Define Register func and Registry object to import go-hooks.
	"github.com/flant/addon-operator/pkg/module_manager"
	addonutils "github.com/flant/addon-operator/pkg/utils"
	"github.com/flant/addon-operator/pkg/values/validation"
	"github.com/flant/addon-operator/sdk"
	"github.com/flant/addon-operator/sdk/registry"
	utils "github.com/flant/shell-operator/pkg/utils/file"
	"github.com/flant/shell-operator/test/hook/context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"golang.org/x/sys/unix"
	yamlv3 "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/deckhouse/deckhouse/testing/library"
	"github.com/deckhouse/deckhouse/testing/library/object_store"
	"github.com/deckhouse/deckhouse/testing/library/sandbox_runner"
	"github.com/deckhouse/deckhouse/testing/library/values_store"
	"github.com/deckhouse/deckhouse/testing/library/values_validation"
)

var (
	globalTmpDir string
	moduleName   string
)

const (
	globalKcovDir = "/deckhouse/testing/kcov-report"
)

func (hec *HookExecutionConfig) KubernetesGlobalResource(kind, name string) object_store.KubeObject {
	return hec.ObjectStore.KubernetesGlobalResource(kind, name)
}

func (hec *HookExecutionConfig) KubernetesResource(kind, namespace, name string) object_store.KubeObject {
	return hec.ObjectStore.KubernetesResource(kind, namespace, name)
}

type ShellOperatorHookConfig struct {
	ConfigVersion interface{} `json:"configVersion,omitempty"`
	Kubernetes    interface{} `json:"kubernetes,omitempty"`
	Schedule      interface{} `json:"schedule,omitempty"`
}

type CustomCRD struct {
	Group      string
	Version    string
	Kind       string
	Namespaced bool
}

type HookExecutionConfig struct {
	tmpDir                   string // FIXME
	HookPath                 string
	GoHook                   sdk.GoHook
	values                   *values_store.ValuesStore
	configValues             *values_store.ValuesStore
	hookConfig               string // <hook> --config output
	KubeExtraCRDs            []CustomCRD
	IsKubeStateInited        bool
	KubeState                string // yaml string
	ObjectStore              object_store.ObjectStore
	KubernetesResourcePatch  KubernetesPatch
	BindingContexts          BindingContextsSlice
	BindingContextController *context.BindingContextController
	extraHookEnvs            []string
	ValuesValidator          *validation.ValuesValidator

	Session   *gexec.Session
	GoHookOut *sdk.HookOutput
}

func (hec *HookExecutionConfig) RegisterCRD(group, version, kind string, namespaced bool) {
	newCRD := CustomCRD{Group: group, Version: version, Kind: kind, Namespaced: namespaced}
	hec.KubeExtraCRDs = append(hec.KubeExtraCRDs, newCRD)
}

func (hec *HookExecutionConfig) ValuesGet(path string) library.KubeResult {
	return hec.values.Get(path)
}

func (hec *HookExecutionConfig) ConfigValuesGet(path string) library.KubeResult {
	return hec.configValues.Get(path)
}

func (hec *HookExecutionConfig) ValuesSet(path string, value interface{}) {
	hec.values.SetByPath(path, value)
}

func (hec *HookExecutionConfig) ConfigValuesSet(path string, value interface{}) {
	hec.configValues.SetByPath(path, value)
}

func (hec *HookExecutionConfig) ValuesDelete(path string) {
	hec.values.DeleteByPath(path)
}

func (hec *HookExecutionConfig) ConfigValuesDelete(path string) {
	hec.configValues.DeleteByPath(path)
}

func (hec *HookExecutionConfig) ValuesSetFromYaml(path string, value []byte) {
	hec.values.SetByPathFromYAML(path, value)
}

func (hec *HookExecutionConfig) ConfigValuesSetFromYaml(path string, value []byte) {
	hec.configValues.SetByPathFromYAML(path, value)
}

func (hec *HookExecutionConfig) AddHookEnv(env string) {
	hec.extraHookEnvs = append(hec.extraHookEnvs, env)
}

func HookExecutionConfigInit(initValues, initConfigValues string) *HookExecutionConfig {
	var err error
	hookEnvs := []string{"ADDON_OPERATOR_NAMESPACE=tests", "DECKHOUSE_POD=tests"}

	hec := new(HookExecutionConfig)
	_, f, _, ok := runtime.Caller(1)
	if !ok {
		panic("can't execute runtime.Caller")
	}
	hec.HookPath = strings.TrimSuffix(f, "_test.go")

	// Use a working directory to retrieve moduleName and modulePath to load OpenAPI schemas.
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("get working directory: %v", err))
	}

	var modulePath string
	if !strings.Contains(wd, "global-hooks") {
		modulePath = filepath.Dir(wd)

		var err error
		moduleName, err = library.GetModuleNameByPath(modulePath)
		if err != nil {
			panic(fmt.Errorf("get module name from working directory: %v", err))
		}
	}
	// TODO Is there a solution for ginkgo to have a shared validator for all tests in module?
	hec.ValuesValidator = validation.NewValuesValidator()
	err = values_validation.LoadOpenAPISchemas(hec.ValuesValidator, moduleName, modulePath)
	if err != nil {
		panic(fmt.Errorf("load module OpenAPI schemas for hook: %v", err))
	}

	// Search golang hook by name.
	goHookPath := hec.HookPath + ".go"
	hasGoHook, err := utils.FileExists(goHookPath)
	if err == nil && hasGoHook {
		goHookName := filepath.Base(goHookPath)
		for _, h := range registry.Registry().Hooks() {
			if strings.Contains(goHookPath, h.Metadata().Path) {
				hec.GoHook = h
				break
			}
		}
		if hec.GoHook == nil {
			panic(fmt.Errorf("go hook '%s' exists but is not registered as '%s'", goHookPath, goHookName))
		}
		hec.HookPath = ""
	}

	hec.KubeExtraCRDs = []CustomCRD{}

	BeforeEach(func() {
		defaultConfigValues := addonutils.Values{
			addonutils.GlobalValuesKey:                   map[string]interface{}{},
			addonutils.ModuleNameToValuesKey(moduleName): map[string]interface{}{},
		}
		configValues, err := addonutils.NewValuesFromBytes([]byte(initConfigValues))
		if err != nil {
			panic(err)
		}
		mergedConfigValuesYaml, err := addonutils.MergeValues(defaultConfigValues, configValues).YamlBytes()
		if err != nil {
			panic(err)
		}
		values, err := addonutils.NewValuesFromBytes([]byte(initValues))
		if err != nil {
			panic(err)
		}
		mergedValuesYaml, err := addonutils.MergeValues(defaultConfigValues, values).YamlBytes()
		if err != nil {
			panic(err)
		}
		hec.configValues, err = values_store.NewStoreFromRawYaml(mergedConfigValuesYaml)
		if err != nil {
			panic(err)
		}
		hec.values, err = values_store.NewStoreFromRawYaml(mergedValuesYaml)
		if err != nil {
			panic(err)
		}
		hec.IsKubeStateInited = false
		hec.BindingContexts.Set()
	})

	// Run --config for shell hook
	if hec.GoHook == nil {
		hookEnvs = append(hookEnvs, "D8_IS_TESTS_ENVIRONMENT=yes")

		stdout := bytes.Buffer{}
		stderr := bytes.Buffer{}
		cmd := &exec.Cmd{
			Path:   hec.HookPath,
			Args:   []string{hec.HookPath, "--config"},
			Env:    append(os.Environ(), hookEnvs...),
			Stdout: &stdout,
			Stderr: &stderr,
		}

		hec.tmpDir, err = ioutil.TempDir(globalTmpDir, "")
		if err != nil {
			panic(err)
		}

		if err := cmd.Run(); err != nil {
			panic(fmt.Errorf("%s\nstdout:\n%s\n\nstderr:\n%s", err, stdout.String(), stderr.String()))
		}

		var config ShellOperatorHookConfig
		err = yaml.Unmarshal(stdout.Bytes(), &config)
		if err != nil {
			panic(err)
		}

		result, err := json.Marshal(config)
		if err != nil {
			panic(err)
		}
		hec.hookConfig = string(result)
	}

	return hec
}

func (hec *HookExecutionConfig) KubeStateSetAndWaitForBindingContexts(newKubeState string, desiredQuantity int) context.GeneratedBindingContexts {
	var contexts context.GeneratedBindingContexts
	var err error
	if !hec.IsKubeStateInited {
		hec.BindingContextController, err = context.NewBindingContextController(hec.hookConfig)
		if err != nil {
			panic(err)
		}

		if hec.GoHook != nil {
			// create GlobalHook or Module and convert its config
			m := hec.GoHook.Metadata()
			// tests are only for schedule and kubernetes bindings, so we can test all hooks as global hooks
			globalHook := module_manager.NewGlobalHook(m.Name, m.Path)
			globalHook.WithGoHook(hec.GoHook)

			var yamlConfigBytes []byte

			goConfig := hec.GoHook.Config()
			if goConfig.YamlConfig != "" {
				yamlConfigBytes = []byte(goConfig.YamlConfig)
			}

			if len(yamlConfigBytes) > 0 {
				err = globalHook.WithConfig(yamlConfigBytes)
				if err != nil {
					panic(fmt.Errorf("fail load hook YAML config: %v", err))
				}
			} else if goConfig != nil {
				err := globalHook.WithGoConfig(goConfig)
				if err != nil {
					panic(fmt.Errorf("fail load hook golang config: %v", err))
				}
			}

			hec.BindingContextController.WithHook(&globalHook.Hook)
		}

		if len(hec.KubeExtraCRDs) > 0 {
			for _, crd := range hec.KubeExtraCRDs {
				hec.BindingContextController.RegisterCRD(crd.Group, crd.Version, crd.Kind, crd.Namespaced)
			}
		}

		contexts, err = hec.BindingContextController.Run(newKubeState)
		if err != nil {
			panic(err)
		}
		hec.IsKubeStateInited = true
	} else {
		if desiredQuantity > 0 {
			contexts, err = hec.BindingContextController.ChangeStateAndWaitForBindingContexts(desiredQuantity, newKubeState)
		} else {
			contexts, err = hec.BindingContextController.ChangeState(newKubeState)
		}
		if err != nil {
			panic(err)
		}
	}
	hec.KubeState = newKubeState
	return contexts
}

func (hec *HookExecutionConfig) KubeStateSet(newKubeState string) context.GeneratedBindingContexts {
	return hec.KubeStateSetAndWaitForBindingContexts(newKubeState, 0)
}

func (hec *HookExecutionConfig) RunSchedule(crontab string) context.GeneratedBindingContexts {
	if hec.BindingContextController == nil {
		return ScheduleBindingContext("Empty Schedule")
	}
	contexts, err := hec.BindingContextController.RunSchedule(crontab)
	if err != nil {
		panic(err)
	}
	return contexts
}

func (hec *HookExecutionConfig) KubeStateToKubeObjects() error {
	var err error
	hec.ObjectStore = make(object_store.ObjectStore)
	dec := yamlv3.NewDecoder(strings.NewReader(hec.KubeState))
	for {
		var t interface{}
		err = dec.Decode(&t)

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if t == nil {
			continue
		}

		var unstructuredObj unstructured.Unstructured
		unstructuredObj.SetUnstructuredContent(t.(map[string]interface{}))
		hec.ObjectStore.PutObject(unstructuredObj.Object, object_store.NewMetaIndex(unstructuredObj.GetKind(), unstructuredObj.GetNamespace(), unstructuredObj.GetName()))
	}
	return nil
}

func (hec *HookExecutionConfig) RunHook() {
	if hec.GoHook != nil {
		hec.RunGoHook()
		return
	}

	var (
		err error

		tmpDir string

		ValuesFile                *os.File
		ConfigValuesFile          *os.File
		ValuesJSONPatchFile       *os.File
		ConfigValuesJSONPatchFile *os.File
		BindingContextFile        *os.File
		KubernetesPatchSetFile    *os.File
		MetricsFile               *os.File

		hookEnvs []string
	)

	err = hec.KubeStateToKubeObjects()
	Expect(err).ShouldNot(HaveOccurred())

	hookEnvs = append(hookEnvs, "ADDON_OPERATOR_NAMESPACE=tests", "DECKHOUSE_POD=tests", "D8_IS_TESTS_ENVIRONMENT=yes", "PATH="+os.Getenv("PATH"))
	hookEnvs = append(hookEnvs, hec.extraHookEnvs...)

	hookCmd := &exec.Cmd{
		Path: hec.HookPath,
		Args: []string{hec.HookPath, "--config"},
		Env:  append(os.Environ(), hookEnvs...),
	}

	hec.Session, err = gexec.Start(hookCmd, nil, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())

	hec.Session.Wait(10)
	Expect(hec.Session.ExitCode()).To(Equal(0))

	if os.Getenv("KCOV_DISABLED") != "yes" {
		// let's re-run --config again, but this time with kcov wrapper
		// it is required since kcov quietly eats all the stdout
		kcovConfigCmd := &exec.Cmd{
			Path: hec.HookPath,
			Args: []string{hec.HookPath, "--config"},
			Dir:  "/deckhouse",
			Env:  append(os.Environ(), hookEnvs...),
		}
		sandbox_runner.Run(kcovConfigCmd,
			sandbox_runner.WithKcovWrapper(globalKcovDir),
			sandbox_runner.AsUser(999, 998),
		)
	}

	out := hec.Session.Out.Contents()
	var parsedConfig json.RawMessage
	Expect(yaml.Unmarshal(out, &parsedConfig)).To(Succeed())

	Expect(hec.values.JSONRepr).ToNot(BeEmpty())
	Expect(hec.configValues.JSONRepr).ToNot(BeEmpty())

	By("Validating initial values")
	Expect(values_validation.ValidateValues(hec.ValuesValidator, moduleName, string(hec.values.JSONRepr))).To(Succeed())
	By("Validating initial config values")
	Expect(values_validation.ValidateValues(hec.ValuesValidator, moduleName, string(hec.configValues.JSONRepr))).To(Succeed())

	tmpDir, err = TempDirWithPerms(globalTmpDir, "", 0o777)
	Expect(err).ShouldNot(HaveOccurred())

	ValuesFile, err = TempFileWithPerms(tmpDir, "", 0o777)
	Expect(err).ShouldNot(HaveOccurred())
	hookEnvs = append(hookEnvs, "VALUES_PATH="+ValuesFile.Name())

	ConfigValuesFile, err = TempFileWithPerms(tmpDir, "", 0o777)
	Expect(err).ShouldNot(HaveOccurred())
	hookEnvs = append(hookEnvs, "CONFIG_VALUES_PATH="+ConfigValuesFile.Name())

	ValuesJSONPatchFile, err = TempFileWithPerms(tmpDir, "", 0o777)
	Expect(err).ShouldNot(HaveOccurred())
	hookEnvs = append(hookEnvs, "VALUES_JSON_PATCH_PATH="+ValuesJSONPatchFile.Name())

	ConfigValuesJSONPatchFile, err = TempFileWithPerms(tmpDir, "", 0o777)
	Expect(err).ShouldNot(HaveOccurred())
	hookEnvs = append(hookEnvs, "CONFIG_VALUES_JSON_PATCH_PATH="+ConfigValuesJSONPatchFile.Name())

	BindingContextFile, err = TempFileWithPerms(tmpDir, "", 0o777)
	Expect(err).ShouldNot(HaveOccurred())
	hookEnvs = append(hookEnvs, "BINDING_CONTEXT_PATH="+BindingContextFile.Name())

	KubernetesPatchSetFile, err = TempFileWithPerms(tmpDir, "", 0o777)
	Expect(err).ShouldNot(HaveOccurred())
	hookEnvs = append(hookEnvs, "D8_TEST_KUBERNETES_PATCH_SET_FILE="+KubernetesPatchSetFile.Name())

	MetricsFile, err = TempFileWithPerms(tmpDir, "", 0o777)
	Expect(err).ShouldNot(HaveOccurred())
	hookEnvs = append(hookEnvs, "METRICS_PATH="+MetricsFile.Name())

	hookCmd = &exec.Cmd{
		Path: hec.HookPath,
		Args: []string{hec.HookPath},
		Dir:  "/deckhouse",
		Env:  hookEnvs,
	}

	options := []sandbox_runner.SandboxOption{
		sandbox_runner.WithFile(ValuesFile.Name(), hec.values.JSONRepr),
		sandbox_runner.WithFile(ConfigValuesFile.Name(), hec.configValues.JSONRepr),
		sandbox_runner.WithFile(BindingContextFile.Name(), []byte(hec.BindingContexts.JSON)),
	}
	if os.Getenv("KCOV_DISABLED") != "yes" {
		options = append(options, sandbox_runner.WithKcovWrapper(globalKcovDir))
		options = append(options, sandbox_runner.AsUser(999, 998))
	}

	hec.Session = sandbox_runner.Run(hookCmd, options...)

	valuesJSONPatchBytes, err := ioutil.ReadAll(ValuesJSONPatchFile)
	Expect(err).ShouldNot(HaveOccurred())
	configValuesJSONPatchBytes, err := ioutil.ReadAll(ConfigValuesJSONPatchFile)
	Expect(err).ShouldNot(HaveOccurred())
	kubernetesPatchBytes, err := ioutil.ReadAll(KubernetesPatchSetFile)
	Expect(err).ShouldNot(HaveOccurred())

	// TODO: take a closer look and refactor into a function
	if len(valuesJSONPatchBytes) != 0 {
		patch, err := addonutils.JsonPatchFromBytes(valuesJSONPatchBytes)
		Expect(err).ShouldNot(HaveOccurred())

		patchedValuesBytes, err := patch.Apply(hec.values.JSONRepr)
		Expect(err).ShouldNot(HaveOccurred())
		hec.values = values_store.NewStoreFromRawJSON(patchedValuesBytes)
	}

	if len(configValuesJSONPatchBytes) != 0 {
		patch, err := addonutils.JsonPatchFromBytes(configValuesJSONPatchBytes)
		Expect(err).ShouldNot(HaveOccurred())

		patchedConfigValuesBytes, err := patch.Apply(hec.configValues.JSONRepr)
		Expect(err).ShouldNot(HaveOccurred())
		hec.configValues = values_store.NewStoreFromRawJSON(patchedConfigValuesBytes)
	}

	By("Validating resulting values")
	Expect(values_validation.ValidateValues(hec.ValuesValidator, moduleName, string(hec.values.JSONRepr))).To(Succeed())
	By("Validating resulting config values")
	Expect(values_validation.ValidateValues(hec.ValuesValidator, moduleName, string(hec.configValues.JSONRepr))).To(Succeed())

	if len(kubernetesPatchBytes) != 0 {
		kubePatch, err := NewKubernetesPatch(kubernetesPatchBytes)
		Expect(err).ShouldNot(HaveOccurred())

		patchedObjects, err := kubePatch.Apply(hec.ObjectStore)
		Expect(err).ToNot(HaveOccurred())

		hec.ObjectStore = patchedObjects
		hec.KubernetesResourcePatch = kubePatch
	}
}

func (hec *HookExecutionConfig) RunGoHook() {
	if hec.GoHook == nil {
		return
	}

	var (
		err error
	)

	err = hec.KubeStateToKubeObjects()
	Expect(err).ShouldNot(HaveOccurred())

	Expect(hec.values.JSONRepr).ToNot(BeEmpty())

	Expect(hec.configValues.JSONRepr).ToNot(BeEmpty())

	values, err := addonutils.NewValuesFromBytes(hec.values.JSONRepr)
	Expect(err).ShouldNot(HaveOccurred())

	convigValues, err := addonutils.NewValuesFromBytes(hec.configValues.JSONRepr)
	Expect(err).ShouldNot(HaveOccurred())

	hookInput := &sdk.HookInput{
		BindingContexts: hec.BindingContexts.BindingContexts,
		Values:          values,
		ConfigValues:    convigValues,
		Envs: map[string]string{
			"ADDON_OPERATOR_NAMESPACE": "tests",
			"DECKHOUSE_POD":            "tests",
			"D8_IS_TESTS_ENVIRONMENT":  "yes",
			"PATH":                     os.Getenv("PATH"),
		},
	}

	out, err := hec.GoHook.Run(hookInput)
	Expect(err).ShouldNot(HaveOccurred())

	if out.MemoryValuesPatches != nil {
		patchedValuesBytes, err := out.MemoryValuesPatches.Apply(hec.values.JSONRepr)
		Expect(err).ShouldNot(HaveOccurred())
		hec.values = values_store.NewStoreFromRawJSON(patchedValuesBytes)
	}

	if out.ConfigValuesPatches != nil {
		patchedConfigValuesBytes, err := out.ConfigValuesPatches.Apply(hec.configValues.JSONRepr)
		Expect(err).ShouldNot(HaveOccurred())
		hec.configValues = values_store.NewStoreFromRawJSON(patchedConfigValuesBytes)
	}

	hec.GoHookOut = out

	// TODO Kubernetes patch not supported for Go Hooks now.
	// https://github.com/flant/shell-operator/issues/94
}

var _ = BeforeSuite(func() {
	var err error

	if os.Getenv("KCOV_DISABLED") == "yes" {
		return
	}
	By("Initing temporary directories")
	unix.Umask(0o000)
	globalTmpDir, err = TempDirWithPerms("", "", 0o777)
	Expect(err).ToNot(HaveOccurred())
	_ = os.Mkdir(globalKcovDir, 0o777)

	dummyDirsFile, err := os.Open("/deckhouse/testing/dummy_dirs")
	if err != nil {
		panic(err)
	}

	sc := bufio.NewScanner(dummyDirsFile)
	for sc.Scan() {
		dir := sc.Text()

		cmd := &exec.Cmd{
			Path: filepath.Join(dir, "dummy"),
			Args: []string{filepath.Join(dir, "dummy")},
			Dir:  "/deckhouse",
		}

		res := sandbox_runner.Run(cmd,
			sandbox_runner.WithKcovWrapper(globalKcovDir),
			sandbox_runner.AsUser(999, 998),
		)

		if res.ExitCode() != 0 {
			panic(fmt.Sprintf(
				"Exit %v\nStdout:\n%s\nStderr:\n%s\n",
				res.ExitCode(),
				string(res.Out.Contents()),
				string(res.Err.Contents()),
			))
		}
	}
	if err := sc.Err(); err != nil {
		panic("scan file error: " + err.Error())
	}
})

var _ = AfterSuite(func() {
	By("Removing temporary directories")
	Expect(os.RemoveAll(globalTmpDir)).Should(Succeed())
})
