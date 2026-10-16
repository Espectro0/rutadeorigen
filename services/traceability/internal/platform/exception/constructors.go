package exception

import (
	"errors"
	"maps"

	"rutadeorigen/traceability/internal/platform/helper"
)

func New(code Code) *Exception {
	kind, ok := KindOf(code)
	if !ok {
		return &Exception{
			code:             CodeInternal,
			kind:             KindInternal,
			technicalMessage: "unregistered error code: " + string(code),
		}
	}

	return &Exception{
		code:             code,
		kind:             kind,
		technicalMessage: string(code),
	}
}

func (e *Exception) WithParams(params map[string]any) *Exception {
	cp := *e
	cp.params = cloneParams(params)
	return &cp
}

func cloneParams(params map[string]any) map[string]any {
	if len(params) == 0 {
		return map[string]any{}
	}
	return maps.Clone(params)
}

func (e *Exception) Is(target error) bool {
	var t *Exception
	ok := errors.As(target, &t)
	return ok && t.code != "" && t.code == e.code
}

func (e *Exception) WithTechnical(message string) *Exception {
	cp := *e
	cp.technicalMessage = helper.ApplyTrim(message)
	return &cp
}

func (e *Exception) WithCause(cause error) *Exception {
	cp := *e
	cp.cause = cause
	return &cp
}
