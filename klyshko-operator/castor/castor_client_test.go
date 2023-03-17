/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package castor

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

	When("when Castor service responds with expected JSON data", func() {

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
			castorClient := NewClient(validCastorURL)
			telemetry, err := castorClient.GetTelemetry(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(telemetry).To(Equal(expectedTelemetry))
		})
	})

	When("when Castor service responds with non-success status code", func() {
		BeforeEach(func() {
			httpmock.Activate()
			httpmock.RegisterResponder(
				"GET",
				fmt.Sprintf("=~^%s/intra-vcp/telemetry", validCastorURL),
				httpmock.NewStringResponder(404, ""),
			)
		})
		It("fails", func() {
			castorClient := NewClient(validCastorURL)
			_, err := castorClient.GetTelemetry(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	When("when Castor service is not available", func() {
		It("fails", func() {
			castorClient := NewClient(invalidCastorURL)
			_, err := castorClient.GetTelemetry(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

})

var _ = Describe("Activating a tuple chunk", func() {

	ctx := context.TODO()

	AfterEach(func() {
		httpmock.DeactivateAndReset()
	})

	When("when Castor service responds with success status code", func() {
		BeforeEach(func() {
			httpmock.Activate()
			httpmock.RegisterResponder(
				"PUT",
				fmt.Sprintf("=~^%s/intra-vcp/tuple-chunks/activate/.*", validCastorURL),
				httpmock.NewStringResponder(200, ""),
			)
		})
		It("succeeds", func() {
			chunkID := uuid.New()
			castorClient := NewClient(validCastorURL)
			err := castorClient.ActivateTupleChunk(ctx, chunkID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	When("when Castor service responds with non-success status code", func() {
		BeforeEach(func() {
			httpmock.Activate()
			httpmock.RegisterResponder(
				"PUT",
				fmt.Sprintf("=~^%s/intra-vcp/tuple-chunks/activate/.*", validCastorURL),
				httpmock.NewStringResponder(404, ""),
			)
		})
		It("fails", func() {
			chunkID := uuid.New()
			castorClient := NewClient(validCastorURL)
			err := castorClient.ActivateTupleChunk(ctx, chunkID)
			Expect(err).To(HaveOccurred())
		})
	})

	When("when Castor service is not available", func() {
		It("fails", func() {
			chunkID := uuid.New()
			castorClient := NewClient(invalidCastorURL)
			err := castorClient.ActivateTupleChunk(ctx, chunkID)
			Expect(err).To(HaveOccurred())
		})
	})

})

var _ = When("Creating a deep copy of a telemetry struct", func() {
	It("succeeds", func() {
		original := &Telemetry{TupleMetrics: []TupleMetrics{
			{
				Available:       0,
				ConsumptionRate: 0,
				TupleType:       "a",
			},
		}}
		deepCopy := original.DeepCopy()
		Expect(deepCopy.TupleMetrics).To(Equal(original.TupleMetrics))
		deepCopy.TupleMetrics[0].Available = 2
		Expect(original.TupleMetrics[0].Available).To(Equal(0))
	})
})
