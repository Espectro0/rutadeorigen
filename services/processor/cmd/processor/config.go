package main

import (
	"errors"
	"fmt"
	"strconv"

	"rutadeorigen/lib/config"
	"rutadeorigen/processor/internal/domain/picture"
	"rutadeorigen/processor/internal/infrastructure/output/external/storage"
)

const (
	defaultMaxFileBytes  = 5 << 20
	defaultMaxMegapixels = 40
	defaultThumbnailSide = 400

	evidenceUploadedQueue = "processor.events"
	oneImageAtATime       = 1
)

var consumedEvents = []string{"evidence.uploaded"}

func loadLimits() (picture.Limits, error) {
	var problems []error

	maxFileBytes := positiveNumber("EVIDENCE_MAX_BYTES", defaultMaxFileBytes, &problems)
	maxMegapixels := positiveNumber("IMAGE_MAX_MEGAPIXELS", defaultMaxMegapixels, &problems)
	thumbnailSide := positiveNumber("THUMBNAIL_SIDE", defaultThumbnailSide, &problems)
	if len(problems) > 0 {
		return picture.DefaultLimits(), errors.Join(problems...)
	}

	return picture.NewLimits(maxFileBytes, int(maxMegapixels), int(thumbnailSide))
}

func loadGarage() (storage.GarageConfig, error) {
	var problems []error
	required := func(key string) string {
		value, err := config.Required(key)
		if err != nil {
			problems = append(problems, err)
		}
		return value
	}

	useSSL, err := config.Bool("GARAGE_USE_SSL", false)
	if err != nil {
		problems = append(problems, err)
	}

	garage := storage.GarageConfig{
		Endpoint:         required("GARAGE_ENDPOINT"),
		UseSSL:           useSSL,
		AccessKey:        required("GARAGE_ACCESS_KEY"),
		SecretKey:        required("GARAGE_SECRET_KEY"),
		Region:           config.String("GARAGE_REGION", "garage"),
		QuarantineBucket: config.String("GARAGE_QUARANTINE_BUCKET", "quarantine"),
		EvidenceBucket:   config.String("GARAGE_EVIDENCE_BUCKET", "evidences"),
	}

	if len(problems) > 0 {
		return storage.GarageConfig{}, errors.Join(problems...)
	}
	return garage, nil
}

func positiveNumber(key string, fallback int64, problems *[]error) int64 {
	raw := config.String(key, "")
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		*problems = append(*problems, fmt.Errorf("%s: %q is not a positive number", key, raw))
		return fallback
	}
	return parsed
}
