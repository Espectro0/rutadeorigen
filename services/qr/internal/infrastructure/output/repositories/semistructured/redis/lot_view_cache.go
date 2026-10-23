package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ ports.LotViewCache = (*LotViewCache)(nil)

const (
	keyPrefix  = "qr:lot:"
	DefaultTTL = 10 * time.Minute
)

type document struct {
	LotID        uuid.UUID `json:"lot_id"`
	Code         string    `json:"code"`
	Variety      string    `json:"variety"`
	Process      string    `json:"process"`
	HarvestDate  time.Time `json:"harvest_date"`
	FarmID       uuid.UUID `json:"farm_id"`
	FarmName     string    `json:"farm_name"`
	FarmAltitude int       `json:"farm_altitude"`
	ProducerID   uuid.UUID `json:"producer_id"`
	ProducerName string    `json:"producer_name"`
	CustodianID  uuid.UUID `json:"custodian_id"`
	CurrentStage string    `json:"current_stage"`
	Status       string    `json:"status"`
	VoidReason   string    `json:"void_reason"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type LotViewCache struct {
	client goredis.Cmdable
	ttl    time.Duration
}

func NewLotViewCache(client goredis.Cmdable, ttl time.Duration) *LotViewCache {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &LotViewCache{client: client, ttl: ttl}
}

func key(id uuid.UUID) string {
	return keyPrefix + id.String()
}

func (c *LotViewCache) Get(ctx context.Context, id uuid.UUID) (*lotview.LotView, error) {
	raw, err := c.client.Get(ctx, key(id)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return lotview.Default(), nil
	}
	if err != nil {
		return lotview.Default(), internalError("get lot view from cache", err)
	}

	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return lotview.Default(), internalError("decode cached lot view", err)
	}
	return toDomain(doc), nil
}

func (c *LotViewCache) Set(ctx context.Context, view *lotview.LotView) error {
	if view.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cache lot view: default view")
	}

	raw, err := json.Marshal(toDocument(view))
	if err != nil {
		return internalError("encode lot view for cache", err)
	}
	if err := c.client.Set(ctx, key(view.LotID()), raw, c.ttl).Err(); err != nil {
		return internalError("set lot view in cache", err)
	}
	return nil
}

func (c *LotViewCache) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.client.Del(ctx, key(id)).Err(); err != nil {
		return internalError("delete lot view from cache", err)
	}
	return nil
}

func toDocument(v *lotview.LotView) document {
	return document{
		LotID:        v.LotID(),
		Code:         v.Code(),
		Variety:      v.Variety(),
		Process:      v.Process().String(),
		HarvestDate:  v.HarvestDate().UTC(),
		FarmID:       v.FarmID(),
		FarmName:     v.FarmName(),
		FarmAltitude: v.FarmAltitude(),
		ProducerID:   v.ProducerID(),
		ProducerName: v.ProducerName(),
		CustodianID:  v.CustodianID(),
		CurrentStage: v.CurrentStage().String(),
		Status:       v.Status().String(),
		VoidReason:   v.VoidReason(),
		Version:      v.Version(),
		CreatedAt:    v.CreatedAt().UTC(),
		UpdatedAt:    v.UpdatedAt().UTC(),
	}
}

func toDomain(d document) *lotview.LotView {
	return lotview.Restore(lotview.RestoreParams{
		LotID:        d.LotID,
		Code:         d.Code,
		Variety:      d.Variety,
		Process:      lotview.ParseProcess(d.Process),
		HarvestDate:  d.HarvestDate.UTC(),
		FarmID:       d.FarmID,
		FarmName:     d.FarmName,
		FarmAltitude: d.FarmAltitude,
		ProducerID:   d.ProducerID,
		ProducerName: d.ProducerName,
		CustodianID:  d.CustodianID,
		CurrentStage: lotview.ParseStage(d.CurrentStage),
		Status:       lotview.ParseStatus(d.Status),
		VoidReason:   d.VoidReason,
		Version:      d.Version,
		CreatedAt:    d.CreatedAt.UTC(),
		UpdatedAt:    d.UpdatedAt.UTC(),
	})
}

func internalError(technicalMessage string, cause error) error {
	return exception.New(exception.CodeInternal).
		WithTechnical(technicalMessage).
		WithCause(cause)
}
