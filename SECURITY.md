# Security Policy

## Supported versions

Security fixes are applied on the latest released version of `WSAuthKit`.

Older releases may not receive backported fixes.

## Reporting a vulnerability

If you believe you found a security issue, please do not open a public issue with exploit details.

Instead:

1. contact the repository maintainer privately
2. include a clear description of the issue
3. provide reproduction steps if possible
4. describe impact and any suggested mitigation

The goal is to validate the report quickly and coordinate a responsible fix before public disclosure.

## Scope

This library is focused on:

- JWT validation in WebSocket handshake flows
- issuer and audience validation
- extraction from `Authorization` and `Sec-WebSocket-Protocol`
- claim injection into request context

Reports outside that scope may be treated as general issues instead of security vulnerabilities.
