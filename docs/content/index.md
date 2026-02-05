---
seo:
  title: Aegis - Lightweight Authentication for Go
  description: A modular, database-agnostic authentication framework for Go with session management, CSRF protection, and official plugins for Email, SMS, OAuth, JWT, and more.
---

::u-page-hero
---
orientation: horizontal
class: "relative overflow-hidden pt-24 pb-48 hero-grid"
---

<div class="edge-glow-top"></div>

#title
<span class="text-black dark:text-white">Aegis</span>

#description
The most robust authentication framework for Go developers. Simple, modular, and built for performance.

#links
  :::u-button
  ---
  to: /get-started/installation
  class: "btn-primary translate-z-0"
  ---
  Get started
  <span class="i-lucide-arrow-right"></span>
  :::

  :::u-button
  ---
  to: https://github.com/theinventorylib/aegis
  variant: ghost
  class: "hover:opacity-60 transition-opacity font-medium"
  ---
  <span class="simple-icons-github mr-2"></span>
  GitHub
  :::

#image
<div class="relative group mt-8 sm:mt-0">
  <div class="relative bg-black/90 rounded-xl border border-white/5 overflow-hidden shadow-2xl backdrop-blur-sm">
    <div class="code-header">
      <div class="code-dots"><div class="dot dot-red"></div><div class="dot dot-yellow"></div><div class="dot dot-green"></div></div>
      <div class="code-filename">main.go</div>
    </div>
    <div class="p-8 font-mono text-[13px] leading-relaxed">
      <span class="text-blue-400">auth</span>, <span class="text-red-400">_</span> := aegis.<span class="text-yellow-400">New</span>(ctx, &config.Config{<br/>
      &nbsp;&nbsp;DB: &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;db,<br/>
      &nbsp;&nbsp;Secret: &nbsp;<span class="text-green-400">"secure-key"</span>,<br/>
      &nbsp;&nbsp;Plugins: []aegis.Plugin{<br/>
      &nbsp;&nbsp;&nbsp;&nbsp;email.<span class="text-yellow-400">New</span>(),<br/>
      &nbsp;&nbsp;&nbsp;&nbsp;oauth.<span class="text-yellow-400">New</span>(),<br/>
      &nbsp;&nbsp;},<br/>
      })
    </div>
  </div>
</div>
::

::u-page-section
---
class: "py-32 border-t border-white/5"
---

#title
<span class="text-blue-500 font-mono text-xs tracking-widest uppercase mb-4 block">Engineered for control</span>
Modern Auth for Go

#description
Aegis provides a lightweight core with a modular plugin architecture inspired by modern best practices.

#features
  :::u-page-feature
  ---
  icon: i-lucide-zap
  color: blue
  class: "border border-white/5 bg-white/5 p-8 rounded-2xl hover:border-blue-500/20 transition-all duration-300"
  ---
  #title
  Zero Magic
  
  #description
  No hidden auto-migrations. You own your schema. Full transparency and control over your database.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-shield-check
  color: blue
  class: "border border-white/5 bg-white/5 p-8 rounded-2xl hover:border-blue-500/20 transition-all duration-300"
  ---
  #title
  Security First
  
  #description
  Argon2id, CSRF protection, and secure sessions built-in. Following industry standard security patterns.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-gauge
  color: blue
  class: "border border-white/5 bg-white/5 p-8 rounded-2xl hover:border-blue-500/20 transition-all duration-300"
  ---
  #title
  Performance
  
  #description
  Type-safe Go API with minimal overhead. Optional Redis caching and efficient session tracking.
  :::
::

::u-page-section
#title
Core Features

#features
  :::u-page-feature
  ---
  icon: i-lucide-database
  ---
  #title
  Database Agnostic
  
  #description
  Works seamlessly with PostgreSQL, MySQL, and SQLite. Use the database you already love with our minimal core schema.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-shield-check
  ---
  #title
  Secure Sessions
  
  #description
  Secure token-based sessions with refresh tokens, Redis caching support, and built-in CSRF protection for web apps.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-puzzle
  ---
  #title
  Modular Plugins
  
  #description
  Extend Aegis with official plugins for Email OTP, SMS, OAuth, JWT, Bearer tokens, Admin management, and multi-tenant Organizations.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-terminal
  ---
  #title
  Powerful CLI
  
  #description
  Manage your database migrations easily. Export migrations in SQL, Goose, or golang-migrate formats for core and any active plugins.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-code-2
  ---
  #title
  Developer Friendly
  
  #description
  No auto-migration magic. Fully typed Go API designed for clarity and control. Focus on building your app, not your auth.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-file-json
  ---
  #title
  OpenAPI Integration
  
  #description
  Auto-generate interactive API documentation with Scalar UI using our official OpenAPI plugin.
  :::
::

::u-page-section
---
class: bg-gray-50 dark:bg-gray-900/50
---

#title
Official Plugins

#description
Extend Aegis with our growing ecosystem of official plugins. Each plugin is self-contained with its own schema, handlers, and migrations.

#features
  :::u-page-feature
  ---
  icon: i-lucide-mail
  color: blue
  ---
  #title
  Email & SMS
  
  #description
  OTP-based verification and email+password authentication for secure user onboarding.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-share-2
  color: cyan
  ---
  #title
  OAuth
  
  #description
  Social login support for Google, GitHub, and more popular providers.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-key
  color: indigo
  ---
  #title
  JWT & Bearer
  
  #description
  Standardized token generation, validation, and rotation for API authentication.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-user-cog
  color: sky
  ---
  #title
  Admin
  
  #description
  Administrative endpoints for user and session management with audit logging.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-users
  color: violet
  ---
  #title
  Organizations
  
  #description
  Multi-tenant support with teams, roles, and member management.
  :::

  :::u-page-feature
  ---
  icon: i-lucide-file-code
  color: teal
  ---
  #title
  OpenAPI
  
  #description
  Interactive documentation for your auth endpoints with Scalar UI.
  :::
::

::u-page-section
#title
Quick Start Example

#description
Get started with Aegis in just a few lines of code.

::code-group
```go [main.go]
package main

import (
    "database/sql"
    "log"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    _ "github.com/lib/pq"
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/core"
    "github.com/theinventorylib/aegis/router"
)

func main() {
    // Connect to database
    db, _ := sql.Open("postgres", 
        "postgres://localhost/aegis?sslmode=disable")
    defer db.Close()
    
    // Initialize Aegis
    auth := aegis.New(&core.Config{
        DB:      db,
        Secret:  "your-secret-key-32-chars-minimum!",
        BaseURL: "http://localhost:8080",
    })
    
    // Create router
    r := chi.NewRouter()
    
    // Mount auth routes
    router.Mount(r, auth)
    
    // Protected route
    r.Group(func(r chi.Router) {
        r.Use(auth.RequireAuth())
        r.Get("/api/profile", profileHandler)
    })
    
    // Start server
    log.Println("Server running on :8080")
    http.ListenAndServe(":8080", r)
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
    user := core.GetUser(r.Context())
    w.Write([]byte("Hello " + user.Email))
}
```

```bash [Install]
# Install Aegis
go get github.com/theinventorylib/aegis

# Install CLI
go install github.com/theinventorylib/aegis/cmd/aegis@latest

# Export migrations
aegis migrate export \
  --driver postgres \
  --format sql \
  --out ./migrations
```

```bash [Usage]
# Sign up
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123!"}'

# Sign in
curl -X POST http://localhost:8080/auth/signin \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123!"}' \
  -c cookies.txt

# Access protected route
curl http://localhost:8080/api/profile -b cookies.txt
```
::
::

::u-page-section
---
align: center
class: bg-gradient-to-b from-gray-50 to-white dark:from-gray-900/50 dark:to-gray-950
---

#title
Ready to get started?

#description
Install Aegis and build secure authentication in minutes. Join the community and start building.

#links
  :::u-button
  ---
  color: primary
  size: lg
  to: /get-started/installation
  trailing-icon: i-lucide-arrow-right
  ---
  Get started
  :::

  :::u-button
  ---
  color: neutral
  size: lg
  to: /get-started/quickstart
  variant: outline
  icon: i-lucide-zap
  ---
  5-Minute Tutorial
  :::

  :::u-button
  ---
  color: neutral
  size: lg
  to: https://github.com/theinventorylib/aegis
  variant: ghost
  icon: simple-icons-github
  ---
  View on GitHub
  :::

