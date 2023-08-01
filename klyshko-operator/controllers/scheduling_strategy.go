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
	"github.com/carbynestack/klyshko/logging"
	"math/rand"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SchedulingStrategy is used to decide for which tuple type a tuple generation job should be launched next. Note that
// strategies are instantiated for each scheduling decision.
type SchedulingStrategy interface {

	// Schedule returns the tuple type for which tuples should be generated next based on the number of tuples
	// available in Castor, a set of tuple-specific policies, and the currently running jobs. In case no tuples should
	// be generated `nil` is returned. Arguments must not be altered by implementations.
	Schedule(ctx context.Context, telemetry castor.Telemetry, policies []v1alpha1.TupleTypePolicy, activeJobs []v1alpha1.TupleGenerationJob) *string
}

// LotterySchedulingStrategy behaves as if it were selecting the tuple type for which to generate tuples next by a
// lottery. More specifically, it can be thought of as assigning lottery tickets to each eligible, i.e, below-threshold
// tuple type, and then selecting a winner by drawing a random ticket.
type LotterySchedulingStrategy struct {
}

// Schedule returns the tuple type selected by this strategy or `nil` if none is eligible.
func (s *LotterySchedulingStrategy) Schedule(
	ctx context.Context, telemetry castor.Telemetry, policies []v1alpha1.TupleTypePolicy,
	activeJobs []v1alpha1.TupleGenerationJob) *string {

	logger := log.FromContext(ctx)

	// Compute aggregate of available and in-flight tuples
	t := telemetry.DeepCopy()
	for _, j := range activeJobs {
		for idx := range t.TupleMetrics {
			if j.Spec.Type == t.TupleMetrics[idx].TupleType {
				t.TupleMetrics[idx].Available = t.TupleMetrics[idx].Available + j.Spec.Count
				break
			}
		}
	}
	logger.V(logging.DEBUG).Info("With in-flight tuple generation jobs", "Metrics.WithInflight", t.TupleMetrics)

	// Create map of policies by tuple type
	policyByType := map[string]v1alpha1.TupleTypePolicy{}
	for _, policy := range policies {
		policyByType[policy.Type] = policy
	}

	// Filter for those tuple types with a policy that are below their threshold and accumulate all weights
	weightSum := 0
	belowThreshold := map[string]castor.TupleMetrics{}
	for _, m := range t.TupleMetrics {
		if policy, exists := policyByType[m.TupleType]; exists {
			if m.Available < policy.Threshold {
				weightSum += policy.Priority
				belowThreshold[m.TupleType] = m
			}
		}
	}
	logger.V(logging.DEBUG).Info("Filtered for eligible types", "TupleTypes.Eligible", belowThreshold, "Weight", weightSum)

	// Terminate early if no tuple type is below threshold
	if len(belowThreshold) == 0 {
		logger.V(logging.DEBUG).Info("Above threshold for all tuple types")
		return nil
	}

	// Randomly select a tuple type with probability determined by their individual weight and the sum of all weights
	r := rand.Intn(weightSum)
	current := 0
	var selectedTupleType *string = nil
	for tupleType := range belowThreshold {
		policy := policyByType[tupleType]
		current += policy.Priority
		if r < current {
			selectedTupleType = &policy.Type
			break
		}
	}

	return selectedTupleType
}
