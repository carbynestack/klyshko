/*
 * Copyright (c) 2022-2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"flag"
	"os"
	"time"

	"github.com/carbynestack/klyshko/castor"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	klyshkov1alpha1 "github.com/carbynestack/klyshko/api/v1alpha1"
	"github.com/carbynestack/klyshko/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(klyshkov1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

var (
	metricsAddr          = flag.String("metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	enableLeaderElection = flag.Bool("leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	probeAddr            = flag.String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	etcdEndpoint         = flag.String("etcd-endpoint", "172.18.1.129:2379", "The address of the etcd service used for cross VCP coordination.")
	etcdDialTimeout      = flag.Int("etcd-dial-timeout", 5, "The timeout (in seconds) for failing to establish a connection to the etcd service.")
	castorURL            = flag.String("castor-url", "http://cs-castor.default.svc.cluster.local:10100", "The base url of the castor service used to upload generated tuples.")
	provisionerImage     = flag.String("provisioner-image", "ghcr.io/carbynestack/klyshko-provisioner:latest", "The name of the provisioner image.")
	vcpIPAddress         = flag.String("vcp-ip-address", "172.18.1.128", "The IP address of the VCP.")
	ingressPortRangeMin  = flag.Uint("ingress-port-range-min", 30500, "The minimum port number for the ingress port range.")
	ingressPortRangeMax  = flag.Uint("ingress-port-range-max", 30504, "The maximum port number for the ingress port range.")
	egressPortRangeMin   = flag.Uint("egress-port-range-min", 30500, "The minimum port number for the egress port range.")
	egressPortRangeMax   = flag.Uint("egress-port-range-max", 30550, "The maximum port number for the egress port range.")
	egressServiceHost    = flag.String("egress-service-host", "istio-egressgateway.istio-system.svc.cluster.local", "The hostname of the Istio egress gateway service.")
	tlsEnabled           = flag.Bool("tls-enabled", false, "Enable TLS for inter-VCP communication.")
	tlsSecretName        = flag.String("tls-secret-name", "vcp-tls-secret", "The name of the secret containing the TLS client and CA certificates.")
)

func main() {

	// Add the zap logger flag set to the CLI
	// (see https://sdk.operatorframework.io/docs/building-operators/golang/references/logging/ for more information)
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     *metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: *probeAddr,
		LeaderElection:         *enableLeaderElection,
		LeaderElectionID:       "operator.klyshko.carbynestack.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{*etcdEndpoint},
		DialTimeout: time.Duration(*etcdDialTimeout) * time.Second,
	})
	if err != nil {
		setupLog.Error(err, "unable to create etcd client", "controller", "TupleGenerationJob")
		os.Exit(1)
	}
	defer func() {
		err := etcdClient.Close()
		setupLog.Error(err, "closing etcd client failed")
	}()

	castorClient := castor.NewClient(*castorURL)
	if err = controllers.NewTupleGenerationJobReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		etcdClient,
		castorClient,
		mgr.GetLogger()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TupleGenerationJob")
		os.Exit(1)
	}

	var tlsConfig *controllers.TLSConfig
	if *tlsEnabled {
		tlsConfig = &controllers.TLSConfig{SecretName: *tlsSecretName}
	}
	ingressPortRange, err := controllers.NewPortRange(uint32(*ingressPortRangeMin), uint32(*ingressPortRangeMax))
	if err != nil {
		setupLog.Error(err, "unable to create ingress port range", "controller", "TupleGenerationTask")
		os.Exit(1)
	}
	egressPortRange, err := controllers.NewPortRange(uint32(*egressPortRangeMin), uint32(*egressPortRangeMax))
	if err != nil {
		setupLog.Error(err, "unable to create egress port range", "controller", "TupleGenerationTask")
		os.Exit(1)
	}
	k8sClient := mgr.GetClient()
	networkManager, err := controllers.NewNetworkManager(ingressPortRange,
		*egressServiceHost,
		egressPortRange,
		tlsConfig,
		k8sClient)
	if err != nil {
		setupLog.Error(err, "unable to create network manager", "controller", "TupleGenerationTask")
		os.Exit(1)
	}

	if err = (&controllers.TupleGenerationTaskReconciler{
		Client:           k8sClient,
		Scheme:           mgr.GetScheme(),
		EtcdClient:       etcdClient,
		ProvisionerImage: *provisionerImage,
		VcpIPAddress:     *vcpIPAddress,
		NetworkManager:   networkManager,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TupleGenerationTask")
		os.Exit(1)
	}

	if err = (&controllers.TupleGenerationSchedulerReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		CastorClient: castorClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TupleGenerationScheduler")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
