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
	isec "istio.io/api/security/v1beta1"
	itype "istio.io/api/type/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
				egressPortManager:  fakeEgressPortManager,
				tlsConfig:          tlsConfig,
			}
		)

		Context("when creating the ingress gateway for a task", func() {

			When("gateway with name already exists", func() {
				var (
					expectedGw = createExpectedIstioIngressGateway(task, 30500, tlsConfig)
				)

				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedGw)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					gw, err := networkManager.getOrCreateIngressGateway(ctx, logger, task)
					Expect(err).NotTo(HaveOccurred())
					Expect(gw).To(Equal(expectedGw))
				})
			})

			When("gateway with name does not exist", func() {

				When("getting a free port fails", func() {
					It("should return an error", func() {
						gw, err := networkManager.getOrCreateIngressGateway(ctx, logger, task)
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
							_, err := networkManager.getOrCreateIngressGateway(ctx, logger, task)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("failed to create gateway"))
						})
					})

					When("creating the gateway succeeds", func() {
						var (
							expectedGw = createExpectedIstioIngressGateway(task, availablePort, tlsConfig)
						)
						BeforeEach(func() {
							fakeK8sClient.CreateReturnError = nil
						})

						It("should create a new one", func() {
							gw, err := networkManager.getOrCreateIngressGateway(ctx, logger, task)
							Expect(err).NotTo(HaveOccurred())
							Expect(gw).To(Equal(expectedGw))
						})
					})
				})
			})
		})

		Context("when creating the authorization policy for a task", func() {
			var (
				tcpPort    = uint32(30500)
				gateway    = createExpectedIstioIngressGateway(task, tcpPort, tlsConfig)
				expectedAp = createExpectedAuthorizationPolicy(task, tcpPort)
			)

			When("authorization policy with name already exists", func() {
				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedAp)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					ap, err := networkManager.getOrCreateAuthorizationPolicy(ctx, logger, task, gateway)
					Expect(err).NotTo(HaveOccurred())
					Expect(ap).To(Equal(expectedAp))
				})
			})

			When("authorization policy with name does not exist", func() {
				When("creating the authorization policy fails", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = fmt.Errorf("failed to create authorization policy")
					})

					It("should return an error", func() {
						_, err := networkManager.getOrCreateAuthorizationPolicy(ctx, logger, task, gateway)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to create authorization policy"))
					})
				})

				When("creating the authorization policy succeeds", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = nil
					})

					It("should create a new one", func() {
						ap, err := networkManager.getOrCreateAuthorizationPolicy(ctx, logger, task, gateway)
						Expect(err).NotTo(HaveOccurred())
						Expect(ap).To(Equal(expectedAp))
					})
				})
			})
		})

		Context("when creating the virtual service for a task", func() {
			var (
				tcpPort = uint32(30500)
				gateway = createExpectedIstioIngressGateway(task, tcpPort, tlsConfig)
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
				playerIp   = "172.18.1.128"
				expectedDr = createExpectedDestinationRule(task, playerID, playerIp, tlsConfig)
			)
			When("destination rule with name already exists", func() {
				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedDr)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					dr, err := networkManager.getOrCreateDestinationRule(ctx, logger, task, playerID, playerIp)
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
						_, err := networkManager.getOrCreateDestinationRule(ctx, logger, task, playerID, playerIp)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to create destination rule"))
					})
				})

				When("creating the destination rule succeeds", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = nil
					})

					It("should create a new one", func() {
						dr, err := networkManager.getOrCreateDestinationRule(ctx, logger, task, playerID, playerIp)
						Expect(err).NotTo(HaveOccurred())
						Expect(dr).To(Equal(expectedDr))
					})
				})
			})
		})

		When("creating service entries for a task", func() {
			var (
				playerID   = uint(1)
				playerIp   = "172.18.1.128"
				playerHost = "172.18.1.128.sslip.io"
				playerPort = uint32(5000)
				expectedSe = createExpectedServiceEntry(task, playerID, playerHost, playerIp, playerPort)
			)

			When("service entry with name already exists", func() {
				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedSe)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					se, err := networkManager.getOrCreateServiceEntry(ctx, logger, task, playerID, playerHost, playerIp, playerPort)
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
						_, err := networkManager.getOrCreateServiceEntry(ctx, logger, task, playerID, playerHost, playerIp, playerPort)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to create service entry"))
					})
				})
				When("creating the service entry succeeds", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = nil
					})

					It("should create a new one", func() {
						se, err := networkManager.getOrCreateServiceEntry(ctx, logger, task, playerID, playerHost, playerIp, playerPort)
						Expect(err).NotTo(HaveOccurred())
						Expect(se).To(Equal(expectedSe))
					})
				})
			})
		})

		When("creating egress virtual service for a task", func() {
			var (
				playerID   = uint(1)
				playerHost = "172.18.1.128.sslip.io"
				playerPort = uint32(5000)
				egressPort = uint32(30500)
				gateway    = createExpectedIstioEgressGateway(task, playerHost, egressPort)
				expectedVs = createExpectedEgressVirtualService(task, gateway, playerID, playerHost, playerPort, egressPort, egressServiceHost)
			)

			When("virtual service with name already exists", func() {
				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedVs)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					vs, err := networkManager.getOrCreateEgressVirtualService(ctx, logger, task, gateway, playerID, playerHost, playerPort)
					Expect(err).NotTo(HaveOccurred())
					Expect(vs).To(Equal(expectedVs))
				})
			})

			When("virtual service with name does not exist", func() {
				When("creating the egress virtual service fails", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = fmt.Errorf("failed to create egress virtual service")
					})

					It("should return an error", func() {
						_, err := networkManager.getOrCreateEgressVirtualService(ctx, logger, task, gateway, playerID, playerHost, playerPort)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to create egress virtual service"))
					})
				})

				When("creating the egress virtual service succeeds", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = nil
					})

					It("should create a new one", func() {
						vs, err := networkManager.getOrCreateEgressVirtualService(ctx, logger, task, gateway, playerID, playerHost, playerPort)
						Expect(err).NotTo(HaveOccurred())
						Expect(vs).To(Equal(expectedVs))
					})
				})
			})
		})

		When("creating the egress gateway for a task", func() {
			var (
				playerPort = uint32(31500)
				playerHost = "172.18.1.128.sslip.io"
				expectedGw = createExpectedIstioEgressGateway(task, playerHost, playerPort)
			)
			When("gateway with name already exists", func() {
				BeforeEach(func() {
					fakeK8sClient.GetReturnObject = InterfaceToUnstructured(expectedGw)
					fakeK8sClient.GetReturnError = nil
				})

				It("should return the existing one", func() {
					gw, err := networkManager.getOrCreateEgressGateway(ctx, logger, task, playerHost, playerPort)
					Expect(err).NotTo(HaveOccurred())
					Expect(gw).To(Equal(expectedGw))
				})
			})

			When("gateway with name does not exist", func() {
				When("creating the gateway fails", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = fmt.Errorf("failed to create gateway")
					})

					It("should return an error", func() {
						_, err := networkManager.getOrCreateEgressGateway(ctx, logger, task, playerHost, playerPort)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("failed to create gateway"))
					})
				})

				When("creating the gateway succeeds", func() {
					BeforeEach(func() {
						fakeK8sClient.CreateReturnError = nil
					})

					It("should create a new one", func() {
						gw, err := networkManager.getOrCreateEgressGateway(ctx, logger, task, playerHost, playerPort)
						Expect(err).NotTo(HaveOccurred())
						Expect(gw).To(Equal(expectedGw))
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
					Expect(createdResources).To(HaveLen(3))
					for _, resource := range createdResources {
						kind := resource.obj.GetObjectKind().GroupVersionKind().Kind
						switch kind {
						case "Gateway":
							Expect(resource.obj).To(
								Equal(InterfaceToUnstructured(
									createExpectedIstioIngressGateway(task, ingressPort, tlsConfig))))
						case "VirtualService":
							Expect(resource.obj).To(
								Equal(InterfaceToUnstructured(
									createExpectedIngressVirtualService(task, createExpectedIstioIngressGateway(task, ingressPort, tlsConfig), ingressPort))))
						case "AuthorizationPolicy":
							expectedUap := InterfaceToUnstructured(createExpectedAuthorizationPolicy(task, ingressPort))
							// we need to set the action to ALLOW manually as
							// there is a marshalling error where the action is
							// not set correctly
							expectedUap.Object["spec"].(map[string]interface{})["action"] = "ALLOW"
							Expect(resource.obj).To(Equal(expectedUap))
						}
					}
				})
			})
		})

		When("creating the egress networking for a task", func() {
			var (
				endpoints = map[uint]string{
					1: "172.20.1.0:30500",
					2: "182.20.1.1:30550",
					3: "192.20.1.2:5000",
				}
			)
			When("getting a free port fails", func() {
				It("should return an error", func() {
					err := networkManager.CreateEgressNetworkingForTask(ctx, task, endpoints)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to get a free egress port"))
				})
			})

			When("getting a free port succeeds", func() {
				var (
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
				When("creating resources succeeds", func() {
					It("should create all egress resources", func() {
						err := networkManager.CreateEgressNetworkingForTask(ctx, task, endpoints)
						Expect(err).NotTo(HaveOccurred())
						createdResources := fakeK8sClient.CreateCallParams
						Expect(createdResources).To(HaveLen(len(endpoints) * 4))
						// as the order when iterating maps is not guaranteed, we have to do
						// some extra work
						endpointsToServe := make(map[uint]string)
						for playerId, endpoint := range endpoints {
							endpointsToServe[playerId] = endpoint
						}
						for i := 0; i < len(endpoints); i++ {
							createdDR := createdResources[i*4+1].obj
							pIdString := createdDR.GetName()[strings.LastIndex(createdDR.GetName(), "-")+1:]
							pId, _ := strconv.ParseUint(pIdString, 10, 32)
							endpoint, ok := endpointsToServe[uint(pId)]
							Expect(ok).To(BeTrue())
							delete(endpointsToServe, uint(pId))
							ipPort := strings.Split(endpoint, ":")
							playerIp := ipPort[0]
							playerHost := fmt.Sprintf("%s.sslip.io", playerIp)
							playerPort, _ := strconv.ParseUint(ipPort[1], 10, 32)
							expectedGateway := createExpectedIstioEgressGateway(task, playerHost, egressPorts[i])
							Expect(createdResources[i*4].obj).To(
								Equal(InterfaceToUnstructured(expectedGateway)))
							Expect(createdDR).To(
								Equal(InterfaceToUnstructured(
									createExpectedDestinationRule(task, uint(pId), playerHost, tlsConfig))))
							expectedSe := createExpectedServiceEntry(task, uint(pId), playerHost, playerIp, uint32(playerPort))
							uese := InterfaceToUnstructured(expectedSe)
							// we need to set the location to MESH_EXTERNAL manually
							// because of a marshalling error where the location is
							// not set correctly
							uese.Object["spec"].(map[string]interface{})["location"] = "MESH_EXTERNAL"
							Expect(createdResources[i*4+2].obj).To(
								Equal(uese))
							Expect(createdResources[i*4+3].obj).To(
								Equal(InterfaceToUnstructured(
									createExpectedEgressVirtualService(task, expectedGateway, uint(pId), playerHost, uint32(playerPort), egressPorts[i], egressServiceHost))))
						}
						Expect(len(endpointsToServe)).To(BeZero())
					})
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
						fmt.Errorf("failed to delete AuthorizationPolicies: %w", deleteErr),
					}
					Expect(err).To(Equal(fmt.Errorf("failed to delete networking resources: %v", expectedResourceDeleteErrs)))
				})
			})
			When("deleting resources succeeds", func() {
				var (
					resourcesExpectedToBeDeleted = []schema.GroupVersionKind{{
						Group:   istioNetworkingGroup,
						Version: istioVersion,
						Kind:    "Gateway",
					}, {
						Group:   istioNetworkingGroup,
						Version: istioVersion,
						Kind:    "VirtualService",
					}, {
						Group:   istioNetworkingGroup,
						Version: istioVersion,
						Kind:    "ServiceEntry",
					}, {
						Group:   istioNetworkingGroup,
						Version: istioVersion,
						Kind:    "DestinationRule",
					}, {
						Group:   istioSecurityGroup,
						Version: istioVersion,
						Kind:    "AuthorizationPolicy",
					}}
				)
				BeforeEach(func() {
					fakeK8sClient.DeleteAllReturnError = nil
				})

				It("should delete all resources", func() {
					err := networkManager.DeleteNetworkingForTask(ctx, task)
					Expect(err).NotTo(HaveOccurred())
					deletedResources := fakeK8sClient.DeleteAllCallParams
					Expect(deletedResources).To(HaveLen(len(resourcesExpectedToBeDeleted)))
					deletedKinds := make([]schema.GroupVersionKind, 0)
					for _, resource := range deletedResources {
						groupVersionKind := resource.obj.GetObjectKind().GroupVersionKind()
						deletedKinds = append(deletedKinds, groupVersionKind)
					}
					Expect(deletedKinds).To(ConsistOf(resourcesExpectedToBeDeleted))
				})
			})
		})
	})
})

func createExpectedEgressVirtualService(task *klyshkov1alpha1.TupleGenerationTask, gateway *IstioGateway, playerID uint, playerHost string, playerPort uint32, egressPort uint32, egressServiceHost string) *IstioVirtualService {
	refLabels := map[string]string{
		TaskLabel: task.Name,
	}
	ivs := NewIstioVirtualService(
		metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-egress-vs-%d", task.Name, playerID),
			Namespace: task.Namespace,
			Labels:    refLabels,
		},
		&v1beta1.VirtualService{
			Hosts: []string{playerHost},
			Gateways: []string{
				"mesh",
				gateway.Name,
			},
			Tcp: []*v1beta1.TCPRoute{
				{
					Match: []*v1beta1.L4MatchAttributes{
						{
							Port:         InterCRGNetworkingPort,
							SourceLabels: refLabels,
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
								gateway.Name,
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

func createExpectedServiceEntry(task *klyshkov1alpha1.TupleGenerationTask, playerID uint, playerHost string, playerIp string, playerPort uint32) *IstioServiceEntry {
	ise := NewIstioServiceEntry(
		metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-se-%d", task.Name, playerID),
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&v1beta1.ServiceEntry{
			Hosts:     []string{playerHost},
			Addresses: []string{fmt.Sprintf("%s/32", playerIp)},
			Ports: []*v1beta1.Port{
				{
					Number:     InterCRGNetworkingPort,
					Protocol:   "TCP",
					TargetPort: 0,
				},
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
								Host: task.Name,
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

func createExpectedIstioIngressGateway(task *klyshkov1alpha1.TupleGenerationTask, port uint32, tlsConfig *TLSConfig) *IstioGateway {
	igw := NewIstioGateway(
		metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ingress-gateway", task.Name),
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
						Name:     fmt.Sprintf("crg-ingress-tcp-%d", port),
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

func createExpectedIstioEgressGateway(task *klyshkov1alpha1.TupleGenerationTask, host string, port uint32) *IstioGateway {
	igw := NewIstioGateway(
		metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-egress-gateway", task.Name),
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
						Name:     fmt.Sprintf("crg-egress-tcp-%d", port),
					},
					Hosts: []string{host},
				},
			},
			Selector: map[string]string{
				"istio": "egressgateway",
			},
		})
	return &igw
}

func createExpectedAuthorizationPolicy(task *klyshkov1alpha1.TupleGenerationTask, port uint32) *IstioAuthorizationPolicy {
	authorizationPolicy := NewIstioAuthorizationPolicy(
		metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-auth-policy", task.Name),
			Namespace: "istio-system",
			Labels: map[string]string{
				TaskNetworkLabel: task.Name,
			},
		},
		&isec.AuthorizationPolicy{
			Action: isec.AuthorizationPolicy_ALLOW,
			Rules: []*isec.Rule{
				{
					To: []*isec.Rule_To{
						{
							Operation: &isec.Operation{
								Ports: []string{
									fmt.Sprintf("%d", port),
								},
							},
						},
					},
				},
			},
			Selector: &itype.WorkloadSelector{
				MatchLabels: map[string]string{
					"istio": "ingressgateway",
				},
			},
		})
	return &authorizationPolicy
}
