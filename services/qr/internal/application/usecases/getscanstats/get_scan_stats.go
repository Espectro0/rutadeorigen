package getscanstats

import (
	"context"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/application/usecases/validators"
	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ UseCase = (*GetScanStats)(nil)

type GetScanStats struct {
	views ports.LotViewRepository
	cache ports.LotViewCache
	scans ports.ScanRepository
	auth  ports.Authorizer
}

func New(views ports.LotViewRepository, cache ports.LotViewCache, scans ports.ScanRepository, auth ports.Authorizer) *GetScanStats {
	return &GetScanStats{views: views, cache: cache, scans: scans, auth: auth}
}

func (uc *GetScanStats) Execute(ctx context.Context, query Query) (*scan.Stats, error) {
	actorID := requestcontext.ActorID(ctx)
	if helper.IsDefaultUUID(actorID) {
		return nil, exception.New(exception.CodeUnauthorized)
	}
	if !scan.IsValidStatsDays(query.Days) {
		return nil, exception.New(exception.CodeStatsInvalidDays)
	}

	view, err := validators.LotViewExists(ctx, uc.cache, uc.views, query.LotID)
	if err != nil {
		return nil, err
	}

	if err := uc.authorize(ctx, actorID, view); err != nil {
		return nil, err
	}

	since := helper.DateUTC(time.Now()).AddDate(0, 0, -(query.Days - 1))
	stats, err := uc.scans.StatsByLot(ctx, view.LotID(), since)
	if err != nil {
		return nil, exception.Wrap(err, "query scan stats")
	}
	return stats, nil
}

func (uc *GetScanStats) authorize(ctx context.Context, actorID uuid.UUID, view *lotview.LotView) error {
	allowed, err := uc.auth.IsAllowed(ctx,
		ports.Subject{ActorID: actorID, Role: actorview.ParseRole(requestcontext.ActorRole(ctx))},
		ports.ActionReadQRStats,
		ports.Resource{Type: ports.ResourceLot, ID: view.LotID(), CustodianID: view.CustodianID(), Stage: view.CurrentStage()})
	if err != nil {
		return exception.Wrap(err, "evaluate read qr stats policy")
	}
	if !allowed {
		return exception.New(exception.CodeForbidden)
	}
	return nil
}
