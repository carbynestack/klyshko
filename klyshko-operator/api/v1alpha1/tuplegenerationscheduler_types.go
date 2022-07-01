/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TupleGenerationSchedulerSpec struct {
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=false
	Concurrency int `json:"concurrency"`

	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=true
	Threshold int `json:"threshold"`
}

type TupleGenerationSchedulerStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type TupleGenerationScheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TupleGenerationSchedulerSpec   `json:"spec,omitempty"`
	Status TupleGenerationSchedulerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type TupleGenerationSchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TupleGenerationScheduler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TupleGenerationScheduler{}, &TupleGenerationSchedulerList{})
}
