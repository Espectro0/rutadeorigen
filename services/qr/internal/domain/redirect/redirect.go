package redirect

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

const idPlaceholder = "{id}"

var allowedStatusCodes = map[int]bool{
	301: true,
	302: true,
	308: true,
}

type Rule struct {
	id             uuid.UUID
	sourcePrefix   string
	targetTemplate string
	statusCode     int
	active         bool
	createdAt      time.Time
}

var defaultRule = Rule{
	id:             helper.DefaultUUID(),
	sourcePrefix:   helper.EmptyString,
	targetTemplate: helper.EmptyString,
	statusCode:     helper.DefaultInt(),
	active:         false,
	createdAt:      helper.DefaultDate(),
}

func Default() *Rule {
	r := defaultRule
	return &r
}

func New(sourcePrefix, targetTemplate string, statusCode int) (*Rule, error) {
	sourcePrefix = helper.ApplyTrim(sourcePrefix)
	targetTemplate = helper.ApplyTrim(targetTemplate)

	if !isValidPrefix(sourcePrefix) || !isValidTemplate(targetTemplate) || !allowedStatusCodes[statusCode] {
		return Default(), exception.New(exception.CodeRedirectInvalidRule)
	}

	return &Rule{
		id:             uuid.New(),
		sourcePrefix:   sourcePrefix,
		targetTemplate: targetTemplate,
		statusCode:     statusCode,
		active:         true,
		createdAt:      time.Now().UTC(),
	}, nil
}

func isValidPrefix(prefix string) bool {
	return len(prefix) > 1 && strings.HasPrefix(prefix, "/") && strings.HasSuffix(prefix, "/")
}

func isValidTemplate(template string) bool {
	if !strings.Contains(template, idPlaceholder) {
		return false
	}
	return strings.HasPrefix(template, "/") ||
		strings.HasPrefix(template, "http://") ||
		strings.HasPrefix(template, "https://")
}

type RestoreParams struct {
	ID             uuid.UUID
	SourcePrefix   string
	TargetTemplate string
	StatusCode     int
	Active         bool
	CreatedAt      time.Time
}

func Restore(p RestoreParams) *Rule {
	return &Rule{
		id:             p.ID,
		sourcePrefix:   p.SourcePrefix,
		targetTemplate: p.TargetTemplate,
		statusCode:     p.StatusCode,
		active:         p.Active,
		createdAt:      p.CreatedAt,
	}
}

type Resolution struct {
	LotID      uuid.UUID
	Location   string
	StatusCode int
}

func (r *Rule) Matches(path string) bool {
	return r.active && strings.HasPrefix(path, r.sourcePrefix)
}

func (r *Rule) Resolve(path string) (Resolution, error) {
	if !r.Matches(path) {
		return Resolution{}, exception.New(exception.CodeRedirectNotFound)
	}

	lotID := helper.ParseUUID(strings.TrimPrefix(path, r.sourcePrefix))
	if helper.IsDefaultUUID(lotID) {
		return Resolution{}, exception.New(exception.CodeLotViewInvalidID)
	}

	return Resolution{
		LotID:      lotID,
		Location:   strings.ReplaceAll(r.targetTemplate, idPlaceholder, lotID.String()),
		StatusCode: r.statusCode,
	}, nil
}

func (r *Rule) ID() uuid.UUID          { return r.id }
func (r *Rule) SourcePrefix() string   { return r.sourcePrefix }
func (r *Rule) TargetTemplate() string { return r.targetTemplate }
func (r *Rule) StatusCode() int        { return r.statusCode }
func (r *Rule) Active() bool           { return r.active }
func (r *Rule) CreatedAt() time.Time   { return r.createdAt }

func (r *Rule) IsDefault() bool {
	return r == nil || helper.IsDefaultUUID(r.id)
}
