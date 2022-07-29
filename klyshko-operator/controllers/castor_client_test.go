/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const validCastorURL = "http://cs-castor.default.svc.cluster.local:10100"
const invalidCastorURL = "http://cs-castor.default.svc.cluster.local:10101"

var _ = Describe("Fetching telemetry", func() {

	ctx := context.TODO()

	AfterEach(func() {
		httpmock.DeactivateAndReset()
	})

	When("when Castor services responds with expected JSON data", func() {

		expectedTelemetry := Telemetry{TupleMetrics: []TupleMetrics{
			{
				Available:       0,
				ConsumptionRate: 0,
				TupleType:       "MULTIPLICATION_TRIPLE_GFP",
			},
		}}

		BeforeEach(func() {
			httpmock.Activate()
			responder, err := httpmock.NewJsonResponder(200, expectedTelemetry)
			Expect(err).NotTo(HaveOccurred())
			httpmock.RegisterResponder(
				"GET",
				fmt.Sprintf("=~^%s/intra-vcp/telemetry", validCastorURL),
				responder,
			)
		})
		It("succeeds", func() {
			castorClient := NewCastorClient(validCastorURL)
			telemetry, err := castorClient.getTelemetry(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(telemetry).To(Equal(expectedTelemetry))
		})
	})

	When("when Castor services responds with non-success status code", func() {
		BeforeEach(func() {
			httpmock.Activate()
			httpmock.RegisterResponder(
				"GET",
				fmt.Sprintf("=~^%s/intra-vcp/telemetry", validCastorURL),
				httpmock.NewStringResponder(404, ""),
			)
		})
		It("fails", func() {
			castorClient := NewCastorClient(validCastorURL)
			_, err := castorClient.getTelemetry(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	When("when Castor service is not available", func() {
		It("fails", func() {
			castorClient := NewCastorClient(invalidCastorURL)
			_, err := castorClient.getTelemetry(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

})

var _ = Describe("Activating a tuple chunk", func() {

	ctx := context.TODO()

	AfterEach(func() {
		httpmock.DeactivateAndReset()
	})

	When("when Castor services responds with success status code", func() {
		BeforeEach(func() {
			httpmock.Activate()
			httpmock.RegisterResponder(
				"PUT",
				fmt.Sprintf("=~^%s/intra-vcp/tuple-chunks/activate/.*", validCastorURL),
				httpmock.NewStringResponder(200, ""),
			)
		})
		It("succeeds", func() {
			chunkId := uuid.New()
			castorClient := NewCastorClient(validCastorURL)
			err := castorClient.activateTupleChunk(ctx, chunkId)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	When("when Castor services responds with bon-success status code", func() {
		BeforeEach(func() {
			httpmock.Activate()
			httpmock.RegisterResponder(
				"PUT",
				fmt.Sprintf("=~^%s/intra-vcp/tuple-chunks/activate/.*", validCastorURL),
				httpmock.NewStringResponder(404, ""),
			)
		})
		It("fails", func() {
			chunkId := uuid.New()
			castorClient := NewCastorClient(validCastorURL)
			err := castorClient.activateTupleChunk(ctx, chunkId)
			Expect(err).To(HaveOccurred())
		})
	})

	When("when Castor service is not available", func() {
		It("fails", func() {
			chunkId := uuid.New()
			castorClient := NewCastorClient(invalidCastorURL)
			err := castorClient.activateTupleChunk(ctx, chunkId)
			Expect(err).To(HaveOccurred())
		})
	})

})
