/*
Copyright (c) 2022-2023 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TupleGenerationTaskState encodes the state of a TupleGenerationTask.
type TupleGenerationTaskState string

const (

	// TaskPreparing means that auxiliary resources are being generated.
	TaskPreparing TupleGenerationTaskState = "Preparing"

	// TaskLaunching means that the tuple generation process is being initiated.
	TaskLaunching TupleGenerationTaskState = "Launching"

	// TaskGenerating means that tuples are being generated.
	TaskGenerating TupleGenerationTaskState = "Generating"

	// TaskProvisioning means that tuples are being uploaded to Castor.
	TaskProvisioning TupleGenerationTaskState = "Provisioning"

	// TaskCompleted means that the task has been finished successfully.
	TaskCompleted TupleGenerationTaskState = "Completed"

	// TaskFailed means that an error occurred while performing the task.
	TaskFailed TupleGenerationTaskState = "Failed"
)

// IsValid returns true if state s is among the defined ones and false otherwise.
func (s TupleGenerationTaskState) IsValid() bool {
	switch s {
	case TaskPreparing, TaskLaunching, TaskGenerating, TaskProvisioning, TaskCompleted, TaskFailed:
		return true
	default:
		return false
	}
}

// TupleGenerationTaskSpec defines the desired state of a TupleGenerationTask.
type TupleGenerationTaskSpec struct {
	PlayerID uint `json:"playerId"`
}

// TupleGenerationTaskStatus defines the observed state of a TupleGenerationTask.
type TupleGenerationTaskStatus struct {
	State    TupleGenerationTaskState `json:"state"`
	Endpoint string                   `json:"endpoint,omitempty"`
}

// Unmarshal parses a JSON serialized TupleGenerationTaskStatus.
func Unmarshal(data []byte) (*TupleGenerationTaskStatus, error) {
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
//+kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TupleGenerationTask is the Schema for the TupleGenerationTask API.
type TupleGenerationTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TupleGenerationTaskSpec   `json:"spec,omitempty"`
	Status TupleGenerationTaskStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TupleGenerationTaskList contains a list of TupleGenerationTasks.
type TupleGenerationTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TupleGenerationTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TupleGenerationTask{}, &TupleGenerationTaskList{})
}
