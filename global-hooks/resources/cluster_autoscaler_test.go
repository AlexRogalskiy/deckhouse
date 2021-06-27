// Copyright 2021 Flant CJSC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*

User-stories:
1. If there is a Deployment kube-system/cluster-autoscaler in cluster, it must not have section `resources.limits` because extended-monitoring will alert at throttling.

*/

package hooks

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "")
}

const (
	initValuesString       = `{}`
	initConfigValuesString = `{}`
)

const (
	stateLimitsAreSet = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-autoscaler
  namespace: kube-system
spec:
  template:
    spec:
      containers:
      - resources:
          requests:
            cpu: 100m
            memory: 300Mi
          limits:
            cpu: 333m
            memory: 333Mi`

	stateLimitsAreUnset = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-autoscaler
  namespace: kube-system
spec:
  template:
    spec:
      containers:
      - resources:
          requests:
            cpu: 100m
            memory: 300Mi`
)

var _ = Describe("Global hooks :: resources/cluster_autoscaler ::", func() {
	f := HookExecutionConfigInit(initValuesString, initConfigValuesString)

	Context("There is no Deployment kube-system/cluster-autoscaler in cluster", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(``))
			f.RunHook()
		})

		It("BINDING_CONTEXT must be empty", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.BindingContexts.Array()).ShouldNot(BeEmpty())
			Expect(f.BindingContexts.Get("0.Objects").Array()).To(BeEmpty())
		})

		Context("Someone created Deployment kube-system/cluster-autoscaler with `spec.template.spec.containers.0.resources.limits`", func() {
			BeforeEach(func() {
				f.BindingContexts.Set(f.KubeStateSet(stateLimitsAreSet))
				f.RunHook()
			})

			It("BINDING_CONTEXT must contain Added event; section `limits` must be deleted", func() {
				Expect(f).To(ExecuteSuccessfully())
				Expect(f.BindingContexts.Array()).ShouldNot(BeEmpty())
				Expect(f.BindingContexts.Get("0.binding").String()).To(Equal("cluster_autoscaler"))
				Expect(f.BindingContexts.Get("0.watchEvent").String()).To(Equal("Added"))
				Expect(f.KubernetesResource("Deployment", "kube-system", "cluster-autoscaler").Field("spec.template.spec.containers.0.resources").Exists()).To(BeTrue())
				Expect(f.KubernetesResource("Deployment", "kube-system", "cluster-autoscaler").Field("spec.template.spec.containers.0.resources.limits").Exists()).To(BeFalse())
			})
		})
	})

	Context("There is Deployment kube-system/cluster-autoscaler in cluster with section `spec.template.spec.containers.0.resources.limits`", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateLimitsAreSet))
			f.RunHook()
		})

		It("BINDING_CONTEXT must contain Synchronization event with cluster-autoscaler Deployment; section `limits` must be deleted", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.BindingContexts.Array()).ShouldNot(BeEmpty())
			Expect(f.BindingContexts.Get("0.binding").String()).To(Equal("cluster_autoscaler"))
			Expect(f.BindingContexts.Get("0.type").String()).To(Equal("Synchronization"))
			Expect(f.BindingContexts.Get("0.objects.0.object.metadata.name").String()).To(Equal("cluster-autoscaler"))
			Expect(f.KubernetesResource("Deployment", "kube-system", "cluster-autoscaler").Field("spec.template.spec.containers.0.resources").Exists()).To(BeTrue())
			Expect(f.KubernetesResource("Deployment", "kube-system", "cluster-autoscaler").Field("spec.template.spec.containers.0.resources.limits").Exists()).To(BeFalse())
		})
	})

	Context("There is Deployment kube-system/cluster-autoscaler in cluster without section `spec.template.spec.containers.0.resources.limits`", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateLimitsAreUnset))
			f.RunHook()
		})

		It("BINDING_CONTEXT must contain Synchronization event with cluster-autoscaler Deployment; section `limits` must be deleted", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.BindingContexts.Array()).ShouldNot(BeEmpty())
			Expect(f.BindingContexts.Get("0.binding").String()).To(Equal("cluster_autoscaler"))
			Expect(f.BindingContexts.Get("0.type").String()).To(Equal("Synchronization"))
			Expect(f.BindingContexts.Get("0.objects.0.object.metadata.name").String()).To(Equal("cluster-autoscaler"))
			Expect(f.KubernetesResource("Deployment", "kube-system", "cluster-autoscaler").Field("spec.template.spec.containers.0.resources").Exists()).To(BeTrue())
			Expect(f.KubernetesResource("Deployment", "kube-system", "cluster-autoscaler").Field("spec.template.spec.containers.0.resources.limits").Exists()).To(BeFalse())
		})

		Context("Someone modified Deployment kube-system/cluster-autoscaler by adding section `spec.template.spec.containers.0.resources.limits`", func() {
			BeforeEach(func() {
				f.BindingContexts.Set(f.KubeStateSet(stateLimitsAreSet))
				f.RunHook()
			})

			It("BINDING_CONTEXT must contain Modified event; section `limits` must be deleted", func() {
				Expect(f).To(ExecuteSuccessfully())
				Expect(f.BindingContexts.Array()).ShouldNot(BeEmpty())
				Expect(f.BindingContexts.Get("0.binding").String()).To(Equal("cluster_autoscaler"))
				Expect(f.BindingContexts.Get("0.watchEvent").String()).To(Equal("Modified"))
				Expect(f.BindingContexts.Get("0.object.metadata.name").String()).To(Equal("cluster-autoscaler"))
				Expect(f.KubernetesResource("Deployment", "kube-system", "cluster-autoscaler").Field("spec.template.spec.containers.0.resources").Exists()).To(BeTrue())
				Expect(f.KubernetesResource("Deployment", "kube-system", "cluster-autoscaler").Field("spec.template.spec.containers.0.resources.limits").Exists()).To(BeFalse())
			})
		})
	})
})
