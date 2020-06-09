package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type DeployerSeletor struct {
	Deployers      *corev1.LocalObjectReference `json:"deployers,omitempty"`
	DeployerLabels *metav1.LabelSelector        `json:"deployerLabels,omitempty"`
}

type PlacementHint struct {
	SchedulerName string                `json:"schedulerName"`
	Priority      *int                  `json:"priority,omitempty"`
	Rules         *runtime.RawExtension `json:"rules,omitempty"`
}

// PlacementRuleSpec defines the desired state of PlacementRule
type PlacementRuleSpec struct {
	DeployerSelector DeployerSeletor `json:",inline"`
	PlacementHints   []PlacementHint `json:"hints,omitempty"`
}

type Recommendation []corev1.ObjectReference

// PlacementRuleStatus defines the observed state of PlacementRule
type PlacementRuleStatus struct {
	Candidates      []corev1.ObjectReference  `json:"candidates,omitempty"`
	Recommendations map[string]Recommendation `json:"recommendations,omitempty"`
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
