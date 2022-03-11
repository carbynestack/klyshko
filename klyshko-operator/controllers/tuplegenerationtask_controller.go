/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
	return uint(playerID) == key.PlayerID, nil
}

//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationtasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationtasks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=klyshko.carbnyestack.io,resources=tuplegenerationtasks/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

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
	} else {
		logger.Info("task exists already")
	}

	// Create roster entry if not existing
	resp, err := r.EtcdClient.Get(ctx, taskKey.ToEtcdKey())
	if err != nil {
		logger.Error(err, "failed to fetch roster entry", "RosterEntry.Key", taskKey)
		return ctrl.Result{}, err
	}
	if resp.Count == 0 {
		_, err = r.EtcdClient.Put(ctx, taskKey.ToEtcdKey(), "")
		if err != nil {
			logger.Error(err, "failed to create roster entry")
			return ctrl.Result{}, err
		}
		logger.Info("roster entry created")
	} else {
		logger.Info("roster entry exists already")
	}

	// Lookup job
	job := &klyshkov1alpha1.TupleGenerationJob{}
	err = r.Get(ctx, taskKey.NamespacedName, job)
	if err != nil {
		logger.Error(err, "failed to lookup job")
		return ctrl.Result{}, err
	}

	// Create Pod
	err = r.createCRGPod(ctx, *taskKey, job, task)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TupleGenerationTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&klyshkov1alpha1.TupleGenerationTask{}).
		Owns(&v1.Pod{}).
		Complete(r)
}

func (r *TupleGenerationTaskReconciler) createCRGPod(ctx context.Context, key RosterEntryKey, job *klyshkov1alpha1.TupleGenerationJob, task *klyshkov1alpha1.TupleGenerationTask) error {
	logger := log.FromContext(ctx).WithValues("Task.Key", key)
	found := &v1.Pod{}
	err := r.Get(ctx, key.NamespacedName, found)
	if err == nil {
		logger.Info("pod already exists")
		return nil
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
			RestartPolicy:         v1.RestartPolicyNever,
			ShareProcessNamespace: pointer.Bool(true),
			Volumes: []v1.Volume{
				{
					Name: "kii",
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{},
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
		return err
	}
	err = r.Create(ctx, pod)
	if err != nil {
		logger.Error(err, "pod creation failed")
		return err
	}
	return nil
}
