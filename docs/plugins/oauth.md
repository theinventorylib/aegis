# OAuth Plugin

The OAuth plugin enables social login with providers like Google, GitHub, Apple, and more. It leverages [Goth](https://github.com/markbates/goth) under the hood.

## Installation

```bash
go get github.com/theinventorylib/aegis/plugins/oauth
```

## Configuration

```go
import (
    "github.com/theinventorylib/aegis/plugins/oauth"
    "github.com/markbates/goth/providers/google"
)

oauthPlugin := oauth.New(&oauth.Config{
    DB: dbProvider,
    Providers: []goth.Provider{
        google.New(clientKey, secret, callbackURL),
    },
})
```

## Usage

Exposes endpoints for:

- `/auth/oauth/{provider}`: Initiates the OAuth flow
- `/auth/oauth/{provider}/callback`: Handles the callback
