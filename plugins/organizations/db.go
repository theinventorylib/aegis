// Package organizations provides team and organization management functionality.
package organizations

import (
	"context"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
)

const (
	// RoleOwner represents the owner role in an organization.
	RoleOwner = "owner"
	// RoleAdmin represents the admin role in an organization.
	RoleAdmin = "admin"
	// RoleMember represents the member role in an organization.
	RoleMember = "member"
)

// ========== ORGANIZATION DATABASE FUNCTIONS ==========

func (p *Plugin) createOrganization(ctx context.Context, name, slug, ownerID string) (*Organization, error) {
	org := &Organization{
		ID:        core.GenerateID(),
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Start transaction
	tx, err := p.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // Ignore rollback error

	// Insert organization
	_, err = tx.Exec(ctx, `
		INSERT INTO auth.organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Add creator as owner
	memberID := core.GenerateID()
	_, err = tx.Exec(ctx, `
		INSERT INTO auth.user_organizations (id, user_id, organization_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, memberID, ownerID, org.ID, RoleOwner, time.Now(), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to add owner: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return org, nil
}

func (p *Plugin) getUserOrganizations(ctx context.Context, userID string) ([]Organization, error) {
	rows, err := p.Query(ctx, `
		SELECT o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM auth.organizations o
		INNER JOIN auth.user_organizations uo ON o.id = uo.organization_id
		WHERE uo.user_id = $1
		ORDER BY o.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var org Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}

	return orgs, nil
}

func (p *Plugin) getOrganization(ctx context.Context, orgID string) (*Organization, error) {
	var org Organization
	err := p.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM auth.organizations
		WHERE id = $1
	`, orgID).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	return &org, nil
}

func (p *Plugin) updateOrganization(ctx context.Context, orgID, name, slug string) error {
	result, err := p.Exec(ctx, `
		UPDATE auth.organizations
		SET name = $1, slug = $2, updated_at = $3
		WHERE id = $4
	`, name, slug, time.Now(), orgID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

func (p *Plugin) deleteOrganization(ctx context.Context, orgID string) error {
	result, err := p.Exec(ctx, `
		DELETE FROM auth.organizations WHERE id = $1
	`, orgID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

// ========== ORGANIZATION MEMBER FUNCTIONS ==========

func (p *Plugin) addOrganizationMember(ctx context.Context, orgID, userID, role string) error {
	memberID := core.GenerateID()
	_, err := p.Exec(ctx, `
		INSERT INTO auth.user_organizations (id, user_id, organization_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, memberID, userID, orgID, role, time.Now(), time.Now())

	return err
}

func (p *Plugin) getOrganizationMembers(ctx context.Context, orgID string) ([]UserOrganization, error) {
	rows, err := p.Query(ctx, `
		SELECT id, user_id, organization_id, role, created_at, updated_at
		FROM auth.user_organizations
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []UserOrganization
	for rows.Next() {
		var member UserOrganization
		if err := rows.Scan(&member.ID, &member.UserID, &member.OrganizationID, &member.Role, &member.CreatedAt, &member.UpdatedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

func (p *Plugin) updateOrganizationMemberRole(ctx context.Context, orgID, userID, role string) error {
	result, err := p.Exec(ctx, `
		UPDATE auth.user_organizations
		SET role = $1, updated_at = $2
		WHERE organization_id = $3 AND user_id = $4
	`, role, time.Now(), orgID, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("member not found")
	}

	return nil
}

func (p *Plugin) removeOrganizationMember(ctx context.Context, orgID, userID string) error {
	result, err := p.Exec(ctx, `
		DELETE FROM auth.user_organizations
		WHERE organization_id = $1 AND user_id = $2
	`, orgID, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("member not found")
	}

	return nil
}

// ========== TEAM FUNCTIONS ==========

func (p *Plugin) createTeam(ctx context.Context, orgID, name, description string) (*Team, error) {
	team := &Team{
		ID:             core.GenerateID(),
		OrganizationID: orgID,
		Name:           name,
		Description:    description,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := p.Exec(ctx, `
		INSERT INTO auth.teams (id, organization_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, team.ID, team.OrganizationID, team.Name, team.Description, team.CreatedAt, team.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return team, nil
}

func (p *Plugin) getOrganizationTeams(ctx context.Context, orgID string) ([]Team, error) {
	rows, err := p.Query(ctx, `
		SELECT id, organization_id, name, description, created_at, updated_at
		FROM auth.teams
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var team Team
		if err := rows.Scan(&team.ID, &team.OrganizationID, &team.Name, &team.Description, &team.CreatedAt, &team.UpdatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}

	return teams, nil
}

func (p *Plugin) getTeam(ctx context.Context, teamID string) (*Team, error) {
	var team Team
	err := p.QueryRow(ctx, `
		SELECT id, organization_id, name, description, created_at, updated_at
		FROM auth.teams
		WHERE id = $1
	`, teamID).Scan(&team.ID, &team.OrganizationID, &team.Name, &team.Description, &team.CreatedAt, &team.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	return &team, nil
}

func (p *Plugin) updateTeam(ctx context.Context, teamID, name, description string) error {
	result, err := p.Exec(ctx, `
		UPDATE auth.teams
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
	`, name, description, time.Now(), teamID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("team not found")
	}

	return nil
}

func (p *Plugin) deleteTeam(ctx context.Context, teamID string) error {
	result, err := p.Exec(ctx, `
		DELETE FROM auth.teams WHERE id = $1
	`, teamID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("team not found")
	}

	return nil
}

// ========== TEAM MEMBER FUNCTIONS ==========

func (p *Plugin) addTeamMember(ctx context.Context, teamID, userID, role string) error {
	memberID := core.GenerateID()
	_, err := p.Exec(ctx, `
		INSERT INTO auth.team_members (id, team_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, memberID, teamID, userID, role, time.Now(), time.Now())

	return err
}

func (p *Plugin) getTeamMembers(ctx context.Context, teamID string) ([]TeamMember, error) {
	rows, err := p.Query(ctx, `
		SELECT id, team_id, user_id, role, created_at, updated_at
		FROM auth.team_members
		WHERE team_id = $1
		ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []TeamMember
	for rows.Next() {
		var member TeamMember
		if err := rows.Scan(&member.ID, &member.TeamID, &member.UserID, &member.Role, &member.CreatedAt, &member.UpdatedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

func (p *Plugin) updateTeamMemberRole(ctx context.Context, teamID, userID, role string) error {
	result, err := p.Exec(ctx, `
		UPDATE auth.team_members
		SET role = $1, updated_at = $2
		WHERE team_id = $3 AND user_id = $4
	`, role, time.Now(), teamID, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("team member not found")
	}

	return nil
}

func (p *Plugin) removeTeamMember(ctx context.Context, teamID, userID string) error {
	result, err := p.Exec(ctx, `
		DELETE FROM auth.team_members
		WHERE team_id = $1 AND user_id = $2
	`, teamID, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("team member not found")
	}

	return nil
}

// ========== PERMISSION CHECK FUNCTIONS ==========

func (p *Plugin) isOrganizationMember(ctx context.Context, userID, orgID string) bool {
	var exists bool
	err := p.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM auth.user_organizations
			WHERE user_id = $1 AND organization_id = $2
		)
	`, userID, orgID).Scan(&exists)

	return err == nil && exists
}

func (p *Plugin) isOwner(ctx context.Context, userID, orgID string) bool {
	var role string
	err := p.QueryRow(ctx, `
		SELECT role FROM auth.user_organizations
		WHERE user_id = $1 AND organization_id = $2
	`, userID, orgID).Scan(&role)

	return err == nil && role == RoleOwner
}

func (p *Plugin) isOwnerOrAdmin(ctx context.Context, userID, orgID string) bool {
	var role string
	err := p.QueryRow(ctx, `
		SELECT role FROM auth.user_organizations
		WHERE user_id = $1 AND organization_id = $2
	`, userID, orgID).Scan(&role)

	return err == nil && (role == RoleOwner || role == RoleAdmin)
}

// ========== DATABASE HELPER FUNCTIONS ==========

// Query executes a query that returns rows.
func (p *Plugin) Query(ctx context.Context, query string, args ...interface{}) (db.Rows, error) {
	return p.database.Query(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row.
func (p *Plugin) QueryRow(ctx context.Context, query string, args ...interface{}) db.Row {
	return p.database.QueryRow(ctx, query, args...)
}

// Exec executes a query that doesn't return rows.
func (p *Plugin) Exec(ctx context.Context, query string, args ...interface{}) (db.Result, error) {
	return p.database.Exec(ctx, query, args...)
}

// Begin starts a new database transaction.
func (p *Plugin) Begin(ctx context.Context) (db.Tx, error) {
	return p.database.Begin(ctx)
}
