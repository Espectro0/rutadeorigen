package helper

func SliceOrEmpty[T any](value []T) []T {
	if value == nil {
		return []T{}
	}
	return value
}

func MapOrEmpty[K comparable, V any](value map[K]V) map[K]V {
	if value == nil {
		return map[K]V{}
	}
	return value
}
