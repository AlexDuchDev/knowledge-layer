package httpserver

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/identity_access"
)

// NavigationVisibility is returned on GET /auth/me for client-side nav filtering.
// Flags mirror effective access checks used by HTTP handlers (domain grant + role action + access level).
type NavigationVisibility struct {
	HasDomainGrant      bool `json:"has_domain_grant"`
	MayPublish          bool `json:"may_publish"`
	MayApprove          bool `json:"may_approve"`
	MayManageSourceFeed bool `json:"may_manage_source_feed"`
	MayRunJob           bool `json:"may_run_job"`
}

func computeNavigationVisibility(ctx context.Context, d *app.Deps, principal uuid.UUID) (NavigationVisibility, error) {
	var out NavigationVisibility
	doms, err := d.Access.DomainIDsWithGrant(ctx, principal)
	if err != nil {
		return out, err
	}
	out.HasDomainGrant = len(doms) > 0
	for _, dom := range doms {
		did := dom
		if !out.MayPublish {
			if ok, err := allowAction(ctx, d, principal, did, "publish"); err != nil {
				return out, err
			} else if ok {
				out.MayPublish = true
			}
		}
		if !out.MayApprove {
			if ok, err := allowAction(ctx, d, principal, did, "approve"); err != nil {
				return out, err
			} else if ok {
				out.MayApprove = true
			}
		}
		if !out.MayManageSourceFeed {
			if ok, err := allowAction(ctx, d, principal, did, "manage_source_feed"); err != nil {
				return out, err
			} else if ok {
				out.MayManageSourceFeed = true
			}
		}
		if !out.MayRunJob {
			if ok, err := allowAction(ctx, d, principal, did, "run_job"); err != nil {
				return out, err
			} else if ok {
				out.MayRunJob = true
			}
		}
	}
	return out, nil
}

func allowAction(ctx context.Context, d *app.Deps, principal uuid.UUID, domainID uuid.UUID, action string) (bool, error) {
	dec, err := d.Access.Evaluate(ctx, identity_access.EvaluateInput{
		PrincipalID:  principal,
		Action:       action,
		ResourceType: "domain",
		DomainID:     &domainID,
	})
	if err != nil {
		return false, err
	}
	return dec.Allow && dec.SensitivityOK, nil
}
