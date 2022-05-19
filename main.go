package main

import (
	"context"
	"flag"

	configv1 "github.com/borchero/switchboard/api/v1"
	"github.com/borchero/switchboard/controllers"
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
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	// >>> core types
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// >>> cert-manager
	utilruntime.Must(certmanager.AddToScheme(scheme))
	// >>> traefik
	utilruntime.Must(traefik.AddToScheme(scheme))
	// >>> external-dns
	groupVersion := schema.GroupVersion{Group: "externaldns.k8s.io", Version: "v1alpha1"}
	scheme.AddKnownTypes(groupVersion,
		&endpoint.DNSEndpoint{},
		&endpoint.DNSEndpointList{},
	)
	metav1.AddToGroupVersion(scheme, groupVersion)
}

func main() {
	var cfgFile string
	flag.StringVar(&cfgFile, "config", "/etc/switchboard/config.yaml", "The config file to use.")
	flag.Parse()

	// Initialize logger
	ctx := context.Background()
	logger := zeus.Logger(ctx)
	defer zeus.Sync()

	// Create manager
	var err error
	options := ctrl.Options{Scheme: scheme}
	var config configv1.Config
	if cfgFile != "" {
		// Load the config file if present
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(cfgFile).OfKind(&config))
		if err != nil {
			logger.Fatal("failed to load config file", zap.Error(err))
		}
	}
	manager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		logger.Fatal("unable to create manager", zap.Error(err))
	}

	controller := controllers.NewIngressRouteReconciler(
		manager.GetClient(), manager.GetScheme(), logger, config.IngressConfig,
	)
	if err := controller.SetupWithManager(manager); err != nil {
		logger.Fatal("unable to start ingress route controller", zap.Error(err))
	}

	if err := manager.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Fatal("unable to set up ready check at /readyz", zap.Error(err))
	}
	if err := manager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Fatal("unable to set up health check at /healthz", zap.Error(err))
	}

	logger.Info("launching manager")
	if err := manager.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Fatal("failed to run manager", zap.Error(err))
	}
	logger.Info("gracefully shut down")
}
