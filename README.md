# WSAuthKit

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logo/logo-dark.png">
    <img src="./assets/logo/logo-light.png" alt="WSAuthKit logo" width="640">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/elton-peixoto-lu/wsauthkit/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/elton-peixoto-lu/wsauthkit/ci.yml?branch=main&style=for-the-badge&label=CI"></a>
  <a href="https://github.com/elton-peixoto-lu/wsauthkit/releases"><img alt="Version" src="https://img.shields.io/github/v/release/elton-peixoto-lu/wsauthkit?display_name=tag&sort=semver&style=for-the-badge&label=Version"></a>
  <a href="https://pkg.go.dev/github.com/elton-peixoto-lu/wsauthkit"><img alt="Go Reference" src="https://img.shields.io/badge/go-reference-007d9c?style=for-the-badge&logo=go"></a>
  <a href="https://proxy.golang.org/github.com/elton-peixoto-lu/wsauthkit/@v/list"><img alt="Go Module" src="https://img.shields.io/badge/go-module-0A1F44?style=for-the-badge&logo=go"></a>
  <a href="https://goreportcard.com/report/github.com/elton-peixoto-lu/wsauthkit"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/elton-peixoto-lu/wsauthkit?style=for-the-badge"></a>
  <a href="./LICENSE"><img alt="License" src="https://img.shields.io/badge/%E2%9A%96%20License-MIT-0A1F44?style=for-the-badge"></a>
</p>

<p align="center">
  Secure WebSocket JWT authentication middleware for Go.
</p>

<p align="center">
  &#x2696; Released under the MIT License
</p>

`WSAuthKit` is a focused Go library that standardizes secure WebSocket authentication without turning your service into an auth framework.

It keeps JWT parsing, issuer validation, audience validation, token extraction, and claim injection out of handlers so real-time services stay small, readable, and consistent.

`WSAuthKit` only handles authentication concerns during the WebSocket handshake. It does not manage connections, rooms, presence, sessions, or message routing.

## Why WSAuthKit

WebSocket authentication is often implemented differently in every service:

- some handlers only validate the token signature
- others forget issuer or audience checks
- `Sec-WebSocket-Protocol` token extraction is easy to miss
- claim parsing logic leaks into application handlers
- gateway and browser handshake edge cases create duplicated code

`WSAuthKit` solves that with one production-oriented middleware layer built specifically for Go backends.

## What WSAuthKit Solves

- eliminates duplicated JWT extraction and validation logic across WebSocket handlers
- enforces consistent `issuer`, `audience`, signature, and expiry validation
- supports handshake patterns commonly used behind gateways and proxies
- keeps handlers focused on business logic by injecting validated claims into context
- provides one reusable auth path for both `net/http` servers and AWS API Gateway WebSocket connect events

## Operational Benefits

- prevents authentication drift across teams and services by standardizing one handshake auth pipeline
- reduces regression risk with repeatable test layers for extraction, validation, and context injection
- centralizes JWT policy updates in one place instead of spreading changes across multiple handlers
- speeds up onboarding by giving new contributors a clear and reusable auth integration path
- keeps code reviews focused on product behavior instead of repeated JWT plumbing

## Features

- JWT validation with `SigningKey`, custom `KeyFunc`, or remote `JWKSURL`
- issuer and audience validation
- provider-agnostic OIDC/JWT support (Dex, OIDC, Auth0, Cognito, Entra ID, etc.)
- token extraction from `Authorization` header
- token extraction from `Sec-WebSocket-Protocol`
- request context claim injection
- standalone auth flow support through the public API
- optional authorization primitives for RBAC and ReBAC composition
- functional and end-to-end test coverage

## Installation

```bash
go get github.com/elton-peixoto-lu/wsauthkit
```

## Check Version

Published module pages:

- `pkg.go.dev`: https://pkg.go.dev/github.com/elton-peixoto-lu/wsauthkit
- `proxy.golang.org` versions list: https://proxy.golang.org/github.com/elton-peixoto-lu/wsauthkit/@v/list

Available versions:

```bash
go list -m -versions github.com/elton-peixoto-lu/wsauthkit
```

Version currently used by your project:

```bash
go list -m github.com/elton-peixoto-lu/wsauthkit
```

## Quick Start

```go
package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/elton-peixoto-lu/wsauthkit"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

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

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	defer conn.Close()

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		reply := append([]byte("user="+claims.Subject+" "), payload...)
		if err := conn.WriteMessage(messageType, reply); err != nil {
			return
		}
	}
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

### Optional typed claim mapping

```go
type Identity struct {
	UserID   string
	TenantID string
	Roles    []string
}

identity, err := wsauthkit.MapClaims(claims, func(r wsauthkit.ClaimReader) (Identity, error) {
	tenantID, err := r.RequiredString("tenant_id")
	if err != nil {
		return Identity{}, err
	}
	roles, _ := r.Strings("roles")

	return Identity{
		UserID:   r.Subject(),
		TenantID: tenantID,
		Roles:    roles,
	}, nil
})
```

## Compatibility

- built on top of standard `net/http`
- intended to run before the WebSocket upgrade
- works naturally with `gorilla/websocket`
- fits API Gateway and proxy-driven handshake patterns
- includes a native `apigateway` adapter for AWS WebSocket connect events

## Examples

- basic authenticated echo server: [`examples/main.go`](./examples/main.go)
- API Gateway style subprotocol extraction with Gorilla WebSocket: [`examples/apigateway/main.go`](./examples/apigateway/main.go)
- API Gateway WebSocket Lambda with OIDC JWKS validation: [`examples/apigateway-lambda-oidc/main.go`](./examples/apigateway-lambda-oidc/main.go)
- API Gateway WebSocket Lambda with OIDC-style JWKS validation: [`examples/apigateway-lambda-oidc/main.go`](./examples/apigateway-lambda-oidc/main.go)

## AWS API Gateway

`WSAuthKit` now includes a focused adapter for API Gateway WebSocket connect events:

```go
auth, err := apigateway.NewAuth(apigateway.Config{
	Issuer:   "https://dex.example.com",
	Audience: "ws-backend",
	JWKSURL:  "https://dex.example.com/keys",
})

claims, err := auth.Authenticate(event)
```

This adapter is intentionally narrow:

- it authenticates API Gateway WebSocket connect events
- it extracts tokens from headers, query string, and `Sec-WebSocket-Protocol`
- it reuses the same JWT validation model as the core library
- it does not manage connection storage or API Gateway callback delivery
- it can be smoke-tested locally with the optional LocalStack flow under `examples/apigateway-lambda-oidc`

## Authorization Extensions (RBAC + ReBAC)

`WSAuthKit` keeps authentication and authorization decoupled.

Use `Authorizer` to plug your policy engine without coupling to a specific IdP:

```go
rbac := wsauthkit.RBACAuthorizer{
	RoleClaim: "roles",
	Policies: map[string][]string{
		"admin": {"*:*"},
		"viewer": {"read:workspace/123"},
	},
}

rebac := wsauthkit.ReBACAuthorizer{
	Checker: myRelationChecker, // e.g. OpenFGA/SpiceDB adapter
}

authorizer := wsauthkit.AnyAuthorizer{rbac, rebac}
allowed, err := authorizer.Authorize(ctx, wsauthkit.AuthorizationInput{
	Claims: claims,
	Action: "read",
	Resource: "workspace/123",
})
```

This pattern enables simple RBAC first, and gradual evolution to ReBAC for multi-tenant and graph-style access rules.

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

Or via `Makefile`:

```bash
make test
```

Integration tests:

```bash
go test ./... -tags integration
```

```bash
make test-integration
```

Functional tests:

```bash
go test ./... -tags functional
```

```bash
make test-functional
```

End-to-end WebSocket tests:

```bash
go test ./... -tags e2e
```

```bash
make test-e2e
```

Real Dex end-to-end test:

```bash
make test-dex-e2e
```

LocalStack smoke test:

```bash
go test ./examples/apigateway-lambda-oidc -tags localstack -v
```

```bash
make test-localstack
```

Run all suites:

```bash
go test ./...
go test ./... -tags integration
go test ./... -tags functional
go test ./... -tags e2e
```

```bash
make test-all
```

Release validation:

```bash
make release-check
```

Environment-specific smoke test notes live in [`docs/test-results/localstack-smoke.md`](./docs/test-results/localstack-smoke.md).

Consolidated AWS + OIDC validation results live in [`docs/test-results/aws-oidc-test-matrix.md`](./docs/test-results/aws-oidc-test-matrix.md).

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

## Changelog

Release history lives in [`CHANGELOG.md`](./CHANGELOG.md).

## Versioning

`WSAuthKit` follows semantic versioning with a practical early-stage policy:

- `v0.x` while the API is still maturing
- patch releases for fixes and non-breaking maintenance
- minor releases for additive, backwards-compatible features
- a future major release when the public API is considered stable and long-lived

## Launch Copy

Short launch post drafts live under [`docs/`](./docs/).

## License

MIT
