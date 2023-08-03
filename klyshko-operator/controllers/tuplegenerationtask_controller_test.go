/*
Copyright (c) 2023 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Getting the inter-CRG networking service endpoint", func() {
	When("the service is not exposed via a load balance", func() {
		It("yields an error", func() {
			svc := &v1.Service{
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeNodePort,
				},
			}
			_, err := endpoint(svc)
			Expect(err).Should(HaveOccurred())
		})
	})
	When("no ingress is available", func() {
		It("returns nil", func() {
			svc := &v1.Service{
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{},
					},
				},
			}
			endpoint, err := endpoint(svc)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(endpoint).To(BeNil())
		})
	})
	When("an IP is attached to the ingress", func() {
		It("returns that IP", func() {
			ip := "127.0.0.1"
			svc := &v1.Service{
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{
								IP: ip,
							},
						},
					},
				},
			}
			endpoint, err := endpoint(svc)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*endpoint).To(Equal(ip))
		})
	})
	When("a hostname is attached to the ingress", func() {
		It("returns that hostname", func() {
			hostname := "apollo"
			svc := &v1.Service{
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{
								Hostname: hostname,
							},
						},
					},
				},
			}
			endpoint, err := endpoint(svc)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*endpoint).To(Equal(hostname))
		})
	})
	When("neither an IP nor a hostname is attached to the ingress", func() {
		It("yields an error", func() {
			svc := &v1.Service{
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{},
						},
					},
				},
			}
			_, err := endpoint(svc)
			Expect(err).Should(HaveOccurred())
		})
	})
})
