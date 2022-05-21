package ext

// AndThen applies the given function if the passed value is non-nil and returns `nil` otherwise.
func AndThen[T any, V any](value *T, f func(T) V) *V {
	if value == nil {
		return nil
	}
	result := f(*value)
	return &result
}
