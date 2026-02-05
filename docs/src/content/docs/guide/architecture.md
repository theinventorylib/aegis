---
title: Architecture
description: Deep dive into Aegis architecture.
---

# Go-Native Auth System with Plugins, sqlc, and Strong Typing

This document describes a **Go-idiomatic architecture** for building an authentication system with **pluggable features**, **strong typing**, **high performance**, and **multi-dialect SQL support**.

## 1. Goals & Constraints

### Goals

- Allow users to **own and manage their database schema**
- Allow **plugins** to define and evolve their own schema
- Support schema extensions (e.g. roles, admin features)
- Preserve **strong typing and compile-time safety**
- Maintain **high performance** (close to raw SQL)

### Constraints (Go & sqlc)

- Types must be known at compile time
- sqlc runs at build time, not runtime
- Packages cannot regenerate code in dependent modules

## 3. High-Level Architecture

```
Application
└── wires core + plugins
    ├── Auth Core
    │   ├── Stable schema
    │   ├── Stable models
    │   └── Store interfaces
    ├── Plugins
    │   ├── Plugin schema
    │   ├── Plugin models
    │   └── Plugin sqlc
    └── Database
        └── User-managed migrations
```
