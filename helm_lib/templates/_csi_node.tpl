{{- /* Usage: {{ include "helm_lib_csi_node_manifests" (list . $config) }} */ -}}
{{- define "helm_lib_csi_node_manifests" }}
  {{- $context := index . 0 }}

  {{- $config := index . 1 }}
  {{- $fullname := $config.fullname | default "csi-node" }}
  {{- $nodeImage := $config.nodeImage | required "$config.nodeImage is required" }}
  {{- $driverFQDN := $config.driverFQDN | required "$config.driverFQDN is required" }}
  {{- $serviceAccount := $config.serviceAccount | default "" }}
  {{- $additionalNodeEnvs := $config.additionalNodeEnvs }}
  {{- $additionalNodeArgs := $config.additionalNodeArgs }}
  {{- $additionalNodeVolumes := $config.additionalNodeVolumes }}
  {{- $additionalNodeVolumeMounts := $config.additionalNodeVolumeMounts }}

  {{- $kubernetesSemVer := semver $context.Values.global.discovery.kubernetesVersion }}

  {{- $driverRegistrarImageName := join "" (list "csiNodeDriverRegistrar" $kubernetesSemVer.Major $kubernetesSemVer.Minor) }}
  {{- $driverRegistrarImageTag := index $context.Values.global.modulesImages.tags.common $driverRegistrarImageName }}
  {{- $driverRegistrarImage := printf "%s:%s" $context.Values.global.modulesImages.registry $driverRegistrarImageTag }}

  {{- if $driverRegistrarImageTag }}
    {{- if (include "helm_lib_cluster_has_non_static_nodes" $context) }}
      {{- if ($context.Values.global.enabledModules | has "vertical-pod-autoscaler-crd") }}
---
apiVersion: autoscaling.k8s.io/v1beta2
kind: VerticalPodAutoscaler
metadata:
  name: {{ $fullname }}
  namespace: d8-{{ $context.Chart.Name }}
{{ include "helm_lib_module_labels" (list $context (dict "app" "csi-node" "workload-resource-policy.deckhouse.io" "every-node")) | indent 2 }}
spec:
  targetRef:
    apiVersion: "apps/v1"
    kind: DaemonSet
    name: {{ $fullname }}
  updatePolicy:
    updateMode: "Auto"
    {{- end }}
---
apiVersion: policy/v1beta1
kind: PodDisruptionBudget
metadata:
  name: {{ $fullname }}
  namespace: d8-{{ $context.Chart.Name }}
{{ include "helm_lib_module_labels" (list $context (dict "app" "csi-node")) | indent 2 }}
spec:
{{ include "helm_lib_pdb_daemonset" $context | indent 2 }}
  selector:
    matchLabels:
      app: {{ $fullname }}
---
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: {{ $fullname }}
  namespace: d8-{{ $context.Chart.Name }}
{{ include "helm_lib_module_labels" (list $context (dict "app" "csi-node")) | indent 2 }}
spec:
  updateStrategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app: {{ $fullname }}
  template:
    metadata:
      labels:
        app: {{ $fullname }}
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - operator: In
                key: node.deckhouse.io/type
                values:
                - Cloud
                - Hybrid
      imagePullSecrets:
      - name: deckhouse-registry
{{ include "helm_lib_priority_class" (tuple $context "system-node-critical") | indent 6 }}
{{ include "helm_lib_tolerations" (tuple $context "any-node-with-no-csi") | indent 6 }}
{{ include "helm_lib_module_pod_security_context_run_as_user_root" . | indent 6 }}
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      containers:
      - name: node-driver-registrar
{{ include "helm_lib_module_container_security_context_read_only_root_filesystem_capabilities_drop_all" . | indent 8 }}
        image: {{ $driverRegistrarImage | quote }}
        args:
        - "--v=5"
        - "--csi-address=/csi/csi.sock"
        - "--kubelet-registration-path=/var/lib/kubelet/csi-plugins/{{ $driverFQDN }}/csi.sock"
        env:
        - name: KUBE_NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - name: plugin-dir
          mountPath: /csi
        - name: registration-dir
          mountPath: /registration
        - name: resolv-conf-volume
          mountPath: /etc/resolv.conf
          readOnly: true
        resources:
          requests:
{{ include "helm_lib_module_ephemeral_storage_logs_with_extra" 10 | indent 12 }}
      - name: node
        securityContext:
          privileged: true
        image: {{ $nodeImage }}
        args:
      {{- if $additionalNodeArgs }}
{{ $additionalNodeArgs | toYaml | indent 8 }}
      {{- end }}
      {{- if $additionalNodeEnvs }}
        env:
{{ $additionalNodeEnvs | toYaml | indent 8 }}
      {{- end }}
        volumeMounts:
        - name: kubelet-dir
          mountPath: /var/lib/kubelet
          mountPropagation: "Bidirectional"
        - name: plugin-dir
          mountPath: /csi
        - name: device-dir
          mountPath: /dev
        - name: resolv-conf-volume
          mountPath: /etc/resolv.conf
          readOnly: true
    {{- if $additionalNodeVolumeMounts }}
{{ $additionalNodeVolumeMounts | toYaml | indent 8 }}
      {{- end }}
        resources:
          requests:
{{ include "helm_lib_module_ephemeral_storage_logs_with_extra" 10 | indent 12 }}
      serviceAccount: {{ $serviceAccount | quote }}
      serviceAccountName: {{ $serviceAccount | quote }}
      volumes:
      - name: registration-dir
        hostPath:
          path: /var/lib/kubelet/plugins_registry/
          type: Directory
      - name: kubelet-dir
        hostPath:
          path: /var/lib/kubelet
          type: Directory
      - name: plugin-dir
        hostPath:
          path: /var/lib/kubelet/csi-plugins/{{ $driverFQDN }}/
          type: DirectoryOrCreate
      - name: device-dir
        hostPath:
          path: /dev
          type: Directory
      - name: resolv-conf-volume
        hostPath:
          path: /var/lib/bashible/resolv/resolv.conf
          type: File
    {{- if $additionalNodeVolumes }}
{{ $additionalNodeVolumes | toYaml | indent 6 }}
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}
