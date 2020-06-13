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
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
)

func (r *ReconcilePlacementRule) ResetDecisionMakingProcess(candidates []corev1.ObjectReference, instance *corev1alpha1.PlacementRule) error {
	now := metav1.Now()
	instance.Status.LastUpdateTime = &now

	instance.Status.Candidates = candidates
	instance.Status.Recommendations = nil
	instance.Status.Eliminators = nil

	return r.client.Status().Update(context.TODO(), instance)
}

func (r *ReconcilePlacementRule) ContinueDecisionMakingProcess(instance *corev1alpha1.PlacementRule) error {
	var err error

	return err
}
