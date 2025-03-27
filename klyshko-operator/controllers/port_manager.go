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
	"github.com/go-logr/logr"
	"istio.io/api/networking/v1beta1"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"sync"
)

type PortManager interface {
	GetFreePort(ctx context.Context) (uint32, error)
}

// NewPortRange creates a new PortRange with the given minimum and maximum port numbers. The minimum port number must be
// non-negative and the maximum port number must be greater than the minimum port number.
func NewPortRange(min, max uint32) (*PortRange, error) {
	if min < 0 || max <= min {
		return nil, fmt.Errorf("invalid port range: %d:%d", min, max)
	}
	return &PortRange{Min: min, Max: max}, nil
}

// PortRange represents a range of port numbers where Min is the minimum port number and Max is the maximum port number
// (inclusive).
type PortRange struct {
	// Min is the minimum port number in the range.
	Min uint32
	// Max is the maximum port number in the range (inclusive).
	Max uint32
}

type usedPortSupplier interface {
	getUsedPortsInRange(ctx context.Context) ([]uint32, error)
}

// NewGatewayPortManager provides a new GatewayPortManager that manages the
// available ports for the Istio gateways. It keeps track of the ports that are
// already in use by the gateways and provides a method to find a free port
// within the configured port range.
func NewGatewayPortManager(k8sClient client.Client, portRange *PortRange) PortManager {
	return &defaultPortManager{
		usedPortSupplier: &gatewayUsedPortSupplier{
			portRange: portRange,
			k8sClient: k8sClient,
			logger:    ctrl.Log.WithName("GatewayPortManager"),
		},
		portRange: portRange,
		mtx:       sync.Mutex{},
	}
}

// NewEgressPortManager provides a new EgressPortManager that manages the
// available ports for the egress traffic. It keeps track of the ports that are
// already in use by the egress gateway and provides a method to find a free
// port within the configured port range.
func NewEgressPortManager(k8sClient client.Client,
	egressServiceHost string,
	egressGatewayName string,
	egressGatewayNamespace string,
	portRange *PortRange) PortManager {
	return &defaultPortManager{
		usedPortSupplier: &egressUsedPortSupplier{
			k8sClient:              k8sClient,
			egressServiceHost:      egressServiceHost,
			egressGatewayName:      egressGatewayName,
			egressGatewayNamespace: egressGatewayNamespace,
			portRange:              portRange,
			logger:                 ctrl.Log.WithName("EgressPortManager"),
		},
		portRange: portRange,
		mtx:       sync.Mutex{},
	}
}

type defaultPortManager struct {
	usedPortSupplier usedPortSupplier
	portRange        *PortRange
	mtx              sync.Mutex
}

// GetFreePort returns a free port within the configured port range. It returns
// an error if no free ports are available.
func (m *defaultPortManager) GetFreePort(ctx context.Context) (uint32, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	usedPorts, err := m.usedPortSupplier.getUsedPortsInRange(ctx)
	if err != nil {
		return 0, err
	}
	if len(usedPorts) == 0 || usedPorts[0] > m.portRange.Min {
		return m.portRange.Min, nil
	}
	if uint32(len(usedPorts)) == m.portRange.Max-m.portRange.Min+1 {
		return 0, errors.New("no free ports available")
	}
	sort.Slice(usedPorts, func(i, j int) bool {
		return usedPorts[i] < usedPorts[j]
	})
	for i := 0; i < len(usedPorts)-1; i++ {
		if usedPorts[i+1]-usedPorts[i] > 1 {
			return usedPorts[i] + 1, nil
		}
	}
	return usedPorts[len(usedPorts)-1] + 1, nil
}

// gatewayUsedPortSupplier provides a method to list the ports that are already
// in use by the gateways.
// Methods are not thread-safe. Public methods are expected to obtain a
// lock on mtx before calling them.
type gatewayUsedPortSupplier struct {
	portRange *PortRange
	k8sClient client.Client
	logger    logr.Logger
}

// getUsedPortsInRange iterates over the list of Istio gateways and collects that ports that are within the port range
// and already in use.
func (m *gatewayUsedPortSupplier) getUsedPortsInRange(ctx context.Context) ([]uint32, error) {
	var usedPorts []uint32

	gateways := IstioGatewayList{}
	err := m.k8sClient.List(ctx, &gateways, client.InNamespace("default"))
	if err != nil {
		m.logger.Error(err, "unable to retrieve the gateways")
		return []uint32{}, err
	}
	for _, gw := range gateways.Items {
		for _, server := range gw.Spec.Servers {
			if server.Port.Number >= m.portRange.Min && server.Port.Number <= m.portRange.Max {
				usedPorts = append(usedPorts, server.Port.Number)
			}
		}
	}
	return usedPorts, nil
}

// egressUsedPortSupplier provides a method to list the ports that are already
// in use to route traffic via the referenced egress gateway.
// Methods are not thread-safe. Public methods are expected to obtain a
// lock on mtx before calling them.
type egressUsedPortSupplier struct {
	k8sClient              client.Client
	egressServiceHost      string
	egressGatewayName      string
	egressGatewayNamespace string
	portRange              *PortRange
	logger                 logr.Logger
}

// getUsedPortsInRange iterates over the list of Istio gateways and collects that ports that are within the port range
// and already in use.
func (m *egressUsedPortSupplier) getUsedPortsInRange(ctx context.Context) ([]uint32, error) {
	var usedPorts []uint32
	vss := IstioVirtualServiceList{}
	err := m.k8sClient.List(ctx, &vss, client.InNamespace(m.egressGatewayNamespace))
	if err != nil {
		m.logger.Error(err, "unable to retrieve the egress gateway")
		return []uint32{}, err
	}
	for _, vs := range vss.Items {
		if !slices.Contains(vs.Spec.GetGateways(), m.egressGatewayName) {
			continue
		}
		usedPorts = append(usedPorts, m.getUsedPortsFromRoutes(vs.Spec.Http)...)
		usedPorts = append(usedPorts, m.getUsedPortsFromRoutes(vs.Spec.Tcp)...)
	}
	return usedPorts, nil
}

func (m *egressUsedPortSupplier) getUsedPortsFromRoutes(routes interface{}) []uint32 {
	switch r := routes.(type) {
	case []*v1beta1.TCPRoute:
		for _, route := range r {
			return m.getUsedPortsFromRouteDestinations(route.Route)
		}
	case []*v1beta1.HTTPRoute:
		for _, route := range r {
			return m.getUsedPortsFromRouteDestinations(route.Route)
		}
	case *v1beta1.Destination:
	}
	return []uint32{}
}

func (m *egressUsedPortSupplier) getUsedPortsFromRouteDestinations(destinations interface{}) []uint32 {
	var usedPorts []uint32
	switch r := destinations.(type) {
	case []*v1beta1.HTTPRouteDestination:
		for _, destination := range r {
			if ok, port := m.destinationPortInRange(destination.Destination); ok {
				usedPorts = append(usedPorts, port)
			}
		}
	case []*v1beta1.RouteDestination:
		for _, destination := range r {
			if ok, port := m.destinationPortInRange(destination.Destination); ok {
				usedPorts = append(usedPorts, port)
			}
		}
	}
	return usedPorts
}

func (m *egressUsedPortSupplier) destinationPortInRange(destination *v1beta1.Destination) (bool, uint32) {
	if destination.Host == m.egressServiceHost {
		if destination.Port.Number >= m.portRange.Min &&
			destination.Port.Number <= m.portRange.Max {
			return true, destination.Port.Number
		}
	}
	return false, 0
}
