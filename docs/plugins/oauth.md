# OAuth Plugin

The OAuth plugin enables social login with providers like Google, GitHub, and more.

## Installation

```go
import "github.com/theinventorylib/aegis/plugins/oauth"
```

## Usage

```go
cfg := oauth.Config{
    Providers: []oauth.Provider{
        oauth.NewGoogleProvider(clientID, clientSecret, callbackURL),
        oauth.NewGitHubProvider(clientID, clientSecret, callbackURL),
    },
}

oauthPlugin := oauth.New(cfg, nil, plugins.DialectPostgres)
aegis.Use(ctx, oauthPlugin)
```
