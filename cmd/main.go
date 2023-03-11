package main

import (
	"context"
	"flag"
	"io/ioutil"

	configv1 "github.com/borchero/switchboard/internal/config/v1"
	"github.com/borchero/switchboard/internal/controllers"
	"github.com/borchero/zeus/pkg/zeus"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/yaml"
)

func main() {
	var cfgFile string
	flag.StringVar(&cfgFile, "config", "/etc/switchboard/config.yaml", "The config file to use.")
	flag.Parse()

	// Initialize logger
	ctx := context.Background()
	logger := zeus.Logger(ctx)
	defer zeus.Sync()

	// Load the config file if available
	var config configv1.Config
	if cfgFile != "" {
		contents, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			logger.Fatal("failed to read config file", zap.Error(err))
		}
		if err := yaml.Unmarshal(contents, &config); err != nil {
			logger.Fatal("failed to parse config file", zap.Error(err))
		}
	}

	// Initialize the options and the schema
	options := ctrl.Options{
		Scheme:                  runtime.NewScheme(),
		LeaderElection:          config.LeaderElection.LeaderElect,
		LeaderElectionID:        config.LeaderElection.ResourceName,
		LeaderElectionNamespace: config.LeaderElection.ResourceNamespace,
		MetricsBindAddress:      config.Metrics.BindAddress,
		HealthProbeBindAddress:  config.Health.HealthProbeBindAddress,
	}
	initScheme(config, options.Scheme)

	// Create the manager
	manager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		logger.Fatal("unable to create manager", zap.Error(err))
	}

	// Create the controllers
	controller, err := controllers.NewIngressRouteReconciler(manager.GetClient(), logger, config)
	if err != nil {
		logger.Fatal("unable to initialize ingress route controller", zap.Error(err))
	}
	if err := controller.SetupWithManager(manager); err != nil {
		logger.Fatal("unable to start ingress route controller", zap.Error(err))
	}

	// Add health check endpoints
	if err := manager.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Fatal("unable to set up ready check at /readyz", zap.Error(err))
	}
	if err := manager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Fatal("unable to set up health check at /healthz", zap.Error(err))
	}

	// Start the manager
	logger.Info("launching manager")
	if err := manager.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Fatal("failed to run manager", zap.Error(err))
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
		groupVersion := schema.GroupVersion{Group: "externaldns.k8s.io", Version: "v1alpha1"}
		scheme.AddKnownTypes(groupVersion,
			&endpoint.DNSEndpoint{},
			&endpoint.DNSEndpointList{},
		)
		metav1.AddToGroupVersion(scheme, groupVersion)
	}
}
