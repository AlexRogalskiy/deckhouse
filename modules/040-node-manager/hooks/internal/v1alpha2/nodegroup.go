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

package v1alpha2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/deckhouse/deckhouse/go_lib/hooks/update"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeGroup is a group of nodes in Kubernetes.
type NodeGroup struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of a node group.
	Spec NodeGroupSpec `json:"spec"`

	// Most recently observed status of the node.
	// Populated by the system.

	Status NodeGroupStatus `json:"status,omitempty"`
}

type NodeGroupSpec struct {
	// Type of nodes in group: Cloud, Static, Hybrid. Field is required.
	NodeType string `json:"nodeType,omitempty"`

	// CRI parameters. Optional.
	CRI CRI `json:"cri,omitempty"`

	// cloudInstances. Optional.
	CloudInstances CloudInstances `json:"cloudInstances,omitempty"`

	// Default labels, annotations and taints for Nodes in NodeGroup. Optional.
	NodeTemplate NodeTemplate `json:"nodeTemplate,omitempty"`

	// Chaos monkey settings. Optional.
	Chaos Chaos `json:"chaos,omitempty"`

	// OperatingSystem. Optional.
	OperatingSystem OperatingSystem `json:"operatingSystem,omitempty"`

	// Disruptions settings for nodes. Optional.
	Disruptions Disruptions `json:"disruptions,omitempty"`

	// Kubelet settings for nodes. Optional.
	Kubelet Kubelet `json:"kubelet,omitempty"`
}

type CRI struct {
	// Container runtime type. Docker, Containerd or NotManaged
	Type string `json:"type,omitempty"`

	// Containerd runtime parameters.
	Containerd *Containerd `json:"containerd,omitempty"`

	// Docker settings for nodes.
	Docker *Docker `json:"docker,omitempty"`

	// NotManaged settings for nodes.
	NotManaged *NotManaged `json:"notManaged,omitempty"`
}

func (c CRI) IsEmpty() bool {
	return c.Type == "" && c.Containerd == nil && c.Docker == nil
}

type Containerd struct {
	// Set the max concurrent downloads for each pull (default 3).
	MaxConcurrentDownloads *int32 `json:"maxConcurrentDownloads,omitempty"`
}

type Docker struct {
	// Set the max concurrent downloads for each pull (default 3).
	MaxConcurrentDownloads *int32 `json:"maxConcurrentDownloads,omitempty"`

	// Enable docker maintenance from bashible.
	Manage *bool `json:"manage,omitempty"`
}

type NotManaged struct {
	// Set custom path to CRI socket
	CriSocketPath *string `json:"criSocketPath,omitempty"`
}

// CloudInstances is an extra parameters for NodeGroup with type Cloud.
type CloudInstances struct {
	// List of availability zones to create instances in.
	Zones []string `json:"zones"`

	// Minimal amount of instances for the group in each zone. Required.
	MinPerZone *int32 `json:"minPerZone,omitempty"`

	// Maximum amount of instances for the group in each zone. Required.
	MaxPerZone *int32 `json:"maxPerZone,omitempty"`

	// Maximum amount of unavailable instances (during rollout) in the group in each zone.
	MaxUnavailablePerZone *int32 `json:"maxUnavailablePerZone,omitempty"`

	// Maximum amount of instances to rollout simultaneously in the group in each zone.
	MaxSurgePerZone *int32 `json:"maxSurgePerZone,omitempty"`

	// Overprovisioned Nodes for this NodeGroup.
	Standby *intstr.IntOrString `json:"standby,omitempty"`

	// Settings for overprovisioned Node holder.
	StandbyHolder StandbyHolder `json:"standbyHolder,omitempty"`

	// Reference to a ClassInstance resource. Required.
	ClassReference ClassReference `json:"classReference"`
}

func (c CloudInstances) IsEmpty() bool {
	return c.Zones == nil &&
		c.MinPerZone == nil &&
		c.MaxPerZone == nil &&
		c.MaxUnavailablePerZone == nil &&
		c.MaxSurgePerZone == nil &&
		c.Standby == nil &&
		c.StandbyHolder.IsEmpty() &&
		c.ClassReference.IsEmpty()
}

type StandbyHolder struct {
	// Describes the amount of resources, that will not be held by standby holder.
	NotHeldResources Resources `json:"notHeldResources,omitempty"`
}

func (s StandbyHolder) IsEmpty() bool {
	return s.NotHeldResources.IsEmpty()
}

type Resources struct {
	// Describes the amount of CPU that will not be held by standby holder on Nodes from this NodeGroup.
	CPU intstr.IntOrString `json:"cpu,omitempty"`

	// Describes the amount of memory that will not be held by standby holder on Nodes from this NodeGroup.
	Memory intstr.IntOrString `json:"memory,omitempty"`
}

func (r Resources) IsEmpty() bool {
	v := r.CPU.String() + r.Memory.String()
	return v == "" || v == "00"
}

type ClassReference struct {
	// Kind of a ClassReference resource: OpenStackInstanceClass, GCPInstanceClass, ...
	Kind string `json:"kind,omitempty"`

	// Name of a ClassReference resource.
	Name string `json:"name,omitempty"`
}

func (c ClassReference) IsEmpty() bool {
	return c.Kind == "" && c.Name == ""
}

type NodeTemplate struct {
	// Annotations is an unstructured key value map that is used as default
	// annotations for Nodes in NodeGroup.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Map of string keys and values that is used as default
	// labels for Nodes in NodeGroup.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels"`

	// Default taints for Nodes in NodeGroup.
	Taints []v1.Taint `json:"taints,omitempty"`
}

func (n NodeTemplate) IsEmpty() bool {
	return n.Annotations == nil && n.Labels == nil && n.Taints == nil
}

// Chaos is a chaos-monkey settings.
type Chaos struct {
	// Chaos monkey mode: DrainAndDelete or Disabled (default).
	Mode string `json:"mode,omitempty"`

	// Chaos monkey wake up period. Default is 6h.
	Period string `json:"period,omitempty"`
}

func (c Chaos) IsEmpty() bool {
	return c.Mode == "" && c.Period == ""
}

type OperatingSystem struct {
	// Enable kernel maintenance from bashible (default true).
	ManageKernel *bool `json:"manageKernel,omitempty"`
}

func (o OperatingSystem) IsEmpty() bool {
	return o.ManageKernel == nil
}

type Disruptions struct {
	// Allow disruptive update mode: Manual or Automatic.
	ApprovalMode string `json:"approvalMode"`

	// Extra settings for Automatic mode.
	Automatic AutomaticDisruptions `json:"automatic,omitempty"`
}

func (d Disruptions) IsEmpty() bool {
	return d.ApprovalMode == "" && d.Automatic.IsEmpty()
}

type AutomaticDisruptions struct {
	// Indicates if Pods should be drained from node before allow disruption.
	DrainBeforeApproval *bool `json:"drainBeforeApproval,omitempty"`
	// Node update windows
	Windows update.Windows `json:"windows,omitempty"`
}

func (a AutomaticDisruptions) IsEmpty() bool {
	return a.DrainBeforeApproval == nil && len(a.Windows) == 0
}

type Kubelet struct {
	// Set the max count of pods per node. Default: 110
	MaxPods *int32 `json:"maxPods,omitempty"`

	// Directory path for managing kubelet files (volume mounts,etc).
	// Default: '/var/lib/kubelet'
	RootDir string `json:"rootDir,omitempty"`
}

func (k Kubelet) IsEmpty() bool {
	return k.MaxPods == nil && k.RootDir == ""
}

type NodeGroupStatus struct {
	// Number of ready Kubernetes nodes in the group.
	Ready int32 `json:"ready,omitempty"`

	// Number of Kubernetes nodes (in any state) in the group.
	Nodes int32 `json:"nodes,omitempty"`

	// Number of instances (in any state) in the group.
	Instances int32 `json:"instances,omitempty"`

	// Number of desired machines in the group.
	Desired int32 `json:"desired,omitempty"`

	// Minimal amount of instances in the group.
	Min int32 `json:"min,omitempty"`

	// Maximum amount of instances in the group.
	Max int32 `json:"max,omitempty"`

	// Number of up-to-date nodes in the group.
	UpToDate int32 `json:"upToDate,omitempty"`

	// Number of overprovisioned instances in the group.
	Standby int32 `json:"standby,omitempty"`

	// Error message about possible problems with the group handling.
	Error string `json:"error,omitempty"`

	// A list of last failures of handled Machines.
	LastMachineFailures []MachineFailure `json:"lastMachineFailures,omitempty"`

	// Status' summary.
	ConditionSummary ConditionSummary `json:"conditionSummary,omitempty"`
}

type MachineFailure struct {
	// Machine's name.
	Name string `json:"name,omitempty"`

	// Machine's ProviderID.
	ProviderID string `json:"providerID,omitempty"`

	// Machine owner's name.
	OwnerRef string `json:"ownerRef,omitempty"`

	// Last operation with machine.
	LastOperation MachineOperation `json:"lastOperation,omitempty"`
}

type MachineOperation struct {
	// Last operation's description.
	Description string `json:"description,omitempty"`

	// Timestamp of last status update for operation.
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`

	// Machine's operation state.
	State string `json:"state,omitempty"`

	// Type of operation.
	Type string `json:"type,omitempty"`
}

type ConditionSummary struct {
	// Status message about group handling.
	StatusMessage string `json:"statusMessage,omitempty"`

	// Summary for the NodeGroup status: True or False
	Ready string `json:"ready,omitempty"`
}

type nodeGroupKind struct{}

func (in *NodeGroupStatus) GetObjectKind() schema.ObjectKind {
	return &nodeGroupKind{}
}

func (f *nodeGroupKind) SetGroupVersionKind(_ schema.GroupVersionKind) {}
func (f *nodeGroupKind) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "deckhouse.io", Version: "v1alpha2", Kind: "NodeGroup"}
}
