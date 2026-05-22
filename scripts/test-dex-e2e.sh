#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/docker-compose.dex.yml"

cleanup() {
  docker compose -f "$COMPOSE_FILE" down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker compose -f "$COMPOSE_FILE" up -d

for i in {1..30}; do
  if curl -fsS "http://127.0.0.1:5556/dex/.well-known/openid-configuration" >/dev/null; then
    break
  fi
  sleep 1
  if [[ "$i" == "30" ]]; then
    echo "Dex did not become ready in time" >&2
    exit 1
  fi
done

cd "$ROOT_DIR"
go test ./... -tags dex_e2e -v
