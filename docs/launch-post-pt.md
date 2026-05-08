# Post de lancamento

Publiquei o `WSAuthKit`, uma biblioteca open source em Go para padronizar autenticacao de WebSocket com JWT.

A ideia veio de um problema recorrente em backend:

- autenticacao de WebSocket quase sempre fica duplicada
- cada servico extrai token de um jeito
- `Authorization` e `Sec-WebSocket-Protocol` costumam virar detalhe esquecido
- issuer e audience nem sempre sao validados com consistencia

O `WSAuthKit` resolve isso com uma camada pequena e focada:

- extracao de token no handshake
- validacao JWT
- validacao de issuer e audience
- claims no contexto
- integracao limpa com `net/http` e `gorilla/websocket`

Repositorio:
`https://github.com/elton-peixoto-lu/wsauthkit`

Instalacao:

```bash
go get github.com/elton-peixoto-lu/wsauthkit
```

Feedback e contribuicoes sao muito bem-vindos.
