package main

import (
	"context"
	"flag"
	"os"

	healthv1alpha1 "github.com/k8s-health-ai/k8s-health-ai/api/v1alpha1"
	"github.com/k8s-health-ai/k8s-health-ai/internal/controller"
	"github.com/k8s-health-ai/k8s-health-ai/internal/llm"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(healthv1alpha1.AddToScheme(scheme))
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	setupLog := ctrl.Log.WithName("setup")

	var metricsAddr string
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "bind address for metrics")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "bind address for health probes")
	flag.Parse()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		setupLog.Error(err, "load kubeconfig")
		os.Exit(1)
	}

	llmProv, err := llm.NewFromEnv(context.Background())
	if err != nil {
		setupLog.Error(err, "llm provider")
		os.Exit(1)
	}

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "kubernetes clientset")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         false,
	})
	if err != nil {
		setupLog.Error(err, "manager")
		os.Exit(1)
	}

	if err := (&controller.PodReconciler{
		Client:    mgr.GetClient(),
		APIReader: mgr.GetAPIReader(),
		K8s:       k8s,
		Scheme:    mgr.GetScheme(),
		LLM:       llmProv,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "healthz")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "readyz")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "manager exited")
		os.Exit(1)
	}
}
