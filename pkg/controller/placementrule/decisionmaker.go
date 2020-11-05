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
	"k8s.io/klog"

	advisorutils "github.com/hybridapp-io/ham-placement/pkg/advisor/utils"
	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
)

const (
	defaultStep = 1
)

type DecisionMaker interface {
	ResetDecisionMakingProcess(candidates []corev1.ObjectReference, instance *corev1alpha1.PlacementRule)
	ContinueDecisionMakingProcess(instance *corev1alpha1.PlacementRule) bool
}

type DefaultDecisionMaker struct {
}

func (d *DefaultDecisionMaker) ResetDecisionMakingProcess(candidates []corev1.ObjectReference, instance *corev1alpha1.PlacementRule) {
	instance.Status.Candidates = candidates
	instance.Status.Eliminators = nil
	instance.Status.Recommendations = nil
}

func (d *DefaultDecisionMaker) ContinueDecisionMakingProcess(instance *corev1alpha1.PlacementRule) bool {
	decisions := d.filterByAdvisorType(instance.Status.Candidates, instance.Spec.Advisors, instance.Status.Recommendations, corev1alpha1.AdvisorTypePredicate)

	if len(decisions) == 0 {
		if len(instance.Status.Decisions) > 0 {
			instance.Status.Decisions = nil
			return true
		}

		return false
	}

	replicas := len(decisions)
	if instance.Spec.Replicas != nil {
		replicas = int(*instance.Spec.Replicas)
	}
	// if valid decision candidates less than target, ignore priority advisors
	if len(decisions) > replicas {
		decisions = d.filterByAdvisorType(decisions, instance.Spec.Advisors, instance.Status.Recommendations, corev1alpha1.AdvisorTypePriority)
	}

	replicas = len(decisions)
	if instance.Spec.Replicas != nil {
		replicas = int(*instance.Spec.Replicas)
	}

	if len(decisions) == replicas || len(instance.Status.Candidates) <= replicas {
		return d.checkAndSetDecisions(decisions, instance)
	}

	d.reduceCandidates(instance)

	klog.Info("New Status: ", instance.Status)

	return true
}

func (d *DefaultDecisionMaker) filterByAdvisorType(candidates []corev1.ObjectReference,
	advisors []corev1alpha1.Advisor, recommendations map[string]corev1alpha1.Recommendation, advtype corev1alpha1.AdvisorType) []corev1.ObjectReference {
	decisions := candidates

	for _, adv := range advisors {
		if adv.Type == nil {
			t := corev1alpha1.AdvisorTypePriority
			adv.Type = &t
		}

		if *adv.Type == advtype {
			recmap := make(map[types.UID]bool)
			for _, or := range recommendations[adv.Name] {
				recmap[or.UID] = true
			}

			var newdecisions []corev1.ObjectReference

			for _, or := range decisions {
				if recmap[or.UID] {
					newdecisions = append(newdecisions, *or.DeepCopy())
				}
			}

			decisions = newdecisions
		}
	}

	return decisions
}

func (d *DefaultDecisionMaker) checkAndSetDecisions(decisions []corev1.ObjectReference, instance *corev1alpha1.PlacementRule) bool {
	if advisorutils.EqualDecisions(decisions, instance.Status.Decisions) {
		return false
	}

	instance.Status.Decisions = decisions

	return true
}

func (d *DefaultDecisionMaker) reduceCandidates(instance *corev1alpha1.PlacementRule) {
	// reduce by predicates first
	candidates := d.filterByAdvisorType(instance.Status.Candidates, instance.Spec.Advisors, instance.Status.Recommendations, corev1alpha1.AdvisorTypePredicate)
	cadweightmap := make(map[string]int)

	for _, or := range candidates {
		cadweightmap[advisorutils.GenKey(or)] = 0
	}

	for _, or := range instance.Status.Candidates {
		if _, ok := cadweightmap[advisorutils.GenKey(or)]; !ok {
			instance.Status.Eliminators = append(instance.Status.Eliminators, *or.DeepCopy())
		}
	}

	if len(candidates) < len(instance.Status.Candidates) {
		instance.Status.Candidates = candidates
		instance.Status.Recommendations = nil

		return
	}

	// calculate weight of all candidates
	for _, adv := range instance.Spec.Advisors {
		if adv.Type == nil {
			t := corev1alpha1.AdvisorTypePriority
			adv.Type = &t
		}

		if *adv.Type == corev1alpha1.AdvisorTypePriority {
			rec := instance.Status.Recommendations[adv.Name]
			if len(rec) == 0 {
				return
			}

			weight := corev1alpha1.DefaultAdvisorWeight
			if adv.Weight != nil {
				weight = int(*adv.Weight)
			}

			for _, or := range rec {
				if _, ok := cadweightmap[advisorutils.GenKey(or.ObjectReference)]; ok {
					//scored recommendations
					if or.Score != nil {
						newweight := (*or.Score / 100) * int16(weight)
						cadweightmap[advisorutils.GenKey(or.ObjectReference)] += int(newweight)
					} else {
						cadweightmap[advisorutils.GenKey(or.ObjectReference)] += weight
					}
				}
			}
		}
	}

	weight := corev1alpha1.DefaultDecisionWeight
	if instance.Spec.DecisionWeight != nil {
		weight = int(*instance.Spec.DecisionWeight)
	}

	for _, or := range instance.Status.Decisions {
		if _, ok := cadweightmap[advisorutils.GenKey(or)]; ok {
			cadweightmap[advisorutils.GenKey(or)] += weight
		}
	}

	// reduce lowest n candidates
	eliminationMap := make(map[string]corev1.ObjectReference)
	step := d.calculateStep()

	var newcandidates []corev1.ObjectReference

	for _, or := range instance.Status.Candidates {
		c := *or.DeepCopy()

		if len(eliminationMap) < step {
			eliminationMap[advisorutils.GenKey(c)] = c
			continue
		}

		for k, el := range eliminationMap {
			if cadweightmap[advisorutils.GenKey(el)] > cadweightmap[advisorutils.GenKey(c)] {
				delete(eliminationMap, k)
				eliminationMap[advisorutils.GenKey(c)] = c
				c = el
			}
		}

		newcandidates = append(newcandidates, c)
	}

	instance.Status.Candidates = newcandidates
	instance.Status.Recommendations = nil

	for _, or := range eliminationMap {
		instance.Status.Eliminators = append(instance.Status.Eliminators, *or.DeepCopy())
	}
}

func (d *DefaultDecisionMaker) calculateStep() int {
	return defaultStep
}
