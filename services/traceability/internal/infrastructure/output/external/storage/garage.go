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

var _ ports.EvidenceStorage = (*Garage)(nil)

type GarageConfig struct {
	// Endpoint is the address the service uses to reach Garage (host:port).
	Endpoint string
	UseSSL   bool
	// PublicURL is the address the browser uses (for example the Cloudflare
	// tunnel). Signatures bind the Host, so presigned URLs are signed for it.
	PublicURL string
	AccessKey string
	SecretKey string
	Region    string
	Bucket    string
}

type Garage struct {
	internal *minio.Client
	signer   *minio.Client
	bucket   string
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

	return &Garage{internal: internal, signer: signer, bucket: cfg.Bucket}, nil
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

// Ping checks that the quarantine bucket is reachable.
func (g *Garage) Ping(ctx context.Context) error {
	ok, err := g.internal.BucketExists(ctx, g.bucket)
	if err != nil {
		return unavailable("check quarantine bucket", err)
	}
	if !ok {
		return exception.New(exception.CodeEvidenceStorageUnavailable).
			WithTechnical("quarantine bucket does not exist")
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
