/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"fmt"
	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
	"github.com/carbynestack/klyshko/castor"
	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clientv3 "go.etcd.io/etcd/client/v3"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"math"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strconv"
	"strings"
	"time"
)

const NumberOfVCPs = 2
const Timeout = 30 * time.Second
const PollingInterval = 1 * time.Second
const SchedulerNamespace = "default"
const SchedulerName = "test-scheduler"
const SchedulerConcurrency = 1
const SchedulerTupleThreshold = 50000

type vcp struct {
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
}

func setupVCP() (*vcp, error) {
	env := vcp{}
	env.testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing:    true,
		CRDDirectoryPaths:        []string{filepath.Join("..", "config", "crd", "bases")},
		AttachControlPlaneOutput: false,
	}
	var err error
	env.cfg, err = env.testEnv.Start()
	if err != nil {
		return nil, err
	}
	err = klyshkov1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}
	env.k8sClient, err = client.New(env.cfg, client.Options{Scheme: scheme.Scheme})
	return &env, err
}

func (vcp *vcp) tearDownVCP() error {
	return vcp.testEnv.Stop()
}

func (vcp *vcp) createVCPConfig(ctx context.Context, name string, namespace string, data map[string]string) {
	vcpConfig := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
	err := vcp.k8sClient.Create(ctx, &vcpConfig)
	if err != nil {
		Fail(fmt.Sprintf("couldn't create VCP configuration: %s", err))
	}
}

func (vcp *vcp) deleteVCPConfig(ctx context.Context, name string, namespace string) {
	vcpConfig := &v1.ConfigMap{}
	err := vcp.k8sClient.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, vcpConfig)
	if err != nil {
		Fail(fmt.Sprintf("couldn't get VCP configuration: %s", err))
	}
	err = vcp.k8sClient.Delete(ctx, vcpConfig)
	if err != nil {
		Fail(fmt.Sprintf("couldn't delete VCP configuration: %s", err))
	}
}

type Controller interface {
	SetupWithManager(manager.Manager) error
}

func (vcp *vcp) setupControllers(ctx context.Context, vcpID int, etcdClient *clientv3.Client, castorURL string) error {
	k8sManager, err := ctrl.NewManager(vcp.cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",                                             // Avoid colliding metrics servers by disabling
		Logger:             logf.Log.WithName(fmt.Sprintf("vcp-%d", vcpID)), // use scoped logger to ease debugging
	})
	if err != nil {
		return err
	}
	castorClient := castor.NewClient(castorURL)
	controllers := []Controller{
		NewTupleGenerationJobReconciler(
			k8sManager.GetClient(), k8sManager.GetScheme(), etcdClient, castorClient, k8sManager.GetLogger()),
		&TupleGenerationTaskReconciler{ // TODO Replace with constructors
			Client:           k8sManager.GetClient(),
			Scheme:           k8sManager.GetScheme(),
			EtcdClient:       etcdClient,
			ProvisionerImage: "carbynestack/klyshko-provisioner:1.0.0-SNAPSHOT",
		},
	}
	if vcpID == 0 {
		controllers = append(controllers, &TupleGenerationSchedulerReconciler{
			Client:       k8sManager.GetClient(),
			Scheme:       k8sManager.GetScheme(),
			CastorClient: castorClient,
		})
	}
	for _, controller := range controllers {
		err := controller.SetupWithManager(k8sManager)
		if err != nil {
			return err
		}
	}
	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
	return nil
}

type vc struct {
	vcps []vcp
	ectd *envtest.Etcd
}

func setupVC(ctx context.Context, numberOfVCPs int) (*vc, error) {
	vc := vc{}
	for i := 0; i < numberOfVCPs; i++ {
		vcp, err := setupVCP()
		if err != nil {
			return nil, err
		}

		// Create the VCP configuration
		vcp.createVCPConfig(ctx, "cs-vcp-config", "default", map[string]string{
			"playerCount": strconv.Itoa(NumberOfVCPs),
			"playerId":    strconv.Itoa(i),
		})

		// Setup controllers using ectd client connected to first VCPs control plane etcd
		if i == 0 {
			vc.ectd = vcp.testEnv.ControlPlane.Etcd
		}
		etcdClient, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{vc.ectd.URL.String()},
			DialTimeout: Timeout,
		})
		err = vcp.setupControllers(ctx, i, etcdClient, "http://cs-castor.default.svc.cluster.local:10100")
		if err != nil {
			return nil, err
		}

		vc.vcps = append(vc.vcps, *vcp)
	}
	return &vc, nil
}

func (vc *vc) teardown() error {
	for _, vcp := range vc.vcps {
		err := vcp.tearDownVCP()
		if err != nil {
			return err
		}
	}
	return nil
}

func setupCastorServiceResponders(numberOfAvailableTuples int) {
	httpmock.Reset()
	httpmock.RegisterResponder(
		"PUT",
		"=~^http://cs-castor.default.svc.cluster.local:10100/intra-vcp/tuple-chunks/activate/.*",
		httpmock.NewStringResponder(200, ""),
	)
	telemetry := castor.Telemetry{TupleMetrics: []castor.TupleMetrics{
		{
			Available:       numberOfAvailableTuples,
			ConsumptionRate: 0,
			TupleType:       "MULTIPLICATION_TRIPLE_GFP",
		},
	}}
	responder, err := httpmock.NewJsonResponder(200, telemetry)
	Expect(err).NotTo(HaveOccurred())
	httpmock.RegisterResponder(
		"GET",
		"=~^http://cs-castor.default.svc.cluster.local:10100/intra-vcp/telemetry",
		responder,
	)
}

var _ = Describe("Generating tuples", func() {

	When("a scheduler is deployed", func() {

		var (
			ctx                context.Context
			cancel             context.CancelFunc
			vc                 *vc
			scheduler          *klyshkov1alpha1.TupleGenerationScheduler
			jobs               []klyshkov1alpha1.TupleGenerationJob
			localTasksByVCP    []klyshkov1alpha1.TupleGenerationTask
			generatorPodsByVCP []v1.Pod
		)

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.TODO())
			httpmock.Activate()
			setupCastorServiceResponders(0)
			var err error
			vc, err = setupVC(ctx, NumberOfVCPs)
			Expect(err).NotTo(HaveOccurred())

			scheduler = createScheduler(ctx, vc)
			jobs = ensureJobCreatedOnEachVcp(ctx, vc, scheduler)

			// Make Castor mock respond from here on with large number of available tuples, to ensure that no other
			// jobs are created
			setupCastorServiceResponders(math.MaxInt32)

			localTasksByVCP = ensureTasksCreatedOnEachVcp(ctx, vc, scheduler, jobs)
			generatorPodsByVCP = ensureGeneratorPodsCreatedOnEachVcp(ctx, vc, localTasksByVCP)
			ensureJobState(ctx, vc, scheduler, uuid.MustParse(jobs[0].Spec.ID), klyshkov1alpha1.JobRunning)
		})

		AfterEach(func() {
			cancel()
			err := vc.teardown()
			Expect(err).NotTo(HaveOccurred())
			httpmock.DeactivateAndReset()
		})

		Context("and the generator pod fails", func() {
			It("fails", func() {
				// Update generator pods to be in PodFailed state
				for i, pod := range generatorPodsByVCP {
					pod.Status.Phase = v1.PodFailed
					Expect(vc.vcps[i].k8sClient.Status().Update(ctx, &pod)).Should(Succeed())
				}
				ensureJobState(ctx, vc, scheduler, uuid.MustParse(jobs[0].Spec.ID), klyshkov1alpha1.JobFailed)
			})
		})

		Context("and the provisioner pod fails", func() {
			It("fails", func() {
				// Update generator pods to be in PodSucceed state
				for i, pod := range generatorPodsByVCP {
					pod.Status.Phase = v1.PodSucceeded
					Expect(vc.vcps[i].k8sClient.Status().Update(ctx, &pod)).Should(Succeed())
				}

				provisionerPodsByVCP := ensureProvisionerPodsCreatedOnEachVcp(ctx, vc, jobs, localTasksByVCP)

				// Update provisioner pods to be in PodFailed state
				for i, pod := range provisionerPodsByVCP {
					pod.Status.Phase = v1.PodFailed
					Expect(vc.vcps[i].k8sClient.Status().Update(ctx, &pod)).Should(Succeed())
				}
				ensureJobState(ctx, vc, scheduler, uuid.MustParse(jobs[0].Spec.ID), klyshkov1alpha1.JobFailed)
			})
		})

		Context("the generator pod succeeds", func() {
			It("succeeds", func() {
				// Update generator pods to be in PodSucceed state
				for i, pod := range generatorPodsByVCP {
					pod.Status.Phase = v1.PodSucceeded
					Expect(vc.vcps[i].k8sClient.Status().Update(ctx, &pod)).Should(Succeed())
				}

				provisionerPodsByVCP := ensureProvisionerPodsCreatedOnEachVcp(ctx, vc, jobs, localTasksByVCP)

				// Update provisioner pods to be in PodSucceed state
				for i, pod := range provisionerPodsByVCP {
					pod.Status.Phase = v1.PodSucceeded
					Expect(vc.vcps[i].k8sClient.Status().Update(ctx, &pod)).Should(Succeed())
				}

				// Ensure that castor activate tuple chunk endpoint has been called on each VCP
				activationURL := fmt.Sprintf("PUT http://cs-castor.default.svc.cluster.local:10100/intra-vcp/tuple-chunks/activate/%s", jobs[0].Spec.ID)
				Eventually(func() bool {
					info := httpmock.GetCallCountInfo()
					return info[activationURL] == NumberOfVCPs
				}, Timeout, PollingInterval).Should(BeTrue())

				// Ensure that resources get deleted.
				// As of https://book-v2.book.kubebuilder.io/reference/envtest.html#testing-considerations garbage
				// collection does not work in envtest. Hence, we can only check that the jobs get deleted and ensure that
				// owner references are set up correctly for all our resources (see respective ensure... methods below).
				for i := 0; i < NumberOfVCPs; i++ {
					key := client.ObjectKey{
						Namespace: jobs[i].GetNamespace(),
						Name:      jobs[i].GetName(),
					}
					Eventually(func() bool {
						return apierrors.IsNotFound(vc.vcps[i].k8sClient.Get(ctx, key, &jobs[i]))
					}, Timeout, PollingInterval).Should(BeTrue())
				}

				// Ensure that proxy tasks get deleted on all VCPs eventually after local tasks are deleted.
				for i := 0; i < NumberOfVCPs; i++ {
					Expect(vc.vcps[i].k8sClient.Delete(ctx, &localTasksByVCP[i])).Should(Succeed())
					for j := 0; j < NumberOfVCPs; j++ {
						if i == j {
							continue
						}
						key := client.ObjectKey{
							Namespace: jobs[j].GetNamespace(),
							Name:      fmt.Sprintf("%s-%d", jobs[j].GetName(), i),
						}
						proxyTask := &klyshkov1alpha1.TupleGenerationTask{}
						Eventually(func() bool {
							return apierrors.IsNotFound(vc.vcps[j].k8sClient.Get(ctx, key, proxyTask))
						}, Timeout, PollingInterval).Should(BeTrue())
					}
				}
			})
		})
	})
})

// Ensures that the job with the given identifier eventually assumes the given state.
func ensureJobState(ctx context.Context, vc *vc, owner *klyshkov1alpha1.TupleGenerationScheduler, jobID uuid.UUID, state klyshkov1alpha1.TupleGenerationJobState) {
	for i := 0; i < NumberOfVCPs; i++ {
		name := client.ObjectKey{
			Namespace: owner.Namespace,
			Name:      fmt.Sprintf("%s-%s", owner.Name, jobID),
		}
		Eventually(func() bool {
			job := &klyshkov1alpha1.TupleGenerationJob{}
			err := vc.vcps[i].k8sClient.Get(ctx, name, job)
			if err != nil {
				return false
			}
			return job.Status.State == state
		}, Timeout, PollingInterval).Should(BeTrue())
	}
}

// Ensures that pods with a certain name eventually become available in each VCP of the given VC and are owned by a
// certain object. Both name and owner are computed from the provided functions that take the VCP identifier as an
// argument.
func ensurePodsCreatedOnEachVcp(ctx context.Context, vc *vc, name func(int) types.NamespacedName, owner func(int) client.Object) []v1.Pod {
	pods := make([]v1.Pod, NumberOfVCPs)
	for i := 0; i < NumberOfVCPs; i++ {
		pod := &v1.Pod{}
		Eventually(func() bool {
			err := vc.vcps[i].k8sClient.Get(ctx, name(i), pod)
			if err != nil {
				return false
			}
			return true
		}, Timeout, PollingInterval).Should(BeTrue())
		expectedOwnerReference := metav1.OwnerReference{
			Kind:               owner(i).GetObjectKind().GroupVersionKind().Kind,
			APIVersion:         owner(i).GetObjectKind().GroupVersionKind().GroupVersion().String(),
			UID:                owner(i).GetUID(),
			Name:               owner(i).GetName(),
			Controller:         pointer.Bool(true),
			BlockOwnerDeletion: pointer.Bool(true),
		}
		Expect(pod.OwnerReferences).To(ContainElement(expectedOwnerReference))
		pods[i] = *pod
	}
	return pods
}

// Ensures that provisioner pods associated with the respective tasks eventually become available for the given job in
// each VCP of the given VC. In addition, it is checked that the pod is owned by the respective task.
func ensureProvisionerPodsCreatedOnEachVcp(ctx context.Context, vc *vc, jobs []klyshkov1alpha1.TupleGenerationJob, localTasks []klyshkov1alpha1.TupleGenerationTask) []v1.Pod {
	return ensurePodsCreatedOnEachVcp(ctx, vc, func(i int) types.NamespacedName {
		return types.NamespacedName{
			Namespace: jobs[i].Namespace,
			Name:      fmt.Sprintf("%s-provisioner", jobs[i].Name),
		}
	}, func(i int) client.Object {
		return &localTasks[i]
	})
}

// Ensures that generator pods associated with the respective tasks eventually become available in each VCP of the
// given VC. In addition, it is checked that the pod is owned by the respective task.
func ensureGeneratorPodsCreatedOnEachVcp(ctx context.Context, vc *vc, localTasks []klyshkov1alpha1.TupleGenerationTask) []v1.Pod {
	return ensurePodsCreatedOnEachVcp(ctx, vc, func(i int) types.NamespacedName {
		return types.NamespacedName{
			Namespace: localTasks[i].Namespace,
			Name:      localTasks[i].Name,
		}
	}, func(i int) client.Object {
		return &localTasks[i]
	})
}

// Ensures that tasks for the respective job resources eventually become available in each VCP of the given VC. It is
// also checked that tasks are owned by the respective job and that the task is in the expected state.
func ensureTasksCreatedOnEachVcp(ctx context.Context, vc *vc, scheduler *klyshkov1alpha1.TupleGenerationScheduler, jobs []klyshkov1alpha1.TupleGenerationJob) []klyshkov1alpha1.TupleGenerationTask {
	localTasks := make([]klyshkov1alpha1.TupleGenerationTask, NumberOfVCPs)
	for i := 0; i < NumberOfVCPs; i++ {
		taskList := &klyshkov1alpha1.TupleGenerationTaskList{}
		opts := []client.ListOption{
			client.InNamespace(scheduler.Namespace),
		}
		Eventually(func() bool {
			err := vc.vcps[i].k8sClient.List(ctx, taskList, opts...)
			if err != nil {
				return false
			}
			allGenerating := true
			for _, t := range taskList.Items {
				allGenerating = allGenerating && (t.Status.State == klyshkov1alpha1.TaskGenerating)
			}
			return len(taskList.Items) == NumberOfVCPs && allGenerating
		}, Timeout, PollingInterval).Should(BeTrue())
		for _, t := range taskList.Items {
			if strings.HasSuffix(t.Name, fmt.Sprintf("-%d", i)) {
				localTasks[i] = t
			}
		}
		Expect(localTasks[i].Status.State).To(Equal(klyshkov1alpha1.TaskGenerating))
		expectedOwnerReference := metav1.OwnerReference{
			Kind:               jobs[i].Kind,
			APIVersion:         jobs[i].APIVersion,
			UID:                jobs[i].UID,
			Name:               jobs[i].Name,
			Controller:         pointer.Bool(true),
			BlockOwnerDeletion: pointer.Bool(true),
		}
		Expect(localTasks[i].OwnerReferences).To(ContainElement(expectedOwnerReference))
	}
	return localTasks
}

// Ensures that a job is created on each VCP of the given VC. Also ensures that job identifiers are the same across all
// VCPs.
func ensureJobCreatedOnEachVcp(ctx context.Context, vc *vc, scheduler *klyshkov1alpha1.TupleGenerationScheduler) []klyshkov1alpha1.TupleGenerationJob {
	jobs := make([]klyshkov1alpha1.TupleGenerationJob, NumberOfVCPs)
	for i := 0; i < NumberOfVCPs; i++ {
		jobList := &klyshkov1alpha1.TupleGenerationJobList{}
		opts := []client.ListOption{
			client.InNamespace(scheduler.Namespace),
		}
		Eventually(func() bool {
			err := vc.vcps[i].k8sClient.List(ctx, jobList, opts...)
			if err != nil {
				return false
			}
			return len(jobList.Items) != 0
		}, Timeout, PollingInterval).Should(BeTrue())
		job := jobList.Items[0]
		Expect(job.Spec.Type).To(Equal("MULTIPLICATION_TRIPLE_GFP"))
		Expect(job.Spec.Count > 0).Should(BeTrue())
		if i > 0 {
			Expect(job.Spec.ID).To(Equal(jobs[0].Spec.ID))
		}
		jobs[i] = job
	}
	return jobs
}

// Creates a scheduler and waits until it becomes available. Checks that created scheduler spec properties are as
// expected.
func createScheduler(ctx context.Context, vc *vc) *klyshkov1alpha1.TupleGenerationScheduler {
	scheduler := &klyshkov1alpha1.TupleGenerationScheduler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SchedulerName,
			Namespace: SchedulerNamespace,
		},
		Spec: klyshkov1alpha1.TupleGenerationSchedulerSpec{
			Concurrency:             SchedulerConcurrency,
			Threshold:               SchedulerTupleThreshold,
			TTLSecondsAfterFinished: 5,
			Generator: klyshkov1alpha1.GeneratorSpec{
				Image: "carbynestack/klyshko-mp-spdz:1.0.0-SNAPSHOT",
			},
		},
	}
	Expect(vc.vcps[0].k8sClient.Create(ctx, scheduler)).Should(Succeed())

	key := types.NamespacedName{Name: SchedulerName, Namespace: SchedulerNamespace}
	createdScheduler := &klyshkov1alpha1.TupleGenerationScheduler{}
	Eventually(func() bool {
		err := vc.vcps[0].k8sClient.Get(ctx, key, createdScheduler)
		if err != nil {
			return false
		}
		return true
	}, Timeout, PollingInterval).Should(BeTrue())
	Expect(createdScheduler.Spec.Threshold).To(Equal(SchedulerTupleThreshold))
	Expect(createdScheduler.Spec.Concurrency).To(Equal(SchedulerConcurrency))
	return createdScheduler
}
