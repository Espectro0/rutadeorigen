package helper

import (
	"strings"
	"unicode/utf8"
)

const EmptyString = ""

func DefaultString() string {
	return EmptyString
}

func TextOrDefault(value *string) string {
	return ValueOrDefault(value, EmptyString)
}

func TextIfEmpty(value, defaultValue string) string {
	if IsEmpty(value) {
		return defaultValue
	}
	return value
}

func ApplyTrim(value string) string {
	return strings.TrimSpace(value)
}

func IsEmpty(value string) bool {
	return ApplyTrim(value) == EmptyString
}

func LengthIsValid(value string, minLength, maxLength int) bool {
	length := utf8.RuneCountInString(ApplyTrim(value))
	return length >= minLength && length <= maxLength
}
