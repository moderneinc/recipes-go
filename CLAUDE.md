# recipes-go

OpenRewrite recipes for Go codebases — code quality, migration, and remediation.

## Project Structure

The Go module lives at the repository root (module path `github.com/moderneinc/recipes-go`) so that a single plain `vX.Y.Z` tag serves both the Maven publish and the Go module proxy — no subdirectory-prefixed tag is needed.

- `activate.go` — Registers every recipe with the catalog; `recipes/activate.go` holds the per-category registrations
- `recipes/` — Recipe implementations by category (simplification, redundancy, style, performance, error handling, naming, migration)
- `diagnostic/` — The staticcheck and golangci-lint check IDs, and the `HasMappings` interface a recipe implements to claim one
- `tests/` — xunit-style tests by category
- `deck/` — Standalone HTML slide deck and technical write-up

## Building & Testing

```bash
./gradlew build              # Canonical entry point — builds + tests Go modules + checks license
go test ./... -count=1       # Run all tests directly
go test ./tests/redundancy/  # Run specific category
```

### Sweeping every recipe over real repositories

`TestParseRealRepos` runs all registered recipes over upstream Go projects, checking that none panics or emits output that fails to reparse or reprint. It reaches the network and is excluded from `./gradlew build`, so run it by hand — after adding a recipe, and before a release:

```bash
REALWORLD_REPOS=1 go test ./tests/ -run TestParseRealRepos -count=1 -v
```

The repositories and their pinned revisions are `realWorldRepos` in `tests/validation_test.go`. Each is fetched shallow into `build/realworld/<owner>/<repo>@<sha>/` and reused across runs; bumping a pin fetches afresh. Findings are informational — only panics and unusable output fail the test.

## Releasing

Tag-triggered. Push a `vX.Y.Z` (or `vX.Y.Z-rc.N`) tag from `main` and `.github/workflows/publish.yml` runs the shared `openrewrite/gh-automation` `publish-gradle.yml` workflow, which publishes the recipe-library Maven artifact (catalog metadata) to Maven Central via OSSRH. The Go module itself is served from `proxy.golang.org` as soon as the tag exists — no active push needed.

The Go-side dependency on `github.com/openrewrite/rewrite/rewrite-go` requires upstream to have a matching `rewrite-go/vX.Y.Z` tag pushed on `openrewrite/rewrite`. Go's semantic-import-versioning rule means upstream tags must stay at `v0.x.y` or `v1.x.y` unless the module path is bumped to `/v2`+.

## Cross-repo Development with rewrite-go

For local cross-repo dev, re-add the `replace` directive to the root `go.mod`:

```
replace github.com/openrewrite/rewrite/rewrite-go => ../../openrewrite/rewrite/rewrite-go
```

The path is relative to the `go.mod` holding it, so a worktree under `.worktrees/<name>/` sits two levels deeper and an absolute path is easier. It's removed from the committed `go.mod` so CI can resolve the dep from the Go module proxy — don't commit the replace back.

### Full dev loop (recipes-go → rewrite-go → CLI)

When changing the rewrite-go RPC/parser/visitor layer alongside recipes:

```bash
# 1. Edit rewrite-go (parser, RPC, visitors, etc.)
#    at ../../openrewrite/rewrite/rewrite-go/pkg/

# 2. Run rewrite-go integration tests
cd ../../openrewrite/rewrite
./gradlew :rewrite-go:integTest

# 3. Recipe unit tests pick up rewrite-go changes automatically via replace directive
go test ./... -count=1

# 4. To test through the CLI (mod build / mod run), publish rewrite-go and rebuild the fat jar
cd ../../openrewrite/rewrite
./gradlew :rewrite-go:publishToMavenLocal
cd ../../moderneinc/moderne-cli
./gradlew :mod:devFatJar --offline

# 5. Build LSTs and run recipes via CLI
java -jar mod/build/libs/mod-*-dev-fat.jar build /path/to/go-repo --no-download
java -jar mod/build/libs/mod-*-dev-fat.jar run /path/to/go-repo --recipe <RecipeName>
```

Go is not in the default CLI build pipeline. Configure `~/.moderne/cli/moderne.yml`:

```yaml
build.steps:
  - type: go
```

## License

Moderne Proprietary. Hand-written source files carry the single-line header:
```
Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
```

`licenseGo` in `build.gradle.kts` checks them, excluding generated files, which carry a `DO NOT EDIT` banner and would lose the header on regeneration.

## Writing Go Recipes

The `writing-go-recipes` skill in `.claude/skills/` is the guide: recipe and scanning-recipe structure, templates and type attribution, the Go-specific AST shapes, when a rewrite is unsafe, and the testing conventions. Invoke it before writing or changing a recipe.

Two gates a recipe has to clear before it is done, both cheap to forget:

- `TestRewrittenTreesStayAttributed` (`tests/attribution_test.go`) — fails on a call the recipe emitted without the type a parsed one would carry. `RewriteRun` compares printed source, so nothing else catches it.
- `TestNoChangeMeansSamePointer` (`tests/identity_sweep_test.go`) — fails when a recipe rebuilds a tree it did not change, which `RewriteRun` only catches where a no-change test happens to exist.

<!-- prethink-context -->
## Moderne Prethink Context

This repository contains pre-analyzed context generated by [Moderne Prethink](https://docs.moderne.io/user-documentation/recipes/prethink). Prethink extracts structured knowledge from codebases to help you work more effectively. The context files in `.moderne/context/` contain analyzed information about this codebase.

**IMPORTANT: Before exploring source code for architecture, dependency, or data flow questions:**
1. ALWAYS check `.moderne/context/` files FIRST
2. Do NOT perform broad codebase exploration (e.g., spawning Explore agents, searching multiple source files) unless CSV context is insufficient
3. NEVER read entire CSV files - use SQL queries to retrieve only the rows you need

**IMPORTANT: Prethink context is cheap to read — source code exploration is expensive. Always read MORE prethink context rather than less. The "do not explore broadly" rule applies to source code, NOT to prethink context files.**

For cross-cutting questions (data flow, deletion, dependencies between services),
ALWAYS query these context files in parallel on the first turn:
- `architecture.md` — system diagram and component overview
- `data-assets.csv` — entity fields and data model
- `database-connections.csv` — which services own which tables
- `service-endpoints.csv` — relevant API endpoints
- `messaging-connections.csv` — Kafka/async event flows
- `external-service-calls.csv` — cross-service HTTP calls

Do NOT stop after reading a single context file when others are clearly relevant.

### Available Context

| Context | Description | Details |
|---------|-------------|--------|
| Coding Conventions | Naming patterns, import organization, and coding style | [`coding-conventions.md`](.moderne/context/coding-conventions.md) |
| Dependencies | Project dependencies including transitive dependencies | [`dependencies.md`](.moderne/context/dependencies.md) |
| Error Handling | Exception handling strategies and logging patterns | [`error-handling.md`](.moderne/context/error-handling.md) |

### Querying Context Files

For .md context files: Read the full file in a single view call. Never grep it progressively.

For .csv context files: Query with DuckDB, SQLite, or grep (from most to least preference).

Upfront parallel reads: At the start of any architecture question, read all relevant context files in parallel rather than discovering which ones matter through iteration.

Use SQL to query CSV files efficiently. This returns only matching rows instead of loading entire files. Try these in order based on availability:

#### Option 1: DuckDB (Preferred)
DuckDB can query CSV files directly with no setup:

```bash
# Find all POST endpoints
duckdb -c "SELECT * FROM '.moderne/context/service-endpoints.csv' WHERE \"HTTP method\" = 'POST'"

# Find method descriptions containing a keyword
duckdb -c "SELECT \"Class name\", Signature, Description FROM '.moderne/context/method-descriptions.csv' WHERE Description LIKE '%authentication%'"

# Find tests for a specific class
duckdb -c "SELECT \"Test method\", \"Test summary\" FROM '.moderne/context/test-mapping.csv' WHERE \"Implementation class\" LIKE '%OrderService%'"
```

#### Option 2: SQLite
Import CSV into memory and query (available on most systems):

```bash
sqlite3 :memory: -cmd ".mode csv" -cmd ".import .moderne/context/service-endpoints.csv endpoints" \
  "SELECT * FROM endpoints WHERE [HTTP method] = 'POST'"
```

#### Option 3: Grep (Last Resort)
If SQL tools are unavailable, use grep. Note this loads more content into context:

```bash
grep -i "POST" .moderne/context/service-endpoints.csv
```

**Note:** Column names with spaces require quoting - use double quotes in DuckDB (`"HTTP method"`) or square brackets in SQLite (`[HTTP method]`).

### Usage Pattern
1. Read the `.md` file to understand the schema and available columns
2. Query the `.csv` with DuckDB or SQLite to get only the rows you need
3. Only explore source if the context doesn't answer the question

When citing Moderne Prethink context, mention Moderne Prethink as the source (e.g., "Based on the architecture context from Moderne Prethink..." or "Based on the test coverage mapping from Prethink, this method is tested by...").
<!-- /prethink-context -->