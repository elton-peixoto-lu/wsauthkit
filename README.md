# WSAuthKit

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logo/logo-dark.png">
    <img src="./assets/logo/logo-light.png" alt="WSAuthKit logo" width="640">
  </picture>
</p>

`WSAuthKit` is a small Go library for secure WebSocket authentication with JWT.

It standardizes the authentication flow that usually gets duplicated across handlers and gateway integrations:

1. extract the token from the handshake request;
2. validate the JWT signature;
3. validate issuer and audience;
4. inject claims into request context;
5. keep the WebSocket handler clean.

## Why it exists

WebSocket auth often drifts into ad-hoc code:

- each service parses headers a bit differently;
- `Sec-WebSocket-Protocol` support is easy to forget;
- handlers end up doing token work that does not belong there;
- issuer and audience checks get skipped or applied inconsistently.

`WSAuthKit` keeps that logic in one focused middleware package with production-oriented defaults.

## Features

- JWT validation with `SigningKey`, custom `KeyFunc`, or remote `JWKSURL`
- issuer and audience validation
- token extraction from `Authorization` header
- token extraction from `Sec-WebSocket-Protocol`
- request context claim injection
- composable middleware with a small API surface

## Installation

```bash
go get github.com/wsauthkit/wsauthkit
```

## Minimal usage

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/wsauthkit/wsauthkit"
)

func main() {
    auth, err := wsauthkit.NewAuth(wsauthkit.Config{
        Issuer:   "https://auth.company.com",
        Audience: "erp-backend",
        JWKSURL:  "https://auth.company.com/certs",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer auth.Close()

    http.Handle("/ws", auth.Middleware(http.HandlerFunc(wsHandler)))

    log.Fatal(http.ListenAndServe(":8080", nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
    claims := wsauthkit.MustClaims(r.Context())

    fmt.Println("user:", claims.Subject)
}
```

## Supported handshake patterns

### Authorization header

```http
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Sec-WebSocket-Protocol

Useful behind API Gateway or browser-driven handshake constraints:

```http
Sec-WebSocket-Protocol: graphql-ws, bearer, eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

`WSAuthKit` also accepts compact forms such as `bearer.<jwt>` when a proxy or client library serializes the token inline.

## API

### Config

```go
type Config struct {
    Issuer       string
    Audience     string
    JWKSURL      string
    SigningKey   any
    KeyFunc      KeyFunc
    Extractors   []TokenExtractor
    ErrorHandler ErrorHandler
}
```

### Standalone flow

You can also use the pieces directly:

```go
claims, err := auth.Authenticate(r)
if err != nil {
    // handle unauthorized
}
```

## Secure defaults

- token expiration is required
- issued-at is validated
- a small clock skew leeway is applied
- token internals are not leaked in HTTP error responses
- the default extractor tries `Authorization` before `Sec-WebSocket-Protocol`

## Testing

Run unit tests by default:

```bash
go test ./...
```

Run functional tests:

```bash
go test ./... -tags functional
```

Run end-to-end WebSocket tests:

```bash
go test ./... -tags e2e
```

Run all suites:

```bash
go test ./...
go test ./... -tags functional
go test ./... -tags e2e
```

## Use cases

- real-time dashboards
- WebSocket APIs
- chat systems
- notification systems
- API Gateway WebSocket integrations

## Project layout

```text
wsauthkit/
|-- auth.go
|-- claims.go
|-- context.go
|-- errors.go
|-- extractor.go
|-- middleware.go
|-- validator.go
`-- examples/
```

## Branding assets

Open-source branding files live under [`assets/`](./assets/README.md).

## License

MIT
