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
	"strings"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
)

const (
	advisorName = "veto"
)

var (
	emptyRecommendatation = corev1.ObjectReference{
		UID: types.UID(0),
	}
)

type vetoRules struct {
	Resources []corev1.ObjectReference `json:"resources"`
}

func (r *ReconcileVetoAdvisor) Recommend(instance *corev1alpha1.PlacementRule) bool {
	var err error

	var vetoadv corev1alpha1.Advisor

	invited := false

	for _, adv := range instance.Spec.Advisors {
		if strings.EqualFold(adv.Name, advisorName) {
			adv.DeepCopyInto(&vetoadv)

			invited = true

			break
		}
	}

	if !invited {
		return false
	}

	if instance.Status.Recommendations == nil {
		instance.Status.Recommendations = make(map[string]corev1alpha1.Recommendation)
	}

	prevrec := instance.Status.Recommendations[advisorName]

	vetorules := &vetoRules{}

	if len(vetoadv.Rules.Raw) == 0 {
	} else {
		err = yaml.Unmarshal(vetoadv.Rules.Raw, vetorules)
		if err != nil {
			klog.Error("Failed to parse veto objects ", err)
			return false
		}
	}

	var rec []corev1.ObjectReference

	for _, or := range instance.Status.Candidates {
		veto := false

		for _, vetoor := range vetorules.Resources {
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

	if len(rec) == 0 {
		rec = append(rec, emptyRecommendatation)
	}

	update := !isEqualRecommendataion(rec, prevrec)

	if update {
		instance.Status.Recommendations[advisorName] = rec
	}

	return update
}

func isEqualRecommendataion(src, dst corev1alpha1.Recommendation) bool {
	if src == nil && dst == nil {
		return true
	}

	if src == nil || dst == nil {
		return false
	}

	srcmap := make(map[types.UID]bool)

	for _, or := range src {
		srcmap[or.UID] = true
	}

	for _, or := range dst {
		if _, ok := srcmap[or.UID]; !ok {
			return false
		}
	}

	return len(srcmap) == 0
}
