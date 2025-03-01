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

/*

User-stories:
1. There is Secret kube-system/audit-policy with audit-policy.yaml set in data, hook must store it to `controlPlaneManager.internal.auditPolicy`.

*/

package hooks

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

var _ = Describe("Modules :: controler-plane-manager :: hooks :: audit_policy ::", func() {
	const (
		initValuesString       = `{"controlPlaneManager":{"internal": {}, "apiserver": {"authn": {}, "authz": {}}}}`
		initConfigValuesString = `{"controlPlaneManager":{"apiserver": {"auditPolicyEnabled": false}}}`
		stateA                 = `
apiVersion: v1
kind: Secret
metadata:
  name: audit-policy
  namespace: kube-system
data:
  audit-policy.yaml: YXBpVmVyc2lvbjogYXVkaXQuazhzLmlvL3YxCmtpbmQ6IFBvbGljeQpydWxlczoKLSBsZXZlbDogTWV0YWRhdGEK
`
		stateB = `
apiVersion: v1
kind: Secret
metadata:
  name: audit-policy
  namespace: kube-system
data:
  audit-policy.yaml: YXBpVmVyc2lvbjogYXVkaXQuazhzLmlvL3YxCmtpbmQ6IFBvbGljeQpydWxlczoKLSBsZXZlbDogTWV0YWRhdGEKICBvbWl0U3RhZ2VzOgogICAgLSAiUmVxdWVzdFJlY2VpdmVkIgo=
`
		invalidPolicy = `
apiVersion: v1
kind: Secret
metadata:
  name: audit-policy
  namespace: kube-system
data:
  audit-policy.yaml: YXBpVmVyc2lvbjogYXVkaXQuazhzLmlvL3YxCmtpbmQ6IFBvbGljeQpydWxlczoKICBzb21rZXk6IGludmFsaWRvbmUK
`
	)

	f := HookExecutionConfigInit(initValuesString, initConfigValuesString)

	Context("Empty cluster", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(``))
			f.RunHook()
		})

		It("Must be executed successfully", func() {
			Expect(f).To(ExecuteSuccessfully())
		})

		It("controlPlaneManager.internal.auditPolicy must be empty", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet("controlPlaneManager.internal.auditPolicy").Exists()).To(BeFalse())
		})
	})

	Context("Invalid policy set", func() {
		BeforeEach(func() {
			f.ValuesSet("controlPlaneManager.apiserver.auditPolicyEnabled", true)
			f.BindingContexts.Set(f.KubeStateSet(invalidPolicy))
			f.RunHook()
		})

		It("Must fail on yaml validation", func() {
			Expect(f).To(Not(ExecuteSuccessfully()))
			Expect(f.GoHookError).Should(MatchError("invalid policy.yaml format"))
		})
	})

	Context("Cluster started with stateA Secret and disabled auditPolicy", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateA))
			f.RunHook()
		})

		It("controlPlaneManager.internal.auditPolicy must be empty", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet("controlPlaneManager.internal.auditPolicy").Exists()).To(BeFalse())
		})
	})

	Context("Cluster started with stateA Secret and not set auditPolicyEnabled", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateA))
			f.ConfigValuesDelete("controlPlaneManager.apiserver.auditPolicyEnabled")
			f.RunHook()
		})

		It("controlPlaneManager.internal.auditPolicy must be empty", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet("controlPlaneManager.internal.auditPolicy").Exists()).To(BeFalse())
		})
	})

	Context("Cluster started with stateA Secret", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateA))
			f.ValuesSet("controlPlaneManager.apiserver.auditPolicyEnabled", true)
			f.RunHook()
		})

		It("controlPlaneManager.internal.auditPolicy must be stateA", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet("controlPlaneManager.internal.auditPolicy").String()).To(Equal("YXBpVmVyc2lvbjogYXVkaXQuazhzLmlvL3YxCmtpbmQ6IFBvbGljeQpydWxlczoKLSBsZXZlbDogTWV0YWRhdGEK"))
		})

		Context("Cluster changed to stateB", func() {
			BeforeEach(func() {
				f.BindingContexts.Set(f.KubeStateSet(stateB))
				f.ValuesSet("controlPlaneManager.apiserver.auditPolicyEnabled", true)
				f.RunHook()
			})

			It("controlPlaneManager.internal.auditPolicy must be stateB", func() {
				Expect(f).To(ExecuteSuccessfully())
				Expect(f.ValuesGet("controlPlaneManager.internal.auditPolicy").String()).To(Equal("YXBpVmVyc2lvbjogYXVkaXQuazhzLmlvL3YxCmtpbmQ6IFBvbGljeQpydWxlczoKLSBsZXZlbDogTWV0YWRhdGEKICBvbWl0U3RhZ2VzOgogICAgLSAiUmVxdWVzdFJlY2VpdmVkIgo="))
			})
		})
	})
})
