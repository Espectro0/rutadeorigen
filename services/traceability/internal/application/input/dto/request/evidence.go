package request

type RequestEvidenceUpload struct {
	LotID       string `json:"-"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type CompleteEvidenceUpload struct {
	LotID      string `json:"-"`
	EvidenceID string `json:"-"`
}

type ListEvidences struct {
	LotID string
}

// RecordEvidenceResult carries the outcome the processor published
// (evidence.processed or evidence.rejected).
type RecordEvidenceResult struct {
	Type         string
	EvidenceID   string
	SHA256       string
	FinalKey     string
	ThumbnailKey string
	Reason       string
	ProcessedAt  string
}
