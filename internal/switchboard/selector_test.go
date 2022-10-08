package switchboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchesNoIngressClass(t *testing.T) {
	selector := NewSelector(nil)
	assert.True(t, selector.Matches(map[string]string{}))
	assert.True(t, selector.Matches(map[string]string{
		"kubernetes.io/ingress.class": "test",
	}))
	assert.False(t, selector.Matches(map[string]string{
		"switchboard.borchero.com/ignore": "true",
	}))
	assert.False(t, selector.Matches(map[string]string{
		"kubernetes.io/ingress.class":     "test",
		"switchboard.borchero.com/ignore": "true",
	}))
	assert.False(t, selector.Matches(map[string]string{
		"switchboard.borchero.com/ignore": "all",
	}))
	assert.False(t, selector.Matches(map[string]string{
		"kubernetes.io/ingress.class":     "test",
		"switchboard.borchero.com/ignore": "all",
	}))
}

func TestMatchesIngressClass(t *testing.T) {
	cls := "ingress"
	selector := NewSelector(&cls)
	assert.False(t, selector.Matches(map[string]string{}))
	assert.False(t, selector.Matches(map[string]string{
		"kubernetes.io/ingress.class": "test",
	}))
	assert.True(t, selector.Matches(map[string]string{
		"kubernetes.io/ingress.class": "ingress",
	}))
	assert.False(t, selector.Matches(map[string]string{
		"switchboard.borchero.com/ignore": "true",
	}))
	assert.False(t, selector.Matches(map[string]string{
		"kubernetes.io/ingress.class":     "test",
		"switchboard.borchero.com/ignore": "true",
	}))
	assert.False(t, selector.Matches(map[string]string{
		"kubernetes.io/ingress.class":     "ingress",
		"switchboard.borchero.com/ignore": "true",
	}))
	assert.False(t, selector.Matches(map[string]string{
		"switchboard.borchero.com/ignore": "all",
	}))
	assert.False(t, selector.Matches(map[string]string{
		"kubernetes.io/ingress.class":     "test",
		"switchboard.borchero.com/ignore": "all",
	}))
	assert.False(t, selector.Matches(map[string]string{
		"kubernetes.io/ingress.class":     "ingress",
		"switchboard.borchero.com/ignore": "all",
	}))
}

func TestMatchesIntegration(t *testing.T) {
	cls := "ingress"
	selector := NewSelector(&cls)

	// Ignore all
	assert.False(t, selector.MatchesIntegration(map[string]string{
		"switchboard.borchero.com/ignore": "true",
	}, "external-dns"))
	assert.False(t, selector.MatchesIntegration(map[string]string{
		"switchboard.borchero.com/ignore": "all",
	}, "external-dns"))

	// Ignore only one
	assert.False(t, selector.MatchesIntegration(map[string]string{
		"switchboard.borchero.com/ignore": "external-dns",
	}, "external-dns"))
	assert.True(t, selector.MatchesIntegration(map[string]string{
		"switchboard.borchero.com/ignore": "cert-manager",
	}, "external-dns"))

	// Ignore multiple
	assert.False(t, selector.MatchesIntegration(map[string]string{
		"switchboard.borchero.com/ignore": "external-dns,cert-manager",
	}, "external-dns"))
	assert.False(t, selector.MatchesIntegration(map[string]string{
		"switchboard.borchero.com/ignore": "external-dns,cert-manager",
	}, "cert-manager"))
	assert.True(t, selector.MatchesIntegration(map[string]string{
		"switchboard.borchero.com/ignore": "external-dns,cert-manager",
	}, "unknown"))

	// Ignore with space in between
	assert.False(t, selector.MatchesIntegration(map[string]string{
		"switchboard.borchero.com/ignore": "external-dns, cert-manager",
	}, "external-dns"))
	assert.False(t, selector.MatchesIntegration(map[string]string{
		"switchboard.borchero.com/ignore": "external-dns, cert-manager",
	}, "cert-manager"))
	assert.True(t, selector.MatchesIntegration(map[string]string{
		"switchboard.borchero.com/ignore": "external-dns, cert-manager",
	}, "unknown"))
}
