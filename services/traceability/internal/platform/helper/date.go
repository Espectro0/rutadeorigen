package helper

import "time"

var defaultDate = time.Time{}

const day = 24 * time.Hour

func DateUTC(value time.Time) time.Time {
	return value.UTC().Truncate(day)
}

func DefaultDate() time.Time {
	return defaultDate
}

func IsFutureDate(value time.Time) bool {
	return DateUTC(value).After(DateUTC(time.Now()))
}

func DateOrDefault(value *time.Time) time.Time {
	return ValueOrDefault(value, defaultDate)
}

func IsDefaultDate(value time.Time) bool {
	return value.IsZero()
}
