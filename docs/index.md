---
layout: home

hero:
  name: "Aegis"
  text: "The Modular Auth Framework for Go"
  tagline: "Type-safe, database-agnostic, and built for performance."
  image:
    src: /logo.png
    alt: Aegis Logo
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/theinventorylib/aegis

features:
  - title: Modular Architecture
    details: Extend functionality with 8+ official plugins including OAuth, JWT, and Multi-tenancy that you can opt-in to.
  - title: Type-Safe & Compiled
    details: Built with Go idioms in mind. No runtime schema magic—everything is strongly typed and checked at compile time.
  - title: Database Agnostic
    details: First-class support for PostgreSQL, MySQL, and SQLite using sqlc for type-safe queries.
  - title: High Performance
    details: Designed for speed with minimal allocation overhead and efficient session management using Redis.
---
