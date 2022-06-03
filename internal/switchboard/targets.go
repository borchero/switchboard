package switchboard

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Target is a type which allows to retrieve a potentially dynamically changing IP from Kubernetes.
type Target interface {
	// IPs returns the IPv4/IPv6 addresses that should be used as targets or an error if the IP
	// addresses cannot be retrieved.
	IPs(ctx context.Context, client client.Client) ([]string, error)
	// NamespacedName returns the namespaced name of the dynamic target service or none if the IP
	// is not retrieved dynamically.
	NamespacedName() *types.NamespacedName
}

//-------------------------------------------------------------------------------------------------
// SERVICE TARGET
//-------------------------------------------------------------------------------------------------

type serviceTarget struct {
	name types.NamespacedName
}

// NewServiceTarget creates a new target which dynamically sources the IP from the provided
// Kubernetes service.
func NewServiceTarget(name, namespace string) Target {
	return serviceTarget{
		name: types.NamespacedName{Name: name, Namespace: namespace},
	}
}

func (t serviceTarget) IPs(ctx context.Context, client client.Client) ([]string, error) {
	// Get service
	var service v1.Service
	if err := client.Get(ctx, t.name, &service); err != nil {
		return nil, fmt.Errorf("failed to query service: %w", err)
	}

	// Get IPs: try to get load balancer IPs, fall back to cluster IPs
	targets := make([]string, 0)
	for _, ingress := range service.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			targets = append(targets, ingress.IP)
		}
	}
	if len(targets) == 0 {
		targets = append(targets, service.Spec.ClusterIPs...)
	}
	return targets, nil
}

func (t serviceTarget) NamespacedName() *types.NamespacedName {
	return &t.name
}

//-------------------------------------------------------------------------------------------------
// STATIC TARGET
//-------------------------------------------------------------------------------------------------

type staticTarget struct {
	ips []string
}

// NewStaticTarget creates a new target which provides the given static IPs. IPs may be IPv4 or
// IPv6 addresses (and any combination thereof).
func NewStaticTarget(ips ...string) Target {
	return staticTarget{ips}
}

func (t staticTarget) IPs(ctx context.Context, client client.Client) ([]string, error) {
	return t.ips, nil
}

func (t staticTarget) NamespacedName() *types.NamespacedName {
	return nil
}
