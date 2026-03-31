# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Wiregenx is a Go CLI tool that generates a standalone DI (dependency injection) container from annotated Go functions. No external DI library (like Google Wire) is needed — the generated code is pure Go.

## Build & Run Commands

```bash
go build ./...                                              # Build
go run . --root ./path --out container/container_gen.go     # Run
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
| `@Inject` | Marks function as a provider (default: singleton) |
| `@Application` | Application root entry point (also a provider) |
| `@Singleton` | Single instance, created in `New()` |
| `@Prototype` | New instance per getter call |
| `@Factory` | Function returns `(T, error)` |

Annotations are Go comments above the function: `// @Inject`

## Architecture

All logic is in `pkg/`. The pipeline is linear:

1. **Scan** (`scanner.go`) — Walks file tree, parses Go AST, finds annotated functions, extracts parameter types and return types from signatures, builds import maps
2. **Resolve imports** (`inject.go`) — Runs `go list -json .` per directory to get canonical import paths, resolves local types
3. **Resolve graph** (`resolver.go`) — Builds dependency graph from canonical type names, topological sort (Kahn's algorithm), detects cycles and missing dependencies
4. **Render** (`render.go`) — Generates `Container` struct with singleton fields, `New()` constructor (eager init in topo order), getter methods, handles import alias collisions, formats with `go/format`
5. **Write** (`inject.go`) — Writes generated file

Entry point: `main.go` → `pkg.Inject()`

## Key Types (type.go)

- `Provider` — Annotated function: name, import path, params (dependencies), return type, scope, factory/app flags
- `TypeRef` — Resolved type reference with `FullName()` for canonical matching (e.g. `*database/sql.DB`)
- `Scope` — `ScopeSingleton` or `ScopePrototype`

## Generated Code Structure

- `Container` struct holds singleton instances as private fields
- `New() (*Container, error)` eagerly initializes singletons in dependency order
- Singleton getters return the stored field
- Prototype getters call the original function each time
- Factory errors are wrapped with `fmt.Errorf("wiregenx: ...")` in `New()`

## Conventions

- Error handling uses `must()` (utils.go) — calls `log.Fatalf`
- Dependency matching is exact canonical type string comparison
- Import aliases avoid collisions by appending numbers (`pkg`, `pkg2`, `pkg3`)
- `_test.go` files and methods (receivers) are excluded from scanning
