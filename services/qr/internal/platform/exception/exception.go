package exception

import "errors"

type Exception struct {
	userMessage      string
	technicalMessage string
	params           map[string]any
	cause            error
	code             Code
	kind             Kind
}

func (e *Exception) Error() string {
	if e.cause == nil {
		return e.technicalMessage
	}
	return e.technicalMessage + ": " + e.cause.Error()
}

func Wrap(err error, technicalMessage string) error {
	if err == nil {
		return nil
	}
	if _, ok := From(err); ok {
		return err
	}
	return New(CodeInternal).
		WithTechnical(technicalMessage).
		WithCause(err)
}

func (e *Exception) Unwrap() error {
	return e.cause
}

func (e *Exception) UserMessage() string      { return e.userMessage }
func (e *Exception) TechnicalMessage() string { return e.technicalMessage }
func (e *Exception) Kind() Kind               { return e.kind }
func (e *Exception) Code() Code               { return e.code }
func (e *Exception) Params() map[string]any   { return cloneParams(e.params) }

func From(err error) (*Exception, bool) {
	var e *Exception
	ok := errors.As(err, &e)
	return e, ok
}

func HasCode(err error, code Code) bool {
	if e, ok := From(err); ok {
		return e.code == code
	}
	return false
}
