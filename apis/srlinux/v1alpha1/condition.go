/*
Copyright 2023 Nokia.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package v1alpha1

import (
	resourcev1alpha1 "github.com/nokia/k8s-ipam/apis/resource/common/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A ConditionType represents a condition type for a given KRM resource
type ConditionType string

// Condition Types.
const (
	// ConditionTypeReady represents the resource ready condition
	ConditionTypeReady ConditionType = "Ready"
)

// A ConditionReason represents the reason a resource is in a condition.
type ConditionReason string

// Reasons a resource is ready or not
const (
	ConditionReasonReady   ConditionReason = "Ready"
	ConditionReasonFailed  ConditionReason = "Failed"
	ConditionReasonUnknown ConditionReason = "Unknown"
)

// Ready returns a condition that indicates the resource is
// ready for use.
func Ready() resourcev1alpha1.Condition {
	return resourcev1alpha1.Condition{Condition: metav1.Condition{
		Type:               string(ConditionTypeReady),
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             string(ConditionReasonReady),
	}}
}

// Unknown returns a condition that indicates the resource is in an
// unknown status.
func Unknown() resourcev1alpha1.Condition {
	return resourcev1alpha1.Condition{Condition: metav1.Condition{
		Type:               string(ConditionTypeReady),
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             string(ConditionReasonUnknown),
	}}
}

// Failed returns a condition that indicates the resource
// failed to get reconciled.
func Failed(msg string) resourcev1alpha1.Condition {
	return resourcev1alpha1.Condition{Condition: metav1.Condition{
		Type:               string(ConditionTypeReady),
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             string(ConditionReasonFailed),
		Message:            msg,
	}}
}
