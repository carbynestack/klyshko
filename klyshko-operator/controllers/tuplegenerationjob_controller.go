/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
	"github.com/google/uuid"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"time"
)

type TupleGenerationJobReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	EtcdClient      *clientv3.Client
	rosterWatcherCh clientv3.WatchChan
}

func NewTupleGenerationJobReconciler(client client.Client, scheme *runtime.Scheme, etcdClient *clientv3.Client) *TupleGenerationJobReconciler {
	r := new(TupleGenerationJobReconciler)
	r.Client = client
	r.Scheme = scheme
	r.EtcdClient = etcdClient
	r.rosterWatcherCh = etcdClient.Watch(context.Background(), rosterKey, clientv3.WithPrefix()) // TODO Close channel?
	go r.handleWatchEvents()
	return r
}

func (r *TupleGenerationJobReconciler) handleWatchEvents() {
	ctx := context.Background()
	for watchResponse := range r.rosterWatcherCh {
		for _, ev := range watchResponse.Events {
			r.handleWatchEvent(ctx, ev)
		}
	}
}

func (r *TupleGenerationJobReconciler) handleWatchEvent(ctx context.Context, ev *clientv3.Event) {
	logger := log.FromContext(ctx)
	key, err := ParseKey(string(ev.Kv.Key))
	if err != nil {
		logger.Error(err, "unexpected etcd watch event", "Event.Key", ev.Kv.Key)
	}
	logger.Info("processing roster event", "Key", key)

	switch k := key.(type) {
	case RosterEntryKey:
		// Skip if update is for local task
		local, err := isLocalTaskKey(ctx, &r.Client, k)
		if err != nil {
			logger.Error(err, "failed to check task type")
			return
		}
		if local {
			return
		}
		r.handleRemoteTaskUpdate(ctx, k, ev)
	case RosterKey:
		r.handleJobUpdate(ctx, k, ev)
	}
}

func (r *TupleGenerationJobReconciler) handleJobUpdate(ctx context.Context, key RosterKey, ev *clientv3.Event) {
	logger := log.FromContext(ctx).WithValues("Key", key)
	switch ev.Type {
	case mvccpb.PUT:
		// Get job spec from etcd K/V pair
		jobSpec := &klyshkov1alpha1.TupleGenerationJobSpec{}
		err := json.Unmarshal(ev.Kv.Value, jobSpec)
		if err != nil {
			logger.Error(err, "failed to unmarshal spec")
			return
		}
		// TODO Create or update depending on whether Job already exists
		err = r.createJob(ctx, key.NamespacedName, jobSpec)
		if err != nil {
			logger.Error(err, "failed to create job")
			return
		}
		logger.Info("job created")
	case mvccpb.DELETE:
		// Delete job iff exists
		found := &klyshkov1alpha1.TupleGenerationJob{}
		err := r.Client.Get(ctx, key.NamespacedName, found)
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Error(err, "job does not exist - ignoring")
				return
			}
			logger.Error(err, "failed to read job resource")
			return
		}
		err = r.Delete(ctx, found)
		if err != nil {
			logger.Error(err, "job deletion failed")
			return
		}
		logger.Info("job deleted")
	}
}

func (r *TupleGenerationJobReconciler) handleRemoteTaskUpdate(ctx context.Context, key RosterEntryKey, ev *clientv3.Event) {
	logger := log.FromContext(ctx).WithValues("Task.Key", key)

	// Lookup job
	job := &klyshkov1alpha1.TupleGenerationJob{}
	err := r.Get(ctx, key.NamespacedName, job)
	if err != nil {
		logger.Error(err, "failed to read job resource")
		return
	}

	taskName := types.NamespacedName{
		Namespace: key.Namespace,
		Name:      r.taskName(job, key.PlayerID),
	}

	switch ev.Type {
	case mvccpb.PUT:
		found := &klyshkov1alpha1.TupleGenerationTask{}
		if err := r.Client.Get(ctx, taskName, found); err == nil {
			// Update local proxy task status
			status, err := klyshkov1alpha1.ParseFromJSON(ev.Kv.Value)
			if err != nil {
				logger.Error(err, "extracting state from roster entry failed")
				return
			}
			found.Status = *status
			err = r.Client.Status().Update(ctx, found)
			if err != nil {
				logger.Error(err, "failed to update proxy task")
				return
			}
		} else {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "failed to fetch task") // TODO Failure requires reconciliation (also above/below), how to do that?
				return
			}

			// Create local proxy for remote task
			task, err := r.taskForJob(job, key.PlayerID)
			if err != nil {
				logger.Error(err, "failed to define proxy task")
				return
			}
			err = r.Create(ctx, task)
			if err != nil {
				logger.Error(err, "failed to create proxy task")
				return
			}
		}
	case mvccpb.DELETE:
		// Delete task for job iff exists
		task := &klyshkov1alpha1.TupleGenerationTask{}
		err = r.Get(ctx, taskName, task)
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Error(err, "proxy task does not exist - ignoring")
				return
			}
			logger.Error(err, "failed to read proxy task resource")
			return
		}
		err = r.Delete(ctx, task)
		if err != nil {
			logger.Error(err, "proxy task deletion failed")
			return
		}
		logger.Info("proxy task deleted")
	}
}

func (r *TupleGenerationJobReconciler) createJob(ctx context.Context, name types.NamespacedName, jobSpec *klyshkov1alpha1.TupleGenerationJobSpec) error {
	logger := log.FromContext(ctx).WithValues("Job.Name", name)
	found := &klyshkov1alpha1.TupleGenerationJob{}
	err := r.Client.Get(ctx, name, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			job := &klyshkov1alpha1.TupleGenerationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name.Name,
					Namespace: name.Namespace,
				},
				Spec: *jobSpec,
			}
			logger.Info("creating a new job")
			return r.Create(ctx, job)
		}
		return err
	}
	return nil
}

//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *TupleGenerationJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	jobKey := RosterKey{
		req.NamespacedName,
	}
	logger := log.FromContext(ctx).WithValues("Job.Key", jobKey)

	// Cleanup if job has been deleted
	job := &klyshkov1alpha1.TupleGenerationJob{}
	err := r.Get(ctx, req.NamespacedName, job)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Job resource not available -> has been deleted
			_, err := r.EtcdClient.Delete(ctx, jobKey.ToEtcdKey())
			if err != nil {
				logger.Error(err, "failed to delete roster")
				return ctrl.Result{}, err
			}
			logger.Info("roster deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "failed to read job resource")
		return ctrl.Result{}, err
	} else {
		logger.Info("job exists already")
	}

	// Create roster if not existing (no etcd transaction needed as remote job creation is triggered by roster creation)
	resp, err := r.EtcdClient.Get(ctx, jobKey.ToEtcdKey())
	if err != nil {
		logger.Error(err, "failed to fetch roster", "Roster.Key", jobKey)
		return ctrl.Result{}, err
	}
	if resp.Count == 0 {
		encoded, err := json.Marshal(job.Spec)
		if err != nil {
			fmt.Println(err, "failed to marshal specification")
			return ctrl.Result{}, err
		}
		_, err = r.EtcdClient.Put(ctx, jobKey.ToEtcdKey(), string(encoded))
		if err != nil {
			logger.Error(err, "failed to create roster")
			return ctrl.Result{}, err
		}
		logger.Info("roster created")
	} else {
		logger.Info("roster exists already")
	}

	// Create local task if not existing
	playerID, err := localPlayerID(ctx, &r.Client, req.Namespace)
	if err != nil {
		logger.Error(err, "can't read playerId from VCP configuration")
		return ctrl.Result{RequeueAfter: 60 * time.Second}, err
	}
	task := &klyshkov1alpha1.TupleGenerationTask{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: job.Namespace,
		Name:      r.taskName(job, playerID),
	}, task)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new local task for job
			task, err = r.taskForJob(job, playerID)
			if err != nil {
				logger.Error(err, "failed to define local task", "Task.Name", task.Name)
				return ctrl.Result{}, err
			}
			err = r.Create(ctx, task)
			if err != nil {
				logger.Error(err, "failed to create local task", "Task.Name", task.Name)
				return ctrl.Result{}, err
			}
			logger.Info("local task created", "Task.Name", task.Name)
			return ctrl.Result{Requeue: true}, nil
		}
		// Error reading resource, requeue
		logger.Error(err, "failed to read task resource")
		return ctrl.Result{}, err
	} else {
		logger.Info("local task exists already")
	}

	// Update job status based on owned task statuses
	tasks := &klyshkov1alpha1.TupleGenerationTaskList{}
	err = r.List(ctx, tasks) // TODO Use ListOption filter based on owner reference, if possible
	if err != nil {
		return ctrl.Result{}, err
	}

	// Collecting tasks owned by job; TODO Move this to task class
	isOwnedBy := func(task klyshkov1alpha1.TupleGenerationTask) bool {
		for _, or := range task.OwnerReferences {
			if or.Name == job.Name {
				return true
			}
		}
		return false
	}
	var ownedBy []klyshkov1alpha1.TupleGenerationTask
	for _, t := range tasks.Items {
		if isOwnedBy(t) {
			ownedBy = append(ownedBy, t)
		}
	}
	logger.Info("Collected statues of owned tasks", "States", ownedBy)

	// Helper functions; TODO Move this to state class?
	allTerminated := func(tasks []klyshkov1alpha1.TupleGenerationTask) bool {
		for _, t := range tasks {
			if t.Status.State != klyshkov1alpha1.TaskCompleted && t.Status.State != klyshkov1alpha1.TaskFailed {
				return false
			}
		}
		return true
	}
	anyFailed := func(tasks []klyshkov1alpha1.TupleGenerationTask) bool {
		for _, t := range tasks {
			if t.Status.State == klyshkov1alpha1.TaskFailed {
				return true
			}
		}
		return false
	}
	var state klyshkov1alpha1.TupleGenerationJobState
	if len(ownedBy) < 2 {
		state = klyshkov1alpha1.JobPending
	} else if !allTerminated(ownedBy) {
		state = klyshkov1alpha1.JobRunning
	} else if anyFailed(ownedBy) {
		state = klyshkov1alpha1.JobFailed
	} else {
		state = klyshkov1alpha1.JobCompleted

		// Activate tuples; TODO How to deal with failures here? Introduce JobActivating state, how to sync between VCPs? Is activation still used in new Castor implementation?
		tupleChunkId, err := uuid.Parse(job.Spec.ID)
		if err != nil {
			logger.Error(err, "invalid job id encountered")
			return ctrl.Result{}, nil
		}
		err = activateTupleChunk(ctx, tupleChunkId)
		if err != nil {
			logger.Error(err, "tuple activation failed")
			return ctrl.Result{}, nil
		}
	}
	job.Status.State = state
	err = r.Status().Update(ctx, job)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("desired state reached")
	return ctrl.Result{}, nil
}

func (r *TupleGenerationJobReconciler) taskName(job *klyshkov1alpha1.TupleGenerationJob, playerID uint) string {
	return job.Name + "-" + strconv.Itoa(int(playerID))
}

func (r *TupleGenerationJobReconciler) taskForJob(job *klyshkov1alpha1.TupleGenerationJob, playerID uint) (*klyshkov1alpha1.TupleGenerationTask, error) {
	task := &klyshkov1alpha1.TupleGenerationTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.taskName(job, playerID),
			Namespace: job.Namespace,
		},
		Spec:   klyshkov1alpha1.TupleGenerationTaskSpec{},
		Status: klyshkov1alpha1.TupleGenerationTaskStatus{},
	}
	err := ctrl.SetControllerReference(job, task, r.Scheme)
	return task, err
}

func (r *TupleGenerationJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&klyshkov1alpha1.TupleGenerationJob{}).
		Owns(&klyshkov1alpha1.TupleGenerationTask{}).
		Complete(r)
}
