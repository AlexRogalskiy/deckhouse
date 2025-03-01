/*
Copyright 2021 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package template

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"sort"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

type ContextBuilder struct {
	ctx context.Context

	stepsDir       string
	stepsStorage   *StepsStorage
	ngStepsStorage *StepsStorage

	registryData     registry
	clusterInputData inputData
	versionMap       map[string]interface{}
	imagesTags       map[string]map[string]string // module: { image_name: tag }

	// debug function injection
	emitStepsOutput func(string, string, map[string]string)
}

func NewContextBuilder(ctx context.Context, stepsTemplatesRootDir string) *ContextBuilder {
	cb := &ContextBuilder{
		ctx:      ctx,
		stepsDir: stepsTemplatesRootDir,
	}

	return cb
}

type bashibleContexts map[string]interface{}

type BashibleContextData struct {
	bashibleContexts // rendered contexts

	Images     map[string]map[string]string `json:"images" yaml:"images"`
	VersionMap map[string]interface{}       `json:"versionMap" yaml:"versionMap"`
	Registry   registry                     `json:"registry" yaml:"registry"`
}

// Map returns context data as map[string]interface{} for proper yaml marshalling
func (bd BashibleContextData) Map() map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range bd.bashibleContexts {
		result[key] = value
	}

	result["images"] = bd.Images
	result["registry"] = bd.Registry
	result["versionMap"] = bd.VersionMap

	return result
}

func (cb *ContextBuilder) SetRegistryData(rd registry) {
	cb.registryData = rd
}

func (cb *ContextBuilder) SetImagesData(data map[string]map[string]string) {
	cb.imagesTags = data
}

func (cb *ContextBuilder) SetVersionMapData(data map[string]interface{}) {
	cb.versionMap = data
}

func (cb *ContextBuilder) SetInputData(data inputData) {
	sort.Strings(data.AllowedBundles)
	sort.Strings(data.AllowedKubernetesVersions)
	sort.Slice(data.NodeGroups, func(i, j int) bool {
		ngi := data.NodeGroups[i]
		ngj := data.NodeGroups[j]

		return ngi.Name() < ngj.Name()
	})
	cb.clusterInputData = data
}

func (cb *ContextBuilder) setStepsOutput(f func(string, string, map[string]string)) {
	cb.emitStepsOutput = f
}

func (cb *ContextBuilder) getCloudProvider() string {
	if cb.clusterInputData.CloudProvider == nil {
		// absent cloud provider means static nodes
		return ""
	}

	cloudProvider, ok := cb.clusterInputData.CloudProvider.(map[string]interface{})
	if !ok {
		return ""
	}
	providerType, ok := cloudProvider["type"].(string)
	if !ok {
		return ""
	}
	return providerType
}

func (cb *ContextBuilder) Build() (BashibleContextData, map[string][]byte, map[string]error) {
	bb := BashibleContextData{
		bashibleContexts: make(map[string]interface{}),
		VersionMap:       cb.versionMap,
		Images:           cb.imagesTags,
		Registry:         cb.registryData,
	}

	ngMap := make(map[string][]byte)
	errorsMap := make(map[string]error)
	hashMap := make(map[string]hash.Hash, len(cb.clusterInputData.NodeGroups))

	cb.stepsStorage = NewStepsStorage(cb.stepsDir, cb.getCloudProvider(), "all")
	cb.ngStepsStorage = NewStepsStorage(cb.stepsDir, cb.getCloudProvider(), "node-group")
	commonContext := &tplContextCommon{
		versionMapWrapper: versionMapFromMap(cb.versionMap),
		RunType:           "Normal",
		// Template:          cb.clusterInputData.Template,
		Normal: normal{
			BootstrapTokenPath: "/var/lib/bashible/bootstrap-token",
			ClusterDomain:      cb.clusterInputData.ClusterDomain,
			ClusterDNSAddress:  cb.clusterInputData.ClusterDNSAddress,
			ApiserverEndpoints: cb.clusterInputData.APIServerEndpoints,
			KubernetesCA:       cb.clusterInputData.KubernetesCA,
		},
		Registry:      cb.registryData,
		Images:        cb.imagesTags,
		PackagesProxy: cb.clusterInputData.PackagesProxy,
	}

	for _, bundle := range cb.clusterInputData.AllowedBundles {
		for _, ng := range cb.clusterInputData.NodeGroups {
			checksumCollector, ok := hashMap[ng.Name()]
			if !ok {
				checksumCollector = sha256.New()
				hashMap[ng.Name()] = checksumCollector
			}
			bundleContextName := fmt.Sprintf("bundle-%s-%s", bundle, ng.Name())
			bashibleContextName := fmt.Sprintf("bashible-%s-%s", bundle, ng.Name())
			bundleNgContext := cb.newBundleNGContext(ng, cb.clusterInputData.Freq, bundle, cb.clusterInputData.CloudProvider, commonContext)
			bb.bashibleContexts[bundleContextName] = bundleNgContext

			bashibleContext, err := cb.newBashibleContext(checksumCollector, bundle, ng, cb.clusterInputData.APIServerEndpoints, cb.versionMap, &bundleNgContext)
			if err != nil {
				errorsMap[bundleContextName] = err
			}
			// bashibleContext always exists. Err is only for checksum generation
			bb.bashibleContexts[bashibleContextName] = bashibleContext
		}

		for _, k8sVersion := range cb.clusterInputData.AllowedKubernetesVersions {
			k8sContextName := fmt.Sprintf("bundle-%s-%s", bundle, k8sVersion)
			bb.bashibleContexts[k8sContextName] = bundleK8sVersionContext{
				tplContextCommon:  commonContext,
				Bundle:            bundle,
				KubernetesVersion: k8sVersion,
				CloudProvider:     cb.clusterInputData.CloudProvider,
			}
		}
	}

	for _, bundle := range cb.clusterInputData.AllowedBundles {
		for _, ng := range cb.clusterInputData.NodeGroups {
			bashibleContextName := fmt.Sprintf("bashible-%s-%s", bundle, ng.Name())
			checksumCollector := hashMap[ng.Name()]
			checksum := fmt.Sprintf("%x", checksumCollector.Sum(nil))
			bcr := bb.bashibleContexts[bashibleContextName]
			bc := bcr.(bashibleContext)
			bc.ConfigurationChecksum = checksum
			bb.bashibleContexts[bashibleContextName] = bc
			ngMap[ng.Name()] = []byte(checksum)
		}
	}

	return bb, ngMap, errorsMap
}

func (cb *ContextBuilder) newBashibleContext(checksumCollector hash.Hash, bundle string, ng nodeGroup, clusterMasterAddresses []string, versionMap map[string]interface{}, bundleNgContext *bundleNGContext) (bashibleContext, error) {
	bc := bashibleContext{
		KubernetesVersion: ng.KubernetesVersion(),
		Bundle:            bundle,
		Normal:            map[string][]string{"apiserverEndpoints": clusterMasterAddresses},
		NodeGroup:         ng,
		// Template:          template,
		RunType: "Normal",

		Images:   cb.imagesTags,
		Registry: &cb.registryData,
	}

	err := cb.generateBashibleChecksum(checksumCollector, bc, bundleNgContext, versionMap)
	if err != nil {
		return bc, errors.Wrap(err, "checksum calc failed")
	}

	return bc, nil
}

func (cb *ContextBuilder) generateBashibleChecksum(checksumCollector hash.Hash, bc bashibleContext, bundleNgContext *bundleNGContext, versionMap map[string]interface{}) error {
	bcData, err := yaml.Marshal(bc)
	if err != nil {
		return errors.Wrap(err, "marshal bashibleContext failed")
	}

	checksumCollector.Write(bcData)

	var res map[string]interface{}

	err = yaml.Unmarshal(bcData, &res)
	if err != nil {
		return errors.Wrap(err, "unmarshal bashibleContext failed")
	}

	// enrich context map with version-map data
	for k, v := range versionMap {
		res[k] = v
	}

	// render steps
	steps, err := cb.stepsStorage.Render(res)
	if err != nil {
		return errors.Wrap(err, "steps render failed")
	}

	if cb.emitStepsOutput != nil {
		cb.emitStepsOutput(bc.Bundle, bc.NodeGroup.Name(), steps)
	}

	stepsData, err := yaml.Marshal(steps)
	if err != nil {
		return errors.Wrap(err, "marshal steps failed")
	}

	checksumCollector.Write(stepsData)

	var bundleRes map[string]interface{}

	bundleData, err := yaml.Marshal(bundleNgContext)
	if err != nil {
		return errors.Wrap(err, "marshal bundleNGContext failed")
	}

	err = yaml.Unmarshal(bundleData, &bundleRes)
	if err != nil {
		return errors.Wrap(err, "unmarshal bundleNGContext failed")
	}

	// render ng steps
	ngSteps, err := cb.ngStepsStorage.Render(bundleRes)
	if err != nil {
		return errors.Wrap(err, "NG steps render failed")
	}

	if cb.emitStepsOutput != nil {
		cb.emitStepsOutput(bc.Bundle, bc.NodeGroup.Name(), ngSteps)
	}

	ngStepsData, err := yaml.Marshal(ngSteps)
	if err != nil {
		return errors.Wrap(err, "marshal NG steps failed")
	}

	checksumCollector.Write(ngStepsData)

	return nil
}

func (cb *ContextBuilder) newBundleNGContext(ng nodeGroup, freq interface{}, bundle string, cloudProvider interface{}, contextCommon *tplContextCommon) bundleNGContext {
	return bundleNGContext{
		tplContextCommon:          contextCommon,
		Bundle:                    bundle,
		KubernetesVersion:         ng.KubernetesVersion(),
		CRI:                       ng.CRIType(),
		NodeGroup:                 ng,
		CloudProvider:             cloudProvider,
		NodeStatusUpdateFrequency: freq,
	}
}

func (rid *registryInputData) FromMap(m map[string][]byte) {
	if v, ok := m["address"]; ok {
		rid.Address = string(v)
	}
	if v, ok := m["path"]; ok {
		rid.Path = string(v)
	}

	if v, ok := m["scheme"]; ok {
		rid.Scheme = string(v)
	}

	if v, ok := m["ca"]; ok {
		rid.CA = string(v)
	}

	if v, ok := m[".dockerconfigjson"]; ok {
		rid.DockerConfig = v
	}
}

func (rid registryInputData) toRegistry() registry {
	var auth string

	if len(rid.DockerConfig) > 0 {
		var dcfg dockerCfg
		err := json.Unmarshal(rid.DockerConfig, &dcfg)
		if err != nil {
			panic(err)
		}

		if registry, ok := dcfg.Auths[rid.Address]; ok {
			auth = registry.Auth
		}
	}

	return registry{
		Address:   rid.Address,
		Path:      rid.Path,
		Scheme:    rid.Scheme,
		CA:        rid.CA,
		DockerCFG: rid.DockerConfig,
		Auth:      auth,
	}
}

func versionMapFromMap(m map[string]interface{}) versionMapWrapper {
	var res versionMapWrapper

	if v, ok := m["bashible"]; ok {
		res.Bashbile = v.(map[string]interface{})
	}

	if v, ok := m["k8s"]; ok {
		res.K8s = v.(map[string]interface{})
	}

	return res
}

type nodeGroup map[string]interface{}

func (ng nodeGroup) Name() string {
	if name, ok := ng["name"]; ok {
		return name.(string)
	}

	return ""
}

func (ng nodeGroup) KubernetesVersion() string {
	if kv, ok := ng["kubernetesVersion"]; ok {
		return kv.(string)
	}

	return ""
}

func (ng nodeGroup) CRIType() string {
	if cri, ok := ng["cri"]; ok {
		if typ, ok := cri.(map[string]interface{})["type"]; ok {
			return typ.(string)
		}
	}

	return ""
}

type bashibleContext struct {
	ConfigurationChecksum string      `json:"configurationChecksum" yaml:"configurationChecksum"`
	KubernetesVersion     string      `json:"kubernetesVersion" yaml:"kubernetesVersion"`
	Bundle                string      `json:"bundle" yaml:"bundle"`
	Normal                interface{} `json:"normal" yaml:"normal"`
	NodeGroup             nodeGroup   `json:"nodeGroup" yaml:"nodeGroup"`
	// Template              interface{} `json:"Template" yaml:"Template"`
	RunType string `json:"runType" yaml:"runType"` // Normal

	// Enrich with images and registry
	Images   map[string]map[string]string `json:"images" yaml:"images"`
	Registry *registry                    `json:"registry" yaml:"registry"`
}

// for appropriate marshalling
type versionMapWrapper struct {
	Bashbile map[string]interface{} `json:"bashible" yaml:"bashible"`
	K8s      map[string]interface{} `json:"k8s" yaml:"k8s"`
}

type tplContextCommon struct {
	versionMapWrapper

	RunType string `json:"runType" yaml:"runType"`
	// Template interface{} `json:"Template" yaml:"Template"`
	Normal normal `json:"normal" yaml:"normal"`

	Images   map[string]map[string]string `json:"images" yaml:"images"`
	Registry registry                     `json:"registry" yaml:"registry"`

	PackagesProxy interface{} `json:"packagesProxy,omitempty" yaml:"packagesProxy,omitempty"`
}

type bundleNGContext struct {
	*tplContextCommon

	Bundle            string    `json:"bundle" yaml:"bundle"`
	KubernetesVersion string    `json:"kubernetesVersion" yaml:"kubernetesVersion"`
	CRI               string    `json:"cri" yaml:"cri"`
	NodeGroup         nodeGroup `json:"nodeGroup" yaml:"nodeGroup"`

	CloudProvider interface{} `json:"cloudProvider,omitempty" yaml:"cloudProvider,omitempty"`

	NodeStatusUpdateFrequency interface{} `json:"nodeStatusUpdateFrequency,omitempty" yaml:"nodeStatusUpdateFrequency,omitempty"`
}

type bundleK8sVersionContext struct {
	*tplContextCommon

	Bundle            string `json:"bundle" yaml:"bundle"`
	KubernetesVersion string `json:"kubernetesVersion" yaml:"kubernetesVersion"`

	CloudProvider interface{} `json:"cloudProvider,omitempty" yaml:"cloudProvider,omitempty"`
}

type normal struct {
	BootstrapTokenPath string   `json:"bootstrapTokenPath" yaml:"bootstrapTokenPath"`
	ClusterDomain      string   `json:"clusterDomain" yaml:"clusterDomain"`
	ClusterDNSAddress  string   `json:"clusterDNSAddress" yaml:"clusterDNSAddress"`
	ApiserverEndpoints []string `json:"apiserverEndpoints" yaml:"apiserverEndpoints"`
	KubernetesCA       string   `json:"kubernetesCA" yaml:"kubernetesCA"`
}

type registry struct {
	Address   string `json:"address" yaml:"address"`
	Path      string `json:"path" yaml:"path"`
	Scheme    string `json:"scheme" yaml:"scheme"`
	CA        string `json:"ca,omitempty" yaml:"ca,omitempty"`
	DockerCFG []byte `json:"dockerCfg" yaml:"dockerCfg"`
	Auth      string `json:"auth" yaml:"auth"`
}

// input from secret
type registryInputData struct {
	Address      string `json:"address" yaml:"address"`
	Path         string `json:"path" yaml:"path"`
	Scheme       string `json:"scheme" yaml:"scheme"`
	CA           string `json:"ca,omitempty" yaml:"ca,omitempty"`
	DockerConfig []byte `json:".dockerconfigjson" yaml:".dockerconfigjson"`
}

type dockerCfg struct {
	Auths map[string]struct {
		Auth string `json:"auth"`
	} `json:"auths"`
}

type inputData struct {
	ClusterDomain      string      `json:"clusterDomain" yaml:"clusterDomain"`
	ClusterDNSAddress  string      `json:"clusterDNSAddress" yaml:"clusterDNSAddress"`
	CloudProvider      interface{} `json:"cloudProvider,omitempty" yaml:"cloudProvider,omitempty"`
	PackagesProxy      interface{} `json:"packagesProxy,omitempty" yaml:"packagesProxy,omitempty"`
	APIServerEndpoints []string    `json:"apiserverEndpoints" yaml:"apiserverEndpoints"`
	KubernetesCA       string      `json:"kubernetesCA" yaml:"kubernetesCA"`
	// Template                  interface{} `json:"Template" yaml:"Template"`
	AllowedBundles            []string    `json:"allowedBundles" yaml:"allowedBundles"`
	AllowedKubernetesVersions []string    `json:"allowedKubernetesVersions" yaml:"allowedKubernetesVersions"`
	NodeGroups                []nodeGroup `json:"nodeGroups" yaml:"nodeGroups"`
	Freq                      interface{} `json:"NodeStatusUpdateFrequency,omitempty" yaml:"NodeStatusUpdateFrequency,omitempty"`
}
