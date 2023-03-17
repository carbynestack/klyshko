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
	"github.com/carbynestack/klyshko/logging"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sort"
)

// SchedulingStrategy is used to decide for which tuple type a tuple generation job should be launched next.
type SchedulingStrategy interface {

	// Schedule returns the tuple type for which tuples should be generated next based on the number of tuples
	// available in Castor and the currently running jobs. In case no tuples should be generated `nil` is returned.
	// Arguments must not be altered by implementations.
	Schedule(ctx context.Context, telemetry castor.Telemetry, activeJobs []v1alpha1.TupleGenerationJob) *string
}

// LeastAvailableFirstStrategy selects the tuple type for which the least tuples are available or being generated and
// for which the number is below a given threshold.
type LeastAvailableFirstStrategy struct {

	// Threshold is the lower bound of available tuples above which tuple types are eligible for being selected by this
	// strategy.
	Threshold int
}

// Schedule returns the tuple type selected by this strategy or `nil` if none is eligible.
func (s *LeastAvailableFirstStrategy) Schedule(ctx context.Context, telemetry castor.Telemetry, activeJobs []v1alpha1.TupleGenerationJob) *string {
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

	// Filter for those that are below threshold
	var belowThreshold []castor.TupleMetrics
	for _, m := range t.TupleMetrics {
		if m.Available < s.Threshold {
			belowThreshold = append(belowThreshold, m)
		}
	}
	logger.V(logging.DEBUG).Info("Filtered for eligible types", "Metrics.Eligible", belowThreshold)

	// Terminate early if no tuple type is below threshold
	if len(belowThreshold) == 0 {
		logger.V(logging.DEBUG).Info("Above threshold for all tuple types")
		return nil
	}

	// Sort ascending by quantity
	sort.Slice(belowThreshold, func(i, j int) bool {
		return belowThreshold[i].Available < belowThreshold[j].Available
	})
	logger.V(logging.DEBUG).Info("Sorted by priority", "Metrics.Sorted", belowThreshold)

	// Return tuple type with minimum number of tuples available or being generated
	return &belowThreshold[0].TupleType
}
