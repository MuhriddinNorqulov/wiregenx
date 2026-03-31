# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Wiregenx is a Go CLI tool that generates a standalone DI (dependency injection) container from annotated Go functions. No external DI library (like Google Wire) is needed — the generated code is pure Go.

## Build & Run Commands

```bash
go build ./...                                                        # Build
go run ./cmd/wiregenx2 --root ./path --out container/container_gen.go # Run
go vet ./...                                                # Static analysis
```

No tests exist yet. No Makefile or CI pipeline.

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-root` | `.` | Directory to scan |
| `-out` | `container_gen.go` | Output file (relative to root) |
| `-pkg` | `container` | Package name for generated file |
| `-no-vendor` | `true` | Skip vendor/ |
| `-no-hidden` | `true` | Skip hidden directories |

## Annotation System

| Annotation | Purpose |
|---|---|
| `@inject` | Marks function as a provider (default: singleton) |
| `@inject(singleton)` | Single instance, created in constructor |
| `@inject(prototype)` | New instance per getter call |
| `@Application("name")` | Application entry point — gets its own named Container |

Annotations are Go comments above the function: `// @inject`, `// @Application("http")`

## Architecture

All logic is in `pkg/`. The pipeline is linear:

1. **Scan** (`scanner.go`) — Walks file tree, parses Go AST, finds annotated functions, extracts parameter types and return types from signatures, builds import maps
2. **Resolve imports** (`inject.go`) — Runs `go list -json .` per directory to get canonical import paths, resolves local types
3. **Resolve graph** (`resolver.go`) — Builds dependency graph from canonical type names, topological sort (Kahn's algorithm), detects cycles and missing dependencies
4. **Render** (`render.go`) — Generates Container structs with singleton fields, constructors (eager init in topo order), getter methods, handles import alias collisions, formats with `go/format`
5. **Write** (`inject.go`) — Writes generated file

Entry point: `cmd/wiregenx2/main.go` → `pkg.Inject()`

## Key Types (type.go)

- `Provider` — Annotated function: name, import path, params (dependencies), return type, scope, app flag
- `TypeRef` — Resolved type reference with `FullName()` for canonical matching (e.g. `*database/sql.DB`)
- `Scope` — `ScopeSingleton` or `ScopePrototype`
- `AppGroup` — One `@Application` provider with its resolved dependency subgraph

## Generated Code Structure

- Without `@Application`: single `Container` struct, `New()` constructor
- With `@Application("name")`: per-app containers (e.g. `HttpContainer`, `AsyncContainer`), each with its own `New{Name}Container()` constructor and only the dependencies that app needs
- Constructors never return error — provider errors cause `panic("wiregenx: ...")`
- Singleton getters return the stored field
- Prototype getters call the original function each time (panic on error)

## Conventions

- Error handling uses `must()` (utils.go) — calls `log.Fatalf`
- Dependency matching is exact canonical type string comparison
- Import aliases avoid collisions by appending numbers (`pkg`, `pkg2`, `pkg3`)
- `_test.go` files and methods (receivers) are excluded from scanning