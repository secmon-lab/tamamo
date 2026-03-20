# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tamamo is an LLM-driven web honeypot generator and server. It generates realistic "internal admin web pages" (login screens, API endpoints, server signatures) using LLM, and serves them as honeypots to capture attacker activity. Scenarios are pre-generated and can be swapped at startup time.

## Common Development Commands

### Building and Testing
- `go vet ./...` - Verify code compiles and check for issues
- `go test ./...` - Run all tests
- `go test ./pkg/path/to/package` - Run tests for specific package
- `task` - Run default tasks (mock generation)
- `task mock` - Generate all mock files

**NEVER run `go build` to verify code.** Use `go vet ./...` instead to check for compile errors.

### Code Quality Checks
When making changes, before finishing the task, always:
- Run `go vet ./...`, `go fmt ./...` to format the code
- Run `golangci-lint run ./...` to check lint error
- Run `gosec -exclude-generated -quiet ./...` to check security issue
- Run tests to ensure no impact on other code

## Important Development Guidelines

### Implementation Completeness
- **NEVER leave incomplete implementations, TODOs, or placeholder code**
- **NEVER skip implementation because it's complex or lengthy**
- **ALWAYS complete the full implementation in one go**
- If a task seems too complex, break it down into smaller steps, but complete ALL steps
- Complexity is not an excuse - implement everything thoroughly
- Long code is acceptable - incomplete code is NOT

### Error Handling
- Use `github.com/m-mizutani/goerr/v2` for error handling
- Must wrap errors with `goerr.Wrap` to maintain error context
- Add helpful variables with `goerr.V` for debugging
- **NEVER check error messages using `strings.Contains(err.Error(), ...)`**
- **ALWAYS use `errors.Is(err, targetErr)` or `errors.As(err, &target)` for error type checking**
- Error discrimination must be done by error types, not by parsing error messages
- **ALWAYS tag errors with `goerr.T(errutil.TagXxx)` from `pkg/utils/errutil`** to enable proper observability:
  - Input validation failures: `errutil.TagValidation`
  - Resource not found: `errutil.TagNotFound`
  - LLM/external service errors: `errutil.TagExternal`
  - Generation failures: `errutil.TagGeneration`
  - Internal errors: `errutil.TagInternal`
  - Example: `goerr.New("scenario missing required field", goerr.V("field", "name"), goerr.T(errutil.TagValidation))`

### Testing with gt Package
- Use `github.com/m-mizutani/gt` package for type-safe testing
- Prefer Helper Driven Testing style over Table Driven Tests
- Use mock implementations from `pkg/domain/mock`
- **NEVER comment out test assertions** - if a test doesn't work, fix it or delete it
- **NEVER use length-only checks** - always verify individual IDs/values explicitly

### Code Visibility
- Do not expose unnecessary methods, variables and types
- Use `export_test.go` to expose items needed only for testing
- Assume that exposed items will be changed. Never expose fields that would be problematic if changed

### Language
All comments and string literals in source code must be in English

## Architecture

### Core Structure
The application follows Domain-Driven Design (DDD) with clean architecture:

- `pkg/domain/` - Domain layer with models, interfaces, and types
- `pkg/service/` - Application services implementing business operations
- `pkg/controller/` - Interface adapters (HTTP honeypot server)
- `pkg/usecase/` - Application use cases orchestrating domain operations
- `pkg/cli/` - CLI command definitions and configuration
- `pkg/utils/` - Shared utilities (error tags, logging)

### Key Components

#### Scenario Model (`pkg/domain/model/scenario/`)
A scenario defines everything the honeypot exposes to attackers:
- `Scenario` - Top-level container with Meta, Pages, and Routes
- `Meta` - Server signature, HTTP headers, theme name
- `Page` - HTML/CSS/JS files for login screens and dashboard
- `Route` - API endpoint definitions with method, status, headers, body

#### Scenario Generation (`pkg/service/generator/`)
Uses gollem Agent with WriteFile tool for iterative file generation:
- `generator.go` - Agent creation and execution
- `tool.go` - WriteFile ToolSet implementation (gollem.ToolSet)
- `prompt.go` - System prompt construction for scenario generation

The LLM Agent autonomously generates scenario.json, routes.json, and pages/*.html files through the WriteFile tool, enabling complex multi-file generation that cannot be done in a single LLM call.

Generation progress is displayed in real-time on the CLI via `Printer` interface, showing tool invocations and agent messages in an AI coding agent style (not traditional logs).

#### Event Emission (`pkg/service/emitter/`)
Attacker activity is reported through the `Emitter` interface, which is extensible for future backends (Pub/Sub, SQS, Kafka):
- `LogEmitter` - Structured log output via slog (always active)
- `WebhookEmitter` - HTTP POST with HMAC-SHA256 signature (`X-Tamamo-Signature: sha256=<hex>`)

#### Honeypot HTTP Server (`pkg/controller/http/`)
- Serves scenario-defined pages and API routes
- Login attempts always "succeed" to avoid alerting attackers
- Post-login dashboard shows perpetual loading state (hang simulation)
- All attacker activity is emitted through `Emitter` interface (log + webhook)

### Application Modes
- `generate` - Generate honeypot scenario data using LLM Agent
- `serve` - Serve a generated scenario as an HTTP honeypot; auto-generates if no scenario specified
- `validate` - Validate scenario data integrity

### Key Interfaces
- `interfaces.Generator` - LLM-based scenario generation abstraction
- `interfaces.Emitter` - Extensible event notification (log, webhook, future: Pub/Sub, SQS)
- `interfaces.Repository` - Scenario read/write abstraction
- `interfaces.Printer` - CLI message display for generation progress

### CLI Configuration
Configuration follows the warren pattern:
- Config structs with `Flags()` and `Configure()` methods
- All config via CLI flags + environment variables
- DI at CLI level using Options pattern

## Testing

- Test files should have `package {name}_test`. Do not use same package name
- Test file name convention is: `xyz.go` → `xyz_test.go`. Other test file names are not allowed.
- Test Skip Policy:
  - **NEVER use `t.Skip()` for anything other than missing environment variables**
  - If a feature is not implemented, write the code, don't skip the test
  - The only acceptable skip pattern: checking for missing environment variables at the beginning of a test

### Test File Checklist (Use this EVERY time)
Before creating or modifying tests:
1. ✓ Is there a corresponding source file for this test file?
2. ✓ Does the test file name match exactly? (`xyz.go` → `xyz_test.go`)
3. ✓ Are all tests for a source file in ONE test file?
4. ✓ No standalone feature/e2e/integration test files?

## Dependencies

- CLI: `github.com/urfave/cli/v3`
- LLM: `github.com/m-mizutani/gollem`
- HTTP Router: `github.com/go-chi/chi/v5`
- Error Handling: `github.com/m-mizutani/goerr/v2`
- Logging: `log/slog` + `github.com/m-mizutani/clog`
- Testing: `github.com/m-mizutani/gt`
