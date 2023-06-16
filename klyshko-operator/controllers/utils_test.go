/*
Copyright (c) 2022-2023 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const myJobName = "my-job"

var _ = When("Getting task names", func() {
	Context("for different players", func() {
		It("should return different names", func() {
			Expect(taskName(myJobName, 1)).NotTo(Equal(taskName(myJobName, 2)))
		})
	})
	Context("for the same player", func() {
		It("should return the same name", func() {
			Expect(taskName(myJobName, 1)).To(Equal(taskName(myJobName, 1)))
		})
	})
})

var _ = When("Getting the task name", func() {
	Context("from a job", func() {
		It("should return the right task name", func() {
			job := &klyshkov1alpha1.TupleGenerationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: myJobName,
				},
			}
			playerID := uint(1)
			Expect(taskNameForJob(job, playerID)).To(Equal(taskName(myJobName, playerID)))
		})
	})
})

var _ = When("Getting the job name", func() {
	Context("from a task name", func() {
		It("should return the job name of the task", func() {
			Expect(jobNameFromTaskName(taskName(myJobName, 1))).To(Equal(myJobName))
		})
	})
})
