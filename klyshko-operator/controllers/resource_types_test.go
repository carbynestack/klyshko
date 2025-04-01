/*
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"fmt"
	"istio.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewIstioGatewayUsingPort(port uint32) *IstioGateway {
	return &IstioGateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1beta1",
			Kind:       "Gateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("gateway-%d", port),
			Namespace: "default",
		},
		Spec: &v1beta1.Gateway{
			Servers: []*v1beta1.Server{
				{
					Port: &v1beta1.Port{
						Number: port,
					},
				},
			},
		},
	}
}

func NewIstioVirtualServiceUsingPort(egressGatewayName string, egressGatewayHost string, tcpPort uint32, httpPort uint32) *IstioVirtualService {
	return &IstioVirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1beta1",
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("virtual-service-%d-%d", tcpPort, httpPort),
			Namespace: "default",
		},
		Spec: &v1beta1.VirtualService{
			Hosts:    []string{"*"},
			Gateways: []string{egressGatewayName},
			Tcp: []*v1beta1.TCPRoute{
				{
					Match: []*v1beta1.L4MatchAttributes{
						{
							Port: tcpPort,
						},
					},
					Route: []*v1beta1.RouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: egressGatewayHost,
								Port: &v1beta1.PortSelector{
									Number: tcpPort,
								},
							},
						},
					},
				},
			},
			Http: []*v1beta1.HTTPRoute{
				{
					Match: []*v1beta1.HTTPMatchRequest{
						{
							Uri: &v1beta1.StringMatch{
								MatchType: &v1beta1.StringMatch_Prefix{
									Prefix: "/",
								},
							},
						},
					},
					Route: []*v1beta1.HTTPRouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: egressGatewayHost,
								Port: &v1beta1.PortSelector{
									Number: httpPort,
								},
							},
						},
					},
				},
			},
		},
	}
}
