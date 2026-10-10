package exception

import "errors"

type Exception struct {
	userMessage      string
	technicalMessage string
	cause            error
	layer            Layer
	kind             Kind
}

func (e *Exception) Error() string {
	if e.cause == nil {
		return e.technicalMessage
	}
	return e.technicalMessage + ": " + e.cause.Error()
}

func Wrap(err error, layer Layer, technicalMessage string) error {
	if err == nil {
		return nil
	}
	if _, ok := From(err); ok {
		return err
	}
	return NewInternal(layer, defaultUserMessage, technicalMessage, err)
}

func (e *Exception) Unwrap() error {
	return e.cause
}

func (e *Exception) UserMessage() string      { return e.userMessage }
func (e *Exception) TechnicalMessage() string { return e.technicalMessage }
func (e *Exception) Layer() Layer             { return e.layer }
func (e *Exception) Kind() Kind               { return e.kind }

func From(err error) (*Exception, bool) {
	var e *Exception
	ok := errors.As(err, &e)
	return e, ok
}
