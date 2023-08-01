/*
Copyright (c) 2022-2023 - for information on the respective copyright owner
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

var _ = Context("LotterySchedulingStrategy", func() {

	// Test data
	strategy := &LotterySchedulingStrategy{}
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
			policies := []v1alpha1.TupleTypePolicy{
				{
					Type:      TupleTypeA,
					Threshold: 1000,
					Priority:  1.0},
				{
					Type:      TupleTypeB,
					Threshold: 10000,
					Priority:  1.0,
				},
			}
			It("returns nil", func() {
				selected := strategy.Schedule(context.Background(), telemetry, policies, noActiveJobs)
				Expect(selected).To(BeNil())
			})
		})

		When("there is a single tuple type below the threshold", func() {
			policies := []v1alpha1.TupleTypePolicy{
				{
					Type:      TupleTypeA,
					Threshold: 2000,
					Priority:  1.0},
				{
					Type:      TupleTypeB,
					Threshold: 100000,
					Priority:  1.0,
				},
			}
			It("returns that tuple type", func() {
				selected := strategy.Schedule(context.Background(), telemetry, policies, noActiveJobs)
				Expect(*selected).To(Equal(TupleTypeB))
			})
		})

		When("there are two tuple types below the threshold", func() {
			policies := []v1alpha1.TupleTypePolicy{
				{
					Type:      TupleTypeA,
					Threshold: 20000,
					Priority:  1},
				{
					Type:      TupleTypeB,
					Threshold: 100000,
					Priority:  9,
				},
			}
			It("returns one of them", func() {
				selected := strategy.Schedule(context.Background(), telemetry, policies, noActiveJobs)
				Expect(*selected).To(BeElementOf(TupleTypeA, TupleTypeB))
			})
			It("returns them according to their priorities", func() {
				counts := map[string]int{
					TupleTypeA: 0,
					TupleTypeB: 0,
				}
				count := 10000
				for i := 0; i < count; i++ {
					selected := strategy.Schedule(context.Background(), telemetry, policies, noActiveJobs)
					Expect(selected).NotTo(BeNil())
					counts[*selected]++
				}
				sumOfPriorities := policies[0].Priority + policies[1].Priority
				Expect(float64(counts[TupleTypeA]) / float64(count)).To(BeNumerically("~", float64(policies[0].Priority)/float64(sumOfPriorities), 0.01))
				Expect(float64(counts[TupleTypeB]) / float64(count)).To(BeNumerically("~", float64(policies[1].Priority)/float64(sumOfPriorities), 0.01))
			})
		})

		When("there is a tuple type with an active job above the threshold", func() {
			policies := []v1alpha1.TupleTypePolicy{
				{
					Type:      TupleTypeA,
					Threshold: 10000,
					Priority:  1.0,
				},
			}
			It("returns nil", func() {
				selected := strategy.Schedule(context.Background(), telemetry, policies, someActiveJobs)
				Expect(selected).To(BeNil())
			})
		})

		When("there is a tuple type with an active job below the threshold", func() {
			policies := []v1alpha1.TupleTypePolicy{
				{
					Type:      TupleTypeA,
					Threshold: 100000,
					Priority:  1.0,
				},
			}
			It("returns that tuple type", func() {
				selected := strategy.Schedule(context.Background(), telemetry, policies, someActiveJobs)
				Expect(*selected).To(Equal(TupleTypeA))
			})
		})

	})

})
