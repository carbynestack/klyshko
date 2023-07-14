/*
Copyright (c) 2022-2023 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/carbynestack/klyshko/logging"
	clientv3 "go.etcd.io/etcd/client/v3"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
)

const (
	// TaskLabel is used to identify target pods for the inter-CRG service
	TaskLabel = "klyshko.carbnyestack.io/task-ref"

	// InterCRGNetworkingPort is the network used for inter-CRG communication.
	InterCRGNetworkingPort = 5000
)

// TupleGenerationTaskReconciler reconciles a TupleGenerationTask object.
type TupleGenerationTaskReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	EtcdClient       *clientv3.Client
	ProvisionerImage string
}

//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationtasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationtasks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationtasks/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile compares the actual state of TupleGenerationTask resources to their desired state and performs actions to
// bring the actual state closer to the desired one.
func (r *TupleGenerationTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("Task.Name", req.Name)
	logger.V(logging.DEBUG).Info("Reconciling tuple generation task")

	taskKey, err := r.taskKeyFromName(req.Namespace, req.Name)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get key for task %v: %w", req.Name, err)
	}

	// Reconcile local task as endpoint might have been assigned for remote task, and we might be able to transition
	// out of TaskPreparing state in case all endpoints are available for the job
	local, err := isLocalTaskKey(ctx, &r.Client, *taskKey)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check type for task %v: %w", req.Name, err)
	}
	if !local {
		localTaskName, err := getLocalTaskName(ctx, &r.Client, req.Name, req.Namespace)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get name of local task for remote task %v: %w", req.Name, err)
		}
		return r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: req.Namespace,
			Name:      localTaskName,
		}})
	}

	// Cleanup if task has been deleted
	task := &klyshkov1alpha1.TupleGenerationTask{}
	err = r.Get(ctx, req.NamespacedName, task)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Task resource not available -> has been deleted
			_, err := r.EtcdClient.Delete(ctx, taskKey.ToEtcdKey())
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to delete roster entry for task %v: %w", req.Name, err)
			}
			logger.V(logging.DEBUG).Info("Roster entry deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, fmt.Errorf("failed to read resource for task %v: %w", req.Name, err)
	}
	logger.V(logging.DEBUG).Info("Task exists already")

	// Create roster entry if not existing
	resp, err := r.EtcdClient.Get(ctx, taskKey.ToEtcdKey())
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to read resource for roster entry with key %v for task %v: %w", taskKey, req.Name, err)
	}
	if resp.Count == 0 {
		status, err := json.Marshal(&klyshkov1alpha1.TupleGenerationTaskStatus{State: klyshkov1alpha1.TaskPreparing})
		_, err = r.EtcdClient.Put(ctx, taskKey.ToEtcdKey(), string(status))
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create roster entry for task %v: %w", req.Name, err)
		}
		logger.V(logging.DEBUG).Info("Roster entry created")
	} else {
		logger.V(logging.DEBUG).Info("Roster entry exists already")
	}
	status, err := r.getStatus(ctx, *taskKey)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("status not available for task %v: %w", req.Name, err)
	}

	// Lookup job that owns the task
	job := &klyshkov1alpha1.TupleGenerationJob{}
	err = r.Get(ctx, taskKey.NamespacedName, job)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to lookup job for task %v: %w", req.Name, err)
	}

	// Update the task status according to state in etcd
	taskStatus, err := r.getStatus(ctx, *taskKey)
	if err != nil {
		return ctrl.Result{}, err
	}
	task.Status = *taskStatus
	if err := r.Status().Update(ctx, task); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to update status for task %v: %w", req.Name, err)
	}

	// Proceed based on current task state. State changes are performed by first invoking setState which updates
	// the state in etcd and then re-enqueueing in order to reflect the updated state in the local task representation.
	switch status.State {
	case klyshkov1alpha1.TaskPreparing:

		// Create persistent volume claim used to store generated tuples, if not existing
		err = r.getOrCreatePVC(ctx, taskKey)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to create PVC for task %v: %w", req.Name, err)
		}

		// Create the service used for inter-CRG networking, if not existing
		svc, err := r.getOrCreateService(ctx, taskKey, task)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to get or create service for task %v: %w", req.Name, err)
		}

		// Collect all known endpoints for all CRGs of the job (local and remote) and decide based
		// on that how to proceed.
		tasksByPlayerID, err := r.tasksForJob(ctx, job)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get task set for task %v: %w", req.Name, err)
		}
		endpoints := r.endpoints(tasksByPlayerID)
		numberOfVCPs, err := numberOfVCPs(ctx, &r.Client, job.Namespace)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get number of VCPs: %w", err)
		}
		switch {
		case len(endpoints) == int(numberOfVCPs): // All endpoints known
			return ctrl.Result{
				Requeue: true,
			}, r.setState(ctx, *taskKey, status, klyshkov1alpha1.TaskLaunching)
		case func() bool { _, ok := endpoints[taskKey.PlayerID]; return !ok }(): // Local endpoint not yet available
			endpoint, err := endpoint(svc)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("can't get endpoint for service %v:%v", svc.Namespace, svc.Name)
			}
			if endpoint == nil {
				logger.V(logging.DEBUG).Info("No endpoint available yet for local task")
				return ctrl.Result{}, nil
			}
			status.Endpoint = fmt.Sprintf("%s:%d", *endpoint, InterCRGNetworkingPort)
			err = r.setStatus(ctx, *taskKey, status)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{
				Requeue: true,
			}, nil
		default: // At least one remote endpoint not available
			return ctrl.Result{}, nil
		}
	case klyshkov1alpha1.TaskLaunching:
		// Create generator pod if not existing
		_, err := r.createGeneratorPod(ctx, *taskKey, job, task)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to create generator pod for task %v: %w", req.Name, err)
		}
		return ctrl.Result{
			Requeue: true,
		}, r.setState(ctx, *taskKey, status, klyshkov1alpha1.TaskGenerating)
	case klyshkov1alpha1.TaskGenerating:
		genPod, err := r.getGeneratorPod(ctx, task)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to get generator pod for task %v: %w", req.Name, err)
		}
		switch genPod.Status.Phase {
		case v1.PodSucceeded:
			// Generation successful, create provisioner pod to upload tuple shares to VCP-local castor
			_, err := r.createProvisionerPod(ctx, *taskKey, job, task)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to create provisioner pod for task %v: %w", req.Name, err)
			}
			return ctrl.Result{
				Requeue: true,
			}, r.setState(ctx, *taskKey, status, klyshkov1alpha1.TaskProvisioning)
		case v1.PodFailed:
			return ctrl.Result{
				Requeue: true,
			}, r.setState(ctx, *taskKey, status, klyshkov1alpha1.TaskFailed)
		}
	case klyshkov1alpha1.TaskProvisioning:
		provPod, err := r.getProvisionerPod(ctx, *taskKey)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to get provisioner pod for task %v: %w", req.Name, err)
		}
		switch provPod.Status.Phase {
		case v1.PodSucceeded:
			return ctrl.Result{
				Requeue: true,
			}, r.setState(ctx, *taskKey, status, klyshkov1alpha1.TaskCompleted)
		case v1.PodFailed:
			return ctrl.Result{
				Requeue: true,
			}, r.setState(ctx, *taskKey, status, klyshkov1alpha1.TaskFailed)
		}
	}

	logger.V(logging.DEBUG).Info("Desired state reached")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TupleGenerationTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&klyshkov1alpha1.TupleGenerationTask{}).
		Owns(&v1.Pod{}).
		Owns(&v1.Service{}).
		Complete(r)
}

// taskKeyFromName creates a RosterEntryKey from the given name and namespace. Expects that the zero-based VCP
// identifier is appended with a hyphen to the name.
func (r *TupleGenerationTaskReconciler) taskKeyFromName(namespace string, name string) (*RosterEntryKey, error) {
	parts := strings.Split(name, "-")
	vcpID := parts[len(parts)-1]
	jobName := strings.Join(parts[:len(parts)-1], "-")
	keyStr := rosterKey + "/" + namespace + "/" + jobName + "/" + vcpID
	key, err := ParseKey(keyStr)
	if err != nil {
		return nil, fmt.Errorf("can't parse task key from '%s': %w", keyStr, err)
	}
	rev, ok := key.(RosterEntryKey)
	if !ok {
		return nil, fmt.Errorf("not a task key: %v", key)
	}
	return &rev, nil
}

// getLocalTaskName returns the name of the local task belonging to the same job as the given task.
func getLocalTaskName(ctx context.Context, client *client.Client, remoteTaskName string, namespace string) (string, error) {
	playerID, err := localPlayerID(ctx, client, namespace)
	if err != nil {
		return "", fmt.Errorf("can't read local player ID: %w", err)
	}
	return taskName(jobNameFromTaskName(remoteTaskName), playerID), nil
}

// isLocalTaskKey checks whether a given key is the key of task running in the local VCP.
func isLocalTaskKey(ctx context.Context, client *client.Client, key RosterEntryKey) (bool, error) {
	playerID, err := localPlayerID(ctx, client, key.Namespace)
	if err != nil {
		return false, fmt.Errorf("can't read local player ID: %w", err)
	}
	return playerID == key.PlayerID, nil
}

// getStatus reads the task status from the respective value stored in etcd.
func (r *TupleGenerationTaskReconciler) getStatus(ctx context.Context, taskKey RosterEntryKey) (*klyshkov1alpha1.TupleGenerationTaskStatus, error) {
	resp, err := r.EtcdClient.Get(ctx, taskKey.ToEtcdKey())
	if err != nil {
		return nil, fmt.Errorf("can't get status from etcd: %w", err)
	}
	if len(resp.Kvs) != 1 {
		return nil, fmt.Errorf("no status available for roster entry: %v", taskKey)
	}
	status, err := klyshkov1alpha1.Unmarshal(resp.Kvs[0].Value)
	if err != nil {
		return nil, fmt.Errorf("parsing status from '%s' failed: %w", string(resp.Kvs[0].Value), err)
	}
	if !status.State.IsValid() {
		return nil, fmt.Errorf("status contains invalid state: %s", status.State)
	}
	return status, nil
}

// setStatus writes the given status to etcd.
func (r *TupleGenerationTaskReconciler) setStatus(ctx context.Context, taskKey RosterEntryKey, status *klyshkov1alpha1.TupleGenerationTaskStatus) error {
	encoded, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshalling status failed: %w", err)
	}
	_, err = r.EtcdClient.Put(ctx, taskKey.ToEtcdKey(), string(encoded))
	if err != nil {
		return fmt.Errorf("storing marshalled status in etcd failed: %w", err)
	}
	return nil
}

// setState updates the given status object with the given state and writes the status to etcd.
func (r *TupleGenerationTaskReconciler) setState(ctx context.Context, taskKey RosterEntryKey, status *klyshkov1alpha1.TupleGenerationTaskStatus, state klyshkov1alpha1.TupleGenerationTaskState) error {
	logger := log.FromContext(ctx).WithValues("Task.Key", taskKey)
	logger.V(logging.DEBUG).Info("Task transitioning into new state", "from", status.State, "to", state)
	status.State = state
	return r.setStatus(ctx, taskKey, status)
}

// pvcName returns the name of the PVC used for the task with the given key.
func pvcName(key RosterEntryKey) string {
	return key.Name + "-" + strconv.Itoa(int(key.PlayerID))
}

// getOrCreatePVC creates a PVC used to transfer tuples between generator and provision pod for a task with the given key.
func (r *TupleGenerationTaskReconciler) getOrCreatePVC(ctx context.Context, key *RosterEntryKey) error {
	logger := log.FromContext(ctx).WithValues("Task.Key", key)
	name := types.NamespacedName{
		Name:      pvcName(*key),
		Namespace: key.Namespace,
	}
	found := &v1.PersistentVolumeClaim{}
	err := r.Get(ctx, name, found)
	if err == nil {
		logger.V(logging.DEBUG).Info("Persistent volume claim already exists")
		return nil
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					"storage": resource.MustParse("100Mi"), // TODO Can this be computed from the number of tuples?
				},
			},
		},
	}
	logger.V(logging.DEBUG).Info("Creating persistent volume claim", "PVC", pvc)
	err = r.Create(ctx, pvc)
	if err != nil {
		return fmt.Errorf("persistent volume claim creation failed for task %v: %w", key, err)
	}
	return nil
}

// provisionerPodName returns the name for the provisioner pod used for the task with the given key.
func (r *TupleGenerationTaskReconciler) provisionerPodName(key RosterEntryKey) string {
	return key.Name + "-provisioner"
}

// getProvisionerPod returns the provisioner pod for the task with given key.
func (r *TupleGenerationTaskReconciler) getProvisionerPod(ctx context.Context, key RosterEntryKey) (*v1.Pod, error) {
	name := types.NamespacedName{
		Name:      r.provisionerPodName(key),
		Namespace: key.Namespace,
	}
	found := &v1.Pod{}
	err := r.Get(ctx, name, found)
	if err != nil {
		return nil, fmt.Errorf("can't get the provisioner pod for task %v: %w", name, err)
	}
	return found, nil
}

// createProvisionerPod creates a provisioner pod for the task with given key. The pod takes the tuples from the PV
// shared with the respective generator pod and uploads them to Castor.
func (r *TupleGenerationTaskReconciler) createProvisionerPod(ctx context.Context, key RosterEntryKey, job *klyshkov1alpha1.TupleGenerationJob, task *klyshkov1alpha1.TupleGenerationTask) (*v1.Pod, error) {
	logger := log.FromContext(ctx).WithValues("Task.Key", key)
	name := types.NamespacedName{
		Name:      r.provisionerPodName(key),
		Namespace: key.Namespace,
	}
	found, err := r.getProvisionerPod(ctx, key)
	if err == nil {
		logger.V(logging.DEBUG).Info("Provisioner pod already exists")
		return found, nil
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:  "provisioner",
				Image: r.ProvisionerImage,
				Env: []v1.EnvVar{
					{
						Name:  "KII_JOB_ID",
						Value: job.Spec.ID,
					},
					{
						Name:  "KII_TUPLE_TYPE",
						Value: job.Spec.Type,
					},
					{
						Name:  "KII_TUPLE_FILE",
						Value: "/kii/tuples",
					},
				},
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      "kii",
						MountPath: "/kii",
					},
				},
			}},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{
				{
					Name: "kii",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName(key),
						},
					},
				},
			},
		},
	}
	logger.V(logging.DEBUG).Info("Creating provisioner pod", "Pod", pod)
	err = ctrl.SetControllerReference(task, pod, r.Scheme)
	if err != nil {
		return nil, fmt.Errorf("setting the owner reference for task %v failed: %w", name, err)
	}
	err = r.Create(ctx, pod)
	if err != nil {
		return nil, fmt.Errorf("pod creation for task %v failed: %w", name, err)
	}
	return pod, nil
}

// getGeneratorPod returns the generator pod for the task with given key.
func (r *TupleGenerationTaskReconciler) getGeneratorPod(ctx context.Context, task *klyshkov1alpha1.TupleGenerationTask) (*v1.Pod, error) {
	found := &v1.Pod{}
	name := types.NamespacedName{
		Name:      task.Name,
		Namespace: task.Namespace,
	}
	err := r.Get(ctx, name, found)
	if err != nil {
		return nil, fmt.Errorf("can't get the generator pod for task %v: %w", name, err)
	}
	return found, nil
}

// createGeneratorPod creates a generator pod for the task with given key. The pod generates tuples according to the
// parameter of the given TupleGenerationJob and stores them on the PV shared with the respective provisioner pod.
func (r *TupleGenerationTaskReconciler) createGeneratorPod(ctx context.Context, key RosterEntryKey, job *klyshkov1alpha1.TupleGenerationJob, task *klyshkov1alpha1.TupleGenerationTask) (*v1.Pod, error) {
	logger := log.FromContext(ctx).WithValues("Task.Key", key)
	found, err := r.getGeneratorPod(ctx, task)
	if err == nil {
		logger.V(logging.DEBUG).Info("Pod already exists")
		return found, nil
	}
	vcpCount, err := numberOfVCPs(ctx, &r.Client, job.Namespace)
	if err != nil {
		return nil, fmt.Errorf("can't get number of VCPs: %w", err)
	}

	// Prepare endpoint environment variables
	tasks, err := r.tasksForJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("failed to get task set: %w", err)
	}
	if int(vcpCount) != len(tasks) {
		return nil, fmt.Errorf("expected %d endpoints but got %d", vcpCount, len(tasks))
	}
	endpoints := r.endpoints(tasks)
	endpointEnvVars := make([]v1.EnvVar, 0, vcpCount)
	for playerID, endpoint := range endpoints {
		endpointEnvVars = append(endpointEnvVars,
			v1.EnvVar{
				Name:  fmt.Sprintf("KII_PLAYER_ENDPOINT_%d", playerID),
				Value: endpoint,
			},
		)
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      task.Name,
			Namespace: task.Namespace,
			Labels: map[string]string{
				TaskLabel: task.Name,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "generator",
					Image:           job.Spec.Generator.Image,
					ImagePullPolicy: job.Spec.Generator.ImagePullPolicy,
					Ports: []v1.ContainerPort{
						{
							ContainerPort: InterCRGNetworkingPort,
						},
					},
					Env: append(
						[]v1.EnvVar{
							{
								Name:  "KII_JOB_ID",
								Value: job.Spec.ID,
							},
							{
								Name:  "KII_PLAYER_COUNT",
								Value: strconv.Itoa(int(vcpCount)),
							},
							{
								Name:  "KII_TUPLE_TYPE",
								Value: job.Spec.Type,
							},
							{
								Name:  "KII_TUPLES_PER_JOB",
								Value: fmt.Sprint(job.Spec.Count),
							},
							{
								Name:  "KII_PLAYER_NUMBER",
								Value: fmt.Sprint(key.PlayerID),
							},
							{
								Name:  "KII_SHARED_FOLDER",
								Value: "/kii",
							},
							{
								Name:  "KII_TUPLE_FILE",
								Value: "/kii/tuples",
							},
						},
						endpointEnvVars...,
					),
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "kii",
							MountPath: "/kii",
						},
						{
							Name:      "params",
							ReadOnly:  true,
							MountPath: "/etc/kii/params",
						},
						{
							Name:      "secret-params",
							ReadOnly:  true,
							MountPath: "/etc/kii/secret-params",
						},
						{
							Name:      "extra-params",
							ReadOnly:  true,
							MountPath: "/etc/kii/extra-params",
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{
				{
					Name: "kii",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName(key),
						},
					},
				},
				{
					Name: "params",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "io.carbynestack.engine.params",
							},
						},
					},
				},
				{
					Name: "secret-params",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "io.carbynestack.engine.params.secret",
						},
					},
				},
				{
					Name: "extra-params",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "io.carbynestack.engine.params.extra",
							},
						},
					},
				},
			},
		},
	}
	logger.V(logging.DEBUG).Info("Creating generator pod", "Pod", pod)
	err = ctrl.SetControllerReference(task, pod, r.Scheme)
	if err != nil {
		return nil, fmt.Errorf("setting the owner reference for task %v failed: %w", task.Name, err)
	}
	err = r.Create(ctx, pod)
	if err != nil {
		return nil, fmt.Errorf("pod creation for task %v failed: %w", task.Name, err)
	}
	return pod, nil
}

func (r *TupleGenerationTaskReconciler) getOrCreateService(ctx context.Context, key *RosterEntryKey, task *klyshkov1alpha1.TupleGenerationTask) (*v1.Service, error) {
	logger := log.FromContext(ctx).WithValues("Task.Key", key)
	name := types.NamespacedName{
		Name:      task.Name,
		Namespace: task.Namespace,
	}
	found := &v1.Service{}
	err := r.Get(ctx, name, found)
	if err == nil {
		logger.V(logging.DEBUG).Info("Service already exists")
		return found, nil
	}
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port:       InterCRGNetworkingPort,
					TargetPort: intstr.FromInt(InterCRGNetworkingPort),
				},
			},
			Selector: map[string]string{
				TaskLabel: task.Name,
			},
			Type: v1.ServiceTypeLoadBalancer,
		},
	}
	logger.V(logging.DEBUG).Info("Creating service", "Service", service)
	err = ctrl.SetControllerReference(task, service, r.Scheme)
	if err != nil {
		return nil, fmt.Errorf("setting the owner reference for task %v failed: %w", task.Name, err)
	}
	err = r.Create(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("service creation failed for task %v: %w", key, err)
	}
	return service, nil
}

// tasksForJob returns the existing and retrievable tasks (remote and local) for the given job.
func (r *TupleGenerationTaskReconciler) tasksForJob(ctx context.Context, job *klyshkov1alpha1.TupleGenerationJob) (map[uint]*klyshkov1alpha1.TupleGenerationTask, error) {
	logger := log.FromContext(ctx).WithValues("Job", job.Name)
	numberOfVCPs, err := numberOfVCPs(ctx, &r.Client, job.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get number of VCPs: %w", err)
	}
	tasksByPlayerId := make(map[uint]*klyshkov1alpha1.TupleGenerationTask, numberOfVCPs)
	for i := uint(0); i < numberOfVCPs; i++ {
		task := &klyshkov1alpha1.TupleGenerationTask{}
		err := r.Client.Get(ctx, types.NamespacedName{
			Namespace: job.Namespace,
			Name:      taskName(job.Name, i),
		}, task)
		if err != nil {
			logger.V(logging.DEBUG).Info("failed to get task", "PlayerID", i, "Error", err)
			continue
		}
		tasksByPlayerId[i] = task
	}
	if logger.V(logging.DEBUG).Enabled() {
		var names []types.NamespacedName
		for _, task := range tasksByPlayerId {
			names = append(names, types.NamespacedName{
				Namespace: task.Namespace,
				Name:      task.Name,
			})
		}
		logger.V(logging.DEBUG).Info("Tasks collected", "Tasks", names)
	}
	return tasksByPlayerId, nil
}

// endpoints returns the non-empty endpoints of all existing and retrievable tasks (remote and local)
// indexed by player number for the given job.
func (r *TupleGenerationTaskReconciler) endpoints(tasks map[uint]*klyshkov1alpha1.TupleGenerationTask) map[uint]string {
	endpoints := make(map[uint]string)
	for playerID, t := range tasks {
		if t.Status.Endpoint != "" {
			endpoints[playerID] = t.Status.Endpoint
		}
	}
	return endpoints
}

// endpoint returns the first ingress point from the given service of type load balancer or nil, iff not available.
// Errors in case of non-LB service type or if neither an IP nor a hostname is available.
func endpoint(service *v1.Service) (*string, error) {
	if service.Spec.Type != v1.ServiceTypeLoadBalancer {
		return nil, fmt.Errorf("service of type load balancer expected, was %s", service.Spec.Type)
	}
	lb := service.Status.LoadBalancer
	if len(lb.Ingress) == 0 {
		return nil, nil
	}
	ingress := lb.Ingress[0]
	switch {
	case ingress.IP != "":
		return &ingress.IP, nil
	case ingress.Hostname != "":
		return &ingress.Hostname, nil
	default:
		return nil, fmt.Errorf("neither IP- nor hostname-based ingress point available")
	}
}
