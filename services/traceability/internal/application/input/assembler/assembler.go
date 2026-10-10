package assembler

type DTOAssembler[D any, Req any, Res any] interface {
	ToDomain(req Req) (D, error)
	ToDTO(domain D) Res
}
