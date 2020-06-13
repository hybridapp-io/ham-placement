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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
)

var (
	defaultResource = schema.GroupVersionResource{
		Resource: "clusters",
		Version:  "v1alpha1",
		Group:    "clusterregistry.k8s.io",
	}
)

func (r *ReconcilePlacementRule) generateCandidates(instance *corev1alpha1.PlacementRule) ([]corev1.ObjectReference, error) {
	if instance == nil {
		return nil, nil
	}

	var candiates []corev1.ObjectReference

	// select by targetLabels, nil = everything
	listopts := metav1.ListOptions{}

	if instance.Spec.TargetLabels != nil {
		selector, err := metav1.LabelSelectorAsSelector(instance.Spec.TargetLabels)
		if err != nil {
			klog.Error("Failed to parse label selector with error: ", err)
			return nil, err
		}

		listopts.LabelSelector = selector.String()
	}

	tl, err := r.dynamicClient.Resource(defaultResource).List(listopts)
	if err != nil {
		klog.Error("Failed to list ", defaultResource.String(), " with error: ", err)
		return nil, err
	}

	// build candidate list, filter targets, nil = everything

	for _, obj := range tl.Items {
		or := corev1.ObjectReference{
			Kind:       obj.GroupVersionKind().Kind,
			Name:       obj.GetName(),
			Namespace:  obj.GetNamespace(),
			APIVersion: obj.GetAPIVersion(),
			UID:        obj.GetUID(),
		}

		pass := true

		// check targets
		if len(instance.Spec.Targets) > 0 {
			pass = false
		}

		for _, t := range instance.Spec.Targets {
			if t.Name != "" && t.Name != or.Name {
				continue
			}

			if t.Namespace != "" && t.Namespace != or.Namespace {
				continue
			}

			pass = true

			break
		}

		if pass {
			candiates = append(candiates, or)
		}
	}

	return candiates, nil
}

func isSameCandidateList(candidates []corev1.ObjectReference, instance *corev1alpha1.PlacementRule) bool {
	if candidates == nil && instance == nil {
		return true
	}

	if candidates == nil || instance == nil {
		return false
	}

	newmap := make(map[types.UID]bool)
	// generate map for src
	for _, or := range candidates {
		newmap[or.UID] = true
	}

	exarray := instance.Status.Candidates
	if len(exarray) > 0 {
		for _, or := range exarray {
			if _, ok := newmap[or.UID]; !ok {
				return false
			}

			delete(newmap, or.UID)
		}
	}

	exarray = instance.Status.Eliminators
	if len(exarray) > 0 {
		for _, or := range exarray {
			if _, ok := newmap[or.UID]; !ok {
				return false
			}

			delete(newmap, or.UID)
		}
	}

	return len(newmap) == 0
}
