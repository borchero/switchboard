package core

import (
	"context"
	"fmt"

	"github.com/borchero/switchboard/api/v1alpha1"
	"github.com/borchero/switchboard/core/utils"
	"go.borchero.com/typewriter"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type serviceReconciler struct {
	*Reconciler
	log typewriter.Logger
}

// RegisterServiceReconciler adds a new reconciliation loop to watch for changes in services to
// properly update DNS records referencing zones.
func RegisterServiceReconciler(
	base *Reconciler, manager ctrl.Manager, log typewriter.Logger,
) error {
	reconciler := &serviceReconciler{base, log.With("services")}
	return ctrl.
		NewControllerManagedBy(manager).
		For(&v1.Service{}).
		Complete(reconciler)
}

func (r *serviceReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// Note that we don't have a deletion function as we do not have finalizers. Also, we explicitly
	// don't set any finalizers as deletion of a resource results in a DNS record unable to update
	// itself anyway.
	var service v1.Service
	return r.doReconcile(
		request, &service, r.log,
		func(log typewriter.Logger) error { return r.update(&service, log) },
		noDelete, r.delete,
	)
}

func (r *serviceReconciler) update(service *v1.Service, logger typewriter.Logger) error {
	ctx := context.Background()

	// 1) First, we list all DNS zone records that reference this service
	name := namespacedServiceName(*service)
	records, err := r.referencingZoneRecords(name)
	if err != nil {
		return err
	}

	// 2) Then, we force-update all the records we found
	for _, record := range records {
		utils.UpdateRollmeAnnotation(&record.ObjectMeta)
		if err := r.client.Update(ctx, &record); err != nil {
			return fmt.Errorf("Failed updating record '%s': %s", record.Name, err)
		}
	}

	return nil
}

func (r *serviceReconciler) delete(name types.NamespacedName, logger typewriter.Logger) error {
	ctx := context.Background()

	// 1) First, we list all DNS zone records that reference this service
	records, err := r.referencingZoneRecords(name)
	if err != nil {
		return err
	}

	// 2) Then, we delete all the records that were found
	for _, record := range records {
		if err := r.client.Delete(ctx, &record); err != nil {
			return fmt.Errorf("Failed deleting record '%s': %s", record.Name, err)
		}
	}

	return nil
}

func (r *serviceReconciler) referencingZoneRecords(
	name types.NamespacedName,
) ([]v1alpha1.DNSZoneRecord, error) {
	ctx := context.Background()

	var records v1alpha1.DNSZoneRecordList
	option := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(indexFieldReferencedService, name.String()),
	}
	if err := r.client.List(ctx, &records, option); err != nil {
		return nil, fmt.Errorf("Failed listing records referencing service: %s", err)
	}

	return records.Items, nil
}

func namespacedServiceName(service v1.Service) types.NamespacedName {
	return types.NamespacedName{
		Name:      service.Name,
		Namespace: service.Namespace,
	}
}
