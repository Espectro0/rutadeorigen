package helper

func IsNil[T any](value *T) bool {
	return value == nil
}

func ValueOrDefault[T any](value *T, defaultValue T) T {
	if IsNil(value) {
		return defaultValue
	}
	return *value
}
