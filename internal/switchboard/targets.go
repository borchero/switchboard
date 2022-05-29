package switchboard

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Target represents a service whose external/internal IP should be used as target for DNS records.
type Target struct {
	name types.NamespacedName
}

// NewTarget creates a new target from the service with the specified name in the given namespace.
func NewTarget(name, namespace string) Target {
	return Target{
		name: types.NamespacedName{Name: name, Namespace: namespace},
	}
}

// IPs returns the IP v4/v6 addresses that should be used as targets or an error if querying the
// IP addresses fails.
func (t Target) IPs(ctx context.Context, client client.Client) ([]string, error) {
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

// NamespacedName returns the namespaced name of the target service.
func (t Target) NamespacedName() types.NamespacedName {
	return t.name
}
