package switchboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	traefiktypes "github.com/traefik/traefik/v2/pkg/types"
	"sigs.k8s.io/external-dns/endpoint"
)

func TestNewHostAggregator(t *testing.T) {
	hosts := NewHostAggregator()
	assert.Equal(t, hosts.Len(), 0)
}

func TestParseTLSHosts(t *testing.T) {
	hosts := NewHostAggregator()
	hosts.ParseTLSHosts(nil)
	assert.Equal(t, hosts.Len(), 0)

	hosts.ParseTLSHosts(&traefik.TLS{
		Domains: []traefiktypes.Domain{{
			Main: "example.com",
			SANs: []string{"www.example.com"},
		}},
	})
	assert.ElementsMatch(t, hosts.Hosts(), []string{"example.com", "www.example.com"})
}

func TestParseRouteHosts(t *testing.T) {
	hosts := NewHostAggregator()
	hosts.ParseRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`example.com`)",
	}})
	assert.ElementsMatch(t, hosts.Hosts(), []string{"example.com"})

	hosts = NewHostAggregator()
	hosts.ParseRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`example.com`, `www.example.com`)",
	}})
	assert.ElementsMatch(t, hosts.Hosts(), []string{"example.com", "www.example.com"})

	hosts = NewHostAggregator()
	hosts.ParseRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`example.com`, `www.example.com`)",
	}, {
		Kind:  "Rule",
		Match: "Host(`v2.example.com`, `www.example.com`) && Prefix(`/test`)",
	}})
	assert.ElementsMatch(
		t, hosts.Hosts(), []string{"example.com", "www.example.com", "v2.example.com"},
	)
}

func TestParseRouteHostsNoop(t *testing.T) {
	hosts := NewHostAggregator()
	hosts.hosts = map[string]struct{}{"example.com": {}}
	hosts.ParseRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`www.example.com`)",
	}})
	assert.ElementsMatch(t, hosts.Hosts(), []string{"example.com"})
}

func TestDNSEndpoints(t *testing.T) {
	hosts := NewHostAggregator()
	hosts.hosts = map[string]struct{}{"example.com": {}, "www.example.com": {}}
	endpoints := hosts.DNSEndpoints("127.0.0.1", 250)
	assert.Len(t, endpoints, 2)
	for _, ep := range endpoints {
		assert.ElementsMatch(t, ep.Targets, []string{"127.0.0.1"})
		assert.Equal(t, ep.RecordTTL, endpoint.TTL(250))
		_, ok := hosts.hosts[ep.DNSName]
		assert.True(t, ok)
	}
}
