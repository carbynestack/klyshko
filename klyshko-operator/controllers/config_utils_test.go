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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = klyshkov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func createVcpConfig(ctx context.Context, name string, namespace string, data map[string]string) {
	vcpConfig := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
	err := k8sClient.Create(ctx, &vcpConfig)
	if err != nil {
		Fail(fmt.Sprintf("couldn't create VCP configuration: %s", err))
	}
}

func deleteVcpConfig(ctx context.Context, name string, namespace string) {
	vcpConfig := &v1.ConfigMap{}
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, vcpConfig)
	if err != nil {
		Fail(fmt.Sprintf("couldn't get VCP configuration: %s", err))
	}
	err = k8sClient.Delete(ctx, vcpConfig)
	if err != nil {
		Fail(fmt.Sprintf("couldn't delete VCP configuration: %s", err))
	}
}

var _ = Describe("Getting the local player Id", func() {
	ctx := context.Background()

	When("when no configuration has been provided", func() {
		It("fails", func() {
			_, err := localPlayerID(ctx, &k8sClient, "default")
			Expect(err).To(HaveOccurred())
		})
	})

	When("when valid configuration has been provided", func() {
		BeforeEach(func() {
			createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
				"playerCount": "2",
				"playerId":    "0",
			})
		})
		AfterEach(func() {
			deleteVcpConfig(ctx, "cs-vcp-config", "default")
		})
		It("gives the right local player ID", func() {
			Expect(localPlayerID(ctx, &k8sClient, "default")).To(Equal(uint(0)))
		})
	})

	When("when the playerId K/V pair is missing", func() {
		BeforeEach(func() {
			createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
				"playerCount": "2",
			})
		})
		AfterEach(func() {
			deleteVcpConfig(ctx, "cs-vcp-config", "default")
		})
		It("fails", func() {
			_, err := localPlayerID(ctx, &k8sClient, "default")
			Expect(err).To(HaveOccurred())
		})
	})

	When("when the playerId is out of range", func() {
		BeforeEach(func() {
			createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
				"playerCount": "2",
				"playerId":    "-1",
			})
		})
		AfterEach(func() {
			deleteVcpConfig(ctx, "cs-vcp-config", "default")
		})
		It("fails", func() {
			_, err := localPlayerID(ctx, &k8sClient, "default")
			Expect(err).To(HaveOccurred())
		})
	})

	When("when the playerId can't be parsed", func() {
		BeforeEach(func() {
			createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
				"playerCount": "2",
				"playerId":    "a1b2",
			})
		})
		AfterEach(func() {
			deleteVcpConfig(ctx, "cs-vcp-config", "default")
		})
		It("fails", func() {
			_, err := localPlayerID(ctx, &k8sClient, "default")
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Getting the number of players", func() {
	ctx := context.Background()

	When("when no configuration has been provided", func() {
		It("fails", func() {
			_, err := numberOfPlayers(ctx, &k8sClient, "default")
			Expect(err).To(HaveOccurred())
		})
	})

	When("when valid configuration has been provided", func() {
		BeforeEach(func() {
			createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
				"playerCount": "2",
				"playerId":    "0",
			})
		})
		AfterEach(func() {
			deleteVcpConfig(ctx, "cs-vcp-config", "default")
		})
		It("gives the right number of Players", func() {
			Expect(numberOfPlayers(ctx, &k8sClient, "default")).To(Equal(uint(2)))
		})
	})

	When("when the playerCount K/V pair is missing", func() {
		BeforeEach(func() {
			createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
				"playerId": "0",
			})
		})
		AfterEach(func() {
			deleteVcpConfig(ctx, "cs-vcp-config", "default")
		})
		It("fails", func() {
			_, err := numberOfPlayers(ctx, &k8sClient, "default")
			Expect(err).To(HaveOccurred())
		})
	})

	When("when the playerCount is out of range", func() {
		BeforeEach(func() {
			createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
				"playerCount": "-1",
				"playerId":    "0",
			})
		})
		AfterEach(func() {
			deleteVcpConfig(ctx, "cs-vcp-config", "default")
		})
		It("fails", func() {
			_, err := numberOfPlayers(ctx, &k8sClient, "default")
			Expect(err).To(HaveOccurred())
		})
	})

	When("when the playerId can't be parsed", func() {
		BeforeEach(func() {
			createVcpConfig(ctx, "cs-vcp-config", "default", map[string]string{
				"playerCount": "a1b2",
				"playerId":    "0",
			})
		})
		AfterEach(func() {
			deleteVcpConfig(ctx, "cs-vcp-config", "default")
		})
		It("fails", func() {
			_, err := numberOfPlayers(ctx, &k8sClient, "default")
			Expect(err).To(HaveOccurred())
		})
	})
})
