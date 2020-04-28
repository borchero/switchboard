package core

import (
	"context"

	"github.com/borchero/switchboard/core/utils"
	"go.borchero.com/typewriter"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler serves as base for all
type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
	cache  utils.BackendCache
}

// NewReconciler initializes a new reconciler that can be shared among specialized reconcilers.
func NewReconciler(manager ctrl.Manager) *Reconciler {
	return &Reconciler{
		client: manager.GetClient(),
		scheme: manager.GetScheme(),
		cache:  utils.NewBackendCache(),
	}
}

func (r *Reconciler) doReconcile(
	request ctrl.Request,
	obj interface {
		runtime.Object
		metav1.Object
	},
	logger typewriter.Logger,
	update func(typewriter.Logger) error,
	delete func(typewriter.Logger) error,
	emptyDelete func(types.NamespacedName, typewriter.Logger) error,
) (ctrl.Result, error) {
	ctx := context.Background()
	if request.Namespace == "" {
		logger = logger.With(request.Name)
	} else {
		logger = logger.With(request.Namespace).With(request.Name)
	}

	// 1) Get reconciled object
	if err := r.client.Get(ctx, request.NamespacedName, obj); err != nil {
		if apierrs.IsNotFound(err) {
			// 1.1) Just ignore, object was deleted
			logger.Info("Object already deleted")
			if err := emptyDelete(request.NamespacedName, logger); err != nil {
				logger.Error("Error occurred while reconciling deleted object", err)
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, nil
		}
		logger.Error("Failed to get reconciled object", err)
		return ctrl.Result{Requeue: true}, nil
	}

	// 2) Run reconciliation
	if obj.GetDeletionTimestamp().IsZero() {
		logger.Info("Updating")
		if err := update(logger); err != nil {
			logger.Error("Failed updating object", err)
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		logger.Info("deleting")
		if err := delete(logger); err != nil {
			logger.Error("Failed deleting object", err)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	logger.Info("Successfully reconciled")

	return ctrl.Result{}, nil
}

/////////////////////////
/// UTILITY FUNCTIONS ///
/////////////////////////

func noDelete(typewriter.Logger) error {
	return nil
}

func noEmptyDelete(types.NamespacedName, typewriter.Logger) error {
	return nil
}
