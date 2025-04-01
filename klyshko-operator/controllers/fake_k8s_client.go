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
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewFakeK8sReader creates a new FakeK8sReader with default failing behavior.
func NewFakeK8sReader() *FakeK8sReader {
	return (&FakeK8sReader{}).Reset()
}

// FakeK8sReader is a fake implementation of the K8sReader interface.
// The Get method supports only the types unstructured.Unstructured and v1.Service.
// The List method supports unstructured.UnstructuredList only.
type FakeK8sReader struct {
	// GetReturnObject is the object that will be returned by the Get method.
	// Get will not type check the object, so it is the caller's responsibility.
	GetReturnObject client.Object
	GetReturnError  error
	// ListReturnObject is the object that will be returned by the List method.
	// List will not type check the object, so it is the caller's responsibility.
	ListReturnObject client.ObjectList
	ListReturnError  error
}

func (fkr *FakeK8sReader) Get(_ context.Context, _ client.ObjectKey, obj client.Object) error {
	if fkr.GetReturnError != nil {
		return fkr.GetReturnError
	}
	switch obj.(type) {
	case *unstructured.Unstructured:
		if fkr.GetReturnObject != nil {
			fkr.GetReturnObject.(*unstructured.Unstructured).DeepCopyInto(obj.(*unstructured.Unstructured))
		}
	case *v1.Service:
		if fkr.GetReturnObject != nil {
			fkr.GetReturnObject.(*v1.Service).DeepCopyInto(obj.(*v1.Service))
		}
	default:
		return fmt.Errorf("Fake8sReader: unsupported object type %T", obj)
	}
	return fkr.GetReturnError
}

func (fkr *FakeK8sReader) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	if fkr.ListReturnObject != nil {
		fkr.ListReturnObject.(*unstructured.UnstructuredList).DeepCopyInto(list.(*unstructured.UnstructuredList))
	}
	return fkr.ListReturnError
}

// Reset resets the FakeK8sReader to its default failing behavior.
func (fkr *FakeK8sReader) Reset() *FakeK8sReader {
	fkr.GetReturnObject = nil
	fkr.GetReturnError = errors.New("fake object not found")
	fkr.ListReturnObject = nil
	fkr.ListReturnError = errors.New("fake list not found")
	return fkr
}

// NewFakeK8sWriter creates a new FakeK8sWriter with default failing behavior.
func NewFakeK8sWriter() *FakeK8sWriter {
	return (&FakeK8sWriter{}).Reset()
}

type CreateParams struct {
	obj     client.Object
	options []client.CreateOption
}

type UpdateParams struct {
	obj     client.Object
	options []client.UpdateOption
}

type DeleteParams struct {
	obj     client.Object
	options []client.DeleteOption
}

type PatchParams struct {
	obj     client.Object
	patch   client.Patch
	options []client.PatchOption
}

type DeleteAllOfParams struct {
	obj     client.Object
	options []client.DeleteAllOfOption
}

type FakeK8sWriter struct {
	CreateReturnError    error
	CreateCallParams     []CreateParams
	UpdateReturnError    error
	UpdateCallParams     []UpdateParams
	DeleteReturnError    error
	DeleteCallParams     []DeleteParams
	PatchReturnError     error
	PatchCallParams      []PatchParams
	DeleteAllReturnError error
	DeleteAllCallParams  []DeleteAllOfParams
}

func (fkw *FakeK8sWriter) Create(ctx context.Context, obj client.Object, options ...client.CreateOption) error {
	fkw.CreateCallParams = append(fkw.CreateCallParams, CreateParams{obj, options})
	return fkw.CreateReturnError
}

func (fkw *FakeK8sWriter) Delete(ctx context.Context, obj client.Object, Params ...client.DeleteOption) error {
	fkw.DeleteCallParams = append(fkw.DeleteCallParams, DeleteParams{obj, Params})
	return fkw.DeleteReturnError
}

func (fkw *FakeK8sWriter) Update(ctx context.Context, obj client.Object, options ...client.UpdateOption) error {
	fkw.UpdateCallParams = append(fkw.UpdateCallParams, UpdateParams{obj, options})
	return fkw.UpdateReturnError
}

func (fkw *FakeK8sWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, options ...client.PatchOption) error {
	fkw.PatchCallParams = append(fkw.PatchCallParams, PatchParams{obj, patch, options})
	return fkw.PatchReturnError
}
func (fkw *FakeK8sWriter) DeleteAllOf(ctx context.Context, obj client.Object, options ...client.DeleteAllOfOption) error {
	fkw.DeleteAllCallParams = append(fkw.DeleteAllCallParams, DeleteAllOfParams{obj, options})
	return fkw.DeleteAllReturnError
}

// Reset resets the FakeK8sWriter to its default failing behavior.
func (fkw *FakeK8sWriter) Reset() *FakeK8sWriter {
	fkw.CreateReturnError = errors.New("fake create failed")
	fkw.UpdateReturnError = errors.New("fake update failed")
	fkw.DeleteReturnError = errors.New("fake delete failed")
	fkw.PatchReturnError = errors.New("fake patch failed")
	fkw.DeleteAllReturnError = errors.New("fake delete all failed")
	fkw.CreateCallParams = make([]CreateParams, 0)
	fkw.UpdateCallParams = make([]UpdateParams, 0)
	fkw.DeleteCallParams = make([]DeleteParams, 0)
	fkw.PatchCallParams = make([]PatchParams, 0)
	fkw.DeleteAllCallParams = make([]DeleteAllOfParams, 0)
	return fkw
}

// NewFakeK8sClient creates a new FakeK8sStatusClient with default failing behavior.
func NewFakeK8sClient() *FakeK8sClient {
	return (&FakeK8sClient{
		NewFakeK8sReader(),
		NewFakeK8sWriter(),
		nil,
	}).Reset()
}

type FakeK8sClient struct {
	*FakeK8sReader
	*FakeK8sWriter
	client.StatusClient
}

func (fkc *FakeK8sClient) Scheme() *runtime.Scheme {
	return nil
}

func (fkc *FakeK8sClient) RESTMapper() meta.RESTMapper {
	return nil
}

// Reset resets the FakeK8sClient to its default failing behavior.
func (fkc *FakeK8sClient) Reset() *FakeK8sClient {
	fkc.FakeK8sReader = fkc.FakeK8sReader.Reset()
	fkc.FakeK8sWriter = fkc.FakeK8sWriter.Reset()
	return fkc
}
