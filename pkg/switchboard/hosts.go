package switchboard

import (
	"fmt"
	"regexp"

	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	"sigs.k8s.io/external-dns/endpoint"
)

const (
	hostRegex = "`((?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9])`"
)

var (
	hostRuleRegex = regexp.MustCompile(
		fmt.Sprintf("(?:Host|HostSNI)\\(%s(?:, *%s)*\\)", hostRegex, hostRegex),
	)
)

// HostAggregator allows to aggregate the hosts from ingress resources.
type HostAggregator struct {
	hosts map[string]struct{}
}

// NewHostAggregator returns a new "empty" host aggregator.
func NewHostAggregator() *HostAggregator {
	return &HostAggregator{hosts: make(map[string]struct{})}
}

// ParseTLSHosts aggregates all hosts found in the provided TLS configuration. If the TLS
// configuration is empty (i.e. `nil`), no hosts are extracted. This method should only be called
// on a freshly initialized aggregator.
func (a *HostAggregator) ParseTLSHosts(config *traefik.TLS) {
	if config != nil {
		for _, domain := range config.Domains {
			a.hosts[domain.Main] = struct{}{}
			for _, san := range domain.SANs {
				a.hosts[san] = struct{}{}
			}
		}
	}
}

// ParseRouteHostsIfRequired aggregates all (unique) hosts found in the provided routes. If the
// aggregator already manages at least one host, this method is a noop, regardless of the routes
// passed as parameters.
func (a *HostAggregator) ParseRouteHostsIfRequired(routes []traefik.Route) {
	if len(a.hosts) > 0 {
		return
	}
	for _, route := range routes {
		if route.Kind == "Rule" {
			for _, matches := range hostRuleRegex.FindAllStringSubmatch(route.Match, -1) {
				for _, match := range matches[1:] {
					if match != "" {
						a.hosts[match] = struct{}{}
					}
				}
			}
		}
	}
}

// Len returns the number of hosts that the aggregator currently manages.
func (a *HostAggregator) Len() int {
	return len(a.hosts)
}

// Hosts returns all hosts managed by this aggregator.
func (a *HostAggregator) Hosts() []string {
	hosts := make([]string, 0, len(a.hosts))
	for host := range a.hosts {
		hosts = append(hosts, host)
	}
	return hosts
}

// DNSEndpoints returns a list of DNS endpoints with the DNS names set to the aggregator's managed
// hosts and the target set to the provided IP. Record ttl is set as passed.
func (a *HostAggregator) DNSEndpoints(target string, ttl endpoint.TTL) []*endpoint.Endpoint {
	endpoints := make([]*endpoint.Endpoint, 0, len(a.hosts))
	for host := range a.hosts {
		endpoints = append(endpoints, &endpoint.Endpoint{
			DNSName:    host,
			Targets:    []string{target},
			RecordType: "A",
			RecordTTL:  ttl,
		})
	}
	return endpoints
}
