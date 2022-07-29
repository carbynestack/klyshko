/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TupleGenerationTaskState string

const (
	TaskLaunching    TupleGenerationTaskState = "Launching"
	TaskGenerating   TupleGenerationTaskState = "Generating"
	TaskProvisioning TupleGenerationTaskState = "Provisioning"
	TaskCompleted    TupleGenerationTaskState = "Completed"
	TaskFailed       TupleGenerationTaskState = "Failed"
)

func (s TupleGenerationTaskState) IsValid() bool {
	switch s {
	case TaskLaunching, TaskGenerating, TaskProvisioning, TaskCompleted, TaskFailed:
		return true
	default:
		return false
	}
}

type TupleGenerationTaskSpec struct {
}

type TupleGenerationTaskStatus struct {
	State TupleGenerationTaskState `json:"state"`
}

func ParseFromJSON(data []byte) (*TupleGenerationTaskStatus, error) {
	status := &TupleGenerationTaskStatus{}
	if err := json.Unmarshal(data, status); err != nil {
		return status, err
	}
	return status, nil
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=tgt;tgtask
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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
