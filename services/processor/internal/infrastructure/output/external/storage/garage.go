package storage

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"rutadeorigen/processor/internal/application/output/ports"
	"rutadeorigen/processor/internal/platform/exception"
)

var (
	_ ports.QuarantineStorage = (*GarageQuarantine)(nil)
	_ ports.EvidenceStorage   = (*GarageEvidences)(nil)
)

type GarageConfig struct {
	Endpoint         string
	UseSSL           bool
	AccessKey        string
	SecretKey        string
	Region           string
	QuarantineBucket string
	EvidenceBucket   string
}

type Garage struct {
	Quarantine *GarageQuarantine
	Evidences  *GarageEvidences
}

type GarageQuarantine struct {
	client *minio.Client
	bucket string
}

type GarageEvidences struct {
	client *minio.Client
	bucket string
}

func NewGarage(cfg GarageConfig) (*Garage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.UseSSL,
		Region:       cfg.Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, unavailable("create garage client", err)
	}

	return &Garage{
		Quarantine: &GarageQuarantine{client: client, bucket: cfg.QuarantineBucket},
		Evidences:  &GarageEvidences{client: client, bucket: cfg.EvidenceBucket},
	}, nil
}

func (g *Garage) Ping(ctx context.Context) error {
	if err := pingBucket(ctx, g.Quarantine.client, g.Quarantine.bucket); err != nil {
		return err
	}
	return pingBucket(ctx, g.Evidences.client, g.Evidences.bucket)
}

func (q *GarageQuarantine) Get(ctx context.Context, key string, maxBytes int64) (ports.QuarantineObject, error) {
	object, err := q.client.GetObject(ctx, q.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return ports.QuarantineObject{}, unavailable("get quarantine object", err)
	}
	defer object.Close()

	data, err := io.ReadAll(io.LimitReader(object, maxBytes))
	if err != nil {
		if isNotFound(err) {
			return ports.QuarantineObject{}, nil
		}
		return ports.QuarantineObject{}, unavailable("read quarantine object", err)
	}
	return ports.QuarantineObject{Exists: true, Data: data}, nil
}

func (q *GarageQuarantine) Remove(ctx context.Context, key string) error {
	if err := q.client.RemoveObject(ctx, q.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		if isNotFound(err) {
			return nil
		}
		return unavailable("remove quarantine object", err)
	}
	return nil
}

func (e *GarageEvidences) Put(ctx context.Context, key, contentType string, data []byte) error {
	_, err := e.client.PutObject(ctx, e.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return unavailable("put evidence object", err)
	}
	return nil
}

func pingBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return unavailable("check bucket "+bucket, err)
	}
	if !exists {
		return exception.New(exception.CodeStorageUnavailable).
			WithTechnical("bucket does not exist: " + bucket)
	}
	return nil
}

func isNotFound(err error) bool {
	code := minio.ToErrorResponse(err).Code
	return code == "NoSuchKey" || code == "NotFound" || strings.EqualFold(code, "nosuchobject")
}

func unavailable(technical string, cause error) error {
	return exception.New(exception.CodeStorageUnavailable).
		WithTechnical(technical).
		WithCause(cause)
}
