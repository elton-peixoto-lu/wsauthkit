# AWS + Keycloak Test Matrix

This document consolidates the latest validation results for the `WSAuthKit` AWS API Gateway + Keycloak flow.

Execution date: **May 8, 2026**

## Scope

- library: `github.com/elton-peixoto-lu/wsauthkit`
- AWS adapter: `github.com/elton-peixoto-lu/wsauthkit/apigateway`
- example app: `examples/apigateway-lambda-keycloak`

## Result Summary

- `unit`: PASS
- `integration`: PASS
- `functional`: PASS
- `e2e`: PASS
- `localstack smoke`: PASS with one expected `SKIP` in the unauthenticated path due to LocalStack WebSocket emulation behavior

## Command Results

### Unit

Command:

```bash
go test ./apigateway -v
```

Observed:

```text
PASS
ok  	github.com/elton-peixoto-lu/wsauthkit/apigateway	2.259s
```

Validated behaviors:

- token extraction from `Authorization`
- token extraction from query string
- token extraction from `Sec-WebSocket-Protocol`
- invalid token rejection
- unauthorized error mapping

### Integration

Command:

```bash
go test ./examples/apigateway-lambda-keycloak -tags integration -v
```

Observed:

```text
accepted websocket connection id=connection-integration sub=integration-user
PASS
ok  	github.com/elton-peixoto-lu/wsauthkit/examples/apigateway-lambda-keycloak	2.638s
```

Validated behaviors:

- Lambda connect handler + adapter wiring
- authenticated connect path
- missing-token rejection path

### Functional

Command:

```bash
go test ./apigateway -tags functional -v
```

Observed:

```text
--- PASS: TestFunctionalAuthenticateWithRemoteJWKS (0.07s)
--- PASS: TestFunctionalAuthenticateWithCustomQueryParameterName (0.09s)
PASS
ok  	github.com/elton-peixoto-lu/wsauthkit/apigateway	2.518s
```

Validated behaviors:

- remote JWKS retrieval and signature validation (`RS256`)
- issuer and audience checks
- custom query parameter extraction

### End-to-End

Command:

```bash
go test ./examples/apigateway-lambda-keycloak -tags e2e -v
```

Observed:

```text
accepted websocket connection id=connection-123 sub=lambda-user
PASS
ok  	github.com/elton-peixoto-lu/wsauthkit/examples/apigateway-lambda-keycloak	0.374s
```

Validated behaviors:

- full connect handler flow with Keycloak-style JWT + JWKS
- invalid-token rejection path

### LocalStack Smoke

Command:

```bash
go test ./examples/apigateway-lambda-keycloak -tags localstack -count=1 -v
```

Observed:

```text
=== RUN   TestLocalStackWebSocketConnectFlow/accepts_valid_authorization_header
=== RUN   TestLocalStackWebSocketConnectFlow/rejects_missing_token
    localstack_test.go:74: localstack accepted unauthenticated $connect; integration and e2e tests cover the rejection path in-process
--- PASS: TestLocalStackWebSocketConnectFlow (72.78s)
    --- PASS: TestLocalStackWebSocketConnectFlow/accepts_valid_authorization_header (0.02s)
    --- SKIP: TestLocalStackWebSocketConnectFlow/rejects_missing_token (0.05s)
PASS
ok  	github.com/elton-peixoto-lu/wsauthkit/examples/apigateway-lambda-keycloak	73.186s
```

Environment facts:

- LocalStack image: `localstack/localstack-pro:latest`
- LocalStack edition: `pro`
- license: activated
- `apigatewayv2`: available

Interpretation:

- the happy path was validated against a real LocalStack WebSocket API
- LocalStack accepted unauthenticated `$connect` in this setup
- strict unauthenticated rejection remains validated by integration/e2e suites in-process

Raw output:

- `docs/test-results/localstack-smoke-latest.txt`

## What WSAuthKit Solved (Evidence)

1. Removed JWT parsing and claim plumbing from handlers.
2. Standardized handshake token extraction across `Authorization`, query string, and `Sec-WebSocket-Protocol`.
3. Enforced issuer/audience/signature checks in one reusable authentication layer.
4. Enabled the same auth model in `net/http` and AWS API Gateway WebSocket events.
5. Added repeatable verification layers (`unit`, `integration`, `functional`, `e2e`, `localstack smoke`) so regressions are detected early.
