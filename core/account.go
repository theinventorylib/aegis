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

	// transactor lets UpdatePassword combine the password rotation and
	// session purge into a single DB transaction. May be nil when the
	// configured stores do not support cross-store transactions, in
	// which case UpdatePassword falls back to a fail-closed sequential
	// path that surfaces session-purge errors instead of swallowing them.
	transactor auth.Transactor

	// logger surfaces non-fatal operational errors (e.g. audit log
	// failures). Defaults to a no-op when not provided.
	logger Logger
}

// NewAccountService creates a new account service with the specified dependencies.
//
// transactor is optional: when non-nil it enables fully atomic password
// updates (account write + session purge in one DB transaction). When
// nil, UpdatePassword still runs both operations but in sequence and
// surfaces session-purge errors instead of swallowing them, so the
// caller can decide whether to retry. logger may be nil; a no-op logger
// is substituted in that case.
func NewAccountService(accountStore auth.AccountStore, sessionStore auth.SessionStore, hashConfig *PasswordHasherConfig, authConfig *AuthConfig, auditLogger AuditLogger, transactor auth.Transactor, logger Logger) *AccountService {
	if logger == nil {
		logger = noopLogger{}
	}
	return &AccountService{
		accountStore: accountStore,
		sessionStore: sessionStore,
		hashConfig:   hashConfig,
		authConfig:   authConfig,
		auditLogger:  auditLogger,
		transactor:   transactor,
		logger:       logger,
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
//   - The new password is hashed with Argon2id before storage.
//   - All existing sessions are invalidated (user must re-login).
//   - The operation is audited for security monitoring.
//
// Atomicity:
//   - When the underlying store supports transactions (the default SQL
//     backend does), the password write and the session purge run in a
//     single DB transaction. Either both happen or neither does, so an
//     attacker holding a valid session can never end up in a state where
//     the new password is in effect but the old session is still alive.
//   - When transactions are not available (custom store implementations
//     that do not implement auth.Transactor), the operations run in
//     sequence; if the session purge fails the password update is
//     rolled back manually by re-saving the previous hash and the
//     original session-purge error is returned. This avoids leaving the
//     account in the dangerous half-rotated state where the password
//     changed but old sessions remain valid.
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

	previousHash := passwordAccount.PasswordHash

	hashed, err := HashPassword(newPassword, s.hashConfig.Argon2Time, s.hashConfig.Argon2Memory, s.hashConfig.Argon2Threads, s.hashConfig.Argon2KeyLength)
	if err != nil {
		return err
	}

	passwordAccount.PasswordHash = hashed
	passwordAccount.UpdatedAt = time.Now()

	if s.transactor != nil {
		if err := s.updatePasswordTx(ctx, *passwordAccount, userID); err != nil {
			return err
		}
	} else {
		if err := s.updatePasswordSequential(ctx, *passwordAccount, previousHash, userID); err != nil {
			return err
		}
	}

	if auditErr := s.auditLogger.LogAuthEvent(ctx, AuditEventPasswordChanged, userID, true, nil); auditErr != nil {
		s.logger.Error("account: failed to write password-change audit event", "user_id", userID, "error", auditErr)
	}
	return nil
}

// updatePasswordTx runs the account update + session purge in a single
// database transaction. Any error rolls back both operations.
func (s *AccountService) updatePasswordTx(ctx context.Context, account auth.Account, userID string) error {
	tx, err := s.transactor.BeginTx(ctx)
	if err != nil {
		return NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to begin password-update transaction", err)
	}
	// Rollback after Commit is a no-op, so this defer is always safe.
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	if err := tx.AccountStore().Update(ctx, account); err != nil {
		return err
	}
	if err := tx.SessionStore().DeleteByUserID(ctx, userID); err != nil {
		return NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to invalidate sessions during password update", err)
	}
	if err := tx.Commit(); err != nil {
		return NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to commit password-update transaction", err)
	}
	return nil
}

// updatePasswordSequential is the fallback used when the configured
// stores do not expose a Transactor. It still fails closed: if the
// session purge fails we attempt to restore the previous password hash
// so the account is not left half-rotated, and we always return the
// original error to the caller.
func (s *AccountService) updatePasswordSequential(ctx context.Context, account auth.Account, previousHash, userID string) error {
	if err := s.accountStore.Update(ctx, account); err != nil {
		return err
	}
	if err := s.sessionStore.DeleteByUserID(ctx, userID); err != nil {
		// Best-effort rollback of the password hash. We log but
		// otherwise ignore restore failures because at this point
		// the password has rotated successfully and only the session
		// purge failed; we must surface the session-purge error.
		account.PasswordHash = previousHash
		account.UpdatedAt = time.Now()
		if restoreErr := s.accountStore.Update(ctx, account); restoreErr != nil {
			s.logger.Error("account: failed to roll back password after session-purge failure",
				"user_id", userID, "session_purge_error", err, "restore_error", restoreErr)
		}
		_ = s.auditLogger.LogAuthEvent(ctx, "session_deletion_failed", userID, false, map[string]any{
			"error":  err.Error(),
			"reason": "password_change",
		})
		return NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to invalidate sessions during password update", err)
	}
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
