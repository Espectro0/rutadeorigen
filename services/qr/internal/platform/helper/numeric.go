package helper

import "cmp"

const (
	defaultInt   = 0
	defaultFloat = 0.0
)

func DefaultInt() int {
	return defaultInt
}

func IntOrDefault(value *int) int {
	return ValueOrDefault(value, defaultInt)
}

func IsDefaultInt(value int) bool {
	return value == defaultInt
}

func DefaultFloat() float64 {
	return defaultFloat
}

func FloatOrDefault(value *float64) float64 {
	return ValueOrDefault(value, defaultFloat)
}

func IsDefaultFloat(value float64) bool {
	return value == defaultFloat
}

func IsInRange[T cmp.Ordered](value, minValue, maxValue T) bool {
	return value >= minValue && value <= maxValue
}
