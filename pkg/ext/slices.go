package ext

// Map is a functional operator to map all values of a slice to a different type.
func Map[T any, V any](values []T, f func(T) V) []V {
	result := make([]V, 0, len(values))
	for _, val := range values {
		result = append(result, f(val))
	}
	return result
}
