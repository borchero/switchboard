package main

import (
	"context"
	"flag"
	"os"

	"github.com/borchero/switchboard/api/v1alpha1"
	"github.com/borchero/switchboard/core"
	"go.borchero.com/typewriter"

	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	clientgoscheme.AddToScheme(scheme)
	metav1.AddToGroupVersion(scheme, certmanager.SchemeGroupVersion)
	certmanager.AddToScheme(scheme)
	metav1.AddToGroupVersion(scheme, v1alpha1.GroupVersion)
	v1alpha1.AddToScheme(scheme)
}

func main() {
	// 1) Setup logging
	logger := typewriter.NewUserLogger("setup")
	ctrlLogger := typewriter.NewUserLogger("ctrl")
	logger.Info("Launching")

	// 2) Parse arguments
	var args struct {
		leaderElection bool
	}
	flag.BoolVar(
		&args.leaderElection, "enable-leader-election", false,
		"Enables leader election for this controller.",
	)
	flag.Parse()

	// 3) Initialize manager
	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: ":8080",
	}
	manager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		logger.Error("Unable to start manager", err)
		os.Exit(1)
	}

	// 4) Setup workers
	reconciler := core.NewReconciler(manager)

	if err := core.RegisterZoneReconciler(reconciler, manager, ctrlLogger); err != nil {
		logger.Error("Unable to register zone reconciler", err)
		os.Exit(1)
	}

	if err := core.RegisterRecordReconciler(reconciler, manager, ctrlLogger); err != nil {
		logger.Error("Unable to register record reconciler", err)
		os.Exit(1)
	}

	if err := core.RegisterZoneRecordReconciler(reconciler, manager, ctrlLogger); err != nil {
		logger.Error("Unable to register zone record reconciler", err)
		os.Exit(1)
	}

	if err := core.RegisterResourceReconciler(reconciler, manager, ctrlLogger); err != nil {
		logger.Error("Unable to register resource reconciler", err)
		os.Exit(1)
	}

	if err := core.RegisterServiceReconciler(reconciler, manager, ctrlLogger); err != nil {
		logger.Error("Unable to register service reconciler", err)
		os.Exit(1)
	}

	if err := core.RegisterNodeReconciler(reconciler, manager, ctrlLogger); err != nil {
		logger.Error("Unable to register node reconciler", err)
		os.Exit(1)
	}

	// 6) Index manager
	ctx := context.Background()
	if err := core.AddIndexes(ctx, manager); err != nil {
		logger.Error("Unable to add indexes", err)
		os.Exit(1)
	}

	// 7) TODO: Add health and readiness checks

	// 8) Start manager
	logger.Info("Starting controllers")
	if err := manager.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error("Unable to start manager", err)
		os.Exit(1)
	}
}
