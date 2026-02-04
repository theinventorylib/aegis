# Core Concepts

Aegis uses a few central concepts to manage authentication.

## User
A **User** is the central identity in the system. A user can have multiple [Accounts](#account) (authentication methods) linked to them.

## Account
An **Account** represents a way a user can log in.
- **Password**: An account with a hashed password.
- **OAuth**: An account linked to a third-party provider (Google, GitHub).

A user can have multiple accounts (e.g., they can have a password login AND a Google login).

## Session
A **Session** is created when a user successfully logs in. It is represented by a secure token, usually stored in a cookie or sent as a header.
Sessions are stored in the database but can be cached in Redis for performance.

## Verification
**Verification** records are short-lived tokens used for email verification, password resets, or OTPs. They have a strict expiry time.
