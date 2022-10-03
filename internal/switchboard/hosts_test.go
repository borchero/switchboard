package switchboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	traefiktypes "github.com/traefik/traefik/v2/pkg/types"
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
	assert.ElementsMatch(t, hosts.Hosts(map[string]string{}).Names, []string{"example.com", "www.example.com"})
}

func TestParseRouteHosts(t *testing.T) {
	hosts := NewHostCollection().WithRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`example.com`)",
	}})
	assert.ElementsMatch(t, hosts.Hosts(map[string]string{}).Names, []string{"example.com"})

	hosts = NewHostCollection().WithRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`example.com`, `www.example.com`)",
	}})
	assert.ElementsMatch(t, hosts.Hosts(map[string]string{}).Names, []string{"example.com", "www.example.com"})

	hosts = NewHostCollection().WithRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`example.com`, `www.example.com`)",
	}, {
		Kind:  "Rule",
		Match: "Host(`v2.example.com`, `www.example.com`) && Prefix(`/test`)",
	}})
	assert.ElementsMatch(
		t, hosts.Hosts(map[string]string{}).Names, []string{"example.com", "www.example.com", "v2.example.com"})
}

func TestHostTargetAnnotation(t *testing.T) {
	hosts := NewHostCollection().WithRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`example.com`)",
	}})
	testService := "TestService"
	assert.Equal(t, *hosts.Hosts(map[string]string{
		"switchboard.borchero.com/target": testService,
	}).Target, testService)
	assert.ElementsMatch(t, hosts.Hosts(map[string]string{
		"switchboard.borchero.com/target": testService,
	}).Names, []string{"example.com"})
}

func TestParseRouteHostsNoop(t *testing.T) {
	hosts := NewHostCollection()
	hosts.hosts = map[string]struct{}{"example.com": {}}
	hosts.WithRouteHostsIfRequired([]traefik.Route{{
		Kind:  "Rule",
		Match: "Host(`www.example.com`)",
	}})
	assert.ElementsMatch(t, hosts.Hosts(map[string]string{}).Names, []string{"example.com"})
}
