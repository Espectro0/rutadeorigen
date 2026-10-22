package helper

import "github.com/google/uuid"

var defaultUUID = uuid.Nil

func DefaultUUID() uuid.UUID {
	return defaultUUID
}

func UUIDOrDefault(value *uuid.UUID) uuid.UUID {
	return ValueOrDefault(value, defaultUUID)
}

func IsDefaultUUID(value uuid.UUID) bool {
	return value == defaultUUID
}
