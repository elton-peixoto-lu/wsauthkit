# WSAuthKit launch post

I just published `WSAuthKit`, an open-source Go library for secure WebSocket JWT authentication.

Problem it solves:

- WebSocket auth logic often gets duplicated across services
- handshake token extraction is implemented inconsistently
- `Authorization` and `Sec-WebSocket-Protocol` handling is easy to get wrong
- issuer and audience validation are often skipped

`WSAuthKit` keeps that logic out of your handlers and provides a small, focused middleware layer for:

- token extraction
- JWT validation
- issuer and audience validation
- claim injection into request context
- clean integration with `net/http` and `gorilla/websocket`

Repository:
`https://github.com/elton-peixoto-lu/wsauthkit`

Install:
```bash
go get github.com/elton-peixoto-lu/wsauthkit
```

Feedback is very welcome.
