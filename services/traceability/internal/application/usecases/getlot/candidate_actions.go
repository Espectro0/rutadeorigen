package getlot

import (
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/lot"
)

type candidate struct {
	action ports.Action
	stage  lot.Stage
}

func candidateActions(l *lot.Lot) []candidate {
	if !l.IsActive() {
		return nil
	}

	current := l.CurrentStage()
	next := current.Next()
	hasNext := next != lot.StageUnknown

	var candidates []candidate
	if hasNext {
		candidates = append(candidates, candidate{action: ports.ActionRegisterStage, stage: next})
	}
	if hasNext && l.HasEvents() {
		candidates = append(candidates, candidate{action: ports.ActionTransferLot, stage: current})
	}
	if current != lot.StageNone {
		candidates = append(candidates, candidate{action: ports.ActionUploadEvidence, stage: current})
	}
	candidates = append(candidates, candidate{action: ports.ActionVoidLot, stage: current})
	if current == lot.StagePackaging {
		candidates = append(candidates, candidate{action: ports.ActionGenerateQR, stage: current})
	}
	return candidates
}
