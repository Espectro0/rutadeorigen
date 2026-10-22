package resolvescan

import (
	"context"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/application/usecases/validators"
	"rutadeorigen/qr/internal/domain/redirect"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ UseCase = (*ResolveScan)(nil)

type ResolveScan struct {
	rules ports.RedirectRepository
	views ports.LotViewRepository
	cache ports.LotViewCache
	scans ports.ScanRepository
}

func New(rules ports.RedirectRepository, views ports.LotViewRepository, cache ports.LotViewCache, scans ports.ScanRepository) *ResolveScan {
	return &ResolveScan{
		rules: rules,
		views: views,
		cache: cache,
		scans: scans,
	}
}

func (uc *ResolveScan) Execute(ctx context.Context, path string) (redirect.Resolution, error) {
	rule, err := uc.matchingRule(ctx, path)
	if err != nil {
		return redirect.Resolution{}, err
	}

	resolution, err := rule.Resolve(path)
	if err != nil {
		return redirect.Resolution{}, err
	}

	if _, err := validators.LotViewExists(ctx, uc.cache, uc.views, resolution.LotID); err != nil {
		return redirect.Resolution{}, err
	}

	uc.recordScan(ctx, resolution)
	return resolution, nil
}

func (uc *ResolveScan) matchingRule(ctx context.Context, path string) (*redirect.Rule, error) {
	rules, err := uc.rules.ListActive(ctx)
	if err != nil {
		return redirect.Default(), exception.Wrap(err, "list active redirect rules")
	}

	for _, rule := range rules {
		if rule.Matches(path) {
			return rule, nil
		}
	}
	return redirect.Default(), exception.New(exception.CodeRedirectNotFound)
}

func (uc *ResolveScan) recordScan(ctx context.Context, resolution redirect.Resolution) {
	scanned, err := scan.New(resolution.LotID)
	if err != nil {
		return
	}
	_ = uc.scans.Save(ctx, scanned)
}
