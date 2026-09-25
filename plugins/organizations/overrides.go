package organizations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/v2/core"
	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
)

// Validation errors returned by SetMemberPermissionOverrides.
var (
	// ErrPermissionRequired means an override was sent without a permission.
	ErrPermissionRequired = errors.New("permission is required")
	// ErrInvalidEffect means the effect was neither "grant" nor "deny".
	ErrInvalidEffect = errors.New("invalid permission effect")
	// ErrDuplicatePermission means the same permission appeared twice.
	ErrDuplicatePermission = errors.New("duplicate permission")
)

// MemberPermissionOverrides returns a member's grant/deny overrides. Returns
// an empty slice when none exist.
func (p *Plugin) MemberPermissionOverrides(ctx context.Context, orgID, userID string) ([]orgtypes.MemberPermissionOverride, error) {
	return p.store.ListMemberPermissionOverrides(ctx, orgID, userID)
}

// SetMemberPermissionOverrides replaces a member's overrides (send an empty
// slice to clear them). The target must be a member of the organization;
// duplicate permissions and unknown effects are rejected before anything is
// written. Empty IDs and timestamps are filled in on overrides in place.
//
// This is the programmatic entry point: it applies no actor-authority cap
// because it has no actor. The HTTP handler applies the cap and the owner
// guard before delegating here.
//
// Replacement is delete-then-insert: a failure part-way through leaves a
// partial set, which the caller can correct by re-sending the desired list.
func (p *Plugin) SetMemberPermissionOverrides(ctx context.Context, orgID, userID string, overrides []orgtypes.MemberPermissionOverride) error {
	seen := make(map[string]bool, len(overrides))
	for i := range overrides {
		perm := core.SanitizeString(overrides[i].Permission, nil)
		if perm == "" {
			return ErrPermissionRequired
		}
		switch overrides[i].Effect {
		case orgtypes.PermissionEffectGrant, orgtypes.PermissionEffectDeny:
		default:
			return fmt.Errorf("%w: %q", ErrInvalidEffect, overrides[i].Effect)
		}
		if seen[perm] {
			return fmt.Errorf("%w: %q", ErrDuplicatePermission, perm)
		}
		seen[perm] = true
		overrides[i].Permission = perm
	}

	if _, err := p.store.GetMember(ctx, userID, orgID); err != nil {
		return err
	}

	if err := p.store.DeleteMemberPermissionOverrides(ctx, orgID, userID); err != nil {
		return err
	}
	now := time.Now()
	for i := range overrides {
		o := &overrides[i]
		if o.ID == "" {
			o.ID = core.GenerateID()
		}
		o.OrganizationID = orgID
		o.UserID = userID
		if o.CreatedAt.IsZero() {
			o.CreatedAt = now
		}
		o.UpdatedAt = now
		if err := p.store.CreateMemberPermissionOverride(ctx, *o); err != nil {
			return err
		}
	}
	return nil
}
