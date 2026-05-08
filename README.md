# WSAuthKit

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logo/logo-dark.png">
    <img src="./assets/logo/logo-light.png" alt="WSAuthKit logo" width="640">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/elton-peixoto-lu/WSAuthKit/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/elton-peixoto-lu/WSAuthKit/ci.yml?branch=main&style=for-the-badge&label=CI"></a>
  <a href="https://github.com/elton-peixoto-lu/WSAuthKit/releases"><img alt="Version" src="https://img.shields.io/github/v/release/elton-peixoto-lu/WSAuthKit?display_name=tag&sort=semver&style=for-the-badge&label=Version"></a>
  <a href="https://pkg.go.dev/github.com/wsauthkit/wsauthkit"><img alt="Go Reference" src="https://img.shields.io/badge/go-reference-0A1F44?style=for-the-badge&logo=go"></a>
  <a href="https://goreportcard.com/report/github.com/elton-peixoto-lu/WSAuthKit"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/elton-peixoto-lu/WSAuthKit?style=for-the-badge"></a>
  <a href="./LICENSE"><img alt="License" src="https://img.shields.io/badge/%E2%9A%96%20License-MIT-0A1F44?style=for-the-badge"></a>
</p>

<p align="center">
  Secure WebSocket JWT authentication middleware for Go.
</p>

`WSAuthKit` is a focused Go library that standardizes secure WebSocket authentication without turning your service into an auth framework.

It keeps JWT parsing, issuer validation, audience validation, token extraction, and claim injection out of handlers so real-time services stay small, readable, and consistent.

## Why WSAuthKit

WebSocket authentication is often implemented differently in every service:

- some handlers only validate the token signature
- others forget issuer or audience checks
- `Sec-WebSocket-Protocol` token extraction is easy to miss
- claim parsing logic leaks into application handlers
- gateway and browser handshake edge cases create duplicated code

`WSAuthKit` solves that with one production-oriented middleware layer built specifically for Go backends.

## Features

- JWT validation with `SigningKey`, custom `KeyFunc`, or remote `JWKSURL`
- issuer and audience validation
- token extraction from `Authorization` header
- token extraction from `Sec-WebSocket-Protocol`
- request context claim injection
- standalone auth flow support through the public API
- functional and end-to-end test coverage

## Installation

```bash
go get github.com/wsauthkit/wsauthkit
```

## Quick Start

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

Authentication flow handled by the middleware:

1. extract token from the handshake request
2. validate JWT signature
3. validate issuer
4. validate audience
5. inject claims into request context
6. continue to the handler

## Supported Handshake Patterns

### Authorization header

```http
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Sec-WebSocket-Protocol

Useful behind API Gateway, browser-driven handshakes, and proxy layers:

```http
Sec-WebSocket-Protocol: graphql-ws, bearer, eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

`WSAuthKit` also supports compact token forms such as `bearer.<jwt>` when intermediaries serialize the token inline.

## Public API

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

```go
token, err := auth.ExtractToken(r)
claims, err := auth.ValidateToken(token)
claims, err = auth.Authenticate(r)
```

## Secure Defaults

- token expiration is required
- issued-at is validated
- issuer and audience checks are enforced when configured
- token internals are not leaked in default HTTP error responses
- the default extractor tries `Authorization` before `Sec-WebSocket-Protocol`
- JWKS-backed validators clean up their background refresh context on `Close()`

## Testing

Unit tests:

```bash
go test ./...
```

Functional tests:

```bash
go test ./... -tags functional
```

End-to-end WebSocket tests:

```bash
go test ./... -tags e2e
```

Run all suites:

```bash
go test ./...
go test ./... -tags functional
go test ./... -tags e2e
```

## Use Cases

- real-time dashboards
- WebSocket APIs
- chat systems
- notification systems
- API Gateway WebSocket integrations

## Project Layout

```text
wsauthkit/
|-- auth.go
|-- claims.go
|-- context.go
|-- errors.go
|-- extractor.go
|-- middleware.go
|-- validator.go
|-- examples/
|-- assets/
`-- scripts/
```

## Security

`WSAuthKit` is intended for infrastructure-grade backend services. Prefer:

- strong signing algorithms such as `RS256` or `ES256`
- strict issuer and audience validation
- remote key rotation through `JWKSURL` where appropriate
- short token lifetimes and explicit claim validation upstream

If you discover a security issue, avoid opening a public issue with exploit details. Share it privately with the repository maintainer first.

## Contributing

Issues and pull requests are welcome. Keep contributions aligned with the project scope:

- small
- idiomatic Go
- focused on WebSocket authentication
- low abstraction and low boilerplate

## Branding Assets

Open-source branding files live under [`assets/`](./assets/README.md).

## License

MIT
