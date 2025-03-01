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

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

var _ = Describe("Modules :: node-manager :: hooks :: set_instance_prefix ::", func() {
	f := HookExecutionConfigInit(`
global:
  clusterConfiguration:
    spec:
      cloud: {}
nodeManager:
  internal: {}
`, `{}`)

	Context("BeforeHelm — nodeManager.instancePrefix isn't set", func() {
		BeforeEach(func() {
			f.ValuesSet("global.clusterConfiguration.cloud.prefix", "global")
			f.BindingContexts.Set(f.GenerateBeforeHelmContext())
			f.RunHook()
		})

		It("Hook must not fail and nodeManager.internal.instancePrefix is 'global'", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet("nodeManager.internal.instancePrefix").String()).To(Equal("global"))
		})
	})

	Context("BeforeHelm — nodeManager.instancePrefix is 'kube'", func() {
		BeforeEach(func() {
			f.ValuesSet("nodeManager.instancePrefix", "kube")
			f.BindingContexts.Set(f.GenerateBeforeHelmContext())
			f.RunHook()
		})

		It(`Hook must not fail and nodeManager.internal.instancePrefix must be 'kube'`, func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet("nodeManager.internal.instancePrefix").String()).To(Equal("kube"))
		})
	})

})
