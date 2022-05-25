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

package utils

import (
	"fmt"
	"strings"

	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	zeroObjectReference = corev1.ObjectReference{
		UID: types.UID(rune(0)),
	}

	EmptyRecommendatation = []corev1.ObjectReference{
		zeroObjectReference,
	}
)

func GenKey(or corev1.ObjectReference) string {
	return string(or.UID)
}

func MakeRecommendation(instance *corev1alpha1.PlacementRule, advisorName string, rec []corev1alpha1.ScoredObjectReference) {
	if instance.Status.Recommendations == nil {
		instance.Status.Recommendations = make(map[string]corev1alpha1.Recommendation)
	}

	instance.Status.Recommendations[advisorName] = rec
}

func IsSameRecommendation(instance *corev1alpha1.PlacementRule, advisorName string, rec []corev1alpha1.ScoredObjectReference) bool {
	if instance.Status.Recommendations == nil && len(rec) == 0 {
		return true
	}

	return EqualRecommendations(instance.Status.Recommendations[advisorName], rec)
}

func EqualRecommendations(src, dst []corev1alpha1.ScoredObjectReference) bool {
	if len(src) == 0 && len(dst) == 0 {
		return true
	}

	if len(src) == 0 || len(dst) == 0 || len(src) != len(dst) {
		return false
	}

	srcmap := make(map[string]bool)

	for _, or := range src {
		srcmap[GenKey(or.ObjectReference)] = true
	}

	for _, or := range dst {
		if _, ok := srcmap[GenKey(or.ObjectReference)]; !ok {
			return false
		}
	}

	return true
}
func EqualDecisions(src, dst []corev1.ObjectReference) bool {
	if len(src) == 0 && len(dst) == 0 {
		return true
	}

	if len(src) == 0 || len(dst) == 0 || len(src) != len(dst) {
		return false
	}

	srcmap := make(map[string]bool)

	for _, or := range src {
		srcmap[GenKey(or)] = true
	}

	for _, or := range dst {
		if _, ok := srcmap[GenKey(or)]; !ok {
			return false
		}
	}

	return true
}

func GetAdvisor(instance *corev1alpha1.PlacementRule, advisorName string) *corev1alpha1.Advisor {
	if instance == nil || advisorName == "" {
		return nil
	}

	if instance.Status.ObservedGeneration != instance.GetGeneration() {
		// only advise on latest
		return nil
	}

	// check if advisor is in the list
	for _, adv := range instance.Spec.Advisors {
		if strings.EqualFold(adv.Name, advisorName) {
			return adv.DeepCopy()
		}
	}

	return nil
}

func Recommended(instance *corev1alpha1.PlacementRule, advisorName string) bool {
	if instance == nil || advisorName == "" {
		return false
	}

	if instance.Status.Recommendations == nil {
		instance.Status.Recommendations = make(map[string]corev1alpha1.Recommendation)
	}

	_, recommended := instance.Status.Recommendations[advisorName]

	return recommended
}
