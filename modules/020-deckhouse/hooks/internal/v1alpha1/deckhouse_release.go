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

package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

const (
	PhasePending   = "Pending"
	PhaseDeployed  = "Deployed"
	PhaseOutdated  = "Outdated"
	PhaseSuspended = "Suspended"
)

// DeckhouseRelease is a deckhouse release object.
type DeckhouseRelease struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Approved bool `json:"approved"`

	Spec DeckhouseReleaseSpec `json:"spec"`

	Status DeckhouseReleaseStatus `json:"status,omitempty"`
}

type DeckhouseReleaseSpec struct {
	Version    string     `json:"version,omitempty"`
	ApplyAfter *time.Time `json:"applyAfter,omitempty"`
}

type DeckhouseReleaseStatus struct {
	Phase          string    `json:"phase,omitempty"`
	Approved       bool      `json:"approved"`
	TransitionTime time.Time `json:"transitionTime,omitempty"`
}

type deckhouseReleaseKind struct{}

func (in *DeckhouseReleaseStatus) GetObjectKind() schema.ObjectKind {
	return &deckhouseReleaseKind{}
}

func (f *deckhouseReleaseKind) SetGroupVersionKind(_ schema.GroupVersionKind) {}
func (f *deckhouseReleaseKind) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "deckhouse.io", Version: "v1alpha1", Kind: "DeckhouseRelease"}
}
