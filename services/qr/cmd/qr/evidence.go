package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"rutadeorigen/lib/config"
	"rutadeorigen/qr/internal/application/usecases/listpublicevidences"
	evidencestorage "rutadeorigen/qr/internal/infrastructure/output/external/storage"
)

const (
	defaultEvidenceViewTTL     = 6 * time.Hour
	defaultEvidencePerLotLimit = 12

	evidenceProcessedBinding = "evidence.processed"
)

type evidenceConfig struct {
	viewTTL   time.Duration
	maxPerLot int
	garage    evidencestorage.GarageConfig
}

func (c evidenceConfig) settings() listpublicevidences.Settings {
	return listpublicevidences.Settings{ViewTTL: c.viewTTL, MaxPerLot: c.maxPerLot}
}

func defaultEvidenceSettings() listpublicevidences.Settings {
	return listpublicevidences.Settings{ViewTTL: defaultEvidenceViewTTL, MaxPerLot: defaultEvidencePerLotLimit}
}

func loadEvidence() (evidenceConfig, error) {
	var problems []string
	required := func(key string) string {
		value, err := config.Required(key)
		if err != nil {
			problems = append(problems, err.Error())
		}
		return value
	}

	viewTTL, err := config.Duration("EVIDENCE_VIEW_TTL", defaultEvidenceViewTTL)
	if err != nil {
		problems = append(problems, err.Error())
	}

	useSSL, err := config.Bool("GARAGE_USE_SSL", false)
	if err != nil {
		problems = append(problems, err.Error())
	}

	maxPerLot := defaultEvidencePerLotLimit
	if raw := config.String("EVIDENCE_MAX_PER_LOT", ""); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			problems = append(problems, fmt.Sprintf("EVIDENCE_MAX_PER_LOT: %q is not a positive number", raw))
		} else {
			maxPerLot = parsed
		}
	}

	cfg := evidenceConfig{
		viewTTL:   viewTTL,
		maxPerLot: maxPerLot,
		garage: evidencestorage.GarageConfig{
			Endpoint:  required("GARAGE_ENDPOINT"),
			UseSSL:    useSSL,
			PublicURL: required("GARAGE_PUBLIC_URL"),
			AccessKey: required("GARAGE_ACCESS_KEY"),
			SecretKey: required("GARAGE_SECRET_KEY"),
			Region:    config.String("GARAGE_REGION", "garage"),
			Bucket:    config.String("GARAGE_EVIDENCE_BUCKET", "evidences"),
		},
	}

	if len(problems) > 0 {
		return evidenceConfig{}, fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return cfg, nil
}
