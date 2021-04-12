package converge

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-multierror"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/deckhouse/deckhouse/candictl/pkg/kubernetes/actions"
	"github.com/deckhouse/deckhouse/candictl/pkg/kubernetes/actions/manifests"
	"github.com/deckhouse/deckhouse/candictl/pkg/kubernetes/client"
	"github.com/deckhouse/deckhouse/candictl/pkg/log"
	"github.com/deckhouse/deckhouse/candictl/pkg/terraform"
	"github.com/deckhouse/deckhouse/candictl/pkg/util/retry"
)

type NodeGroupTerraformState struct {
	State    map[string][]byte
	Settings []byte
}

func GetNodesStateFromCluster(kubeCl *client.KubernetesClient) (map[string]NodeGroupTerraformState, error) {
	extractedState := make(map[string]NodeGroupTerraformState)

	err := retry.StartLoop("Get Nodes Terraform state from Kubernetes cluster", 5, 5, func() error {
		nodeStateSecrets, err := kubeCl.CoreV1().Secrets("d8-system").List(metav1.ListOptions{LabelSelector: "node.deckhouse.io/terraform-state"})
		if err != nil {
			return err
		}

		for _, nodeState := range nodeStateSecrets.Items {
			name := nodeState.Labels["node.deckhouse.io/node-name"]
			if name == "" {
				return fmt.Errorf("can't determine Node name for %q secret", nodeState.Name)
			}

			nodeGroup := nodeState.Labels["node.deckhouse.io/node-group"]
			if nodeGroup == "" {
				return fmt.Errorf("can't determine NodeGroup for %q secret", nodeState.Name)
			}

			if _, ok := extractedState[nodeGroup]; !ok {
				extractedState[nodeGroup] = NodeGroupTerraformState{State: make(map[string][]byte)}
			}

			// TODO: validate, that all secrets from node group have same node-group-settings.json
			nodeGroupTerraformState := extractedState[nodeGroup]
			nodeGroupTerraformState.Settings = nodeState.Data["node-group-settings.json"]

			state := nodeState.Data["node-tf-state.json"]
			nodeGroupTerraformState.State[name] = state

			log.InfoF("nodeGroup=%s nodeName=%s symbols=%v\n", nodeGroup, name, len(state))
			extractedState[nodeGroup] = nodeGroupTerraformState
		}
		return nil
	})
	return extractedState, err
}

func GetClusterStateFromCluster(kubeCl *client.KubernetesClient) ([]byte, error) {
	var state []byte
	err := retry.StartLoop("Get Cluster Terraform state from Kubernetes cluster", 5, 5, func() error {
		clusterStateSecret, err := kubeCl.CoreV1().Secrets("d8-system").Get("d8-cluster-terraform-state", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Return empty state, if there is no state in cluster. Need to skip cluster state apply in converge.
				return nil
			}
			return err
		}

		state = clusterStateSecret.Data["cluster-tf-state.json"]
		return nil
	})
	return state, err
}

// Create secret for node with group settings only.
func CreateNodeTerraformState(kubeCl *client.KubernetesClient, nodeName, nodeGroup string, settings []byte) error {
	task := actions.ManifestTask{
		Name: fmt.Sprintf(`Secret "d8-node-terraform-state-%s"`, nodeName),
		Manifest: func() interface{} {
			return manifests.SecretWithNodeTerraformState(nodeName, nodeGroup, nil, settings)
		},
		CreateFunc: func(manifest interface{}) error {
			_, err := kubeCl.CoreV1().Secrets("d8-system").Create(manifest.(*apiv1.Secret))
			return err
		},
		UpdateFunc: func(manifest interface{}) error {
			_, err := kubeCl.CoreV1().Secrets("d8-system").Update(manifest.(*apiv1.Secret))
			return err
		},
	}
	return retry.StartLoop(fmt.Sprintf("Create Terraform state for Node %q", nodeName), 45, 10, task.CreateOrUpdate)
}

func SaveNodeTerraformState(kubeCl *client.KubernetesClient, nodeName, nodeGroup string, tfState, settings []byte) error {
	if len(tfState) == 0 {
		return fmt.Errorf("Terraform state is not found in outputs.")
	}

	task := actions.ManifestTask{
		Name: fmt.Sprintf(`Secret "d8-node-terraform-state-%s"`, nodeName),
		Manifest: func() interface{} {
			return manifests.SecretWithNodeTerraformState(nodeName, nodeGroup, tfState, settings)
		},
		CreateFunc: func(manifest interface{}) error {
			_, err := kubeCl.CoreV1().Secrets("d8-system").Create(manifest.(*apiv1.Secret))
			return err
		},
		UpdateFunc: func(manifest interface{}) error {
			_, err := kubeCl.CoreV1().Secrets("d8-system").Update(manifest.(*apiv1.Secret))
			return err
		},
	}
	return retry.StartLoop(fmt.Sprintf("Save Terraform state for Node %q", nodeName), 45, 10, task.CreateOrUpdate)
}

func SaveMasterNodeTerraformState(kubeCl *client.KubernetesClient, nodeName string, tfState, devicePath []byte) error {
	if len(tfState) == 0 {
		return fmt.Errorf("Terraform state is not found in outputs.")
	}

	getTerraformStateManifest := func() interface{} {
		return manifests.SecretWithNodeTerraformState(nodeName, masterNodeGroupName, tfState, nil)
	}
	getDevicePathManifest := func() interface{} {
		return manifests.SecretMasterDevicePath(nodeName, devicePath)
	}

	tasks := []actions.ManifestTask{
		{
			Name:     fmt.Sprintf(`Secret "d8-node-terraform-state-%s"`, nodeName),
			Manifest: getTerraformStateManifest,
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("d8-system").Create(manifest.(*apiv1.Secret))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("d8-system").Update(manifest.(*apiv1.Secret))
				return err
			},
		},
		{
			Name:     `Secret "d8-masters-kubernetes-data-device-path"`,
			Manifest: getDevicePathManifest,
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("d8-system").Create(manifest.(*apiv1.Secret))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				data, err := json.Marshal(manifest.(*apiv1.Secret))
				if err != nil {
					return err
				}
				_, err = kubeCl.CoreV1().Secrets("d8-system").Patch(
					"d8-masters-kubernetes-data-device-path",
					types.MergePatchType,
					data,
				)
				return err
			},
		},
	}

	return retry.StartLoop(fmt.Sprintf("Save Terraform state for master Node %s", nodeName), 45, 10, func() error {
		var allErrs *multierror.Error
		for _, task := range tasks {
			if err := task.CreateOrUpdate(); err != nil {
				allErrs = multierror.Append(allErrs, err)
			}
		}
		return allErrs.ErrorOrNil()
	})
}

// SaveNodeIntermediateTerraformState is a method to patch Secret with node state.
// It patches a "node-tf-state" key with terraform state or create a new secret if new node is created.
//
// settings can be nil for master node.
//
// The difference between master node and static node: master node has
// no key "node-group-settings.json" with group settings.
func SaveNodeIntermediateTerraformState(kubeCl *client.KubernetesClient, nodeName, nodeGroup string, outputs *terraform.PipelineOutputs, settings []byte) error {
	if outputs == nil || len(outputs.TerraformState) == 0 {
		return fmt.Errorf("terraform state is not found in outputs")
	}

	task := actions.ManifestTask{
		Name: fmt.Sprintf(`Secret "d8-node-terraform-state-%s"`, nodeName),
		Manifest: func() interface{} {
			return manifests.SecretWithNodeTerraformState(nodeName, nodeGroup, outputs.TerraformState, settings)
		},
		CreateFunc: func(manifest interface{}) error {
			_, err := kubeCl.CoreV1().Secrets("d8-system").Create(manifest.(*apiv1.Secret))
			return err
		},
		PatchData: func() interface{} {
			return manifests.PatchWithNodeTerraformState(outputs.TerraformState)
		},
		PatchFunc: func(patchData []byte) error {
			secretName := manifests.SecretNameForNodeTerraformState(nodeName)
			// MergePatch is used because we need to replace one field in "data".
			_, err := kubeCl.CoreV1().Secrets("d8-system").Patch(secretName, types.MergePatchType, patchData)
			return err
		},
	}
	return retry.StartSilentLoop(fmt.Sprintf("Save intermediate Terraform state for Node %q", nodeName), 45, 10, task.PatchOrCreate)
}

func SaveClusterTerraformState(kubeCl *client.KubernetesClient, outputs *terraform.PipelineOutputs) error {
	if outputs == nil || len(outputs.TerraformState) == 0 {
		return fmt.Errorf("Terraform state is not found in outputs.")
	}

	task := actions.ManifestTask{
		Name:     `Secret "d8-cluster-terraform-state"`,
		Manifest: func() interface{} { return manifests.SecretWithTerraformState(outputs.TerraformState) },
		CreateFunc: func(manifest interface{}) error {
			_, err := kubeCl.CoreV1().Secrets("d8-system").Create(manifest.(*apiv1.Secret))
			return err
		},
		UpdateFunc: func(manifest interface{}) error {
			_, err := kubeCl.CoreV1().Secrets("d8-system").Update(manifest.(*apiv1.Secret))
			return err
		},
	}

	err := retry.StartLoop("Save Cluster Terraform state", 45, 10, task.CreateOrUpdate)
	if err != nil {
		return err
	}

	patch, err := json.Marshal(map[string]interface{}{
		"data": map[string]interface{}{
			"cloud-provider-discovery-data.json": outputs.CloudDiscovery,
		},
	})
	if err != nil {
		return err
	}

	return retry.StartLoop("Update cloud discovery data", 45, 10, func() error {
		_, err = kubeCl.CoreV1().Secrets("kube-system").Patch(
			"d8-provider-cluster-configuration",
			types.MergePatchType,
			patch,
		)
		return err
	})
}

// Save only terraform state, cloud-provider-discovery-data is not updated.
func SaveClusterIntermediateTerraformState(kubeCl *client.KubernetesClient, outputs *terraform.PipelineOutputs) error {
	if outputs == nil || len(outputs.TerraformState) == 0 {
		return fmt.Errorf("terraform state is not found in outputs")
	}

	task := actions.ManifestTask{
		Name: `Secret "d8-cluster-terraform-state"`,
		PatchData: func() interface{} {
			return manifests.PatchWithTerraformState(outputs.TerraformState)
		},
		PatchFunc: func(patch []byte) error {
			// MergePatch is used because we need to replace one field in "data".
			_, err := kubeCl.CoreV1().Secrets("d8-system").Patch(manifests.TerraformClusterStateName, types.MergePatchType, patch)
			return err
		},
	}

	return retry.StartSilentLoop("Save Cluster intermediate Terraform state", 45, 10, task.Patch)
}

func DeleteTerraformState(kubeCl *client.KubernetesClient, secretName string) error {
	return retry.StartLoop(fmt.Sprintf("Delete Terraform state %s", secretName), 45, 10, func() error {
		return kubeCl.CoreV1().Secrets("d8-system").Delete(secretName, &metav1.DeleteOptions{})
	})
}

func GetClusterUUID(kubeCl *client.KubernetesClient) (string, error) {
	var clusterUUID string
	err := retry.StartLoop("Get Cluster UUID from the Kubernetes cluster", 5, 5, func() error {
		uuidConfigMap, err := kubeCl.CoreV1().ConfigMaps("kube-system").Get("d8-cluster-uuid", metav1.GetOptions{})
		if err != nil {
			return err
		}

		clusterUUID = uuidConfigMap.Data["cluster-uuid"]
		return nil
	})
	return clusterUUID, err
}

func NewClusterStateSaver(kubeCl *client.KubernetesClient) *terraform.StateSaver {
	return terraform.NewStateSaver(func(outputs *terraform.PipelineOutputs) error {
		return SaveClusterIntermediateTerraformState(kubeCl, outputs)
	})
}

func NewNodeStateSaver(kubeCl *client.KubernetesClient, nodeName string, nodeGroup string, nodeGroupSettings []byte) *terraform.StateSaver {
	return terraform.NewStateSaver(func(outputs *terraform.PipelineOutputs) error {
		return SaveNodeIntermediateTerraformState(kubeCl, nodeName, nodeGroup, outputs, nodeGroupSettings)
	})
}
