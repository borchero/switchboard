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

type nodeReconciler struct {
	*Reconciler
	log typewriter.Logger
}

// RegisterNodeReconciler adds a new reconciliation loop to watch for changes in nodes to properly
// update DNS records referencing zones.
func RegisterNodeReconciler(base *Reconciler, manager ctrl.Manager, log typewriter.Logger) error {
	reconciler := &nodeReconciler{base, log.With("nodes")}
	return ctrl.
		NewControllerManagedBy(manager).
		For(&v1.Node{}).
		Complete(reconciler)
}

func (r *nodeReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	var node v1.Node
	return r.doReconcile(
		request, &node, r.log,
		func(log typewriter.Logger) error { return r.update(&node, log) },
		noDelete, r.delete,
	)
}

func (r *nodeReconciler) update(node *v1.Node, logger typewriter.Logger) error {
	// 1) Although we would have more information, we just trigger an update for every zone that
	// references a node. This way, we can be sure that every zone record sources the "lowest"
	// available IP at all times.
	return r.updateAllRecordsReferencingNodes()
}

func (r *nodeReconciler) delete(name types.NamespacedName, logger typewriter.Logger) error {
	// 1) We just trigger an update for every zone record that references a node - we don't have
	// any more information here
	return r.updateAllRecordsReferencingNodes()
}

func (r *nodeReconciler) updateAllRecordsReferencingNodes() error {
	ctx := context.Background()

	// 1) Get records
	var records v1alpha1.DNSZoneRecordList
	option := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(indexFieldIPSourceType, "node"),
	}
	if err := r.client.List(ctx, &records, option); err != nil {
		return fmt.Errorf("Failed listing records referencing service: %s", err)
	}

	// 2) Update them
	for _, record := range records.Items {
		utils.UpdateRollmeAnnotation(&record.ObjectMeta)
		if err := r.client.Update(ctx, &record); err != nil {
			return fmt.Errorf("Failed updating record '%s': %s", record.Name, err)
		}
	}

	return nil
}
