$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$resultsDir = Join-Path $repoRoot "docs\test-results"
$resultsFile = Join-Path $resultsDir "localstack-smoke-latest.txt"

New-Item -ItemType Directory -Force -Path $resultsDir | Out-Null

Push-Location $repoRoot
try {
    go test ./examples/apigateway-lambda-keycloak -tags localstack -v | Tee-Object -FilePath $resultsFile
}
finally {
    Pop-Location
}
