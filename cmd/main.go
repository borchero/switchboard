package main

import (
	"flag"
	"log/slog"
	"os"

	configv1 "github.com/borchero/switchboard/internal/config/v1"
	"github.com/borchero/switchboard/internal/controllers"
	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/go-logr/logr"
	traefik "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	externaldnsv1alpha1 "sigs.k8s.io/external-dns/apis/v1alpha1"
	"sigs.k8s.io/yaml"
)

func main() {
	var cfgFile string
	flag.StringVar(&cfgFile, "config", "/etc/switchboard/config.yaml", "The config file to use.")
	flag.Parse()

	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// Set controller-runtime logger to prevent spurious log messages
	ctrl.SetLogger(logr.FromSlogHandler(logger.Handler()))

	// Load the config file if available
	var config configv1.Config
	if cfgFile != "" {
		contents, err := os.ReadFile(cfgFile)
		if err != nil {
			logger.Error("failed to read config file", slog.Any("error", err))
			os.Exit(1)
		}
		if err := yaml.Unmarshal(contents, &config); err != nil {
			logger.Error("failed to parse config file", slog.Any("error", err))
			os.Exit(1)
		}
	}

	// Initialize the options and the schema
	options := ctrl.Options{
		Scheme:                  runtime.NewScheme(),
		LeaderElection:          config.LeaderElection.LeaderElect,
		LeaderElectionID:        config.LeaderElection.ResourceName,
		LeaderElectionNamespace: config.LeaderElection.ResourceNamespace,
		Metrics: server.Options{
			BindAddress: config.Metrics.BindAddress,
		},
		HealthProbeBindAddress: config.Health.HealthProbeBindAddress,
	}
	initScheme(config, options.Scheme)

	// Create the manager
	manager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		logger.Error("unable to create manager", slog.Any("error", err))
		os.Exit(1)
	}

	// Create the controllers
	controller, err := controllers.NewIngressRouteReconciler(manager.GetClient(), logger, config)
	if err != nil {
		logger.Error("unable to initialize ingress route controller", slog.Any("error", err))
		os.Exit(1)
	}
	if err := controller.SetupWithManager(manager); err != nil {
		logger.Error("unable to start ingress route controller", slog.Any("error", err))
		os.Exit(1)
	}

	// Add health check endpoints
	if err := manager.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error("unable to set up ready check at /readyz", slog.Any("error", err))
		os.Exit(1)
	}
	if err := manager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error("unable to set up health check at /healthz", slog.Any("error", err))
		os.Exit(1)
	}

	// Start the manager
	logger.Info("launching manager")
	if err := manager.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error("failed to run manager", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("gracefully shut down")
}

func initScheme(config configv1.Config, scheme *runtime.Scheme) {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(traefik.AddToScheme(scheme))

	if config.Integrations.CertManager != nil {
		utilruntime.Must(certmanager.AddToScheme(scheme))
	}

	if config.Integrations.ExternalDNS != nil {
		utilruntime.Must(externaldnsv1alpha1.AddToScheme(scheme))
	}
}
