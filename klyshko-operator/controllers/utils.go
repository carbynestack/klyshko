/*
Copyright (c) 2022-2023 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
	"strconv"
	"strings"
)

// taskName returns the name of the task for a given player derived from a given job name.
func taskName(jobName string, playerID uint) string {
	return jobName + "-" + strconv.Itoa(int(playerID))
}

// taskNameForJob returns the name of the task for a given player associated with the given job.
func taskNameForJob(job *klyshkov1alpha1.TupleGenerationJob, playerID uint) string {
	return taskName(job.Name, playerID)
}

// jobName returns the name of the job the given task belongs to.
func jobNameFromTaskName(taskName string) string {
	return taskName[:strings.LastIndex(taskName, "-")]
}
