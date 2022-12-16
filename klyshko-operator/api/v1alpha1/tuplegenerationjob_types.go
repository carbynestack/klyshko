/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TupleGenerationJobState encodes the state of a TupleGenerationJob.
type TupleGenerationJobState string

const (
	// JobPending means that not all tasks of the job have been spawned yet.
	JobPending TupleGenerationJobState = "Pending"

	// JobRunning means all tasks for the job have been spawned but have not terminated yet.
	JobRunning TupleGenerationJobState = "Running"

	// JobCompleted means all tasks have completed successfully.
	JobCompleted TupleGenerationJobState = "Completed"

	// JobFailed means that all tasks for the job have terminated but at least on failed.
	JobFailed TupleGenerationJobState = "Failed"
)

// IsValid returns true if state s is among the defined ones and false otherwise.
func (s TupleGenerationJobState) IsValid() bool {
	switch s {
	case JobPending, JobRunning, JobCompleted, JobFailed:
		return true
	default:
		return false
	}
}

// IsDone returns true if s is among the set of TupleGenerationJobState that describe a job that is done, i.e., is
// either JobCompleted or JobFailed, and false otherwise.
func (s TupleGenerationJobState) IsDone() bool {
	return s == JobCompleted || s == JobFailed
}

// TupleGenerationJobSpec defines the desired state of a TupleGenerationJob.
type TupleGenerationJobSpec struct {
	ID string `json:"id"`

	// +kubebuilder:validation:Enum=BIT_GFP;BIT_GF2N;INPUT_MASK_GFP;INPUT_MASK_GF2N;INVERSE_TUPLE_GFP;INVERSE_TUPLE_GF2N;SQUARE_TUPLE_GFP;SQUARE_TUPLE_GF2N;MULTIPLICATION_TRIPLE_GFP;MULTIPLICATION_TRIPLE_GF2N
	Type string `json:"type"`

	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:ExclusiveMinimum=true
	Count int `json:"count"`

	Generator GeneratorSpec `json:"generator"`
}

// TupleGenerationJobStatus defines the observed state of a TupleGenerationJob.
type TupleGenerationJobStatus struct {
	State                   TupleGenerationJobState `json:"state"`
	LastStateTransitionTime metav1.Time             `json:"lastStateTransitionTime"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=tgj;tgjob
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Tuple Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="Tuple Count",type=string,JSONPath=`.spec.count`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TupleGenerationJob is the Schema for the TupleGenerationJob API.
type TupleGenerationJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TupleGenerationJobSpec   `json:"spec,omitempty"`
	Status TupleGenerationJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TupleGenerationJobList contains a list of TupleGenerationJobs.
type TupleGenerationJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TupleGenerationJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TupleGenerationJob{}, &TupleGenerationJobList{})
}
