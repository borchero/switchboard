package ext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAndThenNil(t *testing.T) {
	var test *int
	result := AndThen(test, func(v int) int { return v * 2 })
	assert.Nil(t, result)
}

func TestAndThenNotNil(t *testing.T) {
	test := 2
	result := AndThen(&test, func(v int) int { return v * 2 })
	expected := 4
	assert.Equal(t, &expected, result)
}
