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

// IP returns the IP that should be used as target or an error if querying the IP fails.
func (t Target) IP(ctx context.Context, client client.Client) (string, error) {
	// Get service
	var service v1.Service
	if err := client.Get(ctx, t.name, &service); err != nil {
		return "", fmt.Errorf("failed to query service: %w", err)
	}

	// Get IP: try to get load balancer IP, fall back to cluster IP
	targetIP := service.Spec.ClusterIP
	lbIngress := service.Status.LoadBalancer.Ingress
	if len(lbIngress) > 0 {
		targetIP = lbIngress[0].IP
	}
	return targetIP, nil
}

// NamespacedName returns the namespaced name of the target service.
func (t Target) NamespacedName() types.NamespacedName {
	return t.name
}
