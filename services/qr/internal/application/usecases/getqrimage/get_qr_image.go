package getqrimage

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/application/usecases/validators"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ UseCase = (*GetQRImage)(nil)

type GetQRImage struct {
	views         ports.LotViewRepository
	cache         ports.LotViewCache
	codes         ports.QRCodeRepository
	renderer      ports.QRRenderer
	auth          ports.Authorizer
	publicBaseURL string
}

func New(views ports.LotViewRepository, cache ports.LotViewCache, codes ports.QRCodeRepository, renderer ports.QRRenderer, auth ports.Authorizer, publicBaseURL string) *GetQRImage {
	return &GetQRImage{
		views:         views,
		cache:         cache,
		codes:         codes,
		renderer:      renderer,
		auth:          auth,
		publicBaseURL: publicBaseURL,
	}
}

func (uc *GetQRImage) Execute(ctx context.Context, query Query) (*qrcode.Image, error) {
	actorID := requestcontext.ActorID(ctx)
	if helper.IsDefaultUUID(actorID) {
		return qrcode.DefaultImage(), exception.New(exception.CodeUnauthorized)
	}
	if !query.Size.IsValid() {
		return qrcode.DefaultImage(), exception.New(exception.CodeQRCodeInvalidSize)
	}
	if !query.Format.IsValid() {
		return qrcode.DefaultImage(), exception.New(exception.CodeQRCodeInvalidFormat)
	}

	view, err := validators.LotViewExists(ctx, uc.cache, uc.views, query.LotID)
	if err != nil {
		return qrcode.DefaultImage(), err
	}

	if err := uc.authorize(ctx, actorID, view); err != nil {
		return qrcode.DefaultImage(), err
	}
	if err := view.CanGenerateImage(); err != nil {
		return qrcode.DefaultImage(), err
	}

	code, err := uc.codeFor(ctx, view)
	if err != nil {
		return qrcode.DefaultImage(), err
	}

	data, err := uc.renderer.Render(ctx, ports.RenderRequest{
		Content:         code.Content(),
		ErrorCorrection: code.ErrorCorrection(),
		Size:            query.Size,
		Format:          query.Format,
	})
	if err != nil {
		return qrcode.DefaultImage(), exception.Wrap(err, "render qr image")
	}

	image, err := qrcode.NewImage(data, query.Format, view.Code())
	if err != nil {
		return qrcode.DefaultImage(), internalError(err, "build qr image")
	}
	return image, nil
}

func (uc *GetQRImage) authorize(ctx context.Context, actorID uuid.UUID, view *lotview.LotView) error {
	allowed, err := uc.auth.IsAllowed(ctx,
		ports.Subject{ActorID: actorID},
		ports.ActionReadQRImage,
		ports.Resource{Type: ports.ResourceLot, ID: view.LotID(), CustodianID: view.CustodianID()})
	if err != nil {
		return exception.Wrap(err, "evaluate read qr image policy")
	}
	if !allowed {
		return exception.New(exception.CodeForbidden)
	}
	return nil
}

func (uc *GetQRImage) codeFor(ctx context.Context, view *lotview.LotView) (*qrcode.QRCode, error) {
	existing, err := uc.codes.FindByLotID(ctx, view.LotID())
	if err != nil {
		return qrcode.Default(), exception.Wrap(err, "find qr code")
	}
	if !existing.IsDefault() {
		return existing, nil
	}

	created, err := qrcode.New(view.LotID(), uc.publicBaseURL, qrcode.DefaultErrorCorrection, uc.renderer.LibraryVersion())
	if err != nil {
		return qrcode.Default(), internalError(err, "build qr code")
	}
	if err := uc.codes.Save(ctx, created); err != nil {
		return qrcode.Default(), exception.Wrap(err, "save qr code")
	}
	return created, nil
}

func internalError(err error, message string) error {
	return exception.New(exception.CodeInternal).
		WithTechnical(message).
		WithCause(err)
}
