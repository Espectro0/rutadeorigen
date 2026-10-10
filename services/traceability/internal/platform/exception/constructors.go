package exception

import "rutadeorigen/traceability/internal/platform/helper"

const defaultUserMessage = "An unexpected error occurred, please try again later. If this error persists, please contact support."

func New(kind Kind, layer Layer, userMessage, technicalMessage string, cause error) *Exception {
	userMessage = helper.TextIfEmpty(userMessage, defaultUserMessage)

	return &Exception{
		userMessage:      helper.ApplyTrim(userMessage),
		technicalMessage: helper.ApplyTrim(helper.TextIfEmpty(technicalMessage, userMessage)),
		cause:            cause,
		layer:            layer,
		kind:             kind,
	}
}

func NewValidation(layer Layer, userMessage, technicalMessage string) *Exception {
	return New(KindValidation, layer, userMessage, technicalMessage, nil)
}

func NewNotFound(layer Layer, userMessage, technicalMessage string) *Exception {
	return New(KindNotFound, layer, userMessage, technicalMessage, nil)
}

func NewConflict(layer Layer, userMessage, technicalMessage string) *Exception {
	return New(KindConflict, layer, userMessage, technicalMessage, nil)
}

func NewInternal(layer Layer, userMessage, technicalMessage string, cause error) *Exception {
	return New(KindInternal, layer, userMessage, technicalMessage, cause)
}

func NewForbidden(layer Layer, userMessage, technicalMessage string) *Exception {
	return New(KindForbidden, layer, userMessage, technicalMessage, nil)
}
