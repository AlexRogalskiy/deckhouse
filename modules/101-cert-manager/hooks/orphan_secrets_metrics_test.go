/*
Copyright 2021 Flant CJSC

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

var _ = Describe("Modules :: cert-manager :: hooks :: orphan_secrets_metrics ::", func() {
	const (
		stateCertificates = `
---
apiVersion: certmanager.k8s.io/v1alpha1
kind: Certificate
metadata:
  annotations:
    meta.helm.sh/release-name: dashboard
    meta.helm.sh/release-namespace: d8-system
  labels:
    app: dashboard
    app.kubernetes.io/managed-by: Helm
    heritage: deckhouse
    module: dashboard
  name: dashboard
  namespace: d8-dashboard
spec:
  acme:
    config:
    - domains:
      - dashboard.test
      http01:
        ingressClass: nginx
  dnsNames:
  - dashboard.test
  issuerRef:
    kind: ClusterIssuer
    name: letsencrypt
  secretName: ingress-tls
`
		stateSecrets = `
---
apiVersion: v1
data:
  ca.crt: ""
  tls.crt: LS0tLS1C
  tls.key: LS0tLS1C
kind: Secret
metadata:
  annotations:
    certmanager.k8s.io/alt-names: dashboard.test
    certmanager.k8s.io/certificate-name: dashboard
    certmanager.k8s.io/common-name: dashboard.test
    certmanager.k8s.io/ip-sans: ""
    certmanager.k8s.io/issuer-kind: ClusterIssuer
    certmanager.k8s.io/issuer-name: letsencrypt
  labels:
    certmanager.k8s.io/certificate-name: dashboard
  name: ingress-tls
  namespace: d8-dashboard
type: kubernetes.io/tls
`
	)

	f := HookExecutionConfigInit(`{}`, `{}`)
	f.RegisterCRD("certmanager.k8s.io", "v1alpha1", "Certificate", true)

	Context("Empty cluster", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(``))
			f.RunHook()
		})

		It("Hook must not fail", func() {
			Expect(f).To(ExecuteSuccessfully())
		})
	})
	Context("Secret in cluster, Certificate not in cluster", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateSecrets))
			f.RunHook()
		})

		It("Hook must not fail", func() {
			Expect(f).To(ExecuteSuccessfully())
		})
	})

	Context("Secret in cluster, Certificate in cluster", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateCertificates + stateSecrets))
			f.RunHook()
		})

		It("Hook must not fail", func() {
			Expect(f).To(ExecuteSuccessfully())
		})
	})

})
