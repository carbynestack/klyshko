/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

var _ = Context("TODO", func() { // TODO Description?

	ctx := context.Background()
	var vcp *Vcp

	BeforeEach(func() {
		var err error
		vcp, err = setupVcp()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := vcp.tearDownVcp()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Getting the local player Id", func() {

		When("when no configuration has been provided", func() {
			It("fails", func() {
				_, err := localPlayerID(ctx, &vcp.k8sClient, "default")
				Expect(err).To(HaveOccurred())
			})
		})

		When("when valid configuration has been provided", func() {
			BeforeEach(func() {
				vcp.createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
					"playerCount": "2",
					"playerId":    "0",
				})
			})
			AfterEach(func() {
				vcp.deleteVcpConfig(ctx, "cs-vcp-config", "default")
			})
			It("gives the right local player ID", func() {
				Expect(localPlayerID(ctx, &vcp.k8sClient, "default")).To(Equal(uint(0)))
			})
		})

		When("when the playerId K/V pair is missing", func() {
			BeforeEach(func() {
				vcp.createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
					"playerCount": "2",
				})
			})
			AfterEach(func() {
				vcp.deleteVcpConfig(ctx, "cs-vcp-config", "default")
			})
			It("fails", func() {
				_, err := localPlayerID(ctx, &vcp.k8sClient, "default")
				Expect(err).To(HaveOccurred())
			})
		})

		When("when the playerId is out of range", func() {
			BeforeEach(func() {
				vcp.createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
					"playerCount": "2",
					"playerId":    "-1",
				})
			})
			AfterEach(func() {
				vcp.deleteVcpConfig(ctx, "cs-vcp-config", "default")
			})
			It("fails", func() {
				_, err := localPlayerID(ctx, &vcp.k8sClient, "default")
				Expect(err).To(HaveOccurred())
			})
		})

		When("when the playerId can't be parsed", func() {
			BeforeEach(func() {
				vcp.createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
					"playerCount": "2",
					"playerId":    "a1b2",
				})
			})
			AfterEach(func() {
				vcp.deleteVcpConfig(ctx, "cs-vcp-config", "default")
			})
			It("fails", func() {
				_, err := localPlayerID(ctx, &vcp.k8sClient, "default")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Getting the number of players", func() {

		When("when no configuration has been provided", func() {
			It("fails", func() {
				_, err := numberOfPlayers(ctx, &vcp.k8sClient, "default")
				Expect(err).To(HaveOccurred())
			})
		})

		When("when valid configuration has been provided", func() {
			BeforeEach(func() {
				vcp.createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
					"playerCount": "2",
					"playerId":    "0",
				})
			})
			AfterEach(func() {
				vcp.deleteVcpConfig(ctx, "cs-vcp-config", "default")
			})
			It("gives the right number of Players", func() {
				Expect(numberOfPlayers(ctx, &vcp.k8sClient, "default")).To(Equal(uint(2)))
			})
		})

		When("when the playerCount K/V pair is missing", func() {
			BeforeEach(func() {
				vcp.createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
					"playerId": "0",
				})
			})
			AfterEach(func() {
				vcp.deleteVcpConfig(ctx, "cs-vcp-config", "default")
			})
			It("fails", func() {
				_, err := numberOfPlayers(ctx, &vcp.k8sClient, "default")
				Expect(err).To(HaveOccurred())
			})
		})

		When("when the playerCount is out of range", func() {
			BeforeEach(func() {
				vcp.createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
					"playerCount": "-1",
					"playerId":    "0",
				})
			})
			AfterEach(func() {
				vcp.deleteVcpConfig(ctx, "cs-vcp-config", "default")
			})
			It("fails", func() {
				_, err := numberOfPlayers(ctx, &vcp.k8sClient, "default")
				Expect(err).To(HaveOccurred())
			})
		})

		When("when the playerId can't be parsed", func() {
			BeforeEach(func() {
				vcp.createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
					"playerCount": "a1b2",
					"playerId":    "0",
				})
			})
			AfterEach(func() {
				vcp.deleteVcpConfig(ctx, "cs-vcp-config", "default")
			})
			It("fails", func() {
				_, err := numberOfPlayers(ctx, &vcp.k8sClient, "default")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
