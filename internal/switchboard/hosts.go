package switchboard

import (
	"fmt"

	muxer "github.com/traefik/traefik/v2/pkg/muxer/http"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
)

// HostCollection allows to aggregate the hosts from ingress resources.
type HostCollection struct {
	hosts map[string]struct{}
}

// NewHostCollection returns a new "empty" host collection.
func NewHostCollection() *HostCollection {
	return &HostCollection{hosts: make(map[string]struct{})}
}

// WithTLSHostsIfAvailable aggregates all hosts found in the provided TLS configuration. If the
// TLS configuration is empty (i.e. `nil`), no hosts are extracted. This method should only be
// called on a freshly initialized aggregator.
func (a *HostCollection) WithTLSHostsIfAvailable(config *traefik.TLS) *HostCollection {
	if config != nil {
		for _, domain := range config.Domains {
			a.hosts[domain.Main] = struct{}{}
			for _, san := range domain.SANs {
				a.hosts[san] = struct{}{}
			}
		}
	}
	return a
}

// WithRouteHostsIfRequired aggregates all (unique) hosts found in the provided routes. If the
// aggregator already manages at least one host, this method is a noop, regardless of the routes
// passed as parameters.
func (a *HostCollection) WithRouteHostsIfRequired(
	routes []traefik.Route,
) (*HostCollection, error) {
	if len(a.hosts) > 0 {
		return a, nil
	}
	for _, route := range routes {
		if route.Kind == "Rule" {
			hosts, err := muxer.ParseDomains(route.Match)
			if err != nil {
				return nil, fmt.Errorf("failed to parse domains: %s", err)
			}
			for _, host := range hosts {
				a.hosts[host] = struct{}{}
			}
		}
	}
	return a, nil
}

// Len returns the number of hosts that the aggregator currently manages.
func (a *HostCollection) Len() int {
	return len(a.hosts)
}

// Hosts returns all hosts managed by this aggregator.
func (a *HostCollection) Hosts() []string {
	hosts := make([]string, 0, len(a.hosts))
	for host := range a.hosts {
		hosts = append(hosts, host)
	}
	return hosts
}
