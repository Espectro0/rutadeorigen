package recordevidenceresult

import (
	"context"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*RecordEvidenceResult)(nil)

type RecordEvidenceResult struct {
	evidences ports.EvidenceRepository
}

func New(evidences ports.EvidenceRepository) *RecordEvidenceResult {
	return &RecordEvidenceResult{evidences: evidences}
}

// Execute is idempotent: an evidence that already has a result is left as it
// is, so a redelivered event does nothing.
func (uc *RecordEvidenceResult) Execute(ctx context.Context, cmd Command) error {
	e, err := uc.evidences.FindByID(ctx, cmd.EvidenceID)
	if err != nil {
		return exception.Wrap(err, "find evidence by id")
	}
	if e.IsDefault() {
		return exception.New(exception.CodeEvidenceNotFound)
	}
	if e.Status().IsFinal() {
		return nil
	}

	switch cmd.Outcome {
	case OutcomeProcessed:
		err = e.MarkProcessed(cmd.SHA256, cmd.FinalKey, cmd.ThumbnailKey, cmd.At)
	case OutcomeRejected:
		err = e.MarkRejected(cmd.Reason, cmd.At)
	default:
		err = exception.New(exception.CodeEvidenceInvalidResult)
	}
	if err != nil {
		return err
	}

	if err := uc.evidences.Update(ctx, e); err != nil {
		return exception.Wrap(err, "update evidence result")
	}
	return nil
}
