/*
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"encoding/json"
	"istio.io/api/networking/v1beta1"
	isec "istio.io/api/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const istioNetworkingGroup = "networking.istio.io"
const istioSecurityGroup = "security.istio.io"
const istioVersion = "v1beta1"
const istioNetworkingApiVersion = istioNetworkingGroup + "/" + istioVersion
const istioSecurityApiVersion = istioSecurityGroup + "/" + istioVersion

// This file contains the definition of the istio k8s resources managed by the
// Klyshko operator. This includes the Gateway, AuthorizationPolicy,
// VirtualService, DestinationRule and ServiceEntry resources. However, the
// resources only contain the fields that are relevant to the operator.
//
// This is a hacky workaround to avoid using the Istio client-go library, which
// causes compatible issues with dependencies of other libraries. When
// interacting with the Istio resources, the operator will use the unstructured
// k8s client as resources are not defined in the right package and therefore
// will not be registered with the required GroupVersionKind.

func NewIstioGateway(meta metav1.ObjectMeta, spec *v1beta1.Gateway) IstioGateway {
	return IstioGateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: istioNetworkingApiVersion,
		},
		ObjectMeta: meta,
		Spec:       spec,
	}
}

func NewUnstructuredIstioGateway() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": istioNetworkingApiVersion,
			"kind":       "Gateway",
		},
	}
}

func IstioGatewayFromUnstructured(ugw *unstructured.Unstructured) (*IstioGateway, error) {
	igw := &IstioGateway{}
	return igw, runtime.DefaultUnstructuredConverter.FromUnstructured(ugw.Object, igw)
}

type IstioGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              *v1beta1.Gateway `json:"spec,omitempty"`
}

func NewIstioAuthorizationPolicy(meta metav1.ObjectMeta, spec *isec.AuthorizationPolicy) IstioAuthorizationPolicy {
	return IstioAuthorizationPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AuthorizationPolicy",
			APIVersion: istioSecurityApiVersion,
		},
		ObjectMeta: meta,
		Spec:       spec,
	}
}

func NewUnstructuredIstioAuthorizationPolicy() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": istioSecurityApiVersion,
			"kind":       "AuthorizationPolicy",
		},
	}
}

func IstioAuthorizationPolicyFromUnstructured(uap *unstructured.Unstructured) (*IstioAuthorizationPolicy, error) {
	iap := &IstioAuthorizationPolicy{}
	return iap, runtime.DefaultUnstructuredConverter.FromUnstructured(uap.Object, iap)
}

type IstioAuthorizationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              *isec.AuthorizationPolicy `json:"spec,omitempty"`
}

func NewIstioVirtualService(meta metav1.ObjectMeta, spec *v1beta1.VirtualService) IstioVirtualService {
	return IstioVirtualService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: istioNetworkingApiVersion,
		},
		ObjectMeta: meta,
		Spec:       spec,
	}
}

func NewUnstructuredIstioVirtualService() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": istioNetworkingApiVersion,
			"kind":       "VirtualService",
		},
	}
}

func IstioVirtualServiceFromUnstructured(uvs *unstructured.Unstructured) (*IstioVirtualService, error) {
	ivs := &IstioVirtualService{}
	return ivs, runtime.DefaultUnstructuredConverter.FromUnstructured(uvs.Object, ivs)
}

type IstioVirtualService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              *v1beta1.VirtualService `json:"spec,omitempty"`
}

func NewUnstructuredIstioDestinationRule() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": istioNetworkingApiVersion,
			"kind":       "DestinationRule",
		},
	}
}

func IstioDestinationRuleFromUnstructured(udr *unstructured.Unstructured) (*IstioDestinationRule, error) {
	idr := &IstioDestinationRule{}
	return idr, runtime.DefaultUnstructuredConverter.FromUnstructured(udr.Object, idr)
}

func NewIstioDestinationRule(meta metav1.ObjectMeta, destinationRule *v1beta1.DestinationRule) IstioDestinationRule {
	return IstioDestinationRule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DestinationRule",
			APIVersion: istioNetworkingApiVersion,
		},
		ObjectMeta: meta,
		Spec:       destinationRule,
	}
}

type IstioDestinationRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              *v1beta1.DestinationRule `json:"spec,omitempty"`
}

func NewUnstructuredIstioServiceEntry() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": istioNetworkingApiVersion,
			"kind":       "ServiceEntry",
		},
	}
}

func IstioServiceEntryFromUnstructured(use *unstructured.Unstructured) (*IstioServiceEntry, error) {
	ise := &IstioServiceEntry{}
	return ise, runtime.DefaultUnstructuredConverter.FromUnstructured(use.Object, ise)
}

func NewIstioServiceEntry(meta metav1.ObjectMeta, spec *v1beta1.ServiceEntry) IstioServiceEntry {
	return IstioServiceEntry{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceEntry",
			APIVersion: istioNetworkingApiVersion,
		},
		ObjectMeta: meta,
		Spec:       spec,
	}
}

type IstioServiceEntry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              *v1beta1.ServiceEntry `json:"spec,omitempty"`
}

func InterfaceToUnstructured(igw interface{}) *unstructured.Unstructured {
	var obj map[string]interface{}
	data, _ := json.Marshal(igw)
	_ = json.Unmarshal(data, &obj)
	return &unstructured.Unstructured{
		Object: obj,
	}
}
