package exception

type Kind string

const (
	KindValidation   Kind = "validation"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindInternal     Kind = "internal"
	KindForbidden    Kind = "forbidden"
	KindUnauthorized Kind = "unauthorized"
)
