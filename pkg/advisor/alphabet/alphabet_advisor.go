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

package alphabet

import (
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
)

const (
	advisorName = "alphabet"
)

type objectReferenceIndex struct {
	Items []*corev1.ObjectReference
}

func (oi objectReferenceIndex) Len() int {
	return len(oi.Items)
}

func (oi objectReferenceIndex) Less(x, y int) bool {
	cmp := strings.Compare(oi.Items[x].Name, oi.Items[y].Name)

	if cmp == 0 {
		cmp = strings.Compare(oi.Items[x].Namespace, oi.Items[y].Namespace)
	}

	return cmp < 0
}

func (oi objectReferenceIndex) Swap(x, y int) {
	oi.Items[x], oi.Items[y] = oi.Items[y], oi.Items[x]
}

func (r *ReconcilePlacementRule) Recommend(instance *corev1alpha1.PlacementRule) bool {
	invited := false

	for _, adv := range instance.Spec.Advisors {
		if strings.EqualFold(adv.Name, advisorName) {
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

	if _, ok := instance.Status.Recommendations[advisorName]; ok {
		return false
	}

	ori := objectReferenceIndex{}

	for _, or := range instance.Status.Candidates {
		ori.Items = append(ori.Items, or.DeepCopy())
	}

	sort.Sort(ori)

	reclen := ori.Len()
	if instance.Spec.Replicas != nil {
		if int(*instance.Spec.Replicas) < reclen {
			reclen = int(*instance.Spec.Replicas)
		}
	}

	rec := make([]corev1.ObjectReference, reclen)

	for i, or := range ori.Items {
		if i == reclen {
			break
		}

		rec[i] = *or
	}

	instance.Status.Recommendations[advisorName] = rec

	return true
}
