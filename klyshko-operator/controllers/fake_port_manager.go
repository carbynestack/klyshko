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
)

func NewFakeUsedPortSupplier() *FakeUsedPortSupplier {
	return &FakeUsedPortSupplier{}
}

type FakeUsedPortSupplier struct {
	GetUsedPortsReturnList []uint32
	GetUsedPortsReturnErr  error
}

func (fups *FakeUsedPortSupplier) getUsedPortsInRange(_ context.Context) ([]uint32, error) {
	return fups.GetUsedPortsReturnList, fups.GetUsedPortsReturnErr
}

// Reset resets the FakeUsedPortSupplier to its default failing behavior.
func (fups *FakeUsedPortSupplier) Reset() *FakeUsedPortSupplier {
	fups.GetUsedPortsReturnList = []uint32{}
	fups.GetUsedPortsReturnErr = errors.New("fake returned error")
	return fups
}

// NewFakePortManager creates a new FakePortManager with default failing behavior.
func NewFakePortManager() *FakePortManager {
	return (&FakePortManager{}).Reset()
}

type FreePortResponse struct {
	Port  uint32
	Error error
}

type FakePortManager struct {
	freePortResponses []FreePortResponse
}

func (fpm *FakePortManager) GetFreePort(_ context.Context) (uint32, error) {
	if len(fpm.freePortResponses) == 0 {
		return 0, errors.New("fake port manager not initialized")
	}
	if len(fpm.freePortResponses) == 1 {
		return fpm.freePortResponses[0].Port, fpm.freePortResponses[0].Error
	}
	port, err := fpm.freePortResponses[0].Port, fpm.freePortResponses[0].Error
	fpm.freePortResponses = fpm.freePortResponses[1:]

	return port, err
}

// ReturnOnGetFreePort adds a FreePortResponse to the list of responses that will
// be returned by GetFreePort. If the list is empty, the default failing behavior
// will be used. If multiple responses are added, they will be returned in order,
// and then the last response will be repeated.
// This method is not thread-safe.
func (fpm *FakePortManager) ReturnOnGetFreePort(response FreePortResponse) *FakePortManager {
	fpm.freePortResponses = append(fpm.freePortResponses, response)
	return fpm
}

func (fpm *FakePortManager) Reset() *FakePortManager {
	fpm.freePortResponses = make([]FreePortResponse, 0)
	return fpm
}
