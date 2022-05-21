package ext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	values := []int{1, 2, 3}
	result := Map(values, func(v int) float32 { return float32(v) * 2.5 })
	assert.ElementsMatch(t, result, []float32{2.5, 5.0, 7.5})
}
