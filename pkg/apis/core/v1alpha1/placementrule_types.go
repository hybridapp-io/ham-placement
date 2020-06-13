// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type AdvisorType string

const (
	AdvisorTypeUnknown  AdvisorType = ""
	AdvisorTypeFilter   AdvisorType = "predicate"
	AdvisorTypePriority AdvisorType = "priority"
)

type Advisor struct {
	Name   string                `json:"name"`
	Type   *AdvisorType          `json:"type,omitempty"`
	Weight *int16                `json:"weight,omitempty"`
	Rules  *runtime.RawExtension `json:"rules,omitempty"`
}

// PlacementRuleSpec defines the desired state of PlacementRule
// For different deployer type, the target might be different.
// Default kuberentes target: clusters.clusterregistry.k8s.io
type PlacementRuleSpec struct {
	DeployerType   *string                  `json:"deployerType,omitempty"`   // default: kubernetes
	Targets        []corev1.ObjectReference `json:"targets,omitempty"`        // nil: all
	TargetLabels   *metav1.LabelSelector    `json:"targetLabels,omitempty"`   // nil: all
	DecisionWeight *int16                   `json:"decisionWeight,omitempty"` // nil: 10000
	Replicas       *int16                   `json:"replicas,omitempty"`       // nil: all
	Advisors       []Advisor                `json:"advisors,omitempty"`
}

type Recommendation []corev1.ObjectReference

// PlacementRuleStatus defines the observed state of PlacementRule
type PlacementRuleStatus struct {
	LastUpdateTime  *metav1.Time              `json:"lastUpdateTime,omitempty"`
	Candidates      []corev1.ObjectReference  `json:"candidates,omitempty"`
	Eliminators     []corev1.ObjectReference  `json:"eliminators,omitempty"`
	Recommendations map[string]Recommendation `json:"recommendations,omitempty"` // key: advisor name
	Decisions       []corev1.ObjectReference  `json:"decisions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PlacementRule is the Schema for the placementrules API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=placementrules,scope=Namespaced
type PlacementRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlacementRuleSpec   `json:"spec,omitempty"`
	Status PlacementRuleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PlacementRuleList contains a list of PlacementRule
type PlacementRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlacementRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlacementRule{}, &PlacementRuleList{})
}
