package notification

import "rutadeorigen/notifications/internal/platform/helper"

type Kind string

const (
	KindUnknown           Kind = "unknown"
	KindEvidenceProcessed Kind = "evidence_processed"
	KindEvidenceRejected  Kind = "evidence_rejected"
)

func ParseKind(value string) Kind {
	switch kind := Kind(helper.ApplyLower(helper.ApplyTrim(value))); kind {
	case KindEvidenceProcessed, KindEvidenceRejected:
		return kind
	default:
		return KindUnknown
	}
}

func (k Kind) IsValid() bool {
	return ParseKind(string(k)) != KindUnknown
}

func (k Kind) String() string {
	return string(k)
}
