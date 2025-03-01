---
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver
  namespace: kube-system
  annotations:
    control-plane-manager.deckhouse.io/kubernetes-version: {{ $.clusterConfiguration.kubernetesVersion | quote }}
{{- if hasKey . "images" }}
  {{- if hasKey $.images "kube-apiserver" }}
---
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver
  namespace: kube-system
spec:
  containers:
    - name: kube-apiserver
      image: {{ pluck "kube-apiserver" $.images | first }}
  {{- end }}
{{- end }}
{{- $millicpu := $.resourcesRequestsMilliCpuControlPlane | default 512 -}}
{{- $memory := $.resourcesRequestsMemoryControlPlane | default 536870912 }}
---
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver
  namespace: kube-system
spec:
  containers:
    - name: kube-apiserver
      resources:
        requests:
          cpu: "{{ div (mul $millicpu 33) 100 }}m"
          memory: "{{ div (mul $memory 33) 100 }}"
---
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver
  namespace: kube-system
spec:
  dnsPolicy: ClusterFirstWithHostNet
{{- if $.apiserver.oidcIssuerAddress }}
  {{- if $.apiserver.oidcIssuerURL }}
---
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver
  namespace: kube-system
spec:
  hostAliases:
  - ip: {{ $.apiserver.oidcIssuerAddress }}
    hostnames:
    - {{ trimSuffix "/" (trimPrefix "https://" $.apiserver.oidcIssuerURL) }}
  {{- end }}
{{- end }}

{{- if hasKey . "images" }}
  {{- if hasKey $.images "kube-apiserver-healthcheck" }}
---
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver
  namespace: kube-system
spec:
  containers:
  - name: kube-apiserver
    readinessProbe:
      httpGet:
    {{- if hasKey . "nodeIP" }}
        host: {{ .nodeIP | quote }}
    {{- end }}
        path: /healthz
        port: 3990
        scheme: HTTP
    livenessProbe:
      httpGet:
    {{- if hasKey . "nodeIP" }}
        host: {{ .nodeIP | quote }}
    {{- end }}
        path: /livez
        port: 3990
        scheme: HTTP
    startupProbe:
      httpGet:
    {{- if hasKey . "nodeIP" }}
        host: {{ .nodeIP | quote }}
    {{- end }}
        path: /livez
        port: 3990
        scheme: HTTP
  - name: healthcheck
    image: {{ pluck "kube-apiserver-healthcheck" $.images | first }}
    resources:
      requests:
        cpu: "{{ div (mul $millicpu 2) 100 }}m"
        memory: "{{ div (mul $memory 2) 100 }}"
    livenessProbe:
      httpGet:
        path: /.kube-apiserver-healthcheck/healthz
        port: 3990
    {{- if hasKey . "nodeIP" }}
        host: {{ .nodeIP | quote }}
    {{- end }}
      initialDelaySeconds: 5
      timeoutSeconds: 5
    command:
    - /usr/local/bin/kube-apiserver-healthcheck
    args:
    - --ca-cert=/secrets/ca.crt
    - --client-cert=/secrets/client.crt
    - --client-key=/secrets/client.key
    {{- if hasKey . "nodeIP" }}
    - --listen-address={{ .nodeIP }}
    {{- end }}
    - --listen-port=3990
    {{- if hasKey . "nodeIP" }}
    - --api-server-address={{ .nodeIP }}
    {{- end }}
    - --api-server-port=6443
    volumeMounts:
    - name: healthcheck-secrets-ca
      mountPath: /secrets/ca.crt
      readOnly: true
    - name: healthcheck-secrets-client-crt
      mountPath: /secrets/client.crt
      readOnly: true
    - name: healthcheck-secrets-client-key
      mountPath: /secrets/client.key
      readOnly: true
  volumes:
  - name: healthcheck-secrets-ca
    hostPath:
      path: /etc/kubernetes/pki/ca.crt
      type: File
  - name: healthcheck-secrets-client-crt
    hostPath:
      path: /etc/kubernetes/pki/apiserver-kubelet-client.crt
      type: File
  - name: healthcheck-secrets-client-key
    hostPath:
      path: /etc/kubernetes/pki/apiserver-kubelet-client.key
      type: File
  {{- end }}
{{- end }}
