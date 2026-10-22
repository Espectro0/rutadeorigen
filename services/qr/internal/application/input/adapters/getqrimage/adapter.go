package getqrimage

import (
	"context"

	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	getqrimageuc "rutadeorigen/qr/internal/application/usecases/getqrimage"
)

var _ ports.GetQRImagePort = (*Adapter)(nil)

type Adapter struct {
	useCase   getqrimageuc.UseCase
	assembler Assembler
}

func New(useCase getqrimageuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, assembler: Assembler{}}
}

func (a *Adapter) Execute(ctx context.Context, input request.GetQRImage) (response.QRImage, error) {
	query, err := a.assembler.ToQuery(input)
	if err != nil {
		return response.QRImage{}, err
	}

	image, err := a.useCase.Execute(ctx, query)
	if err != nil {
		return response.QRImage{}, err
	}

	return a.assembler.ToDTO(image), nil
}
