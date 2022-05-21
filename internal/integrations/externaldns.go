package integrations

import (
	"context"
	"fmt"

	"github.com/borchero/switchboard/internal/k8s"
	"github.com/borchero/switchboard/internal/switchboard"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/external-dns/endpoint"
)

type externalDNS struct {
	client client.Client
	target switchboard.Target
	ttl    endpoint.TTL
}

// NewExternalDNS initializes a new external-dns integration whose created DNS endpoints target the
// provided service.
func NewExternalDNS(client client.Client, target switchboard.Target) Integration {
	return &externalDNS{client, target, 300}
}

func (*externalDNS) Name() string {
	return "external-dns"
}

func (*externalDNS) OwnedResource() client.Object {
	return &endpoint.DNSEndpoint{}
}

func (e *externalDNS) WatchedObject() client.Object {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.target.NamespacedName().Name,
			Namespace: e.target.NamespacedName().Namespace,
		},
	}
}

func (e *externalDNS) UpdateResource(
	ctx context.Context, owner metav1.Object, info IngressInfo,
) error {
	// If the ingress specifies no hosts, there should be no endpoint. We try deleting it and
	// ignore any error if it was not found.
	if len(info.Hosts) == 0 {
		dnsEndpoint := endpoint.DNSEndpoint{ObjectMeta: e.objectMeta(owner)}
		if err := k8s.DeleteIfFound(ctx, e.client, &dnsEndpoint); err != nil {
			return fmt.Errorf("failed to delete DNS endpoint: %w", err)
		}
		return nil
	}

	// Get the IP of the target service
	ip, err := e.target.IP(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to query IP for DNS A record: %w", err)
	}

	// Create the endpoint resource
	resource := endpoint.DNSEndpoint{ObjectMeta: e.objectMeta(owner)}
	if _, err := controllerutil.CreateOrPatch(ctx, e.client, &resource, func() error {
		// Meta
		if err := reconcileMetadata(owner, &resource, e.client.Scheme()); err != nil {
			return nil
		}
		// Spec
		resource.Spec.Endpoints = e.endpoints(info.Hosts, ip)
		return nil
	}); err != nil {
		return fmt.Errorf("failed to upsert DNS endpoint: %w", err)
	}
	return nil
}

//-------------------------------------------------------------------------------------------------
// UTILS
//-------------------------------------------------------------------------------------------------

func (e *externalDNS) objectMeta(owner metav1.Object) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      owner.GetName(),
		Namespace: owner.GetNamespace(),
	}
}

func (e *externalDNS) endpoints(hosts []string, target string) []*endpoint.Endpoint {
	endpoints := make([]*endpoint.Endpoint, 0, len(hosts))
	for _, host := range hosts {
		endpoints = append(endpoints, &endpoint.Endpoint{
			DNSName:    host,
			Targets:    []string{target},
			RecordType: "A",
			RecordTTL:  e.ttl,
		})
	}
	return endpoints
}
