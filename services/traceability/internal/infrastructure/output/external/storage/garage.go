package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/platform/exception"
)

var (
	_ ports.EvidenceStorage = (*Garage)(nil)
	_ ports.EvidenceViewer  = (*Garage)(nil)
)

type GarageConfig struct {
	Endpoint  string
	UseSSL    bool
	PublicURL string
	AccessKey string
	SecretKey string
	Region    string
	Bucket    string

	EvidenceBucket string
}

type Garage struct {
	internal *minio.Client
	signer   *minio.Client
	bucket   string

	evidenceBucket string
}

func NewGarage(cfg GarageConfig) (*Garage, error) {
	creds := credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")

	internal, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        creds,
		Secure:       cfg.UseSSL,
		Region:       cfg.Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, unavailable("create garage client", err)
	}

	public, err := url.Parse(cfg.PublicURL)
	if err != nil || public.Host == "" || (public.Scheme != "http" && public.Scheme != "https") {
		return nil, exception.New(exception.CodeInternal).
			WithTechnical(fmt.Sprintf("invalid garage public url %q", cfg.PublicURL))
	}

	signer, err := minio.New(public.Host, &minio.Options{
		Creds:        creds,
		Secure:       public.Scheme == "https",
		Region:       cfg.Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, unavailable("create garage signer", err)
	}

	return &Garage{internal: internal, signer: signer, bucket: cfg.Bucket, evidenceBucket: cfg.EvidenceBucket}, nil
}

func (g *Garage) PresignUpload(ctx context.Context, key, contentType string, ttl time.Duration) (ports.PresignedUpload, error) {
	headers := http.Header{}
	headers.Set("Content-Type", contentType)

	u, err := g.signer.PresignHeader(ctx, http.MethodPut, g.bucket, key, ttl, nil, headers)
	if err != nil {
		return ports.PresignedUpload{}, unavailable("presign evidence upload", err)
	}

	return ports.PresignedUpload{
		URL:       u.String(),
		Method:    http.MethodPut,
		Headers:   map[string]string{"Content-Type": contentType},
		ExpiresAt: time.Now().UTC().Add(ttl),
	}, nil
}

func (g *Garage) Stat(ctx context.Context, key string) (ports.ObjectInfo, error) {
	info, err := g.internal.StatObject(ctx, g.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNotFound(err) {
			return ports.ObjectInfo{}, nil
		}
		return ports.ObjectInfo{}, unavailable("stat quarantine object", err)
	}
	return ports.ObjectInfo{Exists: true, SizeBytes: info.Size}, nil
}

func (g *Garage) Remove(ctx context.Context, key string) error {
	if err := g.internal.RemoveObject(ctx, g.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		if isNotFound(err) {
			return nil
		}
		return unavailable("remove quarantine object", err)
	}
	return nil
}

func (g *Garage) PresignView(ctx context.Context, key string, ttl time.Duration) (ports.ViewURL, error) {
	u, err := g.signer.PresignedGetObject(ctx, g.evidenceBucket, key, ttl, url.Values{})
	if err != nil {
		return ports.ViewURL{}, unavailable("presign evidence view", err)
	}

	return ports.ViewURL{URL: u.String(), ExpiresAt: time.Now().UTC().Add(ttl)}, nil
}

func (g *Garage) Ping(ctx context.Context) error {
	if err := g.pingBucket(ctx, g.bucket); err != nil {
		return err
	}
	return g.pingBucket(ctx, g.evidenceBucket)
}

func (g *Garage) pingBucket(ctx context.Context, bucket string) error {
	ok, err := g.internal.BucketExists(ctx, bucket)
	if err != nil {
		return unavailable("check bucket "+bucket, err)
	}
	if !ok {
		return exception.New(exception.CodeEvidenceStorageUnavailable).
			WithTechnical("bucket does not exist: " + bucket)
	}
	return nil
}

func isNotFound(err error) bool {
	code := minio.ToErrorResponse(err).Code
	return code == "NoSuchKey" || code == "NotFound" || strings.EqualFold(code, "nosuchobject")
}

func unavailable(technical string, cause error) error {
	return exception.New(exception.CodeEvidenceStorageUnavailable).
		WithTechnical(technical).
		WithCause(cause)
}
