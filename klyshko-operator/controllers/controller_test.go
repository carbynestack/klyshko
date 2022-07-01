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
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clientv3 "go.etcd.io/etcd/client/v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strconv"
	"time"
)

const NumberOfVcps = 2

type Vcp struct {
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
}

func setupVcp() (*Vcp, error) {
	env := Vcp{}
	env.testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing:    true,
		CRDDirectoryPaths:        []string{filepath.Join("..", "config", "crd", "bases")},
		AttachControlPlaneOutput: true,
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

func (vcp Vcp) tearDownVcp() error {
	return vcp.testEnv.Stop()
}

func (vcp Vcp) createVcpConfig(ctx context.Context, name string, namespace string, data map[string]string) {
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

func (vcp Vcp) deleteVcpConfig(ctx context.Context, name string, namespace string) {
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

func (vcp Vcp) setupControllers(ctx context.Context, etcdClient *clientv3.Client, castorUrl string) error {
	k8sManager, err := ctrl.NewManager(vcp.cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0", // Avoid colliding metrics servers by disabling
	})
	if err != nil {
		return err
	}
	castorClient := NewCastorClient(castorUrl)
	controllers := []Controller{
		NewTupleGenerationJobReconciler(
			k8sManager.GetClient(), k8sManager.GetScheme(), etcdClient, castorClient),
		&TupleGenerationTaskReconciler{ // TODO Replace with constructors
			Client:     k8sManager.GetClient(),
			Scheme:     k8sManager.GetScheme(),
			EtcdClient: etcdClient,
		},
		&TupleGenerationSchedulerReconciler{
			Client:       k8sManager.GetClient(),
			Scheme:       k8sManager.GetScheme(),
			CastorClient: castorClient,
		},
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

type Vc struct {
	vcps []Vcp
	ectd *envtest.Etcd
}

func setupVc(ctx context.Context, numberOfVcps int) (*Vc, error) {
	vc := Vc{}
	for i := 0; i < numberOfVcps; i++ {
		vcp, err := setupVcp()
		if err != nil {
			return nil, err
		}

		// Create the VCP configuration
		vcp.createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
			"playerCount": strconv.Itoa(1),
			"playerId":    strconv.Itoa(i),
		})

		// Setup controllers using ectd client connected to first VCPs control plane etcd
		if i == 0 {
			vc.ectd = vcp.testEnv.ControlPlane.Etcd
		}
		etcdClient, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{vc.ectd.URL.String()},
			DialTimeout: 5 * time.Second,
		})
		err = vcp.setupControllers(ctx, etcdClient, "http://cs-castor.default.svc.cluster.local:10100")
		if err != nil {
			return nil, err
		}

		vc.vcps = append(vc.vcps, *vcp)
	}
	return &vc, nil
}

func (vc *Vc) teardown() error {
	for _, vcp := range vc.vcps {
		err := vcp.testEnv.Stop()
		if err != nil {
			return err
		}
	}
	return nil
}

var _ = Describe("Generating tuples", func() {
	ctx, cancel := context.WithCancel(context.TODO())
	var vc *Vc

	BeforeEach(func() {
		httpmock.Activate()
		httpmock.RegisterResponder(
			"GET",
			"=~^http://cs-castor.default.svc.cluster.local:10100/intra-vcp/tuple-chunks/activate/.*",
			httpmock.NewStringResponder(200, ""),
		)
		telemetry := Telemetry{TupleMetrics: []TupleMetrics{
			{
				Available:       0,
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
		vc, err = setupVc(ctx, NumberOfVcps)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		cancel()
		err := vc.teardown()
		Expect(err).NotTo(HaveOccurred())
		httpmock.DeactivateAndReset()
	})

	When("a scheduler is deployed", func() {
		It("succeed", func() {
			scheduler := &klyshkov1alpha1.TupleGenerationScheduler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-scheduler",
					Namespace: "default",
				},
				Spec: klyshkov1alpha1.TupleGenerationSchedulerSpec{
					Concurrency: 1,
					Threshold:   50000,
				},
			}
			Expect(vc.vcps[0].k8sClient.Create(ctx, scheduler)).Should(Succeed())

			schedulerLookupKey := types.NamespacedName{Name: "test-scheduler", Namespace: "default"}
			createdScheduler := &klyshkov1alpha1.TupleGenerationScheduler{}
			Eventually(func() bool {
				err := vc.vcps[0].k8sClient.Get(ctx, schedulerLookupKey, createdScheduler)
				if err != nil {
					return false
				}
				return true
			}, 60*time.Second, 5*time.Second).Should(BeTrue())
			time.Sleep(10 * time.Second)
		})
	})
})
