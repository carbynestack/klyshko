/*
Copyright (c) 2022-2023 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TupleTypePolicy specifies the scheduling policy used for a specific tuple type.
type TupleTypePolicy struct {

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Enum=BIT_GFP;BIT_GF2N;INPUT_MASK_GFP;INPUT_MASK_GF2N;INVERSE_TUPLE_GFP;INVERSE_TUPLE_GF2N;SQUARE_TUPLE_GFP;SQUARE_TUPLE_GF2N;MULTIPLICATION_TRIPLE_GFP;MULTIPLICATION_TRIPLE_GF2N
	Type string `json:"type"`

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=true
	Threshold int `json:"threshold"`

	//+kubebuilder:default=1
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=true
	Priority int `json:"priority"`
}

// TupleGenerationSchedulerSpec defines the desired state of a TupleGenerationScheduler.
type TupleGenerationSchedulerSpec struct {

	//+kubebuilder:default=1
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=false
	Concurrency int `json:"concurrency,omitempty"`

	//+kubebuilder:default=600
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=true
	TTLSecondsAfterFinished int `json:"ttlSecondsAfterFinished"`

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinItems=1
	TupleTypePolicies []TupleTypePolicy `json:"policies"`
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
