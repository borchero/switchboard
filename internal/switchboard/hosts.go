package switchboard

import (
	"fmt"
	"regexp"

	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
)

const (
	hostRegex = "`((?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9])`"
)

var (
	hostRuleRegex = regexp.MustCompile(
		fmt.Sprintf("(?:Host|HostSNI)\\(%s(?:, *%s)*\\)", hostRegex, hostRegex),
	)
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
func (a *HostCollection) WithRouteHostsIfRequired(routes []traefik.Route) *HostCollection {
	if len(a.hosts) > 0 {
		return a
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
	return a
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
