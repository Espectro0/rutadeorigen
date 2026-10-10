package helper

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	DateLayout     = time.DateOnly
	DateTimeLayout = time.RFC3339
)

func ParseUUID(value string) uuid.UUID {
	parsed, err := uuid.Parse(ApplyTrim(value))
	if err != nil {
		return defaultUUID
	}
	return parsed
}

func ParseInt(value string) int {
	parsed, err := strconv.Atoi(ApplyTrim(value))
	if err != nil {
		return defaultInt
	}
	return parsed
}

func ParseFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(ApplyTrim(value), 64)
	if err != nil {
		return defaultFloat
	}
	return parsed
}

func ParseDate(value string) time.Time {
	return parseTime(value, DateLayout)
}

func ParseDateTime(value string) time.Time {
	return parseTime(value, DateTimeLayout)
}

func parseTime(value, layout string) time.Time {
	parsed, err := time.Parse(layout, ApplyTrim(value))
	if err != nil {
		return defaultDate
	}
	return parsed
}
