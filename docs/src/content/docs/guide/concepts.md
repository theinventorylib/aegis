---
title: Core Concepts
description: Understanding Users, Accounts, and Sessions.
---

Aegis uses a few central concepts to manage authentication.

## User
A **User** is the central identity in the system. A user can have multiple [Accounts](#account) (authentication methods) linked to them.

## Account
An **Account** represents a way a user can log in.
- **Password**: An account with a hashed password.
- **OAuth**: An account linked to a third-party provider (Google, GitHub).

## Session
A **Session** is created when a user successfully logs in. It is represented by a secure token, usually stored in a cookie.

## Verification
**Verification** records are short-lived tokens used for email verification, password resets, or OTPs.
