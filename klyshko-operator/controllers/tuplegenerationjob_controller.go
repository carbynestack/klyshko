/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go/v4"
	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
	"github.com/carbynestack/klyshko/castor"
	"github.com/carbynestack/klyshko/logging"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

const (
	// headsKey is the key prefix used to store the player head revisions in etcd.
	headsKey = "/klyshko/heads"

	// headRevisionOpRetryPeriod defines the duration between two attempts to store or fetch the head revision in etcd.
	headRevisionOpRetryPeriod = 5 * time.Second
)

// TupleGenerationJobReconciler reconciles a TupleGenerationJob object.
type TupleGenerationJobReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	EtcdClient   *clientv3.Client
	CastorClient *castor.Client
	Logger       logr.Logger
}

// NewTupleGenerationJobReconciler creates a TupleGenerationJobReconciler.
func NewTupleGenerationJobReconciler(client client.Client, scheme *runtime.Scheme, etcdClient *clientv3.Client, castorClient *castor.Client, logger logr.Logger) *TupleGenerationJobReconciler {
	r := &TupleGenerationJobReconciler{
		Client:       client,
		Scheme:       scheme,
		EtcdClient:   etcdClient,
		CastorClient: castorClient,
		Logger:       logger,
	}
	go r.handleWatchEvents()
	return r
}

//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile compares the actual state of TupleGenerationJob resources to their desired state and performs actions to
// bring the actual state closer to the desired one.
func (r *TupleGenerationJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	jobKey := RosterKey{
		req.NamespacedName,
	}
	logger := r.Logger.WithValues("Job.Key", jobKey)
	logger.V(logging.DEBUG).Info("Reconciling tuple generation jobs")

	// Get local player index to be used throughout the reconciliation loop
	playerID, err := localPlayerID(ctx, &r.Client, req.Namespace)
	if err != nil {
		return ctrl.Result{RequeueAfter: 60 * time.Second},
			fmt.Errorf("can't read playerId from VCP configuration for job %v: %w", req.Name, err)
	}

	// Cleanup if job has been deleted
	job := &klyshkov1alpha1.TupleGenerationJob{}
	err = r.Get(ctx, req.NamespacedName, job)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Job resource not available -> has been deleted, delete etcd entry, iff we are the master
			if playerID == 0 {
				_, err := r.EtcdClient.Delete(ctx, jobKey.ToEtcdKey())
				if err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to delete roster for job %v: %w", req.Name, err)
				}
				logger.V(logging.DEBUG).Info("Roster deleted")
			}
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, fmt.Errorf("failed to read resource for job %v: %w", req.Name, err)
	}
	logger.V(logging.DEBUG).Info("Job exists already")

	// Create roster if not existing (no etcd transaction needed as remote job creation is triggered by roster creation)
	resp, err := r.EtcdClient.Get(ctx, jobKey.ToEtcdKey())
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to read resource for roster with key %v for task %v: %w", jobKey, req.Name, err)
	}
	if resp.Count == 0 {
		if playerID != 0 {
			logger.V(logging.DEBUG).Info("Roster not available, retrying later")
			return ctrl.Result{}, nil
		}
		encoded, err := json.Marshal(job.Spec)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to marshal specification for job %v: %w", req.Name, err)
		}
		_, err = r.EtcdClient.Put(ctx, jobKey.ToEtcdKey(), string(encoded))
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create roster for job %v: %w", req.Name, err)
		}
		logger.V(logging.DEBUG).Info("Roster created")
	} else {
		logger.V(logging.DEBUG).Info("Roster exists already")
	}

	// Create local task if not existing
	task := &klyshkov1alpha1.TupleGenerationTask{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: job.Namespace,
		Name:      r.taskNameForJob(job, playerID),
	}, task)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new local task for job
			task, err = r.taskForJob(job, playerID)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to define local task for job %v: %w", req.Name, err)
			}
			err = r.Create(ctx, task)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to create local task for job %v: %w", req.Name, err)
			}
			logger.V(logging.DEBUG).Info("Local task created", "Task.Name", task.Name)
			return ctrl.Result{Requeue: true}, nil
		}
		// Error reading resource, requeue
		return ctrl.Result{}, fmt.Errorf("failed to read task resource for job %v: %w", req.Name, err)
	}
	logger.V(logging.DEBUG).Info("Local task exists already", "Task.Name", task.Name)

	// Update job status based on owned task statuses; TODO That might not scale well in case we have many jobs
	tasks := &klyshkov1alpha1.TupleGenerationTaskList{}
	err = r.List(ctx, tasks)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to fetch task list: %w", err)
	}

	// Collecting tasks owned by job; TODO Consider moving this to task class
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
	logger.V(logging.DEBUG).Info("Collected statuses of owned tasks", "Tasks", ownedBy)

	// Helper functions; TODO Consider moving this to state class
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
	numberOfVCPs, err := numberOfVCPs(ctx, &r.Client, req.Namespace)
	if err != nil {
		return ctrl.Result{RequeueAfter: 60 * time.Second}, fmt.Errorf("can't read playerCount from VCP configuration: %w", err)
	}
	var state klyshkov1alpha1.TupleGenerationJobState
	if uint(len(ownedBy)) < numberOfVCPs {
		state = klyshkov1alpha1.JobPending
	} else if !allTerminated(ownedBy) {
		state = klyshkov1alpha1.JobRunning
	} else if anyFailed(ownedBy) {
		state = klyshkov1alpha1.JobFailed
	} else if job.Status.State != klyshkov1alpha1.JobCompleted {
		state = klyshkov1alpha1.JobCompleted

		// Activate tuples; TODO How to deal with failures here? Introduce JobActivating state, how to sync between VCPs?
		tupleChunkID, err := uuid.Parse(job.Spec.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("invalid job id encountered '%v': %w", job.Spec.ID, err)
		}
		err = r.CastorClient.ActivateTupleChunk(ctx, tupleChunkID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("tuple chunk activation failed for job %v: %w", job.Name, err)
		}
		logger.Info("Job done", "Job", job)
	}
	if state.IsValid() && state != job.Status.State {
		logger.V(logging.DEBUG).Info("State update", "from", job.Status.State, "to", state)
		job.Status.State = state
		job.Status.LastStateTransitionTime = metav1.Now()
		err = r.Status().Update(ctx, job)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("status update failed for job %v: %w", job.Name, err)
		}
	}

	logger.V(logging.DEBUG).Info("Desired state reached")
	return ctrl.Result{}, nil
}

// taskNameForJob returns the name of the task for a given player associated with the given job.
func (r *TupleGenerationJobReconciler) taskNameForJob(job *klyshkov1alpha1.TupleGenerationJob, playerID uint) string {
	return r.taskName(job.Name, playerID)
}

// taskName returns the name of the task for a given player derived from a given job name.
func (r *TupleGenerationJobReconciler) taskName(jobName string, playerID uint) string {
	return jobName + "-" + strconv.Itoa(int(playerID))
}

// taskForJob assembles the TupleGenerationJob resource description for the given job and VCP.
func (r *TupleGenerationJobReconciler) taskForJob(job *klyshkov1alpha1.TupleGenerationJob, playerID uint) (*klyshkov1alpha1.TupleGenerationTask, error) {
	task := &klyshkov1alpha1.TupleGenerationTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.taskNameForJob(job, playerID),
			Namespace: job.Namespace,
		},
		Spec:   klyshkov1alpha1.TupleGenerationTaskSpec{},
		Status: klyshkov1alpha1.TupleGenerationTaskStatus{},
	}
	err := ctrl.SetControllerReference(job, task, r.Scheme)
	if err != nil {
		return nil, fmt.Errorf("setting owner reference failed: %w", err)
	}
	return task, nil
}

// getHeadRevisionKey returns the etcd key of the head revision of the local player.
func (r *TupleGenerationJobReconciler) getHeadRevisionKey(ctx context.Context, namespace string) (string, error) {
	playerID, err := localPlayerID(ctx, &r.Client, namespace)
	if err != nil {
		return "", fmt.Errorf("can't read local player ID: %w", err)
	}
	return fmt.Sprintf("%s/%d", headsKey, playerID), nil
}

// getHeadRevision fetches the head revisions, i.e., the revision of the last processed watch event,
// for the local player from etcd.
func (r *TupleGenerationJobReconciler) getHeadRevision(ctx context.Context, namespace string) (int64, error) {
	key, err := r.getHeadRevisionKey(ctx, namespace)
	if err != nil {
		return 0, err
	}
	var rev int64
	resp, err := r.EtcdClient.Get(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("can't read current revision head: %w", err)
	}
	if resp.Count > 0 {
		rev, _ = binary.Varint(resp.Kvs[0].Value)
	}
	r.Logger.V(logging.DEBUG).Info("Fetched current head revision", "revision", rev, "key", key)
	return rev, nil
}

// setHeadRevision stores the given revision as the head revision for the local player in etcd.
func (r *TupleGenerationJobReconciler) setHeadRevision(ctx context.Context, namespace string, revision int64) error {
	key, err := r.getHeadRevisionKey(ctx, namespace)
	if err != nil {
		return err
	}
	buf := make([]byte, 8)
	bytesWritten := binary.PutVarint(buf, revision)
	_, err = r.EtcdClient.Put(ctx, key, string(buf[:bytesWritten]))
	if err != nil {
		return fmt.Errorf("can't write revision head: %w", err)
	}
	r.Logger.V(logging.DEBUG).Info("Set current head revision", "revision", revision, "key", key)
	return nil
}

// handleWatchEvents handles incoming etcd events and dispatches them individually to handleWatchEvent.
func (r *TupleGenerationJobReconciler) handleWatchEvents() {

	for {
		ctx, cancel := context.WithCancel(context.Background())
		retrySleep := func(err error) {
			r.Logger.Error(err,
				"Failed to fetch / store head revision - sleeping before next attempt",
				"Duration", headRevisionOpRetryPeriod)
			time.Sleep(headRevisionOpRetryPeriod)
			cancel()
		}

		// Read this players head revision and start watching for subsequent events. This will replay
		// historical / missed events, in case the current etcd revision is higher than the head revision.
		revision, err := r.getHeadRevision(ctx, "default")
		if err != nil {
			retrySleep(err)
			continue
		}
		watchRev := revision + 1
		rosterWatcherCh := r.EtcdClient.Watch(ctx, rosterKey, clientv3.WithPrefix(), clientv3.WithRev(watchRev))
		r.Logger.V(logging.DEBUG).Info("Watch registered", "revision", watchRev, "key", rosterKey)

		// Process events
		for watchResponse := range rosterWatcherCh {
			if watchResponse.Err() != nil {
				r.Logger.Error(watchResponse.Err(), "watch failed - reestablishing")
				cancel()
				break
			}
			for _, ev := range watchResponse.Events {
				r.handleWatchEvent(ctx, ev)
			}
			err := r.setHeadRevision(ctx, "default", watchResponse.Header.Revision)
			if err != nil {
				retrySleep(err)
				break
			}
		}
	}
}

// handleWatchEvent inspects the given event and dispatches to handleRemoteTaskUpdate or handleJobUpdate based on the
// type of contained key.
func (r *TupleGenerationJobReconciler) handleWatchEvent(ctx context.Context, ev *clientv3.Event) {
	key, err := ParseKey(string(ev.Kv.Key))
	if err != nil {
		r.Logger.Error(err, "Unexpected etcd watch event", "Event.Key", ev.Kv.Key)
		return
	}
	logger := r.Logger.WithValues("Key", key, "Value", string(ev.Kv.Value), "Type", ev.Type)
	logger.V(logging.DEBUG).Info("Processing roster event")

	switch k := key.(type) {
	case RosterEntryKey:
		// Skip if update is for local task
		local, err := isLocalTaskKey(ctx, &r.Client, k)
		if err != nil {
			logger.Error(err, "Failed to check task type")
			return
		}
		if local {
			return
		}
		r.handleRemoteTaskUpdate(ctx, k, ev)
	case RosterKey:
		r.handleJobUpdate(ctx, k, ev)
	default:
		panic(fmt.Sprintf("Unexpected key type encountered: %v", key))
	}
}

func (r *TupleGenerationJobReconciler) handleJobUpdate(ctx context.Context, key RosterKey, ev *clientv3.Event) {
	logger := r.Logger.WithValues("Key", key)
	switch ev.Type {
	case mvccpb.PUT:
		// Get job spec from etcd K/V pair
		jobSpec := &klyshkov1alpha1.TupleGenerationJobSpec{}
		err := json.Unmarshal(ev.Kv.Value, jobSpec)
		if err != nil {
			logger.Error(err, "Failed to unmarshal spec")
			return
		}
		// TODO Create or update depending on whether Job already exists
		err = r.createJobIfNotExists(ctx, key.NamespacedName, jobSpec)
		if err != nil {
			logger.Error(err, "Failed to create job")
			return
		}
		logger.V(logging.DEBUG).Info("Job created")
	case mvccpb.DELETE:
		// Delete job iff exists
		found := &klyshkov1alpha1.TupleGenerationJob{}
		err := r.Client.Get(ctx, key.NamespacedName, found)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return
			}
			logger.Error(err, "Failed to read job resource")
			return
		}
		err = r.Delete(ctx, found)
		if err != nil {
			logger.Error(err, "Job deletion failed")
			return
		}
		logger.V(logging.DEBUG).Info("Job deleted")
	default:
		panic(fmt.Sprintf("Unexpected etcd event encounter: %v", ev))
	}
}

// handleRemoteTaskUpdate is responsible for creating, updating, and deleting local tasks and proxies for remote tasks.
func (r *TupleGenerationJobReconciler) handleRemoteTaskUpdate(ctx context.Context, key RosterEntryKey, ev *clientv3.Event) {
	logger := r.Logger.WithValues("Task.Key", key)

	// TODO Failure in one of the below handlers requires reconciliation, how to do that?
	switch ev.Type {
	case mvccpb.PUT:

		// Lookup job (requires retry as job might take small period of time to be available from API server)
		job := &klyshkov1alpha1.TupleGenerationJob{}
		err := retry.Do(func() error {
			return r.Get(ctx, key.NamespacedName, job)
		})
		if err != nil {
			logger.Error(err, "Failed to read job resource")
			return
		}
		taskName := types.NamespacedName{
			Namespace: key.Namespace,
			Name:      r.taskNameForJob(job, key.PlayerID),
		}

		found := &klyshkov1alpha1.TupleGenerationTask{}
		if err := r.Client.Get(ctx, taskName, found); err == nil {
			// Update local proxy task status
			status, err := klyshkov1alpha1.Unmarshal(ev.Kv.Value)
			if err != nil {
				logger.Error(err, "Extracting state from roster entry failed", "Value", string(ev.Kv.Value))
				return
			}
			found.Status = *status
			err = r.Client.Status().Update(ctx, found)
			if err != nil {
				logger.Error(err, "Failed to update proxy task")
				return
			}
			logger.V(logging.DEBUG).Info("Updated state", "State.New", status)
		} else {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "Failed to fetch task")
				return
			}

			// Create local proxy for remote task
			task, err := r.taskForJob(job, key.PlayerID)
			if err != nil {
				logger.Error(err, "Failed to define proxy task")
				return
			}
			err = r.Create(ctx, task)
			if err != nil {
				if apierrors.IsAlreadyExists(err) {
					// Resource was not yet created when fetching at the beginning of the method, re-enqueue
					r.handleRemoteTaskUpdate(ctx, key, ev)
				} else {
					logger.Error(err, "Failed to create proxy task")
				}
				return
			}
			logger.V(logging.DEBUG).Info("Proxy task created")
		}
	case mvccpb.DELETE:
		// Delete task for job if exists
		taskName := types.NamespacedName{
			Namespace: key.Namespace,
			Name:      r.taskName(key.Name, key.PlayerID),
		}
		task := &klyshkov1alpha1.TupleGenerationTask{}
		err := r.Get(ctx, taskName, task)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return
			}
			logger.Error(err, "Failed to read proxy task resource")
			return
		}
		err = r.Delete(ctx, task)
		if err != nil {
			logger.Error(err, "Proxy task deletion failed")
			return
		}
		logger.V(logging.DEBUG).Info("Proxy task deleted")
	}
}

// createJobIfNotExists creates a job according to the given TupleGenerationJobSpec.
func (r *TupleGenerationJobReconciler) createJobIfNotExists(ctx context.Context, name types.NamespacedName, jobSpec *klyshkov1alpha1.TupleGenerationJobSpec) error {
	logger := r.Logger.WithValues("Job.Name", name)
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
			logger.V(logging.DEBUG).Info("Creating a new job")
			return r.Create(ctx, job)
		}
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TupleGenerationJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&klyshkov1alpha1.TupleGenerationJob{}).
		Owns(&klyshkov1alpha1.TupleGenerationTask{}).
		Complete(r)
}
