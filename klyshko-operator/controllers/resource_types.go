/*
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"istio.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const istioApiVersion = "networking.istio.io/v1beta1"

// This file contains the definition of the istio k8s resources managed by the
// Klyshko operator. This includes the Gateway, VirtualService, DestinationRule
// and ServiceEntry resources. However, the resources only contain the fields
// that are relevant to the operator.
// This is a hacky workaround to avoid using the Istio client-go library, which
// causes compatible issues with dependencies of other libraries.

func NewIstioGateway(meta metav1.ObjectMeta, spec *v1beta1.Gateway) IstioGateway {
	return IstioGateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: istioApiVersion,
		},
		ObjectMeta: meta,
		Spec:       spec,
	}
}

type IstioGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" `
	Spec              *v1beta1.Gateway `json:"spec,omitempty"`
}

func (g *IstioGateway) DeepCopyObject() runtime.Object {
	if g == nil {
		return nil
	}
	ng := new(IstioGateway)
	*ng = *g
	ng.TypeMeta = g.TypeMeta
	ng.ObjectMeta = g.ObjectMeta
	ng.Spec = g.Spec.DeepCopy()
	return ng
}

func NewIstioVirtualService(meta metav1.ObjectMeta, spec *v1beta1.VirtualService) IstioVirtualService {
	return IstioVirtualService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: istioApiVersion,
		},
		ObjectMeta: meta,
		Spec:       spec,
	}
}

type IstioVirtualService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              *v1beta1.VirtualService `json:"spec,omitempty"`
}

func (vs *IstioVirtualService) DeepCopyObject() runtime.Object {
	if vs == nil {
		return nil
	}
	nvs := new(IstioVirtualService)
	*nvs = *vs
	nvs.TypeMeta = vs.TypeMeta
	nvs.ObjectMeta = vs.ObjectMeta
	if vs.Spec != nil {
		nvs.Spec = vs.Spec.DeepCopy()
	}
	return nvs
}

func NewIstioDestinationRule(meta metav1.ObjectMeta, destinationRule *v1beta1.DestinationRule) IstioDestinationRule {
	return IstioDestinationRule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DestinationRule",
			APIVersion: istioApiVersion,
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

func (dr *IstioDestinationRule) DeepCopyObject() runtime.Object {
	if dr == nil {
		return nil
	}
	ndr := new(IstioDestinationRule)
	*ndr = *dr
	ndr.TypeMeta = dr.TypeMeta
	ndr.ObjectMeta = dr.ObjectMeta
	if dr.Spec != nil {
		ndr.Spec = dr.Spec.DeepCopy()
	}
	return ndr
}

func NewIstioServiceEntry(meta metav1.ObjectMeta, spec *v1beta1.ServiceEntry) IstioServiceEntry {
	return IstioServiceEntry{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceEntry",
			APIVersion: istioApiVersion,
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

func (se *IstioServiceEntry) DeepCopyObject() runtime.Object {
	if se == nil {
		return nil
	}
	nse := new(IstioServiceEntry)
	*nse = *se
	nse.TypeMeta = se.TypeMeta
	nse.ObjectMeta = se.ObjectMeta
	if se.Spec != nil {
		nse.Spec = se.Spec.DeepCopy()
	}
	return nse
}

type IstioGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IstioGateway `json:"items"`
}

func (gl *IstioGatewayList) DeepCopyObject() runtime.Object {
	if gl == nil {
		return nil
	}
	ngl := new(IstioGatewayList)
	*ngl = *gl
	ngl.TypeMeta = gl.TypeMeta
	ngl.ListMeta = gl.ListMeta
	ngl.Items = make([]IstioGateway, len(gl.Items))
	for i := range gl.Items {
		ngl.Items[i] = *gl.Items[i].DeepCopyObject().(*IstioGateway)
	}
	return ngl
}

type IstioVirtualServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IstioVirtualService `json:"items"`
}

func (vsl *IstioVirtualServiceList) DeepCopyObject() runtime.Object {
	if vsl == nil {
		return nil
	}
	nvsl := new(IstioVirtualServiceList)
	*nvsl = *vsl
	nvsl.TypeMeta = vsl.TypeMeta
	nvsl.ListMeta = vsl.ListMeta
	nvsl.Items = make([]IstioVirtualService, len(vsl.Items))
	for i := range vsl.Items {
		nvsl.Items[i] = *vsl.Items[i].DeepCopyObject().(*IstioVirtualService)
	}
	return nvsl
}

//
//type IstioDestinationRuleList struct {
//	metav1.TypeMeta `json:",inline"`
//	metav1.ListMeta `json:"metadata,omitempty"`
//	Items           []IstioDestinationRule `json:"items"`
//}
//
//type ServiceEntryList struct {
//	metav1.TypeMeta `json:",inline"`
//	metav1.ListMeta `json:"metadata,omitempty"`
//	Items           []ServiceEntry `json:"items"`
//}
