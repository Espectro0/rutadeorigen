package assembler

type EntityAssembler[E any, D any] interface {
	ToDomain(entity E) D
	ToEntity(domain D) E
}
