package core

import (
	"context"
	"fmt"
	"time"

	"github.com/borchero/switchboard/api/v1alpha1"
	"github.com/borchero/switchboard/core/utils"
	"go.borchero.com/typewriter"
	"k8s.io/apimachinery/pkg/fields"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type zoneReconciler struct {
	*Reconciler
	log     typewriter.Logger
	factory utils.BackendFactory
}

// RegisterZoneReconciler adds a new reconciliation loop to watch for changes of DNSZone.
func RegisterZoneReconciler(base *Reconciler, manager ctrl.Manager, log typewriter.Logger) error {
	reconciler := &zoneReconciler{
		base,
		log.With("zones"),
		utils.NewBackendFactory(base.client, log.With("backends")),
	}
	return ctrl.
		NewControllerManagedBy(manager).
		For(&v1alpha1.DNSZone{}).
		Complete(reconciler)
}

func (r *zoneReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	var zone v1alpha1.DNSZone
	return r.doReconcile(
		request, &zone, r.log,
		func(log typewriter.Logger) error { return r.update(&zone, log) },
		func(log typewriter.Logger) error { return r.delete(&zone, log) },
		noEmptyDelete,
	)
}

func (r *zoneReconciler) update(zone *v1alpha1.DNSZone, logger typewriter.Logger) error {
	ctx := context.Background()

	// 1) Create backend
	newBackend, err := r.factory.Create(zone.Spec)
	if err != nil {
		return fmt.Errorf("Failed creating backend for current spec: %s", err)
	}

	// 2) Check for changes
	if existingBackend, ok := r.cache.Get(zone.Name); !ok || !newBackend.Equal(existingBackend) {
		// 2.1) Either, the backend does not yet exist, and we just add it to the cache (the other
		// reconcilers will do the rest), or the backend already exists but has changed. In either
		// case, we update the backend
		r.cache.Update(zone.Name, newBackend)
	}

	// 3) In any case, we want to trigger updates of all DNSRecord items to ensure that the correct
	// template values are sourced for DNSZoneRecord items.
	if err := r.updateDNSRecords(zone); err != nil {
		return err
	}

	// 4) Lastly, we update the status to "ready" and attach the finalizer if needed
	// 4.1) Status
	zone.Status = v1alpha1.DNSZoneStatus{Domain: newBackend.Domain()}
	if err := r.client.Status().Update(ctx, zone); err != nil {
		return fmt.Errorf("Failed updating status: %s", err)
	}

	// 4.2) Finalizer
	if utils.AttachFinalizerIfNeeded(&zone.ObjectMeta) {
		if err := r.client.Update(ctx, zone); err != nil {
			return fmt.Errorf("Failed adding finalizer: %s", err)
		}
	}

	return nil
}

func (r *zoneReconciler) delete(zone *v1alpha1.DNSZone, logger typewriter.Logger) error {
	ctx := context.Background()

	// 1) We remove the original zone but keep a "legacy" zone such that we can still delete our
	// resources in the backend.
	deletionName := fmt.Sprintf("*%s", zone.Name)
	backend, ok := r.cache.Get(zone.Name)
	if ok {
		r.cache.Update(deletionName, backend)
	}
	r.cache.Remove(zone.Name)

	// 2) Delete all zone records referring to this zone. This will trigger the DNSRecord resources
	// to try to create them again but that's not this reconciler's problem.
	option := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(indexFieldPartOfZone, zone.Name),
	}
	var list v1alpha1.DNSZoneRecordList
	if err := r.client.List(ctx, &list, option); err != nil {
		return fmt.Errorf("Failed listing owned zone records: %s", err)
	}

	for _, record := range list.Items {
		if err := r.client.Delete(ctx, &record); err != nil {
			return fmt.Errorf("Failed deleting zone record: %s", err)
		}
	}

	// 2.1) We now wait for min(3 seconds + num records * 1 second, 60 seconds) to allow for the
	// graceful removal of all records. FIXME: think of some smarter solution
	duration := 3*time.Second + time.Duration(len(list.Items))*time.Second
	if duration > 60*time.Second {
		duration = 60 * time.Second
	}
	time.Sleep(duration)

	// 2.2) We can now also completely remove the backend
	r.cache.Remove(deletionName)

	// 3) If everything worked out, we can safely remove the finalizer
	if utils.DetachFinalizerIfNeeded(&zone.ObjectMeta) {
		if err := r.client.Update(ctx, zone); err != nil {
			return fmt.Errorf("Failed removing finalizer: %s", err)
		}
	}

	return nil
}

func (r *zoneReconciler) updateDNSRecords(zone *v1alpha1.DNSZone) error {
	ctx := context.Background()

	var records v1alpha1.DNSRecordList
	option := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(indexFieldPartOfZone, zone.Name),
	}
	if err := r.client.List(ctx, &records, option); err != nil {
		return fmt.Errorf("Failed listing DNS records referencing zone: %s", err)
	}

	for _, record := range records.Items {
		// We need to force-update here as, otherwise, no changes occur
		utils.UpdateRollmeAnnotation(&record.ObjectMeta)
		if err := r.client.Update(ctx, &record); err != nil {
			return fmt.Errorf("Failed updating DNS record: %s", err)
		}
	}

	return nil
}
