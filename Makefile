SHELL := /bin/sh

.PHONY: test test-functional test-e2e test-all release-check tidy

test:
	go test ./...

test-functional:
	go test ./... -tags functional

test-e2e:
	go test ./... -tags e2e

test-all: test test-functional test-e2e

release-check: tidy test-all

tidy:
	go mod tidy
