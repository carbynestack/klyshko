/*
Copyright (c) 2023 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TupleTypeSpec declares a tuple type that a generator can generate. It also specifies a batch size that is used by the
// scheduler to decide how many tuples to generate in a single tuple generation job. The batch size should be selected
// on the one hand to be big enough to avoid "trashing", i.e., the situation where a lot of very short running jobs are
// generated, and on the other hand small enough to avoid starvation for other tuple types due to very long job
// runtimes.
type TupleTypeSpec struct {

	// +kubebuilder:validation:Enum=BIT_GFP;BIT_GF2N;INPUT_MASK_GFP;INPUT_MASK_GF2N;INVERSE_TUPLE_GFP;INVERSE_TUPLE_GF2N;SQUARE_TUPLE_GFP;SQUARE_TUPLE_GF2N;MULTIPLICATION_TRIPLE_GFP;MULTIPLICATION_TRIPLE_GF2N
	Type string `json:"type"`

	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=true
	BatchSize int `json:"batchSize"`
}

// TupleGeneratorSpec defines the desired state of TupleGenerator.
type TupleGeneratorSpec struct {
	GeneratorSpec `json:"generator"`

	//+kubebuilder:validation:MinItems=1
	Supports []TupleTypeSpec `json:"supports"`
}

// GetTupleTypeSpec performs a lookup for the given tuple type in the array of supported tuple types.
func (s *TupleGeneratorSpec) GetTupleTypeSpec(tupleType string) *TupleTypeSpec {
	for _, support := range s.Supports {
		if support.Type == tupleType {
			return &support
		}
	}
	return nil
}

// TupleGeneratorStatus defines the observed state of TupleGenerator.
type TupleGeneratorStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=tg;tgenerator
//+kubebuilder:subresource:status

// TupleGenerator is the Schema for the tuplegenerators API.
type TupleGenerator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TupleGeneratorSpec   `json:"spec,omitempty"`
	Status TupleGeneratorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TupleGeneratorList contains a list of TupleGenerator.
type TupleGeneratorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TupleGenerator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TupleGenerator{}, &TupleGeneratorList{})
}
