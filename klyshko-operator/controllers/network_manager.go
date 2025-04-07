/*
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"context"
	"fmt"
	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
	"github.com/carbynestack/klyshko/logging"
	"github.com/go-logr/logr"
	"istio.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"strings"
	"sync"
)

const (
	TaskNetworkLabel = "klyshko.carbnyestack.io/task-network-ref"
)

type TLSConfig struct {
	SecretName string
}

type NetworkManager interface {
	// CreateIngressNetworkingForTask creates the necessary Istio ingress resources
	// for the task if not already
	// It returns the port number where the task is available or an error if the
	// creation failed.
	CreateIngressNetworkingForTask(ctx context.Context, task *klyshkov1alpha1.TupleGenerationTask) (uint32, error)
	// CreateEgressNetworkingForTask creates the necessary Istio egress resources
	// for the task if not already defined based on the resource names.
	// It returns an error if the creation failed.
	CreateEgressNetworkingForTask(ctx context.Context, task *klyshkov1alpha1.TupleGenerationTask, endpoints map[uint]string) error
	// DeleteNetworkingForTask deletes the networking resources for the task.
	// It returns an error if the deletion failed.
	DeleteNetworkingForTask(ctx context.Context, task *klyshkov1alpha1.TupleGenerationTask) error
}

func NewNetworkManager(ingressPortRange *PortRange, egressServiceHost string,
	egressGatewayName string, egressPortRange *PortRange,
	tlsConfig *TLSConfig, k8sClient client.Client) (NetworkManager, error) {

	return &DefaultNetworkManager{
		k8sClient: k8sClient,
		ingressPortManager: NewGatewayPortManager(
			k8sClient,
			ingressPortRange),
		egressServiceHost: egressServiceHost,
		egressGatewayName: egressGatewayName,
		egressPortManager: NewEgressPortManager(
			k8sClient,
			egressServiceHost,
			egressGatewayName,
			egressPortRange),
		tlsConfig: tlsConfig,
		mtx:       sync.Mutex{},
	}, nil
}

// DefaultNetworkManager manages the network. It creates network resources required to make the CRGs available to the players.
type DefaultNetworkManager struct {
	// networkClient is the client used to interact with the network.
	k8sClient client.Client
	// ingressPortManager is the manager used to find available ports for the
	// ingress traffic.
	ingressPortManager PortManager
	// egressServiceHost is the host of the service that is used for egress traffic.
	egressServiceHost string
	// egressGatewayName is the name of the Istio Gateway that is used egress traffic.
	egressGatewayName string
	// egressPortManager is the manager used to find available ports to configure
	// egress routes.
	egressPortManager PortManager
	// tlsConfig is the configuration used to create the TLS secret. If nil, endpoints are not secured.
	tlsConfig *TLSConfig
	mtx       sync.Mutex
}

// CreateIngressNetworkingForTask creates the necessary Istio ingress resources
// for the task.
// It returns the port number where the task is available or an error if the
// creation failed.
func (n *DefaultNetworkManager) CreateIngressNetworkingForTask(ctx context.Context, task *klyshkov1alpha1.TupleGenerationTask) (uint32, error) {
	// Lock the mutex to prevent concurrent access to the ingress port manager
	n.mtx.Lock()
	defer n.mtx.Unlock()
	logger := log.FromContext(ctx).
		WithName("Networking").
		WithName("Ingress").
		WithValues("Service", task.Name).
		V(logging.DEBUG)

	// Create the Gateway
	gateway, err := n.getOrCreateGateway(ctx, logger, task)
	if err != nil {
		logger.Info("Failed to create Gateway", "Error", err)
		return 0, fmt.Errorf("failed to create Gateway: %w", err)
	}
	logger.Info("Gateway created", "Gateway", gateway)
	// Create VirtualService
	virtualService, err := n.getOrCreateVirtualService(ctx, logger, task, gateway)
	if err != nil {
		logger.Info("Failed to create VirtualService", "Error", err)
		return 0, fmt.Errorf("failed to create VirtualService: %w", err)
	}
	logger.Info("VirtualService created", "VirtualService", virtualService)
	return gateway.Spec.Servers[0].Port.Number, nil
}

// CreateEgressNetworkingForTask creates the necessary Istio egress resources
// for the task.
// It returns an error if the creation failed.
func (n *DefaultNetworkManager) CreateEgressNetworkingForTask(ctx context.Context, task *klyshkov1alpha1.TupleGenerationTask, endpoints map[uint]string) error {
	// Lock the mutex to prevent concurrent access to the egress port manager
	n.mtx.Lock()
	defer n.mtx.Unlock()
	logger := log.FromContext(ctx).
		WithName("Networking").
		WithName("Egress").
		WithValues("Task", task.Name).
		V(logging.DEBUG)
	for pId, endpoint := range endpoints {
		// Create egress resources for endpoint
		err := n.createEgressResources(ctx, logger, task, pId, endpoint)
		if err != nil {
			logger.Info("Failed to create Egress resources", "Error", err)
			return fmt.Errorf("failed to create Egress resources for player %d: %w", pId, err)
		}
	}
	return nil
}

// DeleteNetworkingForTask deletes the networking resources for the task.
// It collects errors from the deletion of the resources and returns an error if
// the deletion of any resource type failed.
func (n *DefaultNetworkManager) DeleteNetworkingForTask(ctx context.Context, task *klyshkov1alpha1.TupleGenerationTask) error {
	// Lock the mutex to ensure all resources are deleted before a port may be reused
	n.mtx.Lock()
	defer n.mtx.Unlock()
	logger := log.FromContext(ctx).
		WithName("Networking").
		WithName("Delete").
		WithValues("Task", task.Name).
		V(logging.DEBUG)
	errs := make([]error, 0)
	err := n.k8sClient.DeleteAllOf(
		ctx,
		NewUnstructuredIstioGateway(),
		client.InNamespace(task.Namespace),
		client.MatchingLabels{TaskNetworkLabel: task.Name})
	if err != nil {
		logger.Info("Failed to delete Gateways", "Error", err)
		errs = append(errs, fmt.Errorf("failed to delete Gateways: %w", err))
	}
	err = n.k8sClient.DeleteAllOf(
		ctx,
		NewUnstructuredIstioVirtualService(),
		client.InNamespace(task.Namespace),
		client.MatchingLabels{TaskNetworkLabel: task.Name})
	if err != nil {
		logger.Info("Failed to delete VirtualServices", "Error", err)
		errs = append(errs, fmt.Errorf("failed to delete VirtualServices: %w", err))
	}
	err = n.k8sClient.DeleteAllOf(
		ctx,
		NewUnstructuredIstioDestinationRule(),
		client.InNamespace(task.Namespace),
		client.MatchingLabels{TaskNetworkLabel: task.Name})
	if err != nil {
		logger.Info("Failed to delete ServiceEntries", "Error", err)
		errs = append(errs, fmt.Errorf("failed to delete ServiceEntries: %w", err))
	}
	err = n.k8sClient.DeleteAllOf(
		ctx,
		NewUnstructuredIstioServiceEntry(),
		client.InNamespace(task.Namespace),
		client.MatchingLabels{TaskNetworkLabel: task.Name})
	if err != nil {
		logger.Info("Failed to delete DestinationRules", "Error", err)
		errs = append(errs, fmt.Errorf("failed to delete DestinationRules: %w", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to delete networking resources: %v", errs)
	}
	logger.Info("Deleted all networking resources for task", "Task", task.Name)
	return nil
}

func (n *DefaultNetworkManager) newServer(port uint32) *v1beta1.Server {
	srv := v1beta1.Server{
		Port: &v1beta1.Port{
			Number:   port,
			Protocol: "TCP",
			Name:     fmt.Sprintf("crg-tcp-%d", port),
		},
		Hosts: []string{"*"},
	}
	if n.tlsConfig != nil {
		srv.Tls = &v1beta1.ServerTLSSettings{
			Mode:           v1beta1.ServerTLSSettings_MUTUAL,
			CredentialName: n.tlsConfig.SecretName,
		}
	}
	return &srv
}

func (n *DefaultNetworkManager) getOrCreateGateway(ctx context.Context, logger logr.Logger, task *klyshkov1alpha1.TupleGenerationTask) (*IstioGateway, error) {
	gatewayName := types.NamespacedName{
		Namespace: task.Namespace,
		Name:      fmt.Sprintf("%s-gateway", task.Name),
	}
	logger = logger.WithName("Gateway").
		WithValues("Service", task.Name)
	unstructuredGateway := NewUnstructuredIstioGateway()
	err := n.k8sClient.Get(ctx, gatewayName, unstructuredGateway)
	if err == nil {
		logger.Info("Gateway already exists", "Gateway", unstructuredGateway)
		return IstioGatewayFromUnstructured(unstructuredGateway)
	} else {
		logger.Info("Gateway does not exist", "Gateway", unstructuredGateway, "Error", err)
	}
	// Allocate a free port for the Gateway
	port, err := n.ingressPortManager.GetFreePort(ctx)
	if err != nil {
		logger.Info("Failed to get a free port", "Error", err)
		return nil, fmt.Errorf("failed to get a free port: %w", err)
	}
	selectors := map[string]string{}
	selectors["istio"] = "ingressgateway"
	gateway := NewIstioGateway(
		metav1.ObjectMeta{
			Name:      gatewayName.Name,
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.Gateway{
			Servers: []*v1beta1.Server{
				n.newServer(port),
			},
			Selector: selectors,
		})
	return &gateway, n.k8sClient.Create(ctx, InterfaceToUnstructured(gateway))
}

func (n *DefaultNetworkManager) getOrCreateVirtualService(ctx context.Context, logger logr.Logger, task *klyshkov1alpha1.TupleGenerationTask, gateway *IstioGateway) (*IstioVirtualService, error) {
	vsName := types.NamespacedName{
		Namespace: task.Namespace,
		Name:      fmt.Sprintf("%s-vs", gateway.Name),
	}
	logger = logger.WithName("VirtualService").
		WithValues("Service", task.Name)
	unstructuredVirtualService := NewUnstructuredIstioVirtualService()
	err := n.k8sClient.Get(ctx, vsName, unstructuredVirtualService)
	if err == nil {
		logger.Info("VirtualService already exists", "VirtualService", unstructuredVirtualService)
		return IstioVirtualServiceFromUnstructured(unstructuredVirtualService)
	}
	virtualService := NewIstioVirtualService(
		metav1.ObjectMeta{
			Name:      vsName.Name,
			Namespace: gateway.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.VirtualService{
			Hosts: []string{"*"},
			Gateways: []string{
				gateway.Name,
			},
			Tcp: []*v1beta1.TCPRoute{
				{
					Match: []*v1beta1.L4MatchAttributes{
						{
							Port: gateway.Spec.Servers[0].Port.Number,
						},
					},
					Route: []*v1beta1.RouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: types.NamespacedName{
									Namespace: task.Namespace,
									Name:      task.Name}.
									String(),
								Port: &v1beta1.PortSelector{
									Number: InterCRGNetworkingPort,
								},
							},
						},
					},
				},
			},
		})
	return &virtualService, n.k8sClient.Create(ctx, InterfaceToUnstructured(&virtualService))
}

func (n *DefaultNetworkManager) getOrCreateDestinationRule(ctx context.Context, logger logr.Logger, task *klyshkov1alpha1.TupleGenerationTask, pId uint, host string) (*IstioDestinationRule, error) {
	ruleName := types.NamespacedName{
		Namespace: task.Namespace,
		Name:      fmt.Sprintf("%s-dr-%d", task.Name, pId),
	}
	logger = logger.WithName("DestinationRule").
		WithValues("Name", ruleName)
	unstructuredDestinationRule := NewUnstructuredIstioDestinationRule()
	err := n.k8sClient.Get(ctx, ruleName, unstructuredDestinationRule)
	if err == nil {
		logger.Info("DestinationRule already exists", "DestinationRule", unstructuredDestinationRule)
		return IstioDestinationRuleFromUnstructured(unstructuredDestinationRule)
	}
	destinationRule := NewIstioDestinationRule(
		metav1.ObjectMeta{
			Name:      ruleName.Name,
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.DestinationRule{
			Host: host,
			TrafficPolicy: &v1beta1.TrafficPolicy{
				LoadBalancer: &v1beta1.LoadBalancerSettings{
					LbPolicy: &v1beta1.LoadBalancerSettings_Simple{
						Simple: v1beta1.LoadBalancerSettings_ROUND_ROBIN,
					},
				},
			},
		})
	if n.tlsConfig != nil {
		destinationRule.Spec.TrafficPolicy.Tls = &v1beta1.ClientTLSSettings{
			Mode:           v1beta1.ClientTLSSettings_MUTUAL,
			CredentialName: n.tlsConfig.SecretName,
		}
	}
	return &destinationRule, n.k8sClient.Create(ctx, InterfaceToUnstructured(&destinationRule))
}

func (n *DefaultNetworkManager) createServiceEntry(ctx context.Context, logger logr.Logger, task *klyshkov1alpha1.TupleGenerationTask, pId uint, host string, port uint32) (*IstioServiceEntry, error) {
	seName := types.NamespacedName{
		Namespace: task.Namespace,
		Name:      fmt.Sprintf("%s-se-%d", task.Name, pId),
	}
	logger = logger.WithName("ServiceEntry").
		WithValues("Name", seName)
	unstructuredServiceEntry := NewUnstructuredIstioServiceEntry()
	err := n.k8sClient.Get(ctx, seName, unstructuredServiceEntry)
	if err == nil {
		logger.Info("ServiceEntry already exists", "ServiceEntry", unstructuredServiceEntry)
		return IstioServiceEntryFromUnstructured(unstructuredServiceEntry)
	}
	serviceEntry := NewIstioServiceEntry(
		metav1.ObjectMeta{
			Name:      seName.Name,
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.ServiceEntry{
			Addresses: []string{fmt.Sprintf("%s/32", host)},
			Ports: []*v1beta1.Port{
				// ToDo: Use ServicePort once later versions of IstioAPI are supported
				{
					Number:     port,
					Protocol:   "TCP",
					Name:       fmt.Sprintf("crg-tcp-%d", port),
					TargetPort: port,
				},
			},
			Location:   v1beta1.ServiceEntry_MESH_EXTERNAL,
			Resolution: v1beta1.ServiceEntry_DNS,
		})
	return &serviceEntry, n.k8sClient.Create(ctx, InterfaceToUnstructured(&serviceEntry))
}

func (n *DefaultNetworkManager) getOrCreateEgressVirtualService(ctx context.Context, logger logr.Logger, task *klyshkov1alpha1.TupleGenerationTask, pId uint, host string, port uint32) (*IstioVirtualService, error) {
	vsName := types.NamespacedName{
		Namespace: task.Namespace,
		Name:      fmt.Sprintf("%s-egress-vs-%d", task.Name, pId),
	}
	logger = logger.WithName("VirtualService").
		WithValues("Name", vsName)
	unstructuredVirtualService := NewUnstructuredIstioVirtualService()
	err := n.k8sClient.Get(ctx, vsName, unstructuredVirtualService)
	if err == nil {
		logger.Info("VirtualService already exists", "VirtualService", unstructuredVirtualService)
		return IstioVirtualServiceFromUnstructured(unstructuredVirtualService)
	}
	egressPort, err := n.egressPortManager.GetFreePort(ctx)
	if err != nil {
		logger.Info("Failed to get a free egress port", "Error", err)
		return nil, fmt.Errorf("failed to get a free egress port: %w", err)
	}
	logger.Info("Egress port allocated", "Port", egressPort)
	virtualService := NewIstioVirtualService(
		metav1.ObjectMeta{
			Name:      vsName.Name,
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.VirtualService{
			Hosts: []string{host},
			Gateways: []string{
				"mesh",
				n.egressGatewayName,
			},
			Tcp: []*v1beta1.TCPRoute{
				{
					Match: []*v1beta1.L4MatchAttributes{
						{
							Port: egressPort,
							Gateways: []string{
								"mesh",
							},
						},
					},
					Route: []*v1beta1.RouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: n.egressServiceHost,
								Port: &v1beta1.PortSelector{
									Number: egressPort,
								},
							},
							Weight: 100,
						},
					},
				}, {
					Match: []*v1beta1.L4MatchAttributes{
						{
							Port: egressPort,
							Gateways: []string{
								n.egressGatewayName,
							},
						},
					},
					Route: []*v1beta1.RouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: host,
								Port: &v1beta1.PortSelector{
									Number: port,
								},
							},
							Weight: 100,
						},
					},
				},
			},
		})
	return &virtualService, n.k8sClient.Create(ctx, InterfaceToUnstructured(&virtualService))
}

func (n *DefaultNetworkManager) createEgressResources(ctx context.Context, logger logr.Logger, task *klyshkov1alpha1.TupleGenerationTask, pId uint, endpoint string) error {
	logger = logger.WithName("Endpoint").
		WithValues("playerID", pId, "Endpoint", endpoint)
	hostPort := strings.Split(endpoint, ":")
	if len(hostPort) != 2 {
		logger.Info("Invalid endpoint", "PlayerID", pId, "Endpoint", endpoint)
		return fmt.Errorf("invalid endpoint for player %d: %s", pId, endpoint)
	}
	p, err := strconv.ParseUint(hostPort[1], 10, 32)
	if err != nil {
		logger.Info("Invalid endpoint", "PlayerID", pId, "Endpoint", endpoint)
		return fmt.Errorf("invalid endpoint for player %d: %s", pId, endpoint)
	}
	host := hostPort[0]
	port := uint32(p)
	destinationRule, err := n.getOrCreateDestinationRule(ctx, logger, task, pId, host)
	if err != nil {
		logger.Info("Failed to create Egress DestinationRule", "Error", err)
		return fmt.Errorf("failed to create Egress DestinationRule: %w", err)
	}
	logger.Info("Egress DestinationRule created", "Egress DestinationRule", destinationRule)
	serviceEntry, err := n.createServiceEntry(ctx, logger, task, pId, host, port)
	if err != nil {
		logger.Info("Failed to create Egress ServiceEntry", "Error", err)
		return fmt.Errorf("failed to create Egress ServiceEntry: %w", err)
	}
	logger.Info("Egress ServiceEntry created", "Egress ServiceEntry", serviceEntry)
	virtualService, err := n.getOrCreateEgressVirtualService(ctx, logger, task, pId, host, port)
	if err != nil {
		logger.Info("Failed to create Egress VirtualService", "Error", err)
		return fmt.Errorf("failed to create Egress VirtualService: %w", err)
	}
	logger.Info("Egress VirtualService created", "Egress VirtualService", virtualService)
	return nil
}
