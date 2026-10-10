package exception

type Layer string

const (
	LayerDomain         Layer = "domain"
	LayerApplication    Layer = "application"
	LayerInfrastructure Layer = "infrastructure"
)

type Kind string

const (
	KindValidation Kind = "validation"
	KindNotFound   Kind = "not_found"
	KindConflict   Kind = "conflict"
	KindInternal   Kind = "internal"
	KindForbidden  Kind = "forbidden"
)
