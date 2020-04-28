package core

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/borchero/switchboard/api/v1alpha1"
	"github.com/borchero/switchboard/backends"
	"github.com/borchero/switchboard/core/utils"
	"go.borchero.com/typewriter"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type zoneRecordReconciler struct {
	*Reconciler
	log typewriter.Logger
}

// RegisterZoneRecordReconciler adds a new reconciliation loop to watch for changes of
// DNSZoneRecord.
func RegisterZoneRecordReconciler(
	base *Reconciler, manager ctrl.Manager, log typewriter.Logger,
) error {
	reconciler := &zoneRecordReconciler{base, log.With("zonerecords")}
	return ctrl.
		NewControllerManagedBy(manager).
		For(&v1alpha1.DNSZoneRecord{}).
		Owns(&v1alpha1.DNSResource{}).
		Complete(reconciler)
}

func (r *zoneRecordReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	var record v1alpha1.DNSZoneRecord
	return r.doReconcile(
		request, &record, r.log,
		func(log typewriter.Logger) error { return r.update(&record, log) },
		noDelete, noEmptyDelete,
	)
}

func (r *zoneRecordReconciler) update(
	record *v1alpha1.DNSZoneRecord, logger typewriter.Logger,
) error {
	ctx := context.Background()

	logger.Info("updating")

	// 1) Ensure DNS resource update
	// 1.1) Get zone (needed for the domain)
	backend, ok := r.cache.Get(record.Spec.ZoneName)
	if !ok {
		return fmt.Errorf("Backend zone '%s' not found in cache", record.Spec.ZoneName)
	}

	// 1.2) Get expected IP of DNS resources
	ip, err := r.getIP(record.Spec.IPSource)
	if err != nil {
		return fmt.Errorf("Failed getting IP: %s", err)
	}

	// 1.4) Get expected status, i.e. list all expected DNS resources
	expectedResources, err := r.listExpectedResources(*record, backend, ip)
	if err != nil {
		return err
	}

	// 1.5) Get current status, i.e. list existing DNS resources
	ownedResources, err := r.listOwnedResources(*record)
	if err != nil {
		return err
	}

	// 1.6) Compute diff
	insertset, deletionset := utils.ResourceDiff(expectedResources, ownedResources)

	// 1.7) Create non-existing resources
	if len(insertset) > 0 {
		logger.
			WithV(typewriter.KV("count", strconv.Itoa(len(insertset)))).
			Info("Creating resources")
	}
	for _, insert := range insertset {
		if err := r.client.Create(ctx, &insert); err != nil {
			return fmt.Errorf("Failed creating DNS resource: %s", err)
		}
	}

	// 1.8) Delete falsely existing resources
	if len(deletionset) > 0 {
		logger.
			WithV(typewriter.KV("count", strconv.Itoa(len(insertset)))).
			Info("Deleting resources")
	}
	for _, delete := range deletionset {
		if err := r.client.Delete(ctx, &delete); err != nil {
			return fmt.Errorf("Failed deleting DNS resource: %s", err)
		}
	}

	// 2) Finally, everything is up-to-date. Note that we do not need any finalizers as all this
	// record does is creating other records - which will be deleted by Kubernetes' garbage
	// collection.
	return nil
}

func (r *zoneRecordReconciler) getIP(source v1alpha1.IPSource) (string, error) {
	switch {
	case source.Static != nil:
		return source.Static.IP, nil
	case source.Service != nil:
		ip, err := r.getServiceIP(*source.Service)
		if err != nil {
			return "", fmt.Errorf("Failed getting service IP: %s", err)
		}
		return ip, nil
	case source.Node != nil:
		ip, err := r.getNodeIP(*source.Node)
		if err != nil {
			return "", fmt.Errorf("Failed getting node IP: %s", err)
		}
		return ip, nil
	default:
		return "", fmt.Errorf("IP source not existing")
	}
}

func (r *zoneRecordReconciler) getServiceIP(svc v1alpha1.ServiceIPSource) (string, error) {
	ctx := context.Background()

	// 1) Get service
	var service v1.Service
	if err := r.client.Get(ctx, svc.NamespacedName(), &service); err != nil {
		return "", fmt.Errorf("Failed getting service: %s", err)
	}

	// 2) Extract IP
	switch svc.Type {
	case v1alpha1.ServiceIPTypeCluster:
		if service.Spec.ClusterIP == "" {
			return "", fmt.Errorf("Cluster IP not available")
		}
		return service.Spec.ClusterIP, nil
	case v1alpha1.ServiceIPTypeExternal:
		ingresses := service.Status.LoadBalancer.Ingress
		if len(ingresses) == 0 {
			return "", fmt.Errorf("Load balancer not available")
		}
		return ingresses[0].IP, nil
	default:
		return "", fmt.Errorf("Unknown service type '%s'", svc.Type)
	}
}

func (r *zoneRecordReconciler) getNodeIP(source v1alpha1.NodeIPSource) (string, error) {
	ctx := context.Background()

	// 1) Get all nodes matching the selectors
	selector := labels.NewSelector()
	if source.LabelSelectors != nil {
		for k, v := range source.LabelSelectors {
			requirement, err := labels.NewRequirement(k, selection.Equals, []string{v})
			if err != nil {
				return "", fmt.Errorf("Failed building label selector: %s", err)
			}
			selector.Add(*requirement)
		}
	}

	var nodes v1.NodeList
	if err := r.client.List(ctx, &nodes); err != nil {
		return "", fmt.Errorf("Failed listing nodes: %s", err)
	}

	// 2) Sort node IPs and choose the smallest one
	if len(nodes.Items) == 0 {
		return "", fmt.Errorf("Unable to find any node with specified labels")
	}

	ips := make([]string, 0)
	for _, item := range nodes.Items {
		for _, address := range item.Status.Addresses {
			if (source.Type == v1alpha1.NodeIPTypeExternal && address.Type == v1.NodeExternalIP) ||
				(source.Type == v1alpha1.NodeIPTypeInternal && address.Type == v1.NodeInternalIP) {
				ips = append(ips, address.Address)
				break
			}
		}
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("Unable to find any matching node specifying %s", source.Type)
	}

	sort.Strings(ips)

	return ips[0], nil
}

func (r *zoneRecordReconciler) listExpectedResources(
	record v1alpha1.DNSZoneRecord, zone backends.DNSZone, ip string,
) ([]v1alpha1.DNSResource, error) {
	expectedResources := make([]v1alpha1.DNSResource, 0)
	// 1) A records
	for _, host := range record.Spec.Hosts {
		resource, err := r.resource(record, zone, v1alpha1.DNSTypeA, host, ip)
		if err != nil {
			return nil, fmt.Errorf("Failed listing expected hosts: %s", err)
		}
		expectedResources = append(expectedResources, resource)
	}

	// 2) CNAME records
	primaryDomain := expectedResources[0].Spec.Domain
	for _, cname := range record.Spec.Cnames {
		resource, err := r.resource(record, zone, v1alpha1.DNSTypeCname, cname, primaryDomain)
		if err != nil {
			return nil, fmt.Errorf("Failed listing expected hosts: %s", err)
		}
		expectedResources = append(expectedResources, resource)
	}

	return expectedResources, nil
}

func (r *zoneRecordReconciler) listOwnedResources(
	record v1alpha1.DNSZoneRecord,
) ([]v1alpha1.DNSResource, error) {
	ctx := context.Background()

	var existingResources v1alpha1.DNSResourceList
	option := &client.ListOptions{
		Namespace:     record.Namespace,
		FieldSelector: fields.OneTermEqualSelector(indexFieldOwner, record.Name),
	}
	if err := r.client.List(ctx, &existingResources, option); err != nil {
		return nil, fmt.Errorf("Failed listing owned resources: %s", err)
	}

	return existingResources.Items, nil
}

func (r *zoneRecordReconciler) resource(
	record v1alpha1.DNSZoneRecord, zone backends.DNSZone,
	kind v1alpha1.DNSType, host string, data string,
) (v1alpha1.DNSResource, error) {
	// 1) Get key values
	host = utils.FullHost(host, zone)

	// 2) Generate resource
	resource := v1alpha1.DNSResource{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", record.Name),
			Namespace:    record.Namespace,
		},
		Spec: v1alpha1.DNSResourceSpec{
			ZoneName: record.Spec.ZoneName,
			Domain:   host,
			Type:     kind,
			Data:     data,
			TTL:      record.Spec.TTL,
		},
	}

	// 3) Set owner
	if err := ctrl.SetControllerReference(&record, &resource, r.scheme); err != nil {
		return v1alpha1.DNSResource{}, fmt.Errorf("Failed setting owner: %s", err)
	}

	return resource, nil
}
