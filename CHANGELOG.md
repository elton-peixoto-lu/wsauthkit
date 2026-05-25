# Changelog

All notable changes to `WSAuthKit` will be documented in this file.

The format is inspired by Keep a Changelog and uses semantic versioning.

Current release policy:

- `v0.x` means the library is usable but still evolving
- patch releases are used for fixes and non-breaking adjustments
- minor releases are used for additive improvements
- a future `v1.0.0` will mark a more stable public API contract

## [v0.3.0] - 2026-05-25

### Added

- optional typed claims mapping helper layer with `MapClaims[T]`
- `ClaimReader` typed accessors for common claim projections (`String`, `RequiredString`, `Strings`, `Bool`, `Int64`)
- tests and README example for mapping validated claims into application-specific structs

## [v0.2.1] - 2026-05-22

### Fixed

- renamed the matrix artifact to `docs/test-results/aws-oidc-test-matrix.md` for OIDC-consistent naming
- updated README and documentation references to the new matrix filename

## [v0.2.0] - 2026-05-22

### Added

- provider-agnostic authorization primitives with `RBACAuthorizer`, `ReBACAuthorizer`, and `AnyAuthorizer`
- real Dex end-to-end test flow with Docker (`docker-compose.dex.yml`, `scripts/test-dex-e2e.sh`, `dex_e2e` test tag)
- Dex-style functional coverage (`issuer` + JWKS `/keys`) in core and API Gateway adapter suites

### Changed

- renamed AWS Lambda example from `apigateway-lambda-keycloak` to `apigateway-lambda-oidc`
- standardized OIDC-focused naming across examples and smoke-test scripts
- updated README test commands and examples for provider-agnostic usage

## [v0.1.3] - 2026-05-08

### Added

- consolidated AWS + OIDC validation matrix in `docs/test-results/aws-oidc-test-matrix.md`
- explicit README section describing the concrete problems WSAuthKit solves
- LocalStack smoke-test artifacts and execution notes under `docs/test-results/`

## [v0.1.2] - 2026-05-08

### Fixed

- corrected the published Go module path to `github.com/elton-peixoto-lu/wsauthkit`
- aligned README installation instructions and public links with the canonical lowercase repository path

## [v0.1.1] - 2026-05-08

### Changed

- renamed the repository to lowercase for better Go ecosystem compatibility
- updated module references, badges, and release documentation toward the canonical path

## [v0.1.0] - 2026-05-08

### Added

- initial public release of `WSAuthKit`
- JWT authentication middleware for WebSocket handshakes
- `Authorization` and `Sec-WebSocket-Protocol` token extraction
- issuer and audience validation
- unit, functional, and end-to-end tests
- branding assets and CI workflow
