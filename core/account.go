package core

import (
	"context"
	"database/sql"
	"time"

	"github.com/theinventorylib/aegis/auth"
)

// AccountService manages authentication accounts linked to users.
//
// An "account" represents a specific authentication method for a user:
//   - Credential account (email/password)
//   - OAuth account (Google, GitHub, etc.)
//   - Other provider accounts (SAML, LDAP, etc.)
//
// A single user can have multiple accounts (one per provider), enabling
// multi-provider authentication (login with email OR Google, for example).
//
// Key responsibilities:
//   - Account creation and retrieval
//   - Password updates with session invalidation
//   - Provider-specific account management
//   - Audit logging of account changes
type AccountService struct {
	// accountStore persists account data
	accountStore auth.AccountStore

	// sessionStore manages sessions (for invalidation on password change)
	sessionStore auth.SessionStore

	// hashConfig defines password hashing parameters
	hashConfig *PasswordHasherConfig

	// authConfig holds authentication policies
	authConfig *AuthConfig

	// auditLogger records account management events
	auditLogger AuditLogger
}

// NewAccountService creates a new account service with the specified dependencies.
func NewAccountService(accountStore auth.AccountStore, sessionStore auth.SessionStore, hashConfig *PasswordHasherConfig, authConfig *AuthConfig, auditLogger AuditLogger) *AccountService {
	return &AccountService{
		accountStore: accountStore,
		sessionStore: sessionStore,
		hashConfig:   hashConfig,
		authConfig:   authConfig,
		auditLogger:  auditLogger,
	}
}

// CreateAccount creates a new account
func (s *AccountService) CreateAccount(ctx context.Context, account auth.Account) error {
	// Sanitize external account identifiers
	account.ProviderAccountID = SanitizeString(account.ProviderAccountID, nil)

	if account.ID == "" {
		account.ID = GenerateID()
	}
	if account.CreatedAt.IsZero() {
		account.CreatedAt = time.Now()
	}
	account.UpdatedAt = time.Now()
	return s.accountStore.Create(ctx, account)
}

// UpdatePassword changes a user's password and invalidates existing sessions.
//
// Security considerations:
//   - The new password is hashed with Argon2id before storage
//   - All existing sessions are invalidated (user must re-login)
//   - The operation is audited for security monitoring
//
// Flow:
//  1. Find the user's password account (provider="credentials")
//  2. Hash the new password
//  3. Update the account with the new hash
//  4. Delete all sessions to force re-authentication
//  5. Log the password change event
//
// Parameters:
//   - ctx: Request context
//   - userID: ID of the user whose password is being changed
//   - newPassword: Plaintext new password to hash and store
//
// Returns an error if:
//   - User has no password account (OAuth-only user)
//   - Password hashing fails
//   - Database update fails
//
// Session deletion errors are logged but don't fail the operation.
func (s *AccountService) UpdatePassword(ctx context.Context, userID, newPassword string) error {
	accounts, err := s.accountStore.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	var passwordAccount *auth.Account
	for _, acc := range accounts {
		if acc.Provider == PasswordProvider {
			passwordAccount = &acc
			break
		}
	}

	if passwordAccount == nil {
		return NewAuthError(AuthErrorCodeAccountNotFound, "password account not found")
	}

	hashed, err := HashPassword(newPassword, s.hashConfig.Argon2Time, s.hashConfig.Argon2Memory, s.hashConfig.Argon2Threads, s.hashConfig.Argon2KeyLength)
	if err != nil {
		return err
	}

	passwordAccount.PasswordHash = hashed
	passwordAccount.UpdatedAt = time.Now()

	if err := s.accountStore.Update(ctx, *passwordAccount); err != nil {
		return err
	}

	if err := s.sessionStore.DeleteByUserID(ctx, userID); err != nil {
		// Log the error but don't fail the password update
		// Session invalidation is best-effort - if it fails, existing sessions remain valid
		// until they expire naturally, but the password has been changed successfully
		_ = s.auditLogger.LogAuthEvent(ctx, "session_deletion_failed", userID, "", "", false, map[string]any{
			"error":  err.Error(),
			"reason": "password_change",
		})
	}

	_ = s.auditLogger.LogAuthEvent(ctx, AuditEventPasswordChanged, userID, "", "", true, nil)
	return nil
}

// GetAccountByID retrieves an account by ID
func (s *AccountService) GetAccountByID(ctx context.Context, id string) (auth.Account, error) {
	return s.accountStore.GetByID(ctx, id)
}

// GetAccountsByUserID retrieves all accounts for a user
func (s *AccountService) GetAccountsByUserID(ctx context.Context, userID string) ([]auth.Account, error) {
	return s.accountStore.GetByUserID(ctx, userID)
}

// GetPasswordAccount retrieves the password account for a user
func (s *AccountService) GetPasswordAccount(ctx context.Context, userID string) (auth.Account, error) {
	accounts, err := s.accountStore.GetByUserID(ctx, userID)
	if err != nil {
		return auth.Account{}, err
	}
	for _, acc := range accounts {
		if acc.Provider == PasswordProvider {
			return acc, nil
		}
	}
	return auth.Account{}, sql.ErrNoRows
}

// DeleteAccount deletes a user account by its ID.
func (s *AccountService) DeleteAccount(ctx context.Context, id string) error {
	return s.accountStore.Delete(ctx, id)
}

// VerifyPassword verifies a user's password
func (s *AccountService) VerifyPassword(ctx context.Context, userID, password string) (bool, error) {
	account, err := s.GetPasswordAccount(ctx, userID)
	if err != nil {
		return false, err
	}
	return VerifyPassword(password, account.PasswordHash)
}
