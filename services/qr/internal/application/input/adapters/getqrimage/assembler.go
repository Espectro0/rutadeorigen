package getqrimage

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	getqrimageuc "rutadeorigen/qr/internal/application/usecases/getqrimage"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

type Assembler struct{}

func (Assembler) ToQuery(req request.GetQRImage) (getqrimageuc.Query, error) {
	lotID := helper.ParseUUID(req.LotID)
	if helper.IsDefaultUUID(lotID) {
		return getqrimageuc.Query{}, exception.New(exception.CodeLotViewNotFound)
	}

	size := qrcode.DefaultSize
	if !helper.IsEmpty(req.Size) {
		size = qrcode.ParseSize(helper.ParseInt(req.Size))
	}
	if !size.IsValid() {
		return getqrimageuc.Query{}, exception.New(exception.CodeQRCodeInvalidSize)
	}

	format := qrcode.DefaultFormat
	if !helper.IsEmpty(req.Format) {
		format = qrcode.ParseFormat(req.Format)
	}
	if !format.IsValid() {
		return getqrimageuc.Query{}, exception.New(exception.CodeQRCodeInvalidFormat)
	}

	return getqrimageuc.Query{LotID: lotID, Size: size, Format: format}, nil
}

func (Assembler) ToDTO(image *qrcode.Image) response.QRImage {
	if image.IsDefault() {
		return response.QRImage{}
	}

	return response.QRImage{
		Data:        image.Data(),
		ContentType: image.ContentType(),
		Filename:    image.Filename(),
	}
}
