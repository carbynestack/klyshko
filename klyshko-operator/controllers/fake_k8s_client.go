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
	"github.com/carbynestack/klyshko/api/v1alpha1"
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

type GetObjectResponse struct {
	// Object is the object that will be returned by the Get method.
	// Get will not type check the object, so it is the caller's responsibility.
	Object client.Object
	// Error is the error that will be returned by the Get method.
	Error error
}

type ListObjectResponse struct {
	// List is the object that will be returned by the List method.
	// List will not type check the object, so it is the caller's responsibility.
	List client.ObjectList
	// Error is the error that will be returned by the List method.
	Error error
}

// FakeK8sReader is a fake implementation of the K8sReader interface.
// The Get method supports only the types v1.Pod, v1.ConfigMap,
// v1alpha1.TupleGenerationTask and v1alpha1.TupleGenerator.
// The List method supports unstructured.UnstructuredList only.
// FakeK8sReader is not thread-safe.
type FakeK8sReader struct {
	getResponses  []GetObjectResponse
	listResponses []ListObjectResponse
}

func (fkr *FakeK8sReader) Get(_ context.Context, _ client.ObjectKey, obj client.Object) error {
	if len(fkr.getResponses) == 0 {
		return errors.New("fake object not found")
	}
	getResponse := fkr.getResponses[0]
	if len(fkr.getResponses) > 1 {
		fkr.getResponses = fkr.getResponses[1:]
	}
	if getResponse.Object != nil {
		switch obj.(type) {
		case *unstructured.Unstructured:
			getResponse.Object.(*unstructured.Unstructured).DeepCopyInto(obj.(*unstructured.Unstructured))
		case *v1.ConfigMap:
			getResponse.Object.(*v1.ConfigMap).DeepCopyInto(obj.(*v1.ConfigMap))
		case *v1alpha1.TupleGenerationTask:
			getResponse.Object.(*v1alpha1.TupleGenerationTask).DeepCopyInto(obj.(*v1alpha1.TupleGenerationTask))
		case *v1.Pod:
			getResponse.Object.(*v1.Pod).DeepCopyInto(obj.(*v1.Pod))
		case *v1alpha1.TupleGenerator:
			getResponse.Object.(*v1alpha1.TupleGenerator).DeepCopyInto(obj.(*v1alpha1.TupleGenerator))
			break
		default:
			return fmt.Errorf("Fake8sReader: unsupported object type %T", obj)
		}
	}
	return getResponse.Error
}

func (fkr *FakeK8sReader) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	if len(fkr.listResponses) == 0 {
		return errors.New("fake list not found")
	}
	listResponse := fkr.listResponses[0]
	if len(fkr.listResponses) > 1 {
		fkr.listResponses = fkr.listResponses[1:]
	}
	if listResponse.List != nil {
		listResponse.List.(*unstructured.UnstructuredList).DeepCopyInto(list.(*unstructured.UnstructuredList))
	}
	return listResponse.Error
}

// Reset resets the FakeK8sReader to its default failing behavior.
func (fkr *FakeK8sReader) Reset() *FakeK8sReader {
	fkr.getResponses = make([]GetObjectResponse, 0)
	fkr.listResponses = make([]ListObjectResponse, 0)
	return fkr
}

// AddGetResponse adds a GetObjectResponse to the FakeK8sReader that will
// be returned by Get. If multiple responses are added, they will be returned in
// order, and then the last response will be repeated.
// This method is not thread-safe.
func (fkr *FakeK8sReader) AddGetResponse(getResponse GetObjectResponse) *FakeK8sReader {
	fkr.getResponses = append(fkr.getResponses, getResponse)
	return fkr
}

// AddListResponse adds a ListObjectResponse to the FakeK8sReader that will
// be returned by List. If multiple responses are added, they will be returned in
// order, and then the last response will be repeated.
// This method is not thread-safe.
func (fkr *FakeK8sReader) AddListResponse(listResponse ListObjectResponse) *FakeK8sReader {
	fkr.listResponses = append(fkr.listResponses, listResponse)
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

func (fkw *FakeK8sWriter) Create(_ context.Context, obj client.Object, options ...client.CreateOption) error {
	fkw.CreateCallParams = append(fkw.CreateCallParams, CreateParams{obj, options})
	return fkw.CreateReturnError
}

func (fkw *FakeK8sWriter) Delete(_ context.Context, obj client.Object, Params ...client.DeleteOption) error {
	fkw.DeleteCallParams = append(fkw.DeleteCallParams, DeleteParams{obj, Params})
	return fkw.DeleteReturnError
}

func (fkw *FakeK8sWriter) Update(_ context.Context, obj client.Object, options ...client.UpdateOption) error {
	fkw.UpdateCallParams = append(fkw.UpdateCallParams, UpdateParams{obj, options})
	return fkw.UpdateReturnError
}

func (fkw *FakeK8sWriter) Patch(_ context.Context, obj client.Object, patch client.Patch, options ...client.PatchOption) error {
	fkw.PatchCallParams = append(fkw.PatchCallParams, PatchParams{obj, patch, options})
	return fkw.PatchReturnError
}
func (fkw *FakeK8sWriter) DeleteAllOf(_ context.Context, obj client.Object, options ...client.DeleteAllOfOption) error {
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
