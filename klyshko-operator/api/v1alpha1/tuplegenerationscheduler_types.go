/*
Copyright (c) 2022-2023 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TupleGenerationSchedulerSpec defines the desired state of a TupleGenerationScheduler.
type TupleGenerationSchedulerSpec struct {

	//+kubebuilder:default=1
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=false
	Concurrency int `json:"concurrency,omitempty"`

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=true
	Threshold int `json:"threshold"`

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=true
	TuplesPerJob int `json:"tuplesPerJob"`

	//+kubebuilder:default=600
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=true
	TTLSecondsAfterFinished int `json:"ttlSecondsAfterFinished"`

	Generator GeneratorSpec `json:"generator"`
}

// TupleGenerationSchedulerStatus defines the observed state of a TupleGenerationScheduler.
type TupleGenerationSchedulerStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=tgs;tgscheduler
//+kubebuilder:subresource:status

// TupleGenerationScheduler is the Schema for the TupleGenerationScheduler API.
type TupleGenerationScheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TupleGenerationSchedulerSpec   `json:"spec,omitempty"`
	Status TupleGenerationSchedulerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TupleGenerationSchedulerList contains a list of TupleGenerationSchedulers.
type TupleGenerationSchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TupleGenerationScheduler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TupleGenerationScheduler{}, &TupleGenerationSchedulerList{})
}
