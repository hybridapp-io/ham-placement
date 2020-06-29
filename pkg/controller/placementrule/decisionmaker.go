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
	"k8s.io/klog"

	advisorutils "github.com/hybridapp-io/ham-placement/pkg/advisor/utils"
	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
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
	decisions := d.filteByAdvisorType(instance.Status.Candidates, instance.Spec.Advisors, instance.Status.Recommendations, corev1alpha1.AdvisorTypePredicate)

	klog.Info("Decision fileter: ", decisions)

	replicas := len(decisions)
	if instance.Spec.Replicas != nil {
		replicas = int(*instance.Spec.Replicas)
	}
	// if valid decision candidates less than target, ignore priority advisors
	if len(decisions) > replicas {
		decisions = d.filteByAdvisorType(decisions, instance.Spec.Advisors, instance.Status.Recommendations, corev1alpha1.AdvisorTypePriority)
	}

	replicas = len(decisions)
	if instance.Spec.Replicas != nil {
		replicas = int(*instance.Spec.Replicas)
	}

	klog.Info("Decision prioritized: ", decisions)

	if len(decisions) == replicas || len(instance.Status.Candidates) <= replicas {
		return d.checkAndSetDecisions(decisions, instance)
	}

	d.reduceCandidates(instance)

	klog.Info("New Status: ", instance.Status)

	return true
}

func (d *DefaultDecisionMaker) filteByAdvisorType(candidates []corev1.ObjectReference,
	advisors []corev1alpha1.Advisor, recommendations map[string]corev1alpha1.Recommendation, advtype corev1alpha1.AdvisorType) []corev1.ObjectReference {
	var decisions []corev1.ObjectReference

	hasPredicates := false

	cadmap := make(map[string]corev1.ObjectReference)
	for _, or := range candidates {
		cadmap[advisorutils.GenKey(or)] = or
	}

	for _, adv := range advisors {
		if adv.Type == nil {
			t := corev1alpha1.AdvisorTypePriority
			adv.Type = &t
		}

		if *adv.Type == advtype {
			hasPredicates = true
			rec := recommendations[adv.Name]

			if len(rec) == 0 {
				return nil
			}

			for _, or := range rec {
				if _, ok := cadmap[advisorutils.GenKey(or)]; ok {
					decisions = append(decisions, *or.DeepCopy())
				}
			}
		}
	}

	if !hasPredicates {
		decisions = candidates
	}

	return decisions
}

func (d *DefaultDecisionMaker) checkAndSetDecisions(decisions []corev1.ObjectReference, instance *corev1alpha1.PlacementRule) bool {
	if advisorutils.EqualCandidates(decisions, instance.Status.Decisions) {
		return false
	}

	instance.Status.Decisions = decisions

	return true
}

func (d *DefaultDecisionMaker) reduceCandidates(instance *corev1alpha1.PlacementRule) {
	// reduce by predicates first
	candidates := d.filteByAdvisorType(instance.Status.Candidates, instance.Spec.Advisors, instance.Status.Recommendations, corev1alpha1.AdvisorTypePredicate)
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
				if _, ok := cadweightmap[advisorutils.GenKey(or)]; ok {
					cadweightmap[advisorutils.GenKey(or)] += weight
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
	elmap := make(map[string]corev1.ObjectReference)
	step := d.calculateStep()

	var newcandidates []corev1.ObjectReference

	for _, or := range instance.Status.Candidates {
		c := *or.DeepCopy()

		if len(elmap) < step {
			elmap[advisorutils.GenKey(c)] = c
			continue
		}

		for k, el := range elmap {
			if cadweightmap[advisorutils.GenKey(el)] > cadweightmap[advisorutils.GenKey(c)] {
				delete(elmap, k)
				elmap[advisorutils.GenKey(c)] = c
				c = el
			}
		}

		newcandidates = append(newcandidates, c)
	}

	instance.Status.Candidates = newcandidates
	instance.Status.Recommendations = nil

	for _, or := range elmap {
		instance.Status.Eliminators = append(instance.Status.Eliminators, *or.DeepCopy())
	}
}

func (d *DefaultDecisionMaker) calculateStep() int {
	return 1
}
