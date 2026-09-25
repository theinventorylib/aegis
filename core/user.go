package core

import (
	"context"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
)

// UserService provides high-level user management operations.
// It orchestrates user creation, deletion, and updates while coordinating
// with AccountStore and SessionStore to maintain data consistency.
//
// Key responsibilities:
//   - User CRUD operations
//   - Password account creation during user signup
//   - Cascading deletion of accounts and sessions
//   - Audit logging of user lifecycle events
//
// The service is safe for concurrent use.
type UserService struct {
	// userStore persists user data
	userStore auth.UserStore

	// accountStore manages linked authentication accounts
	accountStore auth.AccountStore

	// sessionStore manages user sessions (for cleanup on deletion)
	sessionStore auth.SessionStore

	// hashConfig defines password hashing parameters
	hashConfig *PasswordHasherConfig

	// authConfig holds authentication policies
	authConfig *AuthConfig

	// auditLogger records user management events
	auditLogger AuditLogger

	// transactor, when non-nil, lets user + credentials-account creation run
	// in a single transaction. Nil falls back to a compensating delete.
	transactor auth.Transactor

	// logger surfaces non-fatal operational errors (e.g. a failed
	// compensating delete). Defaults to a no-op.
	logger Logger

	// sessionCachePurger, when set, clears cached sessions for a user before
	// their DB session rows are deleted. Wired by AuthService.
	sessionCachePurger func(ctx context.Context, userID string) error

	// emailVerificationReset, when set by an email plugin, clears the email
	// verification flag for a new address so a previously-verified flag never
	// carries over after an email change.
	emailVerificationReset func(ctx context.Context, userID, email string) error

	// verification issues and redeems email-change tokens. Wired by AuthService.
	verification *VerificationService

	// emailVerifiedMarker, when set by an email plugin, marks the user's
	// current email verified. Used after a confirmed email change, where
	// control of the new address has just been proven.
	emailVerifiedMarker func(ctx context.Context, userID string) error
}

// newUserService creates a new user service with the specified dependencies.
func newUserService(userStore auth.UserStore, accountStore auth.AccountStore, sessionStore auth.SessionStore, hashConfig *PasswordHasherConfig, authConfig *AuthConfig, auditLogger AuditLogger, transactor auth.Transactor, logger Logger) *UserService {
	if logger == nil {
		logger = noopLogger{}
	}
	return &UserService{
		userStore:    userStore,
		accountStore: accountStore,
		sessionStore: sessionStore,
		hashConfig:   hashConfig,
		authConfig:   authConfig,
		auditLogger:  auditLogger,
		transactor:   transactor,
		logger:       logger,
	}
}

// setSessionCachePurger wires the cache purge run before DeleteUser removes
// the session rows. Set by AuthService.
func (s *UserService) setSessionCachePurger(fn func(ctx context.Context, userID string) error) {
	s.sessionCachePurger = fn
}

// setEmailVerificationReset wires the verification reset run on email change.
// Set by AuthService when an email plugin registers a resetter.
func (s *UserService) setEmailVerificationReset(fn func(ctx context.Context, userID, email string) error) {
	s.emailVerificationReset = fn
}

// setVerificationService wires the verification service used by the
// email-change flow. Called by AuthService during setup.
func (s *UserService) setVerificationService(v *VerificationService) {
	s.verification = v
}

// setEmailVerifiedMarker wires the marker run after a confirmed email change.
// Set by AuthService when an email plugin registers one.
func (s *UserService) setEmailVerifiedMarker(fn func(ctx context.Context, userID string) error) {
	s.emailVerifiedMarker = fn
}

// DeleteUser deletes a user and all associated data (accounts and sessions).
//
// When the store supports transactions, the session/account/user deletes run
// atomically. Otherwise they run in sequence (sessions first to satisfy foreign
// keys). Cached sessions are purged first, best-effort, so Redis cannot serve
// a deleted user until TTL.
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	// Purge cached sessions first: the purge walks the store for the rows.
	if s.sessionCachePurger != nil {
		if err := s.sessionCachePurger(ctx, id); err != nil {
			s.logger.Error("user: failed to purge session cache before delete", "user_id", id, "error", err)
		}
	}

	if s.transactor != nil {
		return s.deleteUserTx(ctx, id)
	}

	// Fallback: sequential deletes, sessions first to satisfy foreign keys.
	if err := s.sessionStore.DeleteByUserID(ctx, id); err != nil {
		return err
	}
	accounts, err := s.accountStore.GetByUserID(ctx, id)
	if err != nil {
		return err
	}
	for _, acc := range accounts {
		if err := s.accountStore.Delete(ctx, acc.ID); err != nil {
			return err
		}
	}
	return s.userStore.Delete(ctx, id)
}

// deleteUserTx deletes sessions, accounts, and the user in one transaction.
func (s *UserService) deleteUserTx(ctx context.Context, id string) error {
	tx, err := s.transactor.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	if err := tx.SessionStore().DeleteByUserID(ctx, id); err != nil {
		return err
	}
	accounts, err := tx.AccountStore().GetByUserID(ctx, id)
	if err != nil {
		return err
	}
	for _, acc := range accounts {
		if err := tx.AccountStore().Delete(ctx, acc.ID); err != nil {
			return err
		}
	}
	if err := tx.UserStore().Delete(ctx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// sanitizeAndPrepareUser applies the standard input sanitization chain plus ID and
// timestamp defaults. Single owner shared by all user-mutating methods.
func sanitizeAndPrepareUser(user auth.User) auth.User {
	user.Name = SanitizeString(user.Name, nil)
	user.Email = SanitizeEmail(user.Email)
	user.Avatar = SanitizeURL(user.Avatar)
	if user.ID == "" {
		user.ID = GenerateID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.UpdatedAt = time.Now()
	return user
}

// CreateUser creates a new user with a password-based authentication account.
//
// This is the primary method for email/password signup flows. It validates the
// password against the configured policy, persists the user, hashes the
// password with the configured Argon2id parameters, and creates a
// password-based account (provider="credentials") linked to the user.
//
// The user and account are created atomically: in one transaction when the
// store supports it, otherwise the user is deleted if account creation fails.
// Returns ErrEmailAlreadyExists if the email is taken.
func (s *UserService) CreateUser(ctx context.Context, user auth.User, password string) (auth.User, error) {
	return s.createUser(ctx, user, password, "")
}

// CreateUserWithUsername creates a user whose credentials account is also
// addressable by a unique username (stored as the account's provider account
// ID). Login accepts either the email or the username.
func (s *UserService) CreateUserWithUsername(ctx context.Context, name, email, username, password string) (auth.User, error) {
	return s.createUser(ctx, auth.User{Name: name, Email: email}, password, username)
}

// createUser is the single owner of user + credentials-account creation.
func (s *UserService) createUser(ctx context.Context, user auth.User, password, username string) (auth.User, error) {
	if err := validatePassword(password, s.authConfig.PasswordPolicy); err != nil {
		return auth.User{}, err
	}

	user = sanitizeAndPrepareUser(user)

	if user.Email != "" {
		if err := ValidateEmail(user.Email); err != nil {
			return auth.User{}, err
		}
		// Friendly duplicate check; the unique constraint stays the backstop.
		// Only a genuine "not found" means the address is free — a store error
		// must not silently proceed as if it were.
		if _, err := s.userStore.GetByEmail(ctx, user.Email); err == nil {
			return auth.User{}, ErrEmailAlreadyExists
		} else if !isNotFound(err) {
			return auth.User{}, err
		}
	}

	username = SanitizeUsername(username, 0)
	if username != "" {
		if len(username) < 3 {
			return auth.User{}, ValidationError{Field: "username", Message: "must be at least 3 characters"}
		}
		if _, err := s.accountStore.GetByProvider(ctx, PasswordProvider, username); err == nil {
			return auth.User{}, ErrUsernameTaken
		} else if !isNotFound(err) {
			return auth.User{}, err
		}
	}

	hashedPassword, err := HashPassword(password, s.hashConfig.Argon2Time, s.hashConfig.Argon2Memory, s.hashConfig.Argon2Threads, s.hashConfig.Argon2KeyLength)
	if err != nil {
		return auth.User{}, err
	}

	if s.transactor != nil {
		return s.createUserTx(ctx, user, hashedPassword, username)
	}

	u, err := s.userStore.Create(ctx, user)
	if err != nil {
		return auth.User{}, err
	}

	account := credentialAccount(u.GetID(), hashedPassword, username)
	if err := s.accountStore.Create(ctx, account); err != nil {
		// Compensate so a failed signup does not leave an orphan user.
		if delErr := s.userStore.Delete(ctx, u.GetID()); delErr != nil {
			s.logger.Error("user: failed to roll back user after account creation failure",
				"user_id", u.GetID(), "error", delErr)
		}
		return auth.User{}, err
	}

	return u, nil
}

// createUserTx runs user + credentials-account creation in one transaction.
func (s *UserService) createUserTx(ctx context.Context, user auth.User, hashedPassword, username string) (auth.User, error) {
	tx, err := s.transactor.BeginTx(ctx)
	if err != nil {
		return auth.User{}, err
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	u, err := tx.UserStore().Create(ctx, user)
	if err != nil {
		return auth.User{}, err
	}
	if err := tx.AccountStore().Create(ctx, credentialAccount(u.GetID(), hashedPassword, username)); err != nil {
		return auth.User{}, err
	}
	if err := tx.Commit(); err != nil {
		return auth.User{}, err
	}
	return u, nil
}

// credentialAccount builds the credentials account row. username (optional)
// is stored as ProviderAccountID so login can resolve it.
func credentialAccount(userID, hashedPassword, username string) auth.Account {
	now := time.Now()
	return auth.Account{
		ID:                GenerateID(),
		UserID:            userID,
		Provider:          PasswordProvider,
		ProviderAccountID: username,
		PasswordHash:      hashedPassword,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// CreateUserWithEmail creates a user with email/password from individual fields,
// delegating to CreateUser.
func (s *UserService) CreateUserWithEmail(ctx context.Context, name, email, password string) (auth.User, error) {
	user := auth.User{
		Name:  name,
		Email: email,
	}
	// CreateUser will handle the sanitization
	return s.CreateUser(ctx, user, password)
}

// CreateUserWithoutPassword creates a new user without any authentication account.
//
// Used for OAuth-only users, admin-created users, or service accounts. The user
// won't be able to log in with email/password until a password account is
// created separately.
func (s *UserService) CreateUserWithoutPassword(ctx context.Context, user auth.User) (auth.User, error) {
	return s.userStore.Create(ctx, sanitizeAndPrepareUser(user))
}

// GetUserByID retrieves a user by their unique ID.
func (s *UserService) GetUserByID(ctx context.Context, id string) (auth.User, error) {
	return s.userStore.GetByID(ctx, id)
}

// GetUserByEmail retrieves a user by their email address.
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (auth.User, error) {
	return s.userStore.GetByEmail(ctx, email)
}

// UpdateUser updates an existing user's information.
func (s *UserService) UpdateUser(ctx context.Context, user auth.User) error {
	// Sanitize user input
	user.Name = SanitizeString(user.Name, nil)
	user.Email = SanitizeEmail(user.Email)
	user.Avatar = SanitizeURL(user.Avatar)

	return s.userStore.Update(ctx, user)
}

// UpdateUserEmail changes a user's email address.
//
// The new address is validated and checked for uniqueness. Any verification
// flag is cleared first (via the wired email plugin) so a previously-verified
// address cannot make the new one look verified. Returns ErrEmailAlreadyExists
// if another account owns the address.
func (s *UserService) UpdateUserEmail(ctx context.Context, userID, email string) error {
	email = SanitizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return err
	}

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.Email == email {
		return nil
	}

	// Friendly duplicate check; the unique constraint stays the backstop.
	if existing, err := s.userStore.GetByEmail(ctx, email); err == nil {
		if existing.GetID() != userID {
			return ErrEmailAlreadyExists
		}
	} else if !isNotFound(err) {
		return err
	}

	// Reset verification before switching the address, so a failure leaves the
	// old address unverified rather than the new one wrongly verified.
	if s.emailVerificationReset != nil {
		if err := s.emailVerificationReset(ctx, userID, email); err != nil {
			return err
		}
	}

	user.Email = email
	user.UpdatedAt = time.Now()
	return s.UpdateUser(ctx, user)
}
