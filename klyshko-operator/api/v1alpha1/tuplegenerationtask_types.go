/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TupleGenerationTaskSpec struct {
}

type TupleGenerationTaskStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=tgt;tgtask
//+kubebuilder:subresource:status

type TupleGenerationTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TupleGenerationTaskSpec   `json:"spec,omitempty"`
	Status TupleGenerationTaskStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type TupleGenerationTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TupleGenerationTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TupleGenerationTask{}, &TupleGenerationTaskList{})
}
