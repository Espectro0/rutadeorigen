package requestevidenceupload

import (
	"context"
	"time"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/access"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/evidence"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*RequestEvidenceUpload)(nil)

type Settings struct {
	MaxBytes  int64
	UploadTTL time.Duration
}

type RequestEvidenceUpload struct {
	actors    ports.ActorRepository
	lots      ports.LotRepository
	evidences ports.EvidenceRepository
	storage   ports.EvidenceStorage
	auth      ports.Authorizer
	publisher ports.EventPublisher
	settings  Settings
}

func New(actors ports.ActorRepository, lots ports.LotRepository, evidences ports.EvidenceRepository,
	storage ports.EvidenceStorage, auth ports.Authorizer, publisher ports.EventPublisher, settings Settings) *RequestEvidenceUpload {
	return &RequestEvidenceUpload{
		actors:    actors,
		lots:      lots,
		evidences: evidences,
		storage:   storage,
		auth:      auth,
		publisher: publisher,
		settings:  settings,
	}
}

func (uc *RequestEvidenceUpload) Execute(ctx context.Context, cmd Command) (Result, error) {
	failed := Result{Evidence: evidence.Default()}

	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return failed, err
	}

	l, err := validators.LotExists(ctx, uc.lots, cmd.LotID)
	if err != nil {
		return failed, err
	}
	if l.CurrentStage() == lot.StageNone {
		return failed, exception.New(exception.CodeEvidenceLotWithoutStage)
	}

	resource := ports.Resource{Type: ports.ResourceLot, ID: l.ID(), CustodianID: l.CustodianID(), Stage: l.CurrentStage()}
	if err := access.Require(ctx, uc.auth, uc.publisher, a, ports.ActionUploadEvidence, resource); err != nil {
		return failed, err
	}
	if !l.IsActive() {
		return failed, exception.New(exception.CodeLotVoided)
	}

	e, err := evidence.New(evidence.NewParams{
		LotID:       l.ID(),
		Stage:       l.CurrentStage(),
		UploadedBy:  a.ID(),
		FileName:    cmd.FileName,
		ContentType: cmd.ContentType,
		SizeBytes:   cmd.SizeBytes,
		MaxBytes:    uc.settings.MaxBytes,
	})
	if err != nil {
		return failed, err
	}

	upload, err := uc.storage.PresignUpload(ctx, e.QuarantineKey(), e.ContentType().String(), uc.settings.UploadTTL)
	if err != nil {
		return failed, exception.Wrap(err, "presign evidence upload")
	}

	if err := uc.evidences.Save(ctx, e); err != nil {
		return failed, exception.Wrap(err, "save evidence")
	}

	return Result{Evidence: e, Upload: upload}, nil
}
