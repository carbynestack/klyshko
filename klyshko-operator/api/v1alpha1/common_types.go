/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import v1 "k8s.io/api/core/v1"

// GeneratorSpec is a description of a Correlated Randomness Generator.
type GeneratorSpec struct {
	// Container image name.
	Image string `json:"image"`

	// Image pull policy specifies under which circumstances the image is pulled from the registry.
	//+kubebuilder:default=IfNotPresent
	//+kubebuilder:validation:Enum=IfNotPresent;Always;Never
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
}
