package response

type Evidence struct {
	ID              string `json:"id"`
	LotID           string `json:"lot_id"`
	Stage           string `json:"stage"`
	UploadedBy      string `json:"uploaded_by"`
	FileName        string `json:"file_name"`
	ContentType     string `json:"content_type"`
	SizeBytes       int64  `json:"size_bytes"`
	Status          string `json:"status"`
	SHA256          string `json:"sha256,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	CreatedAt       string `json:"created_at"`
	ProcessedAt     string `json:"processed_at,omitempty"`
	ThumbnailURL    string `json:"thumbnail_url,omitempty"`
	ImageURL        string `json:"image_url,omitempty"`
	URLExpiresAt    string `json:"url_expires_at,omitempty"`
}

type EvidenceUploadTarget struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt string            `json:"expires_at"`
}

type EvidenceUpload struct {
	Evidence Evidence             `json:"evidence"`
	Upload   EvidenceUploadTarget `json:"upload"`
}

type EvidenceList struct {
	Items []Evidence `json:"items"`
}
