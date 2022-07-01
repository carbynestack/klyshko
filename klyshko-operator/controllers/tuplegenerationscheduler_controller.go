/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"github.com/google/uuid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
)

const MinimumTuplesPerJob = 10000

type TupleGenerationSchedulerReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	CastorClient *CastorClient
}

//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationschedulers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationschedulers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationschedulers/finalizers,verbs=update

func (r *TupleGenerationSchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch scheduler resource
	scheduler := &klyshkov1alpha1.TupleGenerationScheduler{}
	err := r.Get(ctx, req.NamespacedName, scheduler)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Scheduler resource not available -> has been deleted
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "failed to read scheduler resource")
		return ctrl.Result{}, err
	}

	activeJobs, err := r.getMatchingJobs(ctx, func(job klyshkov1alpha1.TupleGenerationJob) bool {
		return !job.Status.State.IsDone()
	})
	if err != nil {
		logger.Error(err, "failed to fetch active jobs")
		return ctrl.Result{}, err
	}

	// Stop if already at maximum concurrency level
	activeJobCount := len(activeJobs)
	if scheduler.Spec.Concurrency <= activeJobCount {
		logger.Info("at maximum concurrency level", "Jobs.Active", activeJobCount, "Scheduler.Concurrency", scheduler.Spec.Concurrency)
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	// Least available first strategy for selecting tuples to create a job for
	// 1. Compute available and in generation number of tuples per type
	// 2. Filter out all above threshold
	// 3. Sort ascending wrt to sum from step 1
	telemetry, err := r.CastorClient.getTelemetry(ctx)
	if err != nil {
		return ctrl.Result{RequeueAfter: 60 * time.Second}, err
	}
	logger.Info("tuple telemetry data fetched", "Metrics.Available", telemetry.TupleMetrics)
	for _, j := range activeJobs {
		for idx := range telemetry.TupleMetrics {
			if j.Spec.Type == telemetry.TupleMetrics[idx].TupleType {
				telemetry.TupleMetrics[idx].Available = telemetry.TupleMetrics[idx].Available + int(j.Spec.Count)
				break
			}
		}
	}
	logger.Info("with in-flight tuple generation jobs", "Metrics.WithInflight", telemetry.TupleMetrics)
	var belowThreshold []TupleMetrics
	for _, m := range telemetry.TupleMetrics {
		if m.Available < scheduler.Spec.Threshold {
			belowThreshold = append(belowThreshold, m)
		}
	}
	logger.Info("filtered for eligible types", "Metrics.Eligible", belowThreshold)
	if len(belowThreshold) == 0 {
		logger.Info("above threshold for all tuple types - do nothing")
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}
	sort.Slice(belowThreshold, func(i, j int) bool {
		return belowThreshold[i].Available < belowThreshold[j].Available
	})
	logger.Info("sorted by priority", "Metrics.Sorted", belowThreshold)

	// Create job for first below threshold
	jobID := uuid.New().String()
	job := &klyshkov1alpha1.TupleGenerationJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "klyshko.carbnyestack.io/v1alpha1",
			Kind:       "TupleGenerationJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      scheduler.Name + "-" + jobID,
			Namespace: req.Namespace,
		},
		Spec: klyshkov1alpha1.TupleGenerationJobSpec{
			ID:    jobID,
			Type:  belowThreshold[0].TupleType,
			Count: MinimumTuplesPerJob, // TODO Make this configurable
		},
	}
	err = ctrl.SetControllerReference(scheduler, job, r.Scheme)
	if err != nil {
		logger.Error(err, "could not set owner reference on job", "Job", job)
		return ctrl.Result{}, err
	}
	err = r.Create(ctx, job)
	if err != nil {
		logger.Error(err, "job creation failed", "Job", job)
		return ctrl.Result{}, err
	}
	logger.Info("job created", "Job", job)

	// Delete all finished jobs
	finishedJobs, err := r.getMatchingJobs(ctx, func(job klyshkov1alpha1.TupleGenerationJob) bool {
		return job.Status.State.IsDone()
	})
	if err != nil {
		logger.Error(err, "failed to fetch finished jobs")
		return ctrl.Result{}, err
	}
	for _, j := range finishedJobs {
		err := r.Delete(ctx, &j)
		if err != nil {
			logger.Error(err, "failed to delete finished job", "Job", j)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *TupleGenerationSchedulerReconciler) getMatchingJobs(ctx context.Context, pred func(klyshkov1alpha1.TupleGenerationJob) bool) ([]klyshkov1alpha1.TupleGenerationJob, error) {
	allJobs := &klyshkov1alpha1.TupleGenerationJobList{}
	err := r.List(ctx, allJobs)
	if err != nil {
		return nil, err
	}
	var matchingJobs []klyshkov1alpha1.TupleGenerationJob
	for _, j := range allJobs.Items {
		if pred(j) {
			matchingJobs = append(matchingJobs, j)
		}
	}
	return matchingJobs, nil
}

func (r *TupleGenerationSchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&klyshkov1alpha1.TupleGenerationScheduler{}).
		Owns(&klyshkov1alpha1.TupleGenerationJob{}).
		Complete(r)
}
