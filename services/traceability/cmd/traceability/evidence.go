package main

import (
	"fmt"
	"strconv"
	"time"

	"rutadeorigen/lib/config"
	"rutadeorigen/traceability/internal/application/usecases/requestevidenceupload"
	evidencestorage "rutadeorigen/traceability/internal/infrastructure/output/external/storage"
)

const (
	defaultEvidenceMaxBytes  = 5 << 20
	defaultEvidenceUploadTTL = 5 * time.Minute
	defaultEvidenceViewTTL   = time.Hour

	evidenceConsumerQueue = "traceability.events"
)

var evidenceResultEvents = []string{"evidence.processed", "evidence.rejected"}

type evidenceConfig struct {
	maxBytes  int64
	uploadTTL time.Duration
	viewTTL   time.Duration
	garage    evidencestorage.GarageConfig
}

func (c evidenceConfig) settings() requestevidenceupload.Settings {
	return requestevidenceupload.Settings{MaxBytes: c.maxBytes, UploadTTL: c.uploadTTL}
}

func loadEvidence() (evidenceConfig, error) {
	var errs []error
	required := func(key string) string {
		v, err := config.Required(key)
		if err != nil {
			errs = append(errs, err)
		}
		return v
	}

	maxBytes := int64(defaultEvidenceMaxBytes)
	if raw := config.String("EVIDENCE_MAX_BYTES", ""); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			errs = append(errs, fmt.Errorf("EVIDENCE_MAX_BYTES: %q is not a positive number", raw))
		} else {
			maxBytes = parsed
		}
	}

	ttl, err := config.Duration("EVIDENCE_UPLOAD_TTL", defaultEvidenceUploadTTL)
	if err != nil {
		errs = append(errs, err)
	}

	viewTTL, err := config.Duration("EVIDENCE_VIEW_TTL", defaultEvidenceViewTTL)
	if err != nil {
		errs = append(errs, err)
	}

	useSSL, err := config.Bool("GARAGE_USE_SSL", false)
	if err != nil {
		errs = append(errs, err)
	}

	cfg := evidenceConfig{
		maxBytes:  maxBytes,
		uploadTTL: ttl,
		viewTTL:   viewTTL,
		garage: evidencestorage.GarageConfig{
			Endpoint:  required("GARAGE_ENDPOINT"),
			UseSSL:    useSSL,
			PublicURL: required("GARAGE_PUBLIC_URL"),
			AccessKey: required("GARAGE_ACCESS_KEY"),
			SecretKey: required("GARAGE_SECRET_KEY"),
			Region:    config.String("GARAGE_REGION", "garage"),
			Bucket:    config.String("GARAGE_QUARANTINE_BUCKET", "quarantine"),

			EvidenceBucket: config.String("GARAGE_EVIDENCE_BUCKET", "evidences"),
		},
	}

	if len(errs) > 0 {
		return evidenceConfig{}, joinErrors(errs)
	}
	return cfg, nil
}

func joinErrors(errs []error) error {
	msg := ""
	for i, err := range errs {
		if i > 0 {
			msg += "; "
		}
		msg += err.Error()
	}
	return fmt.Errorf("%s", msg)
}
