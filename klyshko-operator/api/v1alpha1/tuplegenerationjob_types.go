/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TupleGenerationJobSpec struct {
	ID string `json:"id"`

	// +kubebuilder:validation:Enum=multiplicationtriple_gfp
	Type string `json:"type"`

	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=true
	Count int32 `json:"count"`
}

type TupleGenerationJobStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=tgj;tgjob
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Tuple Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="Tuple Count",type=string,JSONPath=`.spec.count`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type TupleGenerationJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TupleGenerationJobSpec   `json:"spec,omitempty"`
	Status TupleGenerationJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type TupleGenerationJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TupleGenerationJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TupleGenerationJob{}, &TupleGenerationJobList{})
}
