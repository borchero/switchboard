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
}
