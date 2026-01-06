package core

import (
	"context"
	"time"

	"github.com/theinventorylib/aegis/auth"
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
}

// NewUserService creates a new user service with the specified dependencies.
func NewUserService(userStore auth.UserStore, accountStore auth.AccountStore, sessionStore auth.SessionStore, hashConfig *PasswordHasherConfig, authConfig *AuthConfig, auditLogger AuditLogger) *UserService {
	return &UserService{
		userStore:    userStore,
		accountStore: accountStore,
		sessionStore: sessionStore,
		hashConfig:   hashConfig,
		authConfig:   authConfig,
		auditLogger:  auditLogger,
	}
}

// CreateUser creates a new user with a password-based authentication account.
//
// This is the primary method for email/password signup flows. It:
//  1. Assigns a unique ID if not provided
//  2. Sets creation and update timestamps
//  3. Persists the user to storage
//  4. Hashes the password using Argon2id
//  5. Creates a password-based account linked to the user
//
// The password is hashed with the configured Argon2id parameters before storage.
// The account is created with provider="credentials" to distinguish it from
// OAuth accounts.
//
// Parameters:
//   - ctx: Request context for cancellation
//   - user: User model with email, name, etc. (ID optional)
//   - password: Plaintext password to hash and store
//
// Returns the created user. If creation fails, the user account is not created
// (no partial state).
func (s *UserService) CreateUser(ctx context.Context, user auth.User, password string) (auth.User, error) {
	if user.ID == "" {
		user.ID = GenerateID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.UpdatedAt = time.Now()

	u, err := s.userStore.Create(ctx, user)
	if err != nil {
		return u, err
	}

	uid := u.GetID()

	hashedPassword, err := HashPassword(password, s.hashConfig.Argon2Time, s.hashConfig.Argon2Memory, s.hashConfig.Argon2Threads, s.hashConfig.Argon2KeyLength)
	if err != nil {
		return u, err
	}

	account := auth.Account{
		ID:           GenerateID(),
		UserID:       uid,
		Provider:     PasswordProvider,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.accountStore.Create(ctx, account); err != nil {
		return u, err
	}

	return u, nil
}

// CreateUserWithEmail is a convenience method for creating a user with email/password.
//
// This is a simplified wrapper around CreateUser for the common case of email/password
// signup. It constructs the user model from individual fields and delegates to CreateUser.
//
// Parameters:
//   - ctx: Request context
//   - name: User's display name
//   - email: User's email address (should be unique)
//   - password: Plaintext password
//
// Example:
//
//	user, err := userService.CreateUserWithEmail(ctx, "John Doe", "john@example.com", "secret123")
func (s *UserService) CreateUserWithEmail(ctx context.Context, name, email, password string) (auth.User, error) {
	user := auth.User{
		Name:  name,
		Email: email,
	}
	return s.CreateUser(ctx, user, password)
}

// CreateUserWithoutPassword creates a new user without any authentication account.
//
// This is used for OAuth-only users or when accounts will be created separately.
// Common scenarios:
//   - OAuth signup (Google, GitHub, etc.) where password is not needed
//   - Admin-created users where credentials are set later
//   - Service accounts or system users
//
// Note: The user won't be able to log in with email/password until a password
// account is created separately.
//
// Parameters:
//   - ctx: Request context
//   - user: User model (ID will be generated if not provided)
func (s *UserService) CreateUserWithoutPassword(ctx context.Context, user auth.User) (auth.User, error) {
	if user.ID == "" {
		user.ID = GenerateID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.UpdatedAt = time.Now()

	return s.userStore.Create(ctx, user)
}

// DeleteUser deletes a user and all associated data (accounts and sessions).
//
// This performs a cascading delete to maintain referential integrity:
//  1. Delete all sessions (logs user out from all devices)
//  2. Delete all accounts (credentials, OAuth connections, etc.)
//  3. Delete the user record itself
//
// The deletion order ensures that foreign key constraints are satisfied.
// If any step fails, subsequent steps are still attempted (best-effort cleanup).
//
// Parameters:
//   - ctx: Request context
//   - id: User ID to delete
//
// Returns an error only if the user deletion itself fails. Session and account
// deletion errors are logged but don't fail the operation.
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	// First delete all sessions to log user out everywhere
	if err := s.sessionStore.DeleteByUserID(ctx, id); err != nil {
		return err
	}

	// Then delete all linked accounts (credentials, OAuth, etc.)
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
	return s.userStore.Update(ctx, user)
}

// UpdateUserEmail updates a user's email
func (s *UserService) UpdateUserEmail(ctx context.Context, userID, email string) error {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	user.Email = email
	user.UpdatedAt = time.Now()
	return s.UpdateUser(ctx, user)
}
