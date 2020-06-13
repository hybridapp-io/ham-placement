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

package placementrule

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
)

type DecisionMaker interface {
	ResetDecisionMakingProcess(candidates []corev1.ObjectReference, instance *corev1alpha1.PlacementRule)
	ContinueDecisionMakingProcess(instance *corev1alpha1.PlacementRule) bool
}

type DefaultDecisionMaker struct {
}

func (d *DefaultDecisionMaker) ResetDecisionMakingProcess(candidates []corev1.ObjectReference, instance *corev1alpha1.PlacementRule) {
}

func (d *DefaultDecisionMaker) ContinueDecisionMakingProcess(instance *corev1alpha1.PlacementRule) bool {
	updated := false

	cadmap := make(map[types.UID]*corev1.ObjectReference)
	for _, or := range instance.Status.Candidates {
		cadmap[or.UID] = or.DeepCopy()
	}

	for _, rec := range instance.Status.Recommendations {
		recmap := make(map[types.UID]bool)
		for _, ror := range rec {
			recmap[ror.UID] = true
		}

		for k := range cadmap {
			if _, ok := recmap[k]; !ok {
				delete(recmap, k)
			}
		}
	}

	if instance.Spec.Replicas == nil || int(*instance.Spec.Replicas) <= len(cadmap) {
		updated = d.compareAndSetDecisions(cadmap, instance)
	}

	return updated
}

func (d *DefaultDecisionMaker) compareAndSetDecisions(decmap map[types.UID]*corev1.ObjectReference, instance *corev1alpha1.PlacementRule) bool {
	var decisions []corev1.ObjectReference

	updated := false

	replicas := len(decmap)

	if instance.Spec.Replicas != nil {
		replicas = int(*instance.Spec.Replicas)
	}

	for _, dec := range instance.Status.Decisions {
		var or *corev1.ObjectReference

		var ok bool

		if or, ok = decmap[dec.UID]; !ok {
			updated = true
			break
		}

		if len(decisions) == replicas {
			updated = true
			break
		}

		decisions = append(decisions, *or)
	}

	for _, nor := range decmap {
		if len(decisions) == replicas {
			break
		}

		decisions = append(decisions, *nor)
		updated = true
	}

	if updated {
		instance.Status.Decisions = decisions
	}

	return updated
}
