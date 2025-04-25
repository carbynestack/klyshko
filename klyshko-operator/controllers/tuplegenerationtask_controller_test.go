/*
Copyright (c) 2023-2025 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"errors"
	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Getting the inter-CRG networking service endpoint", func() {
	When("the service is not exposed via a load balance", func() {
		It("yields an error", func() {
			svc := &v1.Service{
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeNodePort,
				},
			}
			_, err := endpoint(svc)
			Expect(err).Should(HaveOccurred())
		})
	})
	When("no ingress is available", func() {
		It("returns nil", func() {
			svc := &v1.Service{
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{},
					},
				},
			}
			endpoint, err := endpoint(svc)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(endpoint).To(BeNil())
		})
	})
	When("an IP is attached to the ingress", func() {
		It("returns that IP", func() {
			ip := "127.0.0.1"
			svc := &v1.Service{
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{
								IP: ip,
							},
						},
					},
				},
			}
			endpoint, err := endpoint(svc)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*endpoint).To(Equal(ip))
		})
	})
	When("a hostname is attached to the ingress", func() {
		It("returns that hostname", func() {
			hostname := "apollo"
			svc := &v1.Service{
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{
								Hostname: hostname,
							},
						},
					},
				},
			}
			endpoint, err := endpoint(svc)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*endpoint).To(Equal(hostname))
		})
	})
	When("neither an IP nor a hostname is attached to the ingress", func() {
		It("yields an error", func() {
			svc := &v1.Service{
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{},
						},
					},
				},
			}
			_, err := endpoint(svc)
			Expect(err).Should(HaveOccurred())
		})
	})
})

var _ = Context("TupleGenerationTask controller", func() {
	var (
		ctx              context.Context
		cancel           context.CancelFunc
		fakeSetReference = func(owner, object metav1.Object, scheme *runtime.Scheme) error {
			// do nothing
			return nil
		}
		fakeK8sReader = NewFakeK8sReader()
		fakeK8sWriter = NewFakeK8sWriter()
		fakeK8sClient = &FakeK8sClient{
			FakeK8sReader: fakeK8sReader,
			FakeK8sWriter: fakeK8sWriter,
			StatusClient:  nil,
		}
		rosterKey = RosterEntryKey{
			PlayerID: 0,
			RosterKey: RosterKey{
				NamespacedName: types.NamespacedName{Namespace: "foo", Name: "bar"},
			},
		}
		p0Task = &klyshkov1alpha1.TupleGenerationTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task1",
				Namespace: "default",
			},
			Spec: klyshkov1alpha1.TupleGenerationTaskSpec{
				PlayerID: 0,
			},
		}
		p1Task = &klyshkov1alpha1.TupleGenerationTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task1",
				Namespace: "default",
			},
			Spec: klyshkov1alpha1.TupleGenerationTaskSpec{
				PlayerID: 1,
			},
		}
		job = &klyshkov1alpha1.TupleGenerationJob{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testJob",
			},
			Spec: klyshkov1alpha1.TupleGenerationJobSpec{
				ID:        "testJobId",
				Type:      "bits_gfp",
				Count:     1000,
				Generator: "testGenerator",
			},
		}
		generator = &klyshkov1alpha1.TupleGenerator{
			Spec: klyshkov1alpha1.TupleGeneratorSpec{
				Template: klyshkov1alpha1.TupleGeneratorPodTemplateSpec{
					Spec: klyshkov1alpha1.TupleGeneratorPodSpec{
						Affinity: nil,
						Container: klyshkov1alpha1.TupleGeneratorContainer{
							Image:           "testGeneratorImage",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources:       v1.ResourceRequirements{},
						},
					},
				},
			},
		}
		tlsEnvVarNames = []string{
			"KII_TLS_ENABLED",
			"KII_TLS_CERT",
			"KII_TLS_KEY",
			"KII_TLS_CACERT",
		}
		tlsVolumesAndMounts = []string{
			"certs",
		}
	)
	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.TODO())
		ctrl.SetControllerReference = fakeSetReference
		fakeK8sClient.Reset()
	})
	AfterEach(func() {
		cancel()
		ctrl.SetControllerReference = controllerutil.SetControllerReference
	})
	Describe("creating the generator pod", func() {
		When("pod does not exist", func() {
			var (
				configMap = &v1.ConfigMap{
					Data: map[string]string{
						"playerCount": "2",
						"playerId":    "0",
					},
				}
			)
			BeforeEach(func() {
				fakeK8sReader.
					AddGetResponse(GetObjectResponse{
						Error: errors.New("pod not found"),
					}).
					AddGetResponse(GetObjectResponse{
						Object: configMap,
					}).
					AddGetResponse(GetObjectResponse{
						Object: configMap,
					}).
					AddGetResponse(GetObjectResponse{
						Object: p0Task,
					}).
					AddGetResponse(GetObjectResponse{
						Object: p1Task,
					}).
					AddGetResponse(GetObjectResponse{
						Object: generator,
					})
				fakeK8sWriter.CreateReturnError = nil
			})
			When("TLSConfig is nil", func() {
				var (
					controller = &TupleGenerationTaskReconciler{
						TLSConfig: nil,
						Client:    fakeK8sClient,
					}
				)
				It("should not define any tls related env vars nor mount any secrets", func() {
					pod, err := controller.createGeneratorPod(ctx, rosterKey, job, p0Task)
					Expect(err).NotTo(HaveOccurred())
					Expect(pod).NotTo(BeNil())
					envVarNames := getEnvVarNames(pod.Spec.Containers[0].Env)
					volumeNames := getVolumeNames(pod.Spec.Volumes)
					volumeMountNames := getVolumeMountNames(pod.Spec.Containers[0].VolumeMounts)
					Expect(envVarNames).NotTo(ContainElements(tlsEnvVarNames))
					Expect(volumeNames).NotTo(ContainElements(tlsVolumesAndMounts))
					Expect(volumeMountNames).NotTo(ContainElements(tlsVolumesAndMounts))
				})
			})
			When("TLSConfig is runtime", func() {
				var (
					secretName = "testSecret"
					tlsConfig  = &TLSConfig{
						Mode:       TlsModeRuntime,
						SecretName: secretName,
					}
					controller = &TupleGenerationTaskReconciler{
						TLSConfig: tlsConfig,
						Client:    fakeK8sClient,
					}
				)
				It("should define all tls related env vars and mount the secret", func() {
					pod, err := controller.createGeneratorPod(ctx, rosterKey, job, p0Task)
					Expect(err).NotTo(HaveOccurred())
					Expect(pod).NotTo(BeNil())
					envVarNames := getEnvVarNames(pod.Spec.Containers[0].Env)
					volumeNames := getVolumeNames(pod.Spec.Volumes)
					volumeMountNames := getVolumeMountNames(pod.Spec.Containers[0].VolumeMounts)
					Expect(envVarNames).To(ContainElements(tlsEnvVarNames))
					Expect(volumeNames).To(ContainElements(tlsVolumesAndMounts))
					Expect(volumeMountNames).To(ContainElements(tlsVolumesAndMounts))
				})
			})
		})
	})
})

func getVolumeMountNames(mounts []v1.VolumeMount) []string {
	volumeMountNames := make([]string, len(mounts))
	for i, mount := range mounts {
		volumeMountNames[i] = mount.Name
	}
	return volumeMountNames
}

func getVolumeNames(volumes []v1.Volume) []string {
	volumeNames := make([]string, len(volumes))
	for i, volume := range volumes {
		volumeNames[i] = volume.Name
	}
	return volumeNames
}

func getEnvVarNames(env []v1.EnvVar) []string {
	envVarNames := make([]string, len(env))
	for i, envVar := range env {
		envVarNames[i] = envVar.Name
	}
	return envVarNames
}
