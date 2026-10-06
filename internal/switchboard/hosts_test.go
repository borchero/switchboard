package switchboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	traefik "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	traefiktypes "github.com/traefik/traefik/v3/pkg/types"
)

func TestNewHostCollection(t *testing.T) {
	hosts := NewHostCollection()
	assert.Equal(t, hosts.Len(), 0)
}

func TestParseTLSHosts(t *testing.T) {
	hosts := NewHostCollection().WithTLSHostsIfAvailable(nil)
	assert.Equal(t, hosts.Len(), 0)

	hosts.WithTLSHostsIfAvailable(&traefik.TLS{
		Domains: []traefiktypes.Domain{{
			Main: "example.com",
			SANs: []string{"www.example.com"},
		}},
	})
	assert.ElementsMatch(t, hosts.Hosts(), []string{"example.com", "www.example.com"})
}

func TestParseRouteHosts(t *testing.T) {
	hosts, err := NewHostCollection().WithRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`example.com`)",
	}})
	assert.Nil(t, err)
	assert.ElementsMatch(t, hosts.Hosts(), []string{"example.com"})

	hosts, err = NewHostCollection().WithRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`example.com`, `www.example.com`)",
	}})
	assert.Nil(t, err)
	assert.ElementsMatch(t, hosts.Hosts(), []string{"example.com", "www.example.com"})

	hosts, err = NewHostCollection().WithRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`example.com`, `www.example.com`)",
	}, {
		Kind:  "Rule",
		Match: "Host(`v2.example.com`, `www.example.com`) && PathPrefix(`/test`)",
	}})
	assert.Nil(t, err)
	assert.ElementsMatch(
		t, hosts.Hosts(), []string{"example.com", "www.example.com", "v2.example.com"},
	)

	hosts, err = NewHostCollection().WithRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`service.namespace`, `service`)",
	}})
	assert.Nil(t, err)
	assert.ElementsMatch(t, hosts.Hosts(), []string{"service.namespace", "service"})
}

func TestParseRouteHostsNoop(t *testing.T) {
	hosts := NewHostCollection()
	hosts.hosts = map[string]struct{}{"example.com": {}}
	_, err := hosts.WithRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`www.example.com`)",
	}})
	assert.Nil(t, err)
	assert.ElementsMatch(t, hosts.Hosts(), []string{"example.com"})
}

func TestParseRouteHostsNegation(t *testing.T) {
	cases := []struct {
		name  string
		rule  string
		hosts []string
	}{
		{
			name:  "mixed",
			rule:  "Host(`allowed.example`) && !Host(`blocked.example`)",
			hosts: []string{"allowed.example"},
		},
		{
			name:  "negative only",
			rule:  "!Host(`blocked.example`)",
			hosts: []string{},
		},
		{
			name:  "double negative",
			rule:  "!!Host(`allowed.example`)",
			hosts: []string{"allowed.example"},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			hosts, err := NewHostCollection().WithRouteHostsIfRequired([]traefik.Route{{
				Kind:  "Rule",
				Match: test.rule,
			}})
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, test.hosts, hosts.Hosts())
			}
		})
	}
}
