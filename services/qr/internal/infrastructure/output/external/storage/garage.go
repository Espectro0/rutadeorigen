package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ ports.EvidenceViewer = (*Garage)(nil)

type GarageConfig struct {
	Endpoint  string
	UseSSL    bool
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

func (g *Garage) PresignView(ctx context.Context, key string, ttl time.Duration) (ports.ViewURL, error) {
	u, err := g.signer.PresignedGetObject(ctx, g.bucket, key, ttl, url.Values{})
	if err != nil {
		return ports.ViewURL{}, unavailable("presign evidence view", err)
	}

	return ports.ViewURL{URL: u.String(), ExpiresAt: time.Now().UTC().Add(ttl)}, nil
}

func (g *Garage) Ping(ctx context.Context) error {
	exists, err := g.internal.BucketExists(ctx, g.bucket)
	if err != nil {
		return unavailable("check evidence bucket", err)
	}
	if !exists {
		return exception.New(exception.CodeInternal).WithTechnical("evidence bucket does not exist")
	}
	return nil
}

func unavailable(technical string, cause error) error {
	return exception.New(exception.CodeInternal).
		WithTechnical(technical).
		WithCause(cause)
}
