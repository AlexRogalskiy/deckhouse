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

package hooks

/*
1. No nodes in the cluster;
2. Some nodes exist;
3. There are Pods with control-plane.
*/

import (
	"encoding/base64"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

type input struct {
	nodeVersions               []string
	maxUsedControlPlaneVersion string
	configVersion              string
	controlPlaneVersions       []string
}

type output struct {
	maxUsedControlPlaneVersion string
	effectiveVersion           string
}

func setStateFromTestCase(hec *HookExecutionConfig, caseInput input) {
	const nodeTemplate = `
---
apiVersion: v1
kind: Node
metadata:
  name: test-<<INDEX>>
status:
  nodeInfo:
    kubeletVersion: "<<PLACEHOLDER>>"
`

	const kubeApiserverPodTemplate = `
---
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver-kube-master-<<INDEX>>
  namespace: kube-system
  labels:
    component: kube-apiserver
    tier: control-plane
  annotations:
    control-plane-manager.deckhouse.io/kubernetes-version: "<<PLACEHOLDER>>"
`

	const kubeControllerManagerPodTemplate = `
---
apiVersion: v1
kind: Pod
metadata:
  name: kube-controller-manager-kube-master-<<INDEX>>
  namespace: kube-system
  labels:
    component: kube-controller-manager
    tier: control-plane
  annotations:
    control-plane-manager.deckhouse.io/kubernetes-version: "<<PLACEHOLDER>>"
`

	const kubeSchedulerPodTemplate = `
---
apiVersion: v1
kind: Pod
metadata:
  name: kube-scheduler-kube-master-<<INDEX>>
  namespace: kube-system
  labels:
    component: kube-scheduler
    tier: control-plane
  annotations:
    control-plane-manager.deckhouse.io/kubernetes-version: "<<PLACEHOLDER>>"
`

	const d8ConfigurationSecretTemplate = `
---
apiVersion: v1
kind: Secret
metadata:
  name: d8-cluster-configuration
  namespace: kube-system
data:
  maxUsedControlPlaneKubernetesVersion: "<<PLACEHOLDER_B64>>"
`

	var b strings.Builder

	for index, nodeVersion := range caseInput.nodeVersions {
		nodeManifest := strings.ReplaceAll(nodeTemplate, "<<PLACEHOLDER>>", nodeVersion)
		nodeManifest = strings.ReplaceAll(nodeManifest, "<<INDEX>>", strconv.Itoa(index))

		b.WriteString(nodeManifest)
	}

	for index, controlPlaneVersion := range caseInput.controlPlaneVersions {
		kubeApiserverManifest := strings.ReplaceAll(kubeApiserverPodTemplate, "<<PLACEHOLDER>>", controlPlaneVersion)
		kubeApiserverManifest = strings.ReplaceAll(kubeApiserverManifest, "<<INDEX>>", strconv.Itoa(index))
		b.WriteString(kubeApiserverManifest)

		kubeControllerManager := strings.ReplaceAll(kubeControllerManagerPodTemplate, "<<PLACEHOLDER>>", controlPlaneVersion)
		kubeControllerManager = strings.ReplaceAll(kubeControllerManager, "<<INDEX>>", strconv.Itoa(index))
		b.WriteString(kubeControllerManager)

		kubeSchedulerManifest := strings.ReplaceAll(kubeSchedulerPodTemplate, "<<PLACEHOLDER>>", controlPlaneVersion)
		kubeSchedulerManifest = strings.ReplaceAll(kubeSchedulerManifest, "<<INDEX>>", strconv.Itoa(index))
		b.WriteString(kubeSchedulerManifest)
	}

	b.WriteString(strings.ReplaceAll(d8ConfigurationSecretTemplate, "<<PLACEHOLDER_B64>>", base64.StdEncoding.EncodeToString([]byte(caseInput.maxUsedControlPlaneVersion))))

	hec.ValuesSet("global.clusterConfiguration.kubernetesVersion", caseInput.configVersion)
	hec.BindingContexts.Set(hec.KubeStateSet(b.String()))
}

var _ = Describe("Modules :: controler-plane-manager :: hooks :: get_pki_checksum ::", func() {
	Context("Empty cluster", func() {
		f := HookExecutionConfigInit(`{"controlPlaneManager":{"internal": {}}}`, `{}`)

		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(``))
			f.RunHook()
		})

		It("Hook must fail", func() {
			Expect(f).NotTo(ExecuteSuccessfully())
			Expect(f.GoHookError.Error()).To(BeEquivalentTo("global.clusterConfiguration.kubernetesVersion required"))
		})
	})

	Context("Empty cluster", func() {
		f := HookExecutionConfigInit(`{"controlPlaneManager":{"internal": {}}}`, `{}`)

		DescribeTable("version change",
			func(in input, out output) {
				setStateFromTestCase(f, in)
				f.RunHook()

				Expect(f).To(ExecuteSuccessfully())

				d8ClusterConfigSecret := f.KubernetesResource("Secret", "kube-system", "d8-cluster-configuration")
				decodedMaxUsedKubernetesVersion, err := base64.StdEncoding.DecodeString(d8ClusterConfigSecret.Field("data.maxUsedControlPlaneKubernetesVersion").String())
				Expect(err).To(BeNil())
				Expect(string(decodedMaxUsedKubernetesVersion)).To(Equal(out.maxUsedControlPlaneVersion))

				Expect(f.ValuesGet("controlPlaneManager.internal.effectiveKubernetesVersion").String()).To(Equal(out.effectiveVersion))
			},
			Entry("upgrade: Node version lower than control plane, do not allow to bump effective version and max used version",
				input{
					nodeVersions:               []string{"v1.19.3", "v1.19.1", "v1.19.5", "v1.20.2"},
					maxUsedControlPlaneVersion: "1.20",
					configVersion:              "1.21",
					controlPlaneVersions:       []string{"1.20", "1.20", "1.20"},
				},
				output{
					maxUsedControlPlaneVersion: "1.20",
					effectiveVersion:           "1.20",
				},
			),
			Entry("upgrade: control plane and nodes are on the same version, allow bumping effective version and max used version", input{
				nodeVersions:               []string{"v1.20.10", "v1.20.3", "v1.20.5", "v1.20.2"},
				maxUsedControlPlaneVersion: "1.20",
				configVersion:              "1.21",
				controlPlaneVersions:       []string{"1.20", "1.20", "1.20"},
			},
				output{
					maxUsedControlPlaneVersion: "1.21",
					effectiveVersion:           "1.21",
				},
			),
			Entry("upgrade: control plane and nodes are on the same version (but kube-scheduler is on a lower version), do not bump effective version and max used version",
				input{
					nodeVersions:               []string{"v1.20.10", "v1.20.3", "v1.20.5", "v1.20.2"},
					maxUsedControlPlaneVersion: "1.20",
					configVersion:              "1.21",
					controlPlaneVersions:       []string{"1.20", "1.20", "1.19"},
				},
				output{
					maxUsedControlPlaneVersion: "1.20",
					effectiveVersion:           "1.20",
				},
			),
			Entry("downgrade: control plane and nodes are on the same version, do not lower effective version",
				input{
					nodeVersions:               []string{"v1.20.10", "v1.20.3", "v1.20.5", "v1.20.2"},
					maxUsedControlPlaneVersion: "1.20",
					configVersion:              "1.19",
					controlPlaneVersions:       []string{"1.20", "1.20", "1.20"},
				},
				output{
					maxUsedControlPlaneVersion: "1.20",
					effectiveVersion:           "1.20",
				},
			),
			Entry("downgrade: nodes are downgraded already, lower effective version",
				input{
					nodeVersions:               []string{"v1.20.10", "v1.20.3", "v1.20.5", "v1.20.2"},
					maxUsedControlPlaneVersion: "1.21",
					configVersion:              "1.19",
					controlPlaneVersions:       []string{"1.21", "1.21", "1.21"},
				},
				output{
					maxUsedControlPlaneVersion: "1.21",
					effectiveVersion:           "1.20",
				},
			),
			Entry("downgrade: nodes are downgraded already, but configVersion is 2 minor versions lower, lower effective version by one",
				input{
					nodeVersions:               []string{"v1.20.10", "v1.20.3", "v1.20.5", "v1.20.2"},
					maxUsedControlPlaneVersion: "1.21",
					configVersion:              "1.19",
					controlPlaneVersions:       []string{"1.21", "1.21", "1.21"},
				},
				output{
					maxUsedControlPlaneVersion: "1.21",
					effectiveVersion:           "1.20",
				},
			),
			Entry("downgrade: nodes are downgraded already, but maxUsedControlPlaneVersion does not allow us to downgrade by more than 1",
				input{
					nodeVersions:               []string{"v1.19.4", "v1.19.3", "v1.19.5", "v1.19.2"},
					maxUsedControlPlaneVersion: "1.21",
					configVersion:              "1.19",
					controlPlaneVersions:       []string{"1.20", "1.20", "1.20"},
				},
				output{
					maxUsedControlPlaneVersion: "1.21",
					effectiveVersion:           "1.20",
				},
			),
			Entry("downgrade: nodes are downgraded already, maxUsedControlPlaneVersion does not allow us to downgrade by more than 1, but we already violating maxUsedControlPlaneVersion",
				input{
					nodeVersions:               []string{"v1.19.4", "v1.19.3", "v1.19.5", "v1.19.2"},
					maxUsedControlPlaneVersion: "1.22",
					configVersion:              "1.19",
					controlPlaneVersions:       []string{"1.20", "1.20", "1.20"},
				},
				output{
					maxUsedControlPlaneVersion: "1.22",
					effectiveVersion:           "1.20",
				},
			),
		)
	})
})
