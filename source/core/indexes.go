package core

import (
	"context"

	"github.com/borchero/switchboard/api/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// indexFieldPartOfZone can be used to retrieve records referring to a zone. It is defined on
	// the following types:
	// * v1alpha1.DNSRecord
	// * v1alpha1.DNSZoneRecord
	indexFieldPartOfZone = "custom.partof.zones"

	// indexFieldReferencedService can be used to retrieve records that source a service to find
	// their IP. It is defined on the following types:
	// * v1alpha1.DNSZoneRecord
	indexFieldReferencedService = "custom.referenced.service"

	// indexFieldOwner can be used to retrieve items by the name of their owner. It is defined for
	// the following types:
	// * v1alpha1.DNSZoneRecord
	// * v1alpha1.DNSResource
	// * v1alpha2.Certificate
	indexFieldOwner = "custom.owner"

	// indexFieldIPSourceType can be used to filter zone records by the type of IP they are
	// referencing. It is defined for the following types:
	// * v1alpha1.DNSZoneRecord
	indexFieldIPSourceType = "custom.ipsource.type"
)

type indexer struct {
	client client.FieldIndexer
	err    error
}

func (i *indexer) indexField(
	ctx context.Context, obj runtime.Object, field string, extractValue client.IndexerFunc,
) {
	if i.err != nil {
		return
	}
	i.err = i.client.IndexField(ctx, obj, field, extractValue)
}

// AddIndexes adds indexes to the manager which will speed up this controller significantly.
func AddIndexes(ctx context.Context, manager ctrl.Manager) error {
	// 1) We use the indexer to accumulate errors in the struct instead of checking for err != nil
	// after the addition of every index.
	indexer := indexer{manager.GetFieldIndexer(), nil}

	// 2) indexFieldPartOfZone
	indexer.indexField(ctx, &v1alpha1.DNSRecord{}, indexFieldPartOfZone, extractZoneName)
	indexer.indexField(ctx, &v1alpha1.DNSZoneRecord{}, indexFieldPartOfZone, extractZoneName)

	// 3) indexFieldReferencedService
	indexer.indexField(
		ctx, &v1alpha1.DNSZoneRecord{}, indexFieldReferencedService, extractServiceName,
	)

	// 4) indexFieldOwner
	indexer.indexField(ctx, &v1alpha1.DNSZoneRecord{}, indexFieldOwner, extractOwner)
	indexer.indexField(ctx, &v1alpha1.DNSResource{}, indexFieldOwner, extractOwner)
	indexer.indexField(ctx, &v1alpha2.Certificate{}, indexFieldOwner, extractOwner)

	// 5) indexFieldIPSourceType
	indexer.indexField(ctx, &v1alpha1.DNSZoneRecord{}, indexFieldIPSourceType, extractIPSource)

	// 6) Return the error that occurred when adding indexes (if any)
	return indexer.err
}

func extractZoneName(raw runtime.Object) []string {
	switch obj := raw.(type) {
	case *v1alpha1.DNSRecord:
		zones := make([]string, len(obj.Spec.Zones))
		for i, zone := range obj.Spec.Zones {
			zones[i] = zone.Name
		}
		return zones
	case *v1alpha1.DNSZoneRecord:
		return []string{obj.Spec.ZoneName}
	default:
		return nil
	}
}

func extractServiceName(raw runtime.Object) []string {
	switch obj := raw.(type) {
	case *v1alpha1.DNSZoneRecord:
		service := obj.Spec.IPSource.Service
		if service == nil {
			return nil
		}
		return []string{service.NamespacedName().String()}
	default:
		return nil
	}
}

func extractOwner(raw runtime.Object) []string {
	obj, ok := raw.(metav1.Object)
	if !ok {
		return nil
	}

	owner := metav1.GetControllerOf(obj)
	if owner == nil || owner.APIVersion != v1alpha1.GroupVersion.String() {
		return nil
	}

	switch obj.(type) {
	case *v1alpha1.DNSZoneRecord:
		if owner.Kind == "DNSRecord" {
			return []string{owner.Name}
		}
	case *v1alpha1.DNSResource:
		if owner.Kind == "DNSZoneRecord" {
			return []string{owner.Name}
		}
	case *v1alpha2.Certificate:
		if owner.Kind == "DNSZoneRecord" {
			return []string{owner.Name}
		}
	}

	return nil
}

func extractIPSource(raw runtime.Object) []string {
	switch obj := raw.(type) {
	case *v1alpha1.DNSZoneRecord:
		switch source := obj.Spec.IPSource; {
		case source.Static != nil:
			return []string{"static"}
		case source.Service != nil:
			return []string{"service"}
		case source.Node != nil:
			return []string{"node"}
		default:
			return nil
		}
	default:
		return nil
	}
}
