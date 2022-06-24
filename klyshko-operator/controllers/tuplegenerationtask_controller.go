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
	clientv3 "go.etcd.io/etcd/client/v3"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
)

type TupleGenerationTaskReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	EtcdClient *clientv3.Client
}

func (r *TupleGenerationTaskReconciler) taskKeyFromName(namespace string, name string) (*RosterEntryKey, error) {
	p := strings.Split(name, "-")
	pid := p[len(p)-1]
	jobName := strings.Join(p[:len(p)-1], "-")
	key, err := ParseKey(rosterKey + "/" + namespace + "/" + jobName + "/" + pid)
	if err != nil {
		return nil, err
	}
	rev, ok := key.(RosterEntryKey)
	if !ok {
		return nil, fmt.Errorf("not a task key: %v", key)
	}
	return &rev, nil
}

func isLocalTaskKey(ctx context.Context, client *client.Client, key RosterEntryKey) (bool, error) {
	playerID, err := localPlayerID(ctx, client, key.Namespace)
	if err != nil {
		return false, fmt.Errorf("can't read local player ID: %w", err)
	}
	return playerID == key.PlayerID, nil
}

func (r *TupleGenerationTaskReconciler) getStatus(ctx context.Context, taskKey RosterEntryKey) (*klyshkov1alpha1.TupleGenerationTaskStatus, error) {
	resp, err := r.EtcdClient.Get(ctx, taskKey.ToEtcdKey())
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) != 1 {
		return nil, fmt.Errorf("no status available for roster entry: %v", taskKey)
	}
	status, err := klyshkov1alpha1.ParseFromJSON(resp.Kvs[0].Value)
	if err != nil {
		return nil, err
	}
	if !status.State.IsValid() {
		return nil, fmt.Errorf("status contains invalid state: %s", status.State)
	}
	return status, nil
}

func (r *TupleGenerationTaskReconciler) setStatus(ctx context.Context, taskKey RosterEntryKey, status *klyshkov1alpha1.TupleGenerationTaskStatus) error {
	encoded, err := json.Marshal(status)
	if err != nil {
		return err
	}
	_, err = r.EtcdClient.Put(ctx, taskKey.ToEtcdKey(), string(encoded))
	return err
}

func (r *TupleGenerationTaskReconciler) setState(ctx context.Context, taskKey RosterEntryKey, status *klyshkov1alpha1.TupleGenerationTaskStatus, state klyshkov1alpha1.TupleGenerationTaskState) error {
	logger := log.FromContext(ctx).WithValues("Task.Key", taskKey)
	logger.Info("task transitioning into new state", "from", status.State, "to", state)
	status.State = state
	return r.setStatus(ctx, taskKey, status)
}

//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationtasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationtasks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationtasks/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *TupleGenerationTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("Task.Name", req.Name)
	taskKey, err := r.taskKeyFromName(req.Namespace, req.Name)
	if err != nil {
		logger.Error(err, "failed to get task key")
		return ctrl.Result{}, err
	}

	// Skip if remote task
	local, err := isLocalTaskKey(ctx, &r.Client, *taskKey)
	if err != nil {
		logger.Error(err, "failed to check task type")
		return ctrl.Result{}, err
	}
	if !local {
		return ctrl.Result{}, nil
	}

	// Cleanup if task has been deleted
	task := &klyshkov1alpha1.TupleGenerationTask{}
	err = r.Get(ctx, req.NamespacedName, task)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Task resource not available -> has been deleted
			_, err := r.EtcdClient.Delete(ctx, taskKey.ToEtcdKey())
			if err != nil {
				logger.Error(err, "failed to delete roster entry")
				return ctrl.Result{}, err
			}
			logger.Info("roster entry deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "failed to read task resource")
		return ctrl.Result{}, err
	}
	logger.Info("task exists already")

	// Create roster entry if not existing
	resp, err := r.EtcdClient.Get(ctx, taskKey.ToEtcdKey())
	if err != nil {
		logger.Error(err, "failed to fetch roster entry", "RosterEntry.Key", taskKey)
		return ctrl.Result{}, err
	}
	if resp.Count == 0 {
		status, err := json.Marshal(&klyshkov1alpha1.TupleGenerationTaskStatus{State: klyshkov1alpha1.TaskLaunching})
		_, err = r.EtcdClient.Put(ctx, taskKey.ToEtcdKey(), string(status))
		if err != nil {
			logger.Error(err, "failed to create roster entry")
			return ctrl.Result{}, err
		}
		logger.Info("roster entry created")
	} else {
		logger.Info("roster entry exists already")
	}
	status, err := r.getStatus(ctx, *taskKey)
	if err != nil {
		logger.Error(err, "task status not available")
		return ctrl.Result{}, err
	}

	// Lookup job
	job := &klyshkov1alpha1.TupleGenerationJob{}
	err = r.Get(ctx, taskKey.NamespacedName, job)
	if err != nil {
		logger.Error(err, "failed to lookup job")
		return ctrl.Result{}, err
	}

	// Update the status according to state in etcd
	taskStatus, err := r.getStatus(ctx, *taskKey)
	if err != nil {
		return ctrl.Result{}, err
	}
	task.Status = *taskStatus
	if err := r.Status().Update(ctx, task); err != nil {
		logger.Error(err, "unable to update task status")
		return ctrl.Result{}, err
	}

	if status.State == klyshkov1alpha1.TaskLaunching {
		// Create persistent volume claim used to store generated tuples, if not existing
		err = r.createPVC(ctx, taskKey)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Create generator pod if not existing
		_, err := r.createGeneratorPod(ctx, *taskKey, job, task)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{
			Requeue: true,
		}, r.setState(ctx, *taskKey, status, klyshkov1alpha1.TaskGenerating)
	}

	if status.State == klyshkov1alpha1.TaskGenerating {
		genPod, err := r.getGeneratorPod(ctx, task)
		if err != nil {
			return ctrl.Result{}, err
		}
		switch genPod.Status.Phase {
		case v1.PodSucceeded:
			// Generation successful, create provisioner pod to upload tuple shares to VCP-local castor
			_, err := r.createProvisionerPod(ctx, *taskKey, job, task)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{
				Requeue: true,
			}, r.setState(ctx, *taskKey, status, klyshkov1alpha1.TaskProvisioning)
		case v1.PodFailed:
			return ctrl.Result{
				Requeue: true,
			}, r.setState(ctx, *taskKey, status, klyshkov1alpha1.TaskFailed)
		}
	}

	if status.State == klyshkov1alpha1.TaskProvisioning {
		provPod, err := r.getProvisionerPod(ctx, *taskKey)
		if err != nil {
			return ctrl.Result{}, err
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

	logger.Info("desired state reached")
	return ctrl.Result{}, nil
}

func (r *TupleGenerationTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&klyshkov1alpha1.TupleGenerationTask{}).
		Owns(&v1.Pod{}).
		Complete(r)
}

func pvcName(key RosterEntryKey) string {
	return key.Name + "-" + strconv.Itoa(int(key.PlayerID))
}

func (r *TupleGenerationTaskReconciler) createPVC(ctx context.Context, key *RosterEntryKey) error {
	logger := log.FromContext(ctx).WithValues("Task.Key", key)
	name := types.NamespacedName{
		Name:      pvcName(*key),
		Namespace: key.Namespace,
	}
	found := &v1.PersistentVolumeClaim{}
	err := r.Get(ctx, name, found)
	if err == nil {
		logger.Info("persistent volume claim already exists")
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
					"storage": resource.MustParse("100Mi"),
				},
			},
		},
	}
	logger.Info("creating persistent volume claim", "PVC", pvc)
	err = r.Create(ctx, pvc)
	if err != nil {
		logger.Error(err, "persistent volume claim creation failed")
		return err
	}
	return nil
}

func provisionerPodName(key RosterEntryKey) string {
	return key.Name + "-provisioner"
}

func (r *TupleGenerationTaskReconciler) getProvisionerPod(ctx context.Context, key RosterEntryKey) (*v1.Pod, error) {
	name := types.NamespacedName{
		Name:      provisionerPodName(key),
		Namespace: key.Namespace,
	}
	found := &v1.Pod{}
	err := r.Get(ctx, name, found)
	return found, err
}

func (r *TupleGenerationTaskReconciler) createProvisionerPod(ctx context.Context, key RosterEntryKey, job *klyshkov1alpha1.TupleGenerationJob, task *klyshkov1alpha1.TupleGenerationTask) (*v1.Pod, error) {
	logger := log.FromContext(ctx).WithValues("Task.Key", key)
	name := types.NamespacedName{
		Name:      provisionerPodName(key),
		Namespace: key.Namespace,
	}
	found, err := r.getProvisionerPod(ctx, key)
	if err == nil {
		logger.Info("provisioner pod already exists")
		return found, nil
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:  "generator",
				Image: "carbynestack/klyshko-provisioner:1.0.0-SNAPSHOT",
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
	logger.Info("creating provisioner pod", "Pod", pod)
	err = ctrl.SetControllerReference(task, pod, r.Scheme)
	if err != nil {
		logger.Error(err, "setting owner reference failed")
		return nil, err
	}
	err = r.Create(ctx, pod)
	if err != nil {
		logger.Error(err, "provisioner pod creation failed")
		return nil, err
	}
	return pod, nil
}

func (r *TupleGenerationTaskReconciler) getGeneratorPod(ctx context.Context, task *klyshkov1alpha1.TupleGenerationTask) (*v1.Pod, error) {
	found := &v1.Pod{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      task.Name,
		Namespace: task.Namespace,
	}, found)
	return found, err
}

func (r *TupleGenerationTaskReconciler) createGeneratorPod(ctx context.Context, key RosterEntryKey, job *klyshkov1alpha1.TupleGenerationJob, task *klyshkov1alpha1.TupleGenerationTask) (*v1.Pod, error) {
	logger := log.FromContext(ctx).WithValues("Task.Key", key)
	found, err := r.getGeneratorPod(ctx, task)
	if err == nil {
		logger.Info("pod already exists")
		return found, nil
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      task.Name,
			Namespace: task.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "generator",
					Image: "carbynestack/klyshko-mp-spdz:1.0.0-SNAPSHOT", // TODO Read from config
					Command: []string{
						"/bin/bash",
						"-c",
					},
					Args: []string{
						"./kii-run.sh",
					},
					Env: []v1.EnvVar{
						{
							Name:  "KII_JOB_ID",
							Value: job.Spec.ID,
						},
						{
							Name:  "KII_PLAYER_COUNT",
							Value: "2", // TODO Read from config
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
							Value: "/kii", // TODO Read from config
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
	logger.Info("creating pod", "Pod", pod)
	err = ctrl.SetControllerReference(task, pod, r.Scheme)
	if err != nil {
		logger.Error(err, "setting owner reference failed")
		return nil, err
	}
	err = r.Create(ctx, pod)
	if err != nil {
		logger.Error(err, "pod creation failed")
		return nil, err
	}
	return pod, nil
}
