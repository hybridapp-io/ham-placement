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

import corev1 "k8s.io/api/core/v1"

func GenKey(or corev1.ObjectReference) string {
	return string(or.UID)
}

func EqualCandidates(src, dst []corev1.ObjectReference) bool {
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
