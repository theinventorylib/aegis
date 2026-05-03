package auth

import authtypes "github.com/theinventorylib/aegis/auth/types"

// Interface aliases re-export the storage interfaces from auth/types at the
// top-level auth package so callers do not need to import the sub-package.

// UserStore defines the persistence interface for User records: create, fetch
// by ID and email, update, soft-delete, paginated list, and count. Implement
// this interface to replace the built-in SQL store with a custom backend.
type UserStore = authtypes.UserStore

// AccountStore defines the persistence interface for Account records: create,
// fetch by ID, by user ID, and by provider+providerAccountID, update, and
// delete. Implement this to manage provider-linked accounts in a custom store.
type AccountStore = authtypes.AccountStore

// VerificationStore defines the persistence interface for Verification tokens:
// create, lookup by token, list by identifier, invalidate, delete, and clean up
// expired tokens. Used by email verification, password reset, and OTP flows.
type VerificationStore = authtypes.VerificationStore

// SessionStore defines the persistence interface for Session records: create,
// fetch by ID, token, and refresh token, list by user ID, delete, and clean up
// expired sessions. Implement this to manage sessions in a custom store.
type SessionStore = authtypes.SessionStore

// Transactor is an optional capability re-exported from auth/types. Stores
// that support cross-store transactions implement this; callers should treat
// support as optional via Auth.Transactor.
type Transactor = authtypes.Transactor

// Tx is a re-export of the transaction handle returned by Transactor.BeginTx.
type Tx = authtypes.Tx
