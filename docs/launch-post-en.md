# Launch post

I just published `WSAuthKit`, an open-source Go library for standardized WebSocket JWT authentication.

It comes from a very practical backend problem:

- WebSocket auth logic often gets duplicated across services
- every service extracts handshake tokens a little differently
- `Authorization` and `Sec-WebSocket-Protocol` handling is easy to get wrong
- issuer and audience validation are often inconsistent

`WSAuthKit` keeps that concern small and focused:

- token extraction during the handshake
- JWT validation
- issuer and audience validation
- claims injected into request context
- clean integration with `net/http` and `gorilla/websocket`

Repository:
`https://github.com/elton-peixoto-lu/wsauthkit`

Install:

```bash
go get github.com/elton-peixoto-lu/wsauthkit
```

Feedback and contributions are very welcome.
