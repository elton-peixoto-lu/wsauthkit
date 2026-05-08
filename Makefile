SHELL := /bin/sh

.PHONY: test test-integration test-functional test-e2e test-localstack test-all release-check tidy

test:
	go test ./...

test-integration:
	go test ./... -tags integration

test-functional:
	go test ./... -tags functional

test-e2e:
	go test ./... -tags e2e

test-localstack:
	go test ./examples/apigateway-lambda-keycloak -tags localstack -v

test-all: test test-integration test-functional test-e2e

release-check: tidy test-all

tidy:
	go mod tidy
