package switchboard

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const HostnameKey = "external-dns.alpha.kubernetes.io/hostname"

// Target is a type which allows to retrieve a potentially dynamically changing IP from Kubernetes.
type Target interface {
	// Targets returns the IPv4/IPv6 addresses or hostnames that should be used as targets or an
	// error if the addresses/hostnames cannot be retrieved.
	Targets(ctx context.Context, client client.Client) ([]string, error)
	// NamespacedName returns the namespaced name of the dynamic target service or none if the IP
	// is not retrieved dynamically.
	NamespacedName() *types.NamespacedName
}

//-------------------------------------------------------------------------------------------------
// ANNOTATION HOSTNAME
//-------------------------------------------------------------------------------------------------

// SplitHostnameAnnotation splits a comma-separated hostname annotation string into a slice of hostnames.
// It trims any leading or trailing whitespace and removes any spaces within the anno
func SplitHostnameAnnotation(input string) []string {
	return strings.Split(strings.TrimSpace(strings.ReplaceAll(input, " ", "")), ",")
}

func HostnamesFromAnnotations(input map[string]string) []string {
	return extractHostnamesFromAnnotations(input, HostnameKey)
}

func extractHostnamesFromAnnotations(input map[string]string, key string) []string {
	annotation, ok := input[key]
	if !ok {
		return nil
	}
	return SplitHostnameAnnotation(annotation)
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

func (t serviceTarget) Targets(ctx context.Context, client client.Client) ([]string, error) {
	// Get service
	var service v1.Service
	if err := client.Get(ctx, t.name, &service); err != nil {
		return nil, fmt.Errorf("failed to query service: %w", err)
	}
	return t.targetsFromService(service), nil
}

func (serviceTarget) targetsFromService(service v1.Service) []string {
	targets := make([]string, 0)

	// Prioritize annotation hostnames.
	hostnameList := HostnamesFromAnnotations(service.Annotations)
	for _, hostname := range hostnameList {
		// We cannot have more than one CNAME record, the annotation overwrites everything
		targets = []string{hostname}
		break
	}

	// Try to get load balancer IPs/hostnames...
	if len(targets) == 0 {
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.Hostname != "" {
				// We cannot have more than one CNAME record, the hostname overwrites everything
				targets = []string{ingress.Hostname}
				break
			}
			if ingress.IP != "" {
				targets = append(targets, ingress.IP)
			}
		}
	}

	// ...fall back to cluster IPs
	if len(targets) == 0 {
		targets = append(targets, service.Spec.ClusterIPs...)
	}
	return targets
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

func (t staticTarget) Targets(ctx context.Context, client client.Client) ([]string, error) {
	return t.ips, nil
}

func (staticTarget) NamespacedName() *types.NamespacedName {
	return nil
}
