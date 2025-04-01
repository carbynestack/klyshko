/*
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"context"
	"errors"
	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
)

// NewFakeNetworkManager creates a new FakeNetworkManager with default failing behavior.
func NewFakeNetworkManager() *FakeNetworkManager {
	return (&FakeNetworkManager{}).Reset()
}

type FakeNetworkManager struct {
	createIngressReturnPort uint32
	createIngressReturnErr  error
	createEgressReturnErr   error
	deleteReturnErr         error
}

func (hnm *FakeNetworkManager) CreateIngressNetworkingForTask(_ context.Context, _ *klyshkov1alpha1.TupleGenerationTask) (uint32, error) {
	return hnm.createIngressReturnPort, hnm.createIngressReturnErr
}

func (hnm *FakeNetworkManager) CreateEgressNetworkingForTask(_ context.Context, _ *klyshkov1alpha1.TupleGenerationTask, _ map[uint]string) error {
	return hnm.createEgressReturnErr
}

func (hnm *FakeNetworkManager) DeleteNetworkingForTask(_ context.Context, _ *klyshkov1alpha1.TupleGenerationTask) error {
	return hnm.deleteReturnErr
}

func (hnm *FakeNetworkManager) DoReturnOnCreateIngressNetworkingForTask(port uint32, err error) *FakeNetworkManager {
	hnm.createIngressReturnPort = port
	hnm.createIngressReturnErr = err
	return hnm
}

func (hnm *FakeNetworkManager) DoReturnOnCreateEgressNetworkingForTask(err error) *FakeNetworkManager {
	hnm.createEgressReturnErr = err
	return hnm
}

func (hnm *FakeNetworkManager) DoReturnOnDeleteNetworkingForTask(err error) *FakeNetworkManager {
	hnm.deleteReturnErr = err
	return hnm
}

// Reset resets the FakeNetworkManager to its default failing behavior.
func (hnm *FakeNetworkManager) Reset() *FakeNetworkManager {
	hnm.createIngressReturnPort = 0
	hnm.createIngressReturnErr = errors.New("fake ingress not created")
	hnm.createEgressReturnErr = errors.New("fake egress not created")
	hnm.deleteReturnErr = errors.New("fake networking not deleted")
	return hnm
}
