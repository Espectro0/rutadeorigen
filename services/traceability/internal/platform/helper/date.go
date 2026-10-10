package helper

import "time"

var defaultDate = time.Time{}

func DefaultDate() time.Time {
	return defaultDate
}

func DateOrDefault(value *time.Time) time.Time {
	return ValueOrDefault(value, defaultDate)
}

func IsDefaultDate(value time.Time) bool {
	return value.IsZero()
}
