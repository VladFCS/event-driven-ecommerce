# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go microservices workspace. Service entrypoints live in `cmd/<service>/main.go` (currently `catalog-service` and `inventory-service`; other service folders are scaffolded). Keep business logic under `internal/<service>/` with the existing split: `domain/`, `service/`, `repository/`, and `handler/`. Protobuf contracts live in `api/proto/`, and generated gRPC stubs are committed under `gen/<service>/v1/`. Treat `gen/` as generated output; edit the `.proto` files, not the `.pb.go` files. `deployments/` is reserved for runtime manifests.

## Build, Test, and Development Commands
Use the standard Go toolchain plus the provided Make target:

- `go test ./...` runs the full package tree and is the current baseline verification command.
- `go build ./...` checks that all packages compile.
- `go run ./cmd/catalog-service` starts the catalog gRPC service on `GRPC_PORT` (default `50051`).
- `go run ./cmd/inventory-service` starts the inventory gRPC service on `GRPC_PORT` (default `50052`).
- `make proto` regenerates Go and gRPC code from `api/proto/**/*.proto`.

## Coding Style & Naming Conventions
Format all Go code with `gofmt ./...` before opening a PR. Follow idiomatic Go naming: exported identifiers in `PascalCase`, unexported helpers in `camelCase`, and lowercase package names. Match the current package layout by keeping validation in `service/`, persistence in `repository/`, and transport logic in `handler/`. Prefer structured logging with `log/slog` and pass `context.Context` through service boundaries.

## Testing Guidelines
There are no committed `*_test.go` files yet, so new changes should add tests alongside the package they cover. Use Go’s `testing` package, favor table-driven tests, and focus first on service validation, repository behavior, and gRPC error mapping. Run `go test ./...` locally before pushing.

## Commit & Pull Request Guidelines
Recent history uses Conventional Commit prefixes such as `feat: add service to order-service`. Keep that format (`feat:`, `fix:`, `chore:`) with short, imperative summaries. PRs should explain the affected service, note any contract changes, list verification commands run, and include regenerated `gen/` files whenever `.proto` definitions change.
