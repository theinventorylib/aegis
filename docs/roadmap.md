# Aegis TODO

## v1.0.0 Release - Code Quality Improvements

### Quick Wins (COMPLETED ✅)
These easy fixes reduced linting issues from 71 to 66:

- [x] **goconst (1 issue)** - ✅ Extracted role constants in `plugins/organizations/db.go`
  - Added `RoleOwner`, `RoleAdmin`, `RoleMember` constants
  
- [x] **godot (3 issues in production)** - ✅ Fixed comment periods in `db/provider.go`
  - Fixed Row, Rows, and Result interface comments
  - Note: 3 new godot issues in `examples/` (excluded)

- [x] **prealloc (1 issue)** - ✅ Preallocated slice in `core/testing.go:53`
  - Optimized memory allocation in MockDB.ListUsers

### Documentation (30 minutes)
Add missing godoc comments for exported APIs:

- [ ] **revive (43 issues)** - Add godoc comments to exported functions/types
  - `core/keys.go` - NewStaticKeyManager, NewRedisKeyManager, methods
  - `core/testing.go` - NewMockDB
  - `core/types.go` - User type comment
  - `core/middleware.go` - Fix unused parameter warnings
  - `db/provider.go` - Add package comment
  - `migrations/core.go` - Add comments for exported functions
  - `migrations/exporter.go` - Add comments for exported functions
  - `plugins/` - Add comments for various exported methods
  - Fix stuttering names (e.g., `db.DBProvider` -> consider `db.Provider`)
  - Remove unused parameters or rename to `_`

### Code Refactoring (Optional)
Review code duplication for potential refactoring:

- [ ] **dupl (7 issues)** - Review and potentially refactor duplicated code
  - `db/sql.go` - GetSession vs GetSessionByRefreshToken (lines 349-408)
  - `plugins/oauth/db.go` - GetConnection methods (lines 55-112)
  - `plugins/organizations/handlers.go` - Handler auth patterns (lines 63-396)
  
  **Note**: These may be intentionally similar with slight differences. Assess if abstraction improves or complicates code.

### Security Review (Already Assessed)
These gosec warnings are false positives but documented for reference:

- [x] **gosec (7 issues)** - Reviewed, all acceptable
  - `core/password.go:109` - Integer conversion is safe (validated values)
  - `examples/` - HTTP timeouts - demo code only
  - `migrations/exporter.go` - File permissions (0755/0644) - migration files need to be readable

### Non-Issues
These are either already excluded or intentionally configured:

- [x] **errcheck (9 issues)** - 6 in `examples/` (excluded), 3 others already have `_` prefix
  - Examples are excluded per `.golangci.yml` configuration

---

## Notes

- **Beta.2 Release**: All CI-blocking issues resolved, safe to ship
- **v1.0.0 Release**: Address documentation and quick wins for polish
- Total issues: 71 (down from 91 initially)
  - 9 errcheck (6 in excluded examples)
  - 43 revive (docs/style)
  - 7 dupl (acceptable patterns)
  - 7 gosec (false positives)
  - 3 godot (cosmetic)
  - 1 goconst (minor)
  - 1 prealloc (micro-optimization)
