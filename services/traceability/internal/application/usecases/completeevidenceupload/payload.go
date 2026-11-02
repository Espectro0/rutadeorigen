package completeevidenceupload

import "github.com/google/uuid"

type Uploader struct {
	ActorID uuid.UUID `json:"actor_id"`
	Role    string    `json:"role"`
	Name    string    `json:"name"`
	Email   string    `json:"email"`
}

type EvidenceUploadedPayload struct {
	EvidenceID    uuid.UUID `json:"evidence_id"`
	LotID         uuid.UUID `json:"lot_id"`
	LotCode       string    `json:"lot_code"`
	Stage         string    `json:"stage"`
	FileName      string    `json:"file_name"`
	ContentType   string    `json:"content_type"`
	SizeBytes     int64     `json:"size_bytes"`
	QuarantineKey string    `json:"quarantine_key"`
	Uploader      Uploader  `json:"uploader"`
}
