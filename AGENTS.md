# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go backend for travel inspiration generation. The entrypoint is [`cmd/server`](./cmd/server), which wires config, Mongo, routing, and services. Core application code lives under [`internal/`](./internal):

- `agent/`: specialized LLM and Wikipedia agents
- `orchestrator/`: chat flow orchestration
- `service/`: application-facing business services
- `dao/`, `mongo/`, `model/`, `dto/`: persistence and data contracts
- `handler/`, `router/`: HTTP layer
- `config/`: config loading from `config.json`

Docs live in [`docs/`](./docs). Runtime configuration is stored in [`config.json`](./config.json).

## Build, Test, and Development Commands
- `go test ./...`: run the full test suite.
- `go test ./internal/agent ./internal/orchestrator ./internal/service`: fast validation for the main chat pipeline.
- `go run ./cmd/server`: start the API server locally.
- `make mongo`: start the local MongoDB container defined in `Makefile`.
- `make mongo-stop`: stop the local MongoDB container.
- `make mongo-shell`: open a Mongo shell against the local container.

## Coding Style & Naming Conventions
Use standard Go style with tabs and `gofmt`. Run `gofmt -w <file>` before submitting changes. Keep package names lowercase and short. Use `CamelCase` for exported identifiers and `mixedCaps` for unexported ones. New orchestration logic should go in `internal/orchestrator`; single-purpose model/tool wrappers belong in `internal/agent`.

## Testing Guidelines
Tests use Go’s built-in `testing` package and should live beside the code as `*_test.go`. Prefer focused unit tests for agents, DTO mapping, and orchestration helpers. Name tests clearly, for example `TestKeywordAgentExtractKeywordsFromOutput`. Add tests for new parsing, prompt-shaping, or persistence-mapping behavior.

## Commit & Pull Request Guidelines
Recent history favors short, imperative commit messages such as `refactor to multi-agent` or `add intent agent, modified resp dto`. Keep commits scoped to one logical change. PRs should include:

- a brief summary of user-visible behavior changes
- affected packages or endpoints
- test commands you ran
- sample request/response payloads when API contracts changed

## Security & Configuration Tips
Do not hardcode secrets in code. Keep API keys, Mongo settings, and Wikipedia settings in `config.json` or environment-specific config management. If you change config shape, update both `internal/config/config.go` and `config.json`.
