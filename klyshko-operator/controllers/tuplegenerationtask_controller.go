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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	// TODO Create POD using KII

	return ctrl.Result{}, nil
}

func (r *TupleGenerationTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&klyshkov1alpha1.TupleGenerationTask{}).
		Complete(r)
}
