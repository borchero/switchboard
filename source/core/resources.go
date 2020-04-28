package core

import (
	"context"
	"fmt"

	"github.com/borchero/switchboard/api/v1alpha1"
	"github.com/borchero/switchboard/backends"
	"github.com/borchero/switchboard/core/utils"
	"go.borchero.com/typewriter"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type resourceReconciler struct {
	*Reconciler
	log typewriter.Logger
}

// RegisterResourceReconciler adds a new reconciliation loop to watch for changes of DNSResource.
func RegisterResourceReconciler(
	base *Reconciler, manager ctrl.Manager, log typewriter.Logger,
) error {
	reconciler := &resourceReconciler{base, log.With("resources")}
	return ctrl.
		NewControllerManagedBy(manager).
		For(&v1alpha1.DNSResource{}).
		Complete(reconciler)
}

func (r *resourceReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	var resource v1alpha1.DNSResource
	return r.doReconcile(
		request, &resource, r.log,
		func(log typewriter.Logger) error { return r.update(&resource, log) },
		func(log typewriter.Logger) error { return r.delete(&resource, log) },
		func(types.NamespacedName, typewriter.Logger) error { return nil },
	)
}

func (r *resourceReconciler) update(
	resource *v1alpha1.DNSResource, logger typewriter.Logger,
) error {
	ctx := context.Background()

	// 1) Get backend (potentially, an already deleted backend)
	backend, ok := r.cache.Get(resource.Spec.ZoneName)
	if !ok {
		return fmt.Errorf("Backend zone '%s' not found in cache", resource.Spec.ZoneName)
	}

	// 2) Update the backend
	record := r.backendRecord(resource.Spec)
	if err := backend.Update(record); err != nil {
		return fmt.Errorf("Failed updating backend: %s", err)
	}

	// 3) Update the resource if needed
	// 3.1) Status
	resource.Status = v1alpha1.DNSResourceStatus{Ready: true}
	if err := r.client.Status().Update(ctx, resource); err != nil {
		return fmt.Errorf("Failed updating status: %s", err)
	}

	// 3.2) Finalizer
	if utils.AttachFinalizerIfNeeded(&resource.ObjectMeta) {
		if err := r.client.Update(ctx, resource); err != nil {
			return fmt.Errorf("Failed adding finalizer: %s", err)
		}
	}

	return nil
}

func (r *resourceReconciler) delete(
	resource *v1alpha1.DNSResource, logger typewriter.Logger,
) error {
	ctx := context.Background()

	// 1) Get backend (potentially a deleted one). If it cannot be retrieved, we just move on as we
	// consider this to be sufficient for deletion.
	backend, ok := r.cache.Get(resource.Spec.ZoneName)
	if !ok {
		// This is the format for already deleted backends
		backend, ok = r.cache.Get(fmt.Sprintf("*%s", resource.Spec.ZoneName))
	}
	if ok {
		// 2) We got the backend, delete
		if err := backend.Delete(r.backendRecord(resource.Spec)); err != nil {
			return fmt.Errorf("Failed deleting from backend: %s", err)
		}
	}

	// 3) In any case, we need to remove the finalizer
	if utils.DetachFinalizerIfNeeded(&resource.ObjectMeta) {
		if err := r.client.Update(ctx, resource); err != nil {
			return fmt.Errorf("Failed removing finalizer: %s", err)
		}
	}

	// 4) And we're done
	return nil
}

func (*resourceReconciler) backendRecord(spec v1alpha1.DNSResourceSpec) backends.DNSRecord {
	return backends.DNSRecord{
		Name: spec.Domain,
		Type: string(spec.Type),
		TTL:  int(spec.TTL),
		Data: spec.Data,
	}
}
