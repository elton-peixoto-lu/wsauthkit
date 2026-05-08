# WSAuthKit launch post

Acabei de publicar o `WSAuthKit`, uma biblioteca open source em Go para autenticação segura de WebSocket com JWT.

Problema que ela resolve:

- autenticação de WebSocket costuma ficar duplicada entre serviços
- extração de token no handshake é implementada de forma inconsistente
- `Authorization` e `Sec-WebSocket-Protocol` nem sempre são tratados direito
- issuer e audience acabam sendo esquecidos

O `WSAuthKit` mantém isso fora do handler e entrega um middleware pequeno, idiomático e focado:

- extração de token
- validação JWT
- validação de issuer e audience
- injeção de claims no contexto
- integração limpa com `net/http` e `gorilla/websocket`

Repo:
`https://github.com/elton-peixoto-lu/wsauthkit`

Instalação:
```bash
go get github.com/elton-peixoto-lu/wsauthkit
```

Feedback é muito bem-vindo.
