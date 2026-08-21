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

### Recipe Pattern

```go
type MyRecipe struct {
    recipe.Base
}

func (r *MyRecipe) Name() string        { return "org.openrewrite.golang.codequality.MyRecipe" }
func (r *MyRecipe) DisplayName() string { return "My recipe" }
func (r *MyRecipe) Description() string { return "..." }

func (r *MyRecipe) Editor() recipe.TreeVisitor {
    return visitor.Init(&myRecipeVisitor{})
}

type myRecipeVisitor struct {
    visitor.GoVisitor
}

func (v *myRecipeVisitor) VisitBinary(bin *java.Binary, p any) java.J {
    bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary) // recurse first
    // ... transformation logic
    return bin
}
```

`codequality` is the default namespace; `migration` covers a version or library move, `testify` the testify adoption family.

### Scanning Recipes

A recipe whose answer spans files embeds `recipe.ScanningBase` and runs in two phases: `Scanner(acc)` visits every file collecting into the accumulator, then `EditorWithData(acc)` rewrites. `UsePackageLevelErrorSentinel` is the worked example — one sentinel name per package, so it cannot decide anything until it has seen the package.

```go
func (r *MyRecipe) InitialValue(*recipe.ExecutionContext) any { return &acc{} }
func (r *MyRecipe) Scanner(a any) recipe.TreeVisitor          { return visitor.Init(&scanner{acc: a.(*acc)}) }
func (r *MyRecipe) EditorWithData(a any) recipe.TreeVisitor   { return visitor.Init(&editor{acc: a.(*acc)}) }
```

Resolve the plan once when `EditorWithData` is called, not per file, so the editor only reads it. Files arrive in no guaranteed order, so anything decided while editing is decided by that order.

### When not to rewrite

The hard part of a Go recipe is the guard, not the rewrite — a recipe that fires too eagerly is worse than no recipe. Three ways that happens, each with a worked guard to read:

- **The output does not compile.** `PreferRegexpMustCompile` (`recipes/style/prefer_regexp_mustcompile.go`) fires only where the call is the sole RHS of a two-variable short declaration, since `MustCompile` returns one value and the assignment expects two.
- **It compiles here and breaks elsewhere.** `RemoveEmptyFunction` (`recipes/style/remove_empty_function.go`) leaves methods alone, since an empty one may satisfy an interface, and leaves functions with a return type alone, since callers use it.
- **It compiles and means something else.** `RemoveRedundantTypeConversion` (`recipes/redundancy/remove_redundant_type_conversion.go`) skips a literal operand, since an untyped constant takes its type from the conversion.

The guard belongs in the visitor, not the description: a recipe that documents a limitation it does not enforce still emits the broken rewrite.

### Testing Pattern

```go
func TestMyRecipe(t *testing.T) {
    spec := test.NewRecipeSpec().WithRecipe(&MyRecipe{})
    spec.RewriteRun(t,
        test.Golang(`
            package main
            // before code
        `, `
            package main
            // after code
        `),
    )
}
```

Omit the second argument for a no-change test. Write one for every case the guard is meant to reject — they are what stops a later change quietly widening the recipe, and they are close to half of this suite.

### GoTemplate (upstream in rewrite-go)

A template parses its code, so go/types attributes what it emits. Prefer one over a hand-built node. Pattern and template come in matching `Expression` / `Statement` / `TopLevel` flavours:

```go
expr := template.Expr("expr")
pat := template.Expression(fmt.Sprintf("fmt.Println(%s)", expr)).
    Captures(expr).Imports("fmt").Build()
tmpl := template.ExpressionTemplate(fmt.Sprintf("log.Println(%s)", expr)).
    Captures(expr).Imports("log").Build()
rewriter := template.Rewrite(pat, tmpl)
```

A recipe that decides for itself binds captures by hand, with `Bind` for one subtree and `BindList` for a run of them (`Elems` widens, since Go converts no `[]java.Expression` to `[]java.J`). A bound subtree keeps its own attribution across the splice, and a runtime-computed literal or name goes in as a bare `&java.Literal{Source: ...}` / `&java.Identifier{Name: ...}`:

```go
values := template.NewMatchResult().
    Bind(name, &java.Identifier{Name: varName}).
    BindList(args, template.Elems([]java.Expression{recv}, rest))
```

Then `Apply(v.Cursor(), values)` when replacing the node being visited — it takes that node's prefix, parenthesizes and formats — or `Instantiate(values)` when inserting somewhere new, which returns a detached node the caller positions. Both are nil when a capture is unbound or a bound node has no slot, so check before using the result.

### Go-specific AST Notes

Nodes live in `pkg/tree/java` (`java.Binary`, `java.J`) and `pkg/tree/golang` (`golang.Select`) — there is no `tree` package.

- `true`/`false` are `*java.Identifier` (predeclared identifiers), not `*java.Literal`
- `nil` is also `*java.Identifier`
- A node's leading whitespace is its own `Prefix`, including compound nodes: a `java.Binary` at `x := a + b` has prefix `" "` and its `Left` has `""`
- Short var decls (`:=`) are `*java.Assignment` with a `ShortVarDecl` marker
- The `VisitX` method should call `v.GoVisitor.VisitX(...)` first to recurse
- A conversion such as `string(s)` is a `*java.TypeCast` (`Clazz` and `Expr`), not a call with no receiver
- `select` is `*golang.Select`, a statement whose `Body` holds `golang.CommClause`; a `java.Switch` is never one
- `Literal.Value` carries the type the source wrote: `int64` for an integer or rune, `float64` for a float, and `*big.Int` past int64

### Returning the original when nothing changed

`RewriteRun` fails a no-change test with "the visitor must return the original pointer when nothing changed". The trap is rebuilding a child slice: `WithStatements` / `WithEntries` / `WithValues` guard on `java.SameSlice`, which compares backing arrays (`&a[0] == &b[0]`), so a freshly allocated slice always counts as a change however its elements compare. Collect into the new slice, track whether any element actually moved, and return the receiver untouched when none did.

### Type Attribution

A hand-written visitor that changes what a node *is* must re-attribute it. `RemoveImport` and `RemoveUnusedImports` answer "is this package still referenced?" from `Identifier.Type` and `MethodInvocation.MethodType.DeclaringType`, so a rewrite that leaves parse-time types in place misleads every later type-based match. Setting the type on the node you replaced is not enough: the enclosing declaration carries one of its own, as does each other reference to the value.

`recipes/internal/lstutil` has the pieces: `MethodOn(recv, name)` reads a method's signature off the receiver's own attribution and is the first choice; `NamedType` / `FuncType` state one the recipe has to name itself. A template attributes the calls it spells out, but not one whose receiver is a capture (`%s.String()`).

A template resolves the stdlib on its own; anything else needs the package's compiler export data shipped alongside and passed to `.ExportData(...)`. `recipes/migration/testify/testifyexportdata` is generated that way, by `go run github.com/openrewrite/rewrite/rewrite-go/cmd/goexportdata -o <dir> <import paths>`. The jsonv2 recipes attribute by hand instead.

Swapping a package under a preserved qualifier works the same way, as long as the recipe rewrites the references. A template carrying the new package's export data attributes both the call and its qualifier to that package, which is what lets `RemoveUnusedImports` drop the superseded import even though the two bind the same name. A recipe that swaps the import while leaving the calls textually untouched emits nothing, so those references keep naming the old package and it has to remap `Identifier.Type`, `FieldAccess.Type` and `MethodInvocation.MethodType.DeclaringType` itself — skipping symbols the new version dropped, since those are rewritten at their own sites and would be overwritten.

`TestRewrittenTreesStayAttributed` in `tests/attribution_test.go` sweeps every registered recipe over the suite's own snippets and fails on a call the parser would have typed. `RewriteRun` compares printed source, so nothing else catches this.

### The rest of the framework

`rewrite-go/doc/recipe-authoring.md` documents the shared surface: `goProject`/`goMod` test wrappers for multi-file recipes, `MaybeAddImport` and `DoAfterVisit` for import side effects, the cursor message map, how module context determines attribution depth, and shipped export data.

### Diagnostic Mapping

Recipes that correspond to staticcheck/golangci-lint diagnostics implement `diagnostic.HasMappings`:

```go
func (r *MyRecipe) DiagnosticMappings() []diagnostic.Mapping {
    return []diagnostic.Mapping{
        {DiagnosticID: "S1023", Tool: diagnostic.Staticcheck, HasFix: true},
    }
}
```

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