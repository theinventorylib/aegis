# Bearer Plugin

The Bearer plugin adds support for authenticating requests using standard Bearer tokens.

## Installation

```go
import "github.com/theinventorylib/aegis/plugins/bearer"
```

## Usage

```go
bearerPlugin := bearer.New(bearerConfig)
aegis.Use(ctx, bearerPlugin)
```

## Description
This plugin is useful for machine-to-machine authentication or mobile apps that prefer long-lived tokens over cookies.
