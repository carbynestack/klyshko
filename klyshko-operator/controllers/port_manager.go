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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sort"
)

var egressSelector, _ = fields.ParseSelector("spec.selector.istio=egressgateway")

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
func NewGatewayPortManager(k8sReader client.Reader, portRange *PortRange) PortManager {
	return &defaultPortManager{
		usedPortSupplier: &gatewayUsedPortSupplier{
			portRange: portRange,
			k8sReader: k8sReader,
			logger: log.FromContext(context.TODO()).
				WithName("GatewayPortManager"),
		},
		portRange: portRange,
	}
}

// NewEgressPortManager provides a new EgressPortManager that manages the
// available ports for the egress traffic. It keeps track of the ports that are
// already in use by the egress gateway and provides a method to find a free
// port within the configured port range.
func NewEgressPortManager(k8sReader client.Client,
	egressServiceHost string,
	portRange *PortRange) PortManager {
	return &defaultPortManager{
		usedPortSupplier: &egressUsedPortSupplier{
			k8sReader:         k8sReader,
			egressServiceHost: egressServiceHost,
			portRange:         portRange,
			logger:            ctrl.Log.WithName("EgressPortManager"),
		},
		portRange: portRange,
	}
}

type defaultPortManager struct {
	usedPortSupplier usedPortSupplier
	portRange        *PortRange
}

// GetFreePort returns a free port within the configured port range. It returns
// an error if no free ports are available.
func (m *defaultPortManager) GetFreePort(ctx context.Context) (uint32, error) {
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
type gatewayUsedPortSupplier struct {
	portRange *PortRange
	k8sReader client.Reader
	logger    logr.Logger
}

// getUsedPortsInRange iterates over the list of Istio gateways and collects that ports that are within the port range
// and already in use.
func (m *gatewayUsedPortSupplier) getUsedPortsInRange(ctx context.Context) ([]uint32, error) {
	var usedPorts = make([]uint32, 0)
	gateways := &unstructured.UnstructuredList{}
	gateways.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1beta1",
		Kind:    "Gateway",
	})
	err := m.k8sReader.List(ctx, gateways, client.InNamespace("default"))
	if err != nil {
		m.logger.Error(err, "unable to list gateways")
		return usedPorts, err
	}
	for _, gw := range gateways.Items {
		igw := IstioGateway{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(gw.Object, &igw)
		if err != nil {
			m.logger.Error(err, "unable to convert unstructured gateway to typed gateway")
			continue
		}
		m.logger.Info("processing gateway", "gateway", igw)
		for _, server := range igw.Spec.Servers {
			if server.Port.Number >= m.portRange.Min && server.Port.Number <= m.portRange.Max {
				usedPorts = append(usedPorts, server.Port.Number)
			}
		}
	}
	return usedPorts, nil
}

// egressUsedPortSupplier provides a method to list the ports that are already
// in use to route traffic via the referenced egress gateway.
type egressUsedPortSupplier struct {
	k8sReader         client.Reader
	egressServiceHost string
	portRange         *PortRange
	logger            logr.Logger
}

// getUsedPortsInRange iterates over the list of Istio gateways and collects that ports that are within the port range
// and already in use.
func (m *egressUsedPortSupplier) getUsedPortsInRange(ctx context.Context) ([]uint32, error) {
	usedPorts := make([]uint32, 0)
	gws := &unstructured.UnstructuredList{}
	gws.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   istioNetworkingGroup,
		Version: istioVersion,
		Kind:    "Gateway",
	})
	err := m.k8sReader.List(ctx, gws)
	if err != nil {
		m.logger.Error(err, "unable to list gateways")
		return []uint32{}, err
	}
	for _, ugw := range gws.Items {
		igw, err := IstioGatewayFromUnstructured(&ugw)
		if err != nil {
			m.logger.Error(err, "unable to convert unstructured gateway to IstioGateway")
			continue
		}
		m.logger.Info("gateway", "gateway", igw)
		if igw.Spec.Selector == nil || igw.Spec.Selector["istio"] != "egressgateway" {
			m.logger.Info("not an egress gateway - skip")
			continue
		}
		usedPorts = append(usedPorts, m.getUsedPortsFromServers(igw.Spec.Servers)...)
	}
	return usedPorts, nil
}

func (m *egressUsedPortSupplier) getUsedPortsFromServers(servers []*v1beta1.Server) []uint32 {
	usedPorts := make([]uint32, 0)
	for _, server := range servers {
		if server.Port.Number >= m.portRange.Min &&
			server.Port.Number <= m.portRange.Max {
			usedPorts = append(usedPorts, server.Port.Number)
		}
	}
	return usedPorts
}
