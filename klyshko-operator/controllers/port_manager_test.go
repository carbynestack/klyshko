/*
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync"
)

var _ = Describe("Managing ports", func() {
	var (
		ctx                  context.Context
		cancel               context.CancelFunc
		portRange            = &PortRange{Min: 30500, Max: 30504}
		fakeUsedPortSupplier *FakeUsedPortSupplier
		portManager          *defaultPortManager
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.TODO())
		fakeUsedPortSupplier = NewFakeUsedPortSupplier()
		portManager = &defaultPortManager{
			usedPortSupplier: fakeUsedPortSupplier,
			portRange:        portRange,
			mtx:              sync.Mutex{},
		}
	})

	AfterEach(func() {
		cancel()
	})

	When("creating a new port manager", func() {
		When("max port is less than min port", func() {
			It("should return an error", func() {
				portRange, err := NewPortRange(30504, 30500)
				Expect(portRange).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})

		When("max and min ports are valid", func() {
			It("should return a new port manager", func() {
				var min, max uint32
				min, max = 30500, 30504
				portRange, err := NewPortRange(min, max)
				Expect(portRange).ToNot(BeNil())
				Expect(err).NotTo(HaveOccurred())
				Expect(portRange.Min).To(Equal(min))
				Expect(portRange.Max).To(Equal(max))
			})
		})
	})

	When("getting a free port", func() {

		When("usedPortSupplier fails to get used ports", func() {
			BeforeEach(func() {
				fakeUsedPortSupplier.Reset()
			})

			It("should return an error", func() {
				port, err := portManager.GetFreePort(ctx)
				Expect(port).To(Equal(uint32(0)))
				Expect(err).To(HaveOccurred())
			})
		})

		When("all ports are in use", func() {
			BeforeEach(func() {
				fakeUsedPortSupplier.GetUsedPortsReturnErr = nil
				usedPorts := make([]uint32, portRange.Max-portRange.Min+1)
				for i := range usedPorts {
					usedPorts[i] = portRange.Min + uint32(i)
				}
				fakeUsedPortSupplier.GetUsedPortsReturnList = usedPorts
			})

			It("should return an error", func() {
				port, err := portManager.GetFreePort(ctx)
				Expect(port).To(Equal(uint32(0)))
				Expect(err).To(HaveOccurred())
			})
		})

		When("some ports are in use", func() {
			BeforeEach(func() {
				fakeUsedPortSupplier.GetUsedPortsReturnErr = nil
				items := []uint32{30500, 30502}
				fakeUsedPortSupplier.GetUsedPortsReturnList = items
			})

			It("should return the first free port", func() {
				port, err := portManager.GetFreePort(ctx)
				Expect(port).To(Equal(uint32(30501)))
				Expect(err).To(BeNil())
			})
		})
	})
})

var _ = Describe("Finding used ports", func() {
	var (
		ctx           context.Context
		cancel        context.CancelFunc
		portRange     = &PortRange{Min: 30500, Max: 30504}
		fakeK8sReader *FakeK8sReader
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.TODO())
		fakeK8sReader = NewFakeK8sReader()
	})

	AfterEach(func() {
		cancel()
		fakeK8sReader.Reset()
	})

	Describe("for ingress gateways", func() {
		var (
			portSupplier usedPortSupplier
		)

		BeforeEach(func() {
			portSupplier = &gatewayUsedPortSupplier{
				portRange: portRange,
				k8sReader: fakeK8sReader,
				logger:    ctrl.Log.WithName("test").WithName("GatewayPortManager"),
			}
		})

		When("getting a free port", func() {

			When("k8s fails to list the gateways", func() {
				BeforeEach(func() {
					fakeK8sReader.Reset()
				})

				It("should return an error", func() {
					_, err := portSupplier.getUsedPortsInRange(ctx)
					Expect(err).To(HaveOccurred())
				})
			})

			When("no gateways exist", func() {
				BeforeEach(func() {
					fakeK8sReader.ListReturnError = nil
					fakeK8sReader.ListReturnObject = &unstructured.UnstructuredList{}
				})

				It("should return an empty list", func() {
					ports, err := portSupplier.getUsedPortsInRange(ctx)
					Expect(ports).To(Equal([]uint32{}))
					Expect(err).To(BeNil())
				})
			})

			When("gateways exist", func() {
				BeforeEach(func() {
					fakeK8sReader.ListReturnError = nil
					items := make([]unstructured.Unstructured, 4)
					items[0] = *InterfaceToUnstructured(
						NewIstioGatewayUsingPort(30000))
					items[1] = *InterfaceToUnstructured(
						NewIstioGatewayUsingPort(30500))
					items[2] = *InterfaceToUnstructured(
						NewIstioGatewayUsingPort(30502))
					items[3] = *InterfaceToUnstructured(
						NewIstioGatewayUsingPort(30505))
					fakeK8sReader.ListReturnObject = &unstructured.UnstructuredList{
						Items: items,
					}
				})

				It("should return the used ports in range", func() {
					ports, err := portSupplier.getUsedPortsInRange(ctx)
					Expect(ports).To(Equal([]uint32{30500, 30502}))
					Expect(err).To(BeNil())
				})
			})
		})
	})

	Describe("for egress gateways", func() {
		var (
			portSupplier      usedPortSupplier
			egressGatewayName = "test-egressgateway"
			egressServiceHost = "test-egressgateway.default.svc.cluster.local"
		)

		BeforeEach(func() {
			portSupplier = &egressUsedPortSupplier{
				portRange:         portRange,
				k8sReader:         fakeK8sReader,
				egressServiceHost: egressServiceHost,
				egressGatewayName: egressGatewayName,
				logger:            ctrl.Log.WithName("test").WithName("EgressPortManager"),
			}
		})

		When("getting a free port", func() {

			When("k8s fails to list the virtual services", func() {
				BeforeEach(func() {
					fakeK8sReader.Reset()
				})

				It("should return an error", func() {
					ports, err := portSupplier.getUsedPortsInRange(ctx)
					Expect(ports).To(Equal([]uint32{}))
					Expect(err).To(HaveOccurred())
				})
			})

			When("no virtual service exist", func() {
				BeforeEach(func() {
					fakeK8sReader.ListReturnError = nil
					fakeK8sReader.ListReturnObject = &unstructured.UnstructuredList{}
				})

				It("should return an empty list", func() {
					ports, err := portSupplier.getUsedPortsInRange(ctx)
					Expect(ports).To(Equal([]uint32{}))
					Expect(err).To(BeNil())
				})
			})

			When("virtual services exist", func() {
				BeforeEach(func() {
					fakeK8sReader.ListReturnError = nil
					items := make([]unstructured.Unstructured, 2)
					items[0] = *InterfaceToUnstructured(
						NewIstioVirtualServiceUsingPort(egressGatewayName, egressServiceHost, 30400, 30500))
					items[1] = *InterfaceToUnstructured(
						NewIstioVirtualServiceUsingPort(egressGatewayName, egressServiceHost, 30502, 30505))
					fakeK8sReader.ListReturnObject = &unstructured.UnstructuredList{
						Items: items,
					}
				})

				It("should return the used ports in range", func() {
					ports, err := portSupplier.getUsedPortsInRange(ctx)
					Expect(ports).To(Equal([]uint32{30500, 30502}))
					Expect(err).To(BeNil())
				})
			})
		})
	})
})
