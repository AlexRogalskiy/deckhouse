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
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	"github.com/flant/shell-operator/pkg/kube_events_manager/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/versioning"
	audit "k8s.io/apiserver/pkg/apis/audit/v1"
	"sigs.k8s.io/yaml"
)

var systemNamespaces = []string{"kube-system", "d8-cert-manager", "d8-chrony", "d8-cloud-instance-manager", "d8-cloud-provider-aws", "d8-cloud-provider-azure", "d8-cloud-provider-gcp", "d8-cloud-provider-openstack", "d8-cloud-provider-vsphere", "d8-cloud-provider-yandex", "d8-cni-flannel", "d8-cni-simple-bridge", "d8-descheduler", "d8-flant-integration", "d8-ingress-nginx", "d8-istio", "d8-keepalived", "d8-local-path-provisioner", "d8-log-shipper", "d8-metallb", "d8-monitoring", "d8-network-gateway", "d8-okmeter", "d8-openvpn", "d8-operator-prometheus", "d8-pod-reloader", "d8-system", "d8-user-authn", "d8-user-authz", "d8-upmeter"}
var systemSA = []string{"system:serviceaccount:d8-system:deckhouse"}

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue:        moduleQueue,
	OnBeforeHelm: &go_hook.OrderedConfig{Order: 10},
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:       "kube_audit_policy_secret",
			ApiVersion: "v1",
			Kind:       "Secret",
			NamespaceSelector: &types.NamespaceSelector{
				NameSelector: &types.NameSelector{
					MatchNames: []string{"kube-system"},
				},
			},
			NameSelector: &types.NameSelector{
				MatchNames: []string{"audit-policy"},
			},
			FilterFunc: filterAuditSecret,
		},
	},
}, handleAuditPolicy)

func filterAuditSecret(unstructured *unstructured.Unstructured) (go_hook.FilterResult, error) {
	var sec v1.Secret

	err := sdk.FromUnstructured(unstructured, &sec)
	if err != nil {
		return nil, err
	}

	data := sec.Data["audit-policy.yaml"]

	return data, nil
}

func appendDropRule(policy *audit.Policy, resource audit.GroupResources) {
	rule := audit.PolicyRule{
		Level: audit.LevelNone,
		Resources: []audit.GroupResources{
			resource,
		},
	}
	policy.Rules = append(policy.Rules, rule)
}

func handleAuditPolicy(input *go_hook.HookInput) error {
	var policy audit.Policy

	// Drop events on endpoints and events resources.
	// todo(31337Ghost) consider to drop only particular endpoints events + mcm triggered
	// https://github.com/kubernetes/kubernetes/blob/master/cluster/gce/gci/configure-helper.sh#L1195
	appendDropRule(&policy, audit.GroupResources{
		Group:     "",
		Resources: []string{"endpoints", "events"},
	})
	// Drop leader elections based on leases resource.
	appendDropRule(&policy, audit.GroupResources{
		Group:     "coordination.k8s.io",
		Resources: []string{"leases"},
	})
	// Drop cert-manager's leader elections based on configmap resources.
	appendDropRule(&policy, audit.GroupResources{
		Group:         "",
		Resources:     []string{"configmaps"},
		ResourceNames: []string{"cert-manager-cainjector-leader-election", "cert-manager-controller"},
	})
	// Drop verticalpodautoscalercheckpoints.
	appendDropRule(&policy, audit.GroupResources{
		Group:     "autoscaling.k8s.io",
		Resources: []string{"verticalpodautoscalercheckpoints"},
	})

	// Drop everything related to d8-upmeter namespace.
	{
		rule := audit.PolicyRule{
			Level:      audit.LevelNone,
			Namespaces: []string{"d8-upmeter"},
		}
		policy.Rules = append(policy.Rules, rule)
	}
	// Drop upmeterhookprobes.
	appendDropRule(&policy, audit.GroupResources{
		Group:     "deckhouse.io",
		Resources: []string{"upmeterhookprobes"},
	})

	// A rule collecting logs about actions of service accounts from system namespaces.
	{
		rule := audit.PolicyRule{
			Level:      audit.LevelMetadata,
			Verbs:      []string{"create", "update", "patch", "delete"},
			Users:      systemSA,
			UserGroups: []string{"system:serviceaccounts"},
			OmitStages: []audit.Stage{
				audit.StageRequestReceived,
			},
		}
		policy.Rules = append(policy.Rules, rule)
	}
	// A rule collecting logs about all actions taken on the resources from system namespaces.
	{
		rule := audit.PolicyRule{
			Level:      audit.LevelMetadata,
			Verbs:      []string{"create", "update", "patch", "delete"},
			Namespaces: systemNamespaces,
			OmitStages: []audit.Stage{
				audit.StageRequestReceived,
			},
		}
		policy.Rules = append(policy.Rules, rule)
	}

	policyEnabled := input.Values.Get("controlPlaneManager.apiserver.auditPolicyEnabled")
	snap := input.Snapshots["kube_audit_policy_secret"]

	if policyEnabled.Bool() && len(snap) > 0 {
		data := snap[0].([]byte)

		var p audit.Policy
		err := yaml.UnmarshalStrict(data, &p)
		if err != nil {
			return fmt.Errorf("invalid audit-policy.yaml format: %s", err)
		}

		policy.OmitStages = append(policy.OmitStages, p.OmitStages...)
		policy.Rules = append(policy.Rules, p.Rules...)
	}

	schema := runtime.NewScheme()
	builder := runtime.SchemeBuilder{
		audit.AddToScheme,
	}
	err := builder.AddToScheme(schema)
	if err != nil {
		return err
	}
	serializer := json.NewSerializerWithOptions(
		json.DefaultMetaFactory, schema, schema,
		json.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)
	buf := bytes.NewBuffer(nil)
	versioningCodec := versioning.NewDefaultingCodecForScheme(schema, serializer, nil, nil, nil)
	err = versioningCodec.Encode(&policy, buf)
	if err != nil {
		return fmt.Errorf("invalid final Policy format: %s", err)
	}

	data := strings.Replace(buf.String(), "metadata:\n  creationTimestamp: null\n", "", 1)

	input.Values.Set("controlPlaneManager.internal.auditPolicy", base64.StdEncoding.EncodeToString([]byte(data)))

	return nil
}
