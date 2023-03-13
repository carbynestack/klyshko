/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/
package controllers

import (
	"context"
	"github.com/carbynestack/klyshko/api/v1alpha1"
	"github.com/carbynestack/klyshko/castor"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

var _ = Context("LeastAvailableFirstStrategy", func() {

	// Test data
	const TupleTypeA = "A"
	const TupleTypeB = "B"
	telemetry := castor.Telemetry{
		TupleMetrics: []castor.TupleMetrics{
			{
				Available:       5000,
				ConsumptionRate: 0,
				TupleType:       TupleTypeA,
			},
			{
				Available:       50000,
				ConsumptionRate: 0,
				TupleType:       TupleTypeB,
			},
		},
	}
	someActiveJobs := []v1alpha1.TupleGenerationJob{
		{Spec: v1alpha1.TupleGenerationJobSpec{
			Type:  TupleTypeA,
			Count: 30000,
		}},
	}
	var noActiveJobs []v1alpha1.TupleGenerationJob

	Describe("Selecting a tuple type", func() {

		When("there is no tuple type below the threshold", func() {
			It("returns nil", func() {
				strategy := &LeastAvailableFirstStrategy{Threshold: 1000}
				selected := strategy.Schedule(context.Background(), telemetry, noActiveJobs)
				Expect(selected).To(BeNil())
			})
		})

		When("there is a single tuple type below the threshold", func() {
			It("returns that tuple type", func() {
				strategy := &LeastAvailableFirstStrategy{Threshold: 10000}
				selected := strategy.Schedule(context.Background(), telemetry, noActiveJobs)
				Expect(*selected).To(Equal(TupleTypeA))
			})
		})

		When("there are two tuple types below the threshold", func() {
			It("returns the tuple type with less tuples available", func() {
				strategy := &LeastAvailableFirstStrategy{Threshold: 100000}
				selected := strategy.Schedule(context.Background(), telemetry, noActiveJobs)
				Expect(*selected).To(Equal(TupleTypeA))
			})
		})

		When("there is a tuple type with an active job above the threshold", func() {
			It("returns nil", func() {
				strategy := &LeastAvailableFirstStrategy{Threshold: 30000}
				selected := strategy.Schedule(context.Background(), telemetry, someActiveJobs)
				Expect(selected).To(BeNil())
			})
		})

		When("there is a tuple type with an active job below the threshold", func() {
			It("returns that tuple type", func() {
				strategy := &LeastAvailableFirstStrategy{Threshold: 40000}
				selected := strategy.Schedule(context.Background(), telemetry, someActiveJobs)
				Expect(*selected).To(Equal(TupleTypeA))
			})
		})

	})

})
