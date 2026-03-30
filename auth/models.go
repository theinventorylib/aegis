package auth

import authtypes "github.com/theinventorylib/aegis/auth/types"

// Type aliases re-export the core domain types from auth/types at the top-level
// auth package so callers can use auth.User, auth.Account, etc. without importing
// the sub-package directly.

// User is the core Aegis user identity model. It holds the user's unique ID,
// display Name, Email, Avatar URL, Disabled status, extensible Metadata, and
// CreatedAt/UpdatedAt timestamps. A user may have multiple linked Accounts, one
// per authentication provider.
type User = authtypes.User

// Account represents a provider-specific authentication account linked to a User.
// One user may have multiple Accounts — one per provider (e.g., email/password,
// Google, GitHub). Each Account stores the provider name, provider-assigned user
// ID, credential or token data, and optional expiry information.
type Account = authtypes.Account

// Verification is a time-limited token used for email verification, password
// reset, OTP delivery, and other flows that require confirming an identifier
// before proceeding. Tokens are scoped by Identifier, Type, and ExpiresAt.
type Verification = authtypes.Verification

// Session represents an authenticated user session. It holds a session token,
// an optional refresh token, an expiry time, and optional metadata such as the
// client IP address and user-agent string.
type Session = authtypes.Session
