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

package veto

import (
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	advisorutils "github.com/hybridapp-io/ham-placement/pkg/advisor/utils"
	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
)

const (
	advisorName = "veto"
)

var defaultScore = int16(corev1alpha1.DefaultScore)

type vetoRules struct {
	Resources []corev1.ObjectReference `json:"resources"`
}

func (r *ReconcileVetoAdvisor) doRecommend(candidates, bl []corev1.ObjectReference) []corev1.ObjectReference {
	var rec []corev1.ObjectReference

	for _, or := range candidates {
		veto := false

		for _, vetoor := range bl {
			if vetoor.Name != "" && vetoor.Name != or.Name {
				continue
			}

			if vetoor.Namespace != "" && vetoor.Namespace != or.Namespace {
				continue
			}

			if vetoor.Name == "" && vetoor.Namespace == "" {
				continue
			}

			veto = true

			break
		}

		if !veto {
			rec = append(rec, *or.DeepCopy())
		}
	}

	return rec
}

func (r *ReconcileVetoAdvisor) Recommend(instance *corev1alpha1.PlacementRule, vetoadv *corev1alpha1.Advisor) []corev1alpha1.ScoredObjectReference {
	if vetoadv.Rules == nil || (vetoadv.Rules.Object == nil && len(vetoadv.Rules.Raw) == 0) {
		return r.getScoredObjectReferences(instance.Status.Candidates)
	}

	vetorules := &vetoRules{}

	if len(vetoadv.Rules.Raw) != 0 {
		err := yaml.Unmarshal(vetoadv.Rules.Raw, vetorules)
		if err != nil {
			klog.Error("Failed to parse veto objects ", err)
			return r.getScoredObjectReferences(instance.Status.Candidates)
		}
	}

	rec := r.doRecommend(instance.Status.Candidates, vetorules.Resources)

	if len(rec) == 0 {
		rec = advisorutils.EmptyRecommendatation
	}

	return r.getScoredObjectReferences(rec)
}

func (r *ReconcileVetoAdvisor) getScoredObjectReferences(references []corev1.ObjectReference) []corev1alpha1.ScoredObjectReference {
	rec := make([]corev1alpha1.ScoredObjectReference, len(references))
	for i, or := range references {

		rec[i] = corev1alpha1.ScoredObjectReference{
			ObjectReference: or,
			Score:           &defaultScore,
		}
	}
	return rec
}
