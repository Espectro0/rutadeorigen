package authorizer

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	stringadapter "github.com/casbin/casbin/v2/persist/string-adapter"
	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
)

//go:embed casbin_model.conf
var casbinModel string

//go:embed casbin_policy.csv
var casbinPolicy string

var _ ports.Authorizer = (*Casbin)(nil)

type Casbin struct {
	enforcer *casbin.Enforcer
}

type subjectAttributes struct {
	ID   string
	Role string
}

type resourceAttributes struct {
	Type        string
	OwnerID     string
	CustodianID string
	Stage       string
}

func NewCasbin() (*Casbin, error) {
	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		return nil, fmt.Errorf("load casbin model: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(m, stringadapter.NewAdapter(casbinPolicy))
	if err != nil {
		return nil, fmt.Errorf("load casbin policy: %w", err)
	}

	return &Casbin{enforcer: enforcer}, nil
}

func MustNewCasbin() *Casbin {
	c, err := NewCasbin()
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Casbin) IsAllowed(_ context.Context, subject ports.Subject, action ports.Action, resource ports.Resource) (bool, error) {
	allowed, err := c.enforcer.Enforce(
		subjectAttributes{ID: identifier(subject.ActorID), Role: string(subject.Role)},
		resourceAttributes{
			Type:        string(resource.Type),
			CustodianID: identifier(resource.CustodianID),
			Stage:       string(resource.Stage),
		},
		string(action),
	)
	if err != nil {
		return false, fmt.Errorf("enforce %s: %w", action, err)
	}
	return allowed, nil
}

func identifier(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}
