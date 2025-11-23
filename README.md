# Aegis

**Aegis** is a lightweight, production-ready authentication framework for Go with a **modular plugin architecture**.

[![CI](https://github.com/theinventorylib/aegis/actions/workflows/ci.yml/badge.svg)](https://github.com/theinventorylib/aegis/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/theinventorylib/aegis.svg)](https://pkg.go.dev/github.com/theinventorylib/aegis)

---

## 📚 Documentation

**[Visit the Full Documentation Site](https://theinventorylib.github.io/aegis/)** (or browse the [`docs/`](./docs) folder locally).

### Quick Links

- [**Getting Started**](./docs/getting-started.md)
- [**Core Concepts**](./docs/core-concepts.md)
- [**Plugins**](./docs/plugins.md)
- [**CLI Reference**](./docs/cli.md)
- [**Contributing**](./docs/contributing.md)

---

## ✨ Features

- **Minimal Core Schema**: Only 5 essential tables.
- **Database Agnostic**: Works with PostgreSQL, MySQL, SQLite.
- **Plugin System**: Email, SMS, OAuth, Password, and more.
- **Secure**: Argon2id hashing, JWT sessions, CSRF protection.
- **Developer Friendly**: No auto-migration magic, fully typed API.

## 🚀 Quick Install

```bash
go get github.com/theinventorylib/aegis
```

For the CLI tool:

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING](./docs/contributing.md) for details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.
