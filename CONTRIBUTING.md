# Contributing to WSAuthKit

Thanks for your interest in contributing to `WSAuthKit`.

This project aims to stay:

- small
- idiomatic Go
- focused on WebSocket authentication
- easy to adopt in production systems

Please keep that scope in mind when proposing changes.

## Before opening a pull request

Make sure your change:

- solves a real WebSocket authentication problem
- keeps the public API small and explicit
- avoids turning the library into a framework
- prefers composition over complex abstraction
- includes tests when behavior changes

## Local development

Run unit tests:

```bash
go test ./...
```

Run functional tests:

```bash
go test ./... -tags functional
```

Run end-to-end tests:

```bash
go test ./... -tags e2e
```

## Pull request guidelines

- keep PRs focused and reviewable
- explain the problem, not only the code change
- update documentation when the public API or behavior changes
- prefer straightforward naming over clever naming
- avoid unnecessary dependencies

## Good contribution examples

- improving token extraction behavior
- tightening validation defaults
- adding tests for real handshake scenarios
- improving documentation and examples
- fixing production-facing bugs

## Changes that are less likely to be accepted

- large framework-style configuration layers
- broad plugin systems
- unrelated utility helpers
- speculative abstractions without a concrete use case

## Questions and proposals

If you want to propose a larger change, open an issue first so the scope can be discussed before implementation.
