# LocalStack Smoke Test

This smoke test validates the closest local approximation of the AWS WebSocket connect flow currently covered by the repository:

1. start LocalStack with Lambda and API Gateway services
2. build the example Lambda from `examples/apigateway-lambda-keycloak`
3. expose a JWKS endpoint from the host to emulate Keycloak key discovery
4. create an API Gateway WebSocket API with a `$connect` Lambda integration
5. connect with a valid JWT and verify the WebSocket upgrade succeeds
6. connect without a token and verify the handshake is rejected

## Command

```powershell
pwsh -File .\scripts\run-localstack-smoke.ps1
```

## Prerequisites

- Docker Desktop or a compatible Docker engine
- Docker Compose
- LocalStack support for API Gateway V2 WebSocket APIs
- Docker socket mounted into the LocalStack container

## Notes

- LocalStack documents API Gateway V2 WebSocket support under the Base plan.
- This suite is optional and intentionally excluded from CI because it depends on Docker and LocalStack feature availability.
- The latest raw output is stored in `docs/test-results/localstack-smoke-latest.txt`.

## Latest Result

Latest execution on May 8, 2026 with `localstack/localstack-pro` and an activated license:

- LocalStack started successfully
- edition reported by `/_localstack/info`: `pro`
- `apigatewayv2` reported as `available`
- Lambda packaging and API provisioning completed
- WebSocket connect with a valid JWT succeeded
- unauthenticated connect was still accepted by LocalStack instead of surfacing the Lambda `401`

Interpretation:

- the smoke test validates the happy path against a real LocalStack WebSocket API
- the rejection path remains covered by the repository's integration and e2e tests
- the current LocalStack WebSocket emulation does not appear to enforce the `$connect` unauthorized response in the same way as AWS
