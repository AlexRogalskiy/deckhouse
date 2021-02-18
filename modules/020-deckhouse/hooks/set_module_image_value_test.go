package hooks

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

var _ = Describe("Modules :: deckhouse :: hooks :: set module image value ::", func() {
	f := HookExecutionConfigInit(`
global:
  deckhouseVersion: "12345"
  modulesImages:
    registry: registry.flant.com/sys/antiopa
deckhouse:
  internal:
    currentReleaseImageName: "test"
`, `{}`)

	Context("With Deckhouse pod", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSetAndWaitForBindingContexts(`
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: deckhouse
    heritage: deckhouse
    module: deckhouse
  name: deckhouse
  namespace: d8-system
spec:
  template:
    spec:
      containers:
      - name: deckhouse
        image: registry.flant.com/sys/antiopa/dev:test
`, 1))
			f.RunHook()
		})

		It("Should run", func() {
			Expect(f).To(ExecuteSuccessfully())
			deployment := f.KubernetesResource("Deployment", "d8-system", "deckhouse")
			Expect(deployment.Exists()).To(BeTrue())
			Expect(f.ValuesGet("deckhouse.internal.currentReleaseImageName").String()).To(Equal("registry.flant.com/sys/antiopa/dev:test"))
		})

	})

})
