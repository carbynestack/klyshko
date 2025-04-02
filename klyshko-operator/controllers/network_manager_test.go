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
	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"istio.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"strings"
)

var _ = Describe("When managing network resources", func() {

	var (
		ctx                    context.Context
		cancel                 context.CancelFunc
		fakeK8sClient          = NewFakeK8sClient()
		fakeIngressPortManager = NewFakePortManager()
		fakeEgressPortManager  = NewFakePortManager()
		task                   = &klyshkov1alpha1.TupleGenerationTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task1",
				Namespace: "default",
			},
			Spec: klyshkov1alpha1.TupleGenerationTaskSpec{
				PlayerID: 1,
			},
		}
		egressServiceHost = "test.gateway.host"
		egressGatewayName = "test-gateway"
		logger            = ctrl.Log.WithName("Test").WithName("NetworkManager")
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.TODO())
		fakeIngressPortManager.Reset()
		fakeEgressPortManager.Reset()
		fakeK8sClient.Reset()
	})

	AfterEach(func() {
		cancel()
	})

	Context("with no tls configuration", func() {
		var (
			tlsConfig      *TLSConfig = nil
			networkManager            = &DefaultNetworkManager{
				k8sClient:          fakeK8sClient,
				ingressPortManager: fakeIngressPortManager,
				egressServiceHost:  egressServiceHost,
				egressGatewayName:  egressGatewayName,
				egressPortManager:  fakeEgressPortManager,
				tlsConfig:          tlsConfig,
			}
		)

		Context("when creating the gateway for a task", func() {

			When("gateway with name already exists", func() {
				var (
					expectedGw = createExpectedIstioGateway(task, 30500, tlsConfig)
				)

				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedGw)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					gw, err := networkManager.getOrCreateGateway(ctx, logger, task)
					Expect(err).NotTo(HaveOccurred())
					Expect(gw).To(Equal(expectedGw))
				})
			})

			When("gateway with name does not exist", func() {

				When("getting a free port fails", func() {
					It("should return an error", func() {
						gw, err := networkManager.getOrCreateGateway(ctx, logger, task)
						Expect(gw).To(BeNil())
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to get a free port"))
					})
				})

				When("getting a free port succeeds", func() {
					var (
						availablePort = uint32(30500)
					)

					BeforeEach(func() {
						fakeIngressPortManager.ReturnOnGetFreePort(
							FreePortResponse{
								availablePort,
								nil,
							})
					})

					When("creating the gateway fails", func() {
						BeforeEach(func() {
							fakeK8sClient.CreateReturnError = fmt.Errorf("failed to create gateway")
						})

						It("should return an error", func() {
							_, err := networkManager.getOrCreateGateway(ctx, logger, task)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("failed to create gateway"))
						})
					})

					When("creating the gateway succeeds", func() {
						var (
							expectedGw = createExpectedIstioGateway(task, availablePort, tlsConfig)
						)
						BeforeEach(func() {
							fakeK8sClient.CreateReturnError = nil
						})

						It("should create a new one", func() {
							gw, err := networkManager.getOrCreateGateway(ctx, logger, task)
							Expect(err).NotTo(HaveOccurred())
							Expect(gw).To(Equal(expectedGw))
						})
					})
				})
			})
		})

		Context("when creating the virtual service for a task", func() {
			var (
				tcpPort = uint32(30500)
				gateway = createExpectedIstioGateway(task, tcpPort, tlsConfig)
			)
			When("virtual service with name already exists", func() {
				var (
					expectedVs = NewIstioVirtualServiceUsingPort("test-gateway", "test.gateway.host", 30500, 30501)
				)

				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedVs)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					vs, err := networkManager.getOrCreateVirtualService(ctx, logger, task, gateway)
					Expect(err).NotTo(HaveOccurred())
					Expect(vs).To(Equal(expectedVs))
				})
			})

			When("virtual service with name does not exist", func() {
				When("creating the virtual service fails", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = fmt.Errorf("failed to create virtual service")
					})

					It("should return an error", func() {
						_, err := networkManager.getOrCreateVirtualService(ctx, logger, task, gateway)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to create virtual service"))
					})
				})

				When("creating the virtual service succeeds", func() {
					var (
						expectedVs = createExpectedIngressVirtualService(task, gateway, tcpPort)
					)
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = nil
					})

					It("should create a new one", func() {
						vs, err := networkManager.getOrCreateVirtualService(ctx, logger, task, gateway)
						Expect(err).NotTo(HaveOccurred())
						Expect(vs).To(Equal(expectedVs))
					})
				})
			})
		})

		When("creating destination rules for a task", func() {
			var (
				playerID   = uint(1)
				playerHost = "172.18.1.128"
				expectedDr = createExpectedDestinationRule(task, playerID, playerHost, tlsConfig)
			)
			When("destination rule with name already exists", func() {
				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedDr)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					dr, err := networkManager.getOrCreateDestinationRule(ctx, logger, task, playerID, playerHost)
					Expect(err).NotTo(HaveOccurred())
					Expect(dr).To(Equal(expectedDr))
				})
			})

			When("destination rule with name does not exist", func() {
				When("creating the destination rule fails", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = fmt.Errorf("failed to create destination rule")
					})

					It("should return an error", func() {
						_, err := networkManager.getOrCreateDestinationRule(ctx, logger, task, playerID, playerHost)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to create destination rule"))
					})
				})

				When("creating the destination rule succeeds", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = nil
					})

					It("should create a new one", func() {
						dr, err := networkManager.getOrCreateDestinationRule(ctx, logger, task, playerID, playerHost)
						Expect(err).NotTo(HaveOccurred())
						Expect(dr).To(Equal(expectedDr))
					})
				})
			})
		})

		When("creating service entries for a task", func() {
			var (
				playerID   = uint(1)
				playerHost = "172.18.1.128"
				playerPort = uint32(5000)
				expectedSe = createExpectedServiceEntry(task, playerID, playerHost, playerPort)
			)

			When("service entry with name already exists", func() {
				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedSe)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					se, err := networkManager.createServiceEntry(ctx, logger, task, playerID, playerHost, playerPort)
					Expect(err).NotTo(HaveOccurred())
					Expect(se).To(Equal(expectedSe))
				})
			})
			When("service entry with name does not exist", func() {
				When("creating the service entry fails", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = fmt.Errorf("failed to create service entry")
					})

					It("should return an error", func() {
						_, err := networkManager.createServiceEntry(ctx, logger, task, playerID, playerHost, playerPort)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to create service entry"))
					})
				})
				When("creating the service entry succeeds", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = nil
					})

					It("should create a new one", func() {
						se, err := networkManager.createServiceEntry(ctx, logger, task, playerID, playerHost, playerPort)
						Expect(err).NotTo(HaveOccurred())
						Expect(se).To(Equal(expectedSe))
					})
				})
			})
		})

		When("creating egress virtual service for a task", func() {

			var (
				playerID   = uint(1)
				playerHost = "172.18.1.128"
				playerPort = uint32(5000)
				egressPort = uint32(30500)
				expectedVs = createExpectedEgressVirtualService(task, playerID, playerHost, playerPort, egressPort, egressGatewayName, egressServiceHost)
			)

			When("virtual service with name already exists", func() {
				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedVs)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					vs, err := networkManager.getOrCreateEgressVirtualService(ctx, logger, task, playerID, playerHost, playerPort)
					Expect(err).NotTo(HaveOccurred())
					Expect(vs).To(Equal(expectedVs))
				})
			})

			When("virtual service with name does not exist", func() {

				When("getting a free port fails", func() {
					It("should return an error", func() {
						_, err := networkManager.getOrCreateEgressVirtualService(ctx, logger, task, playerID, playerHost, playerPort)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to get a free egress port"))
					})
				})

				When("getting a free port succeeds", func() {
					var (
						availableEgressPort = uint32(30500)
					)
					BeforeEach(func() {
						fakeEgressPortManager.ReturnOnGetFreePort(
							FreePortResponse{
								availableEgressPort,
								nil,
							})
					})

					When("creating the egress virtual service fails", func() {
						BeforeEach(func() {
							fakeK8sClient.CreateReturnError = fmt.Errorf("failed to create egress virtual service")
						})

						It("should return an error", func() {
							_, err := networkManager.getOrCreateEgressVirtualService(ctx, logger, task, playerID, playerHost, playerPort)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("failed to create egress virtual service"))
						})
					})

					When("creating the egress virtual service succeeds", func() {
						BeforeEach(func() {
							fakeK8sClient.CreateReturnError = nil
						})

						It("should create a new one", func() {
							vs, err := networkManager.getOrCreateEgressVirtualService(ctx, logger, task, playerID, playerHost, playerPort)
							Expect(err).NotTo(HaveOccurred())
							Expect(vs).To(Equal(expectedVs))
						})
					})
				})
			})
		})
	})

	Context("with tls configuration", func() {
		var (
			tlsConfig = &TLSConfig{
				SecretName: "test-secret",
			}
			networkManager = &DefaultNetworkManager{
				k8sClient:          fakeK8sClient,
				ingressPortManager: fakeIngressPortManager,
				egressServiceHost:  egressServiceHost,
				egressGatewayName:  egressGatewayName,
				egressPortManager:  fakeEgressPortManager,
				tlsConfig:          tlsConfig,
			}
		)
		When("creating the ingress networking for a task", func() {
			When("creating resources succeeds", func() {
				var (
					ingressPort = uint32(30500)
				)
				BeforeEach(func() {
					fakeIngressPortManager.ReturnOnGetFreePort(
						FreePortResponse{
							ingressPort,
							nil,
						})
					fakeK8sClient.CreateReturnError = nil
				})

				It("should create all resources and return the ingress port", func() {
					port, err := networkManager.CreateIngressNetworkingForTask(ctx, task)
					Expect(err).NotTo(HaveOccurred())
					Expect(port).To(Equal(ingressPort))
					createdResources := fakeK8sClient.CreateCallParams
					Expect(createdResources).To(HaveLen(2))
					for _, resource := range createdResources {
						kind := resource.obj.GetObjectKind().GroupVersionKind().Kind
						switch kind {
						case "Gateway":
							Expect(resource.obj).To(
								Equal(InterfaceToUnstructured(
									createExpectedIstioGateway(task, ingressPort, tlsConfig))))
						case "VirtualService":
							Expect(resource.obj).To(
								Equal(InterfaceToUnstructured(
									createExpectedIngressVirtualService(task, createExpectedIstioGateway(task, ingressPort, tlsConfig), ingressPort))))
						}
					}
				})
			})
		})

		When("creating the egress networking for a task", func() {
			When("creating resources succeeds", func() {
				var (
					endpoints = map[uint]string{
						1: "172.20.1.0:30500",
						2: "182.20.1.1:30550",
						3: "192.20.1.2:5000",
					}
					egressPorts = []uint32{
						10000,
						20000,
						30000,
					}
				)
				BeforeEach(func() {
					for _, p := range egressPorts {
						fakeEgressPortManager.ReturnOnGetFreePort(
							FreePortResponse{
								p,
								nil,
							})
					}
					fakeEgressPortManager.ReturnOnGetFreePort(
						FreePortResponse{
							0,
							errors.New("too many requests"),
						})
					fakeK8sClient.CreateReturnError = nil
				})

				It("should create all resources and return the ingress port", func() {
					err := networkManager.CreateEgressNetworkingForTask(ctx, task, endpoints)
					Expect(err).NotTo(HaveOccurred())
					createdResources := fakeK8sClient.CreateCallParams
					Expect(createdResources).To(HaveLen(len(endpoints) * 3))
					// as the order when iterating maps is not guaranteed, we have to do
					// some extra work
					endpointsToServe := make(map[uint]string)
					for playerId, endpoint := range endpoints {
						endpointsToServe[playerId] = endpoint
					}
					for i := 0; i < len(endpoints); i++ {
						generatedDR := createdResources[i*3].obj
						pIdString := generatedDR.GetName()[strings.LastIndex(generatedDR.GetName(), "-")+1:]
						pId, _ := strconv.ParseUint(pIdString, 10, 32)
						endpoint, ok := endpointsToServe[uint(pId)]
						Expect(ok).To(BeTrue())
						delete(endpointsToServe, uint(pId))
						hostPort := strings.Split(endpoint, ":")
						playerPort, _ := strconv.ParseUint(hostPort[1], 10, 32)
						Expect(generatedDR).To(
							Equal(InterfaceToUnstructured(
								createExpectedDestinationRule(task, uint(pId), hostPort[0], tlsConfig))))
						Expect(createdResources[i*3+1].obj).To(
							Equal(InterfaceToUnstructured(
								createExpectedServiceEntry(task, uint(pId), hostPort[0], uint32(playerPort)))))
						Expect(createdResources[i*3+2].obj).To(
							Equal(InterfaceToUnstructured(
								createExpectedEgressVirtualService(task, uint(pId), hostPort[0], uint32(playerPort), egressPorts[i], egressGatewayName, egressServiceHost))))
					}
					Expect(len(endpointsToServe)).To(BeZero())
				})
			})
		})

		When("deleting networking for a task", func() {
			When("deleting resources fails", func() {
				It("should return an error", func() {
					deleteErr := fmt.Errorf("failed to delete resources")
					fakeK8sClient.DeleteAllReturnError = fmt.Errorf("failed to delete resources")
					err := networkManager.DeleteNetworkingForTask(ctx, task)
					Expect(err).To(HaveOccurred())
					expectedResourceDeleteErrs := []error{
						fmt.Errorf("failed to delete Gateways: %w", deleteErr),
						fmt.Errorf("failed to delete VirtualServices: %w", deleteErr),
						fmt.Errorf("failed to delete ServiceEntries: %w", deleteErr),
						fmt.Errorf("failed to delete DestinationRules: %w", deleteErr),
					}
					Expect(err).To(Equal(fmt.Errorf("failed to delete networking resources: %v", expectedResourceDeleteErrs)))
				})
			})
			When("deleting resources succeeds", func() {
				BeforeEach(func() {
					fakeK8sClient.DeleteAllReturnError = nil
				})

				It("should delete all resources", func() {
					err := networkManager.DeleteNetworkingForTask(ctx, task)
					Expect(err).NotTo(HaveOccurred())
					deletedResources := fakeK8sClient.DeleteAllCallParams
					Expect(deletedResources).To(HaveLen(4))
					deletedKinds := make([]string, 0)
					for _, resource := range deletedResources {
						groupVersionKind := resource.obj.GetObjectKind().GroupVersionKind()
						Expect(groupVersionKind.Group).To(Equal(istioGroup))
						Expect(groupVersionKind.Version).To(Equal(istioVersion))
						deletedKinds = append(deletedKinds, groupVersionKind.Kind)
					}
					Expect(deletedKinds).To(ConsistOf("Gateway", "VirtualService", "ServiceEntry", "DestinationRule"))
				})
			})
		})
	})
})

func createExpectedEgressVirtualService(task *klyshkov1alpha1.TupleGenerationTask, playerID uint, playerHost string, playerPort uint32, egressPort uint32, egressGatewayName string, egressServiceHost string) *IstioVirtualService {
	ivs := NewIstioVirtualService(
		metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-egress-vs-%d", task.Name, playerID),
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.VirtualService{
			Hosts: []string{playerHost},
			Gateways: []string{
				"mesh",
				egressGatewayName,
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
								Host: egressServiceHost,
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
								egressGatewayName,
							},
						},
					},
					Route: []*v1beta1.RouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: playerHost,
								Port: &v1beta1.PortSelector{
									Number: playerPort,
								},
							},
							Weight: 100,
						},
					},
				},
			},
		})
	return &ivs
}

func createExpectedServiceEntry(task *klyshkov1alpha1.TupleGenerationTask, playerID uint, playerHost string, playerPort uint32) *IstioServiceEntry {
	ise := NewIstioServiceEntry(
		metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-se-%d", task.Name, playerID),
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.ServiceEntry{
			Addresses: []string{fmt.Sprintf("%s/32", playerHost)},
			Ports: []*v1beta1.Port{
				{
					Number:     playerPort,
					Protocol:   "TCP",
					Name:       fmt.Sprintf("crg-tcp-%d", playerPort),
					TargetPort: playerPort,
				},
			},
			Location:   v1beta1.ServiceEntry_MESH_EXTERNAL,
			Resolution: v1beta1.ServiceEntry_DNS,
		})
	return &ise
}

func createExpectedDestinationRule(task *klyshkov1alpha1.TupleGenerationTask, playerID uint, playerHost string, tlsConfig *TLSConfig) *IstioDestinationRule {
	idr := NewIstioDestinationRule(
		metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-dr-%d", task.Name, playerID),
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.DestinationRule{
			Host: playerHost,
			TrafficPolicy: &v1beta1.TrafficPolicy{
				LoadBalancer: &v1beta1.LoadBalancerSettings{
					LbPolicy: &v1beta1.LoadBalancerSettings_Simple{
						Simple: v1beta1.LoadBalancerSettings_ROUND_ROBIN,
					},
				},
			},
		})
	if tlsConfig != nil {
		idr.Spec.TrafficPolicy.Tls = &v1beta1.ClientTLSSettings{
			Mode:           v1beta1.ClientTLSSettings_MUTUAL,
			CredentialName: tlsConfig.SecretName,
		}
	}
	return &idr
}

func createExpectedIngressVirtualService(task *klyshkov1alpha1.TupleGenerationTask, gateway *IstioGateway, port uint32) *IstioVirtualService {
	ivs := NewIstioVirtualService(
		metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vs", gateway.Name),
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.VirtualService{
			Hosts:    []string{"*"},
			Gateways: []string{gateway.Name},
			Tcp: []*v1beta1.TCPRoute{
				{
					Match: []*v1beta1.L4MatchAttributes{
						{
							Port: port,
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
	return &ivs
}

func createExpectedIstioGateway(task *klyshkov1alpha1.TupleGenerationTask, port uint32, tlsConfig *TLSConfig) *IstioGateway {
	igw := NewIstioGateway(
		metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-gateway", task.Name),
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.Gateway{
			Servers: []*v1beta1.Server{
				{
					Port: &v1beta1.Port{
						Number:   port,
						Protocol: "TCP",
						Name:     fmt.Sprintf("crg-tcp-%d", port),
					},
					Hosts: []string{"*"},
				},
			},
			Selector: map[string]string{
				"istio": "ingressgateway",
			},
		})
	if tlsConfig != nil {
		igw.Spec.Servers[0].Tls = &v1beta1.ServerTLSSettings{
			Mode:           v1beta1.ServerTLSSettings_MUTUAL,
			CredentialName: tlsConfig.SecretName,
		}
	}
	return &igw
}
