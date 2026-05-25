<p align="center">
  <a href="https://docs.openrewrite.org">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://github.com/openrewrite/rewrite/raw/main/doc/logo-oss-dark.svg">
      <source media="(prefers-color-scheme: light)" srcset="https://github.com/openrewrite/rewrite/raw/main/doc/logo-oss-light.svg">
      <img alt="OpenRewrite Logo" src="https://github.com/openrewrite/rewrite/raw/main/doc/logo-oss-light.svg" width='600px'>
    </picture>
  </a>
</p>

<div align="center">
  <h1>recipes-go</h1>
</div>

This project implements a [Rewrite module](https://github.com/openrewrite/rewrite) that provides automated code quality, migration, and remediation recipes for Go codebases. Built on the [rewrite-go](https://github.com/openrewrite/rewrite) framework, these recipes analyze and transform Go source code at the AST level, enabling safe, large-scale automated refactoring.

**Note**: These recipes are currently only supported via the [Moderne CLI](https://docs.moderne.io/user-documentation/moderne-cli/getting-started/cli-intro) or the [Moderne Platform](https://docs.moderne.io/user-documentation/moderne-platform/getting-started/running-your-first-recipe). The Moderne CLI is free to use for open-source repositories. If your repository is closed-source, you will need to obtain a license to use the CLI or the Moderne Platform. [Please contact Moderne to learn more](https://www.moderne.ai/contact-us).

## Prerequisites

Go is not part of the default Moderne CLI build pipeline. You must opt in by adding a `build.steps` configuration. Create or edit `~/.moderne/cli/moderne.yml`:

```yaml
build.steps:
  - type: go
```

Alternatively, place a `.moderne/moderne.yml` file in your repository root with the same content.

## Installation

Install the recipes using the Moderne CLI's `mod config recipes go install` command. Go recipe modules are resolved from source via their Go module path.

### From a release tag

```bash
mod config recipes go install github.com/moderneinc/recipes-go/recipes-code-quality@v0.1.0
```

### Latest (no version specified)

```bash
mod config recipes go install github.com/moderneinc/recipes-go/recipes-code-quality
```

### From a specific version

```bash
mod config recipes go install github.com/moderneinc/recipes-go/recipes-code-quality@v0.2.0
```

### Removing recipes

```bash
mod config recipes go delete github.com/moderneinc/recipes-go/recipes-code-quality
```

### Viewing installed recipes

```bash
mod config recipes list
```

### Building and running recipes

Build an LST first, then run recipes against it:

```bash
# Build the LST for your Go repository
mod build . --no-download

# Run a specific recipe
mod run . --recipe org.openrewrite.golang.codequality.PreferErrorsIsOverEquality

# Run a simplification recipe
mod run . --recipe org.openrewrite.golang.codequality.SimplifyBooleanExpression
```

## Overview

**213 recipes** across 6 categories with **595 tests**.

| Category | Recipes | Description |
|---|---|---|
| **Style** | 73 | Enforce conventions, detect code smells, security patterns, resource management |
| **Simplification** | 61 | Modernize code with newer stdlib APIs, simplify expressions, migrate deprecated APIs |
| **Error Handling** | 27 | `errors.Is`/`errors.As` migration, error wrapping, sentinel extraction |
| **Redundancy** | 23 | Remove dead code, redundant operations, unreachable statements |
| **Performance** | 20 | Loop optimizations, allocation hoisting, format string improvements |
| **Naming** | 9 | Receiver names, stuttering, constants, getter prefixes, error variables |

## Recipe Highlights

### Simplification

- **ioutil to io/os migration**: `ioutil.ReadAll` to `io.ReadAll`, `ioutil.ReadFile` to `os.ReadFile`, etc. (8 recipes)
- **strings/bytes Index to Contains family**: `strings.Index(s, sub) != -1` to `strings.Contains(s, sub)` (7 recipes)
- **Boolean simplification**: `x == true` to `x`, `x == false` to `!x`, `!!x` to `x`
- **fmt.Sprintf optimization**: `fmt.Sprintf("%s", s)` to `s`, `fmt.Sprintf("%v", x)` to `fmt.Sprint(x)`
- **Go 1.21+**: `sort.Sort(sort.IntSlice(s))` to `sort.Ints(s)`, `math.Min(a, b)` to `min(a, b)`
- **Structured logging**: `log.Println(x)` to `slog.Info(x)` (Go 1.21+)

### Error Handling

- **errors.Is migration**: `err == io.EOF` to `errors.Is(err, io.EOF)` for 8+ common sentinels
- **errors.As migration**: `if myErr, ok := err.(*T); ok` to `var myErr *T; if errors.As(err, &myErr)`
- **Error wrapping**: `return err` to `return fmt.Errorf("funcName: %w", err)`
- **Sentinel extraction**: Inline `errors.New("msg")` to package-level `var ErrMsg` declarations
- **fmt.Errorf verb**: `fmt.Errorf("...: %s", err)` to `fmt.Errorf("...: %w", err)`
- **Swallowed error**: `if err != nil { return }` to `if err != nil { return err }`

### Style & Security

- **Resource management**: Auto-insert `defer f.Close()`, `defer rows.Close()`, `defer ticker.Stop()`, etc.
- **TLS enforcement**: `InsecureSkipVerify: true` to `InsecureSkipVerify: false`
- **File permissions**: `0777` to `0755`
- **Credential remediation**: `var password = "hunter2"` to `var password = os.Getenv("PASSWORD")`
- **Doc comments**: Auto-generate `// FuncName ...` stubs for exported functions
- **Raw regex strings**: `regexp.Compile("\\d+")` to `` regexp.Compile(`\d+`) ``

### Redundancy

- **Dead code removal**: Unreachable code after return, empty loops/switches/defaults, self-assignments
- **Control flow simplification**: `if true {body}` to `body`, `if len(s) > 0 { for range s }` to `for range s`
- **Goroutine simplification**: `go func() { f() }()` to `go f()`
- **Map clearing**: `for k := range m { delete(m, k) }` to `clear(m)`

### Performance

- **Regex hoisting**: `regexp.MustCompile("pattern")` inside loops hoisted before loop
- **Format string optimization**: `fmt.Sprintf("%d", n)` to `strconv.Itoa(n)`
- **Loop detection**: Allocations, defer, goroutine launches, lock acquisition in loops

### Naming

- **Receiver names**: `func (self *Foo) Bar()` to `func (f *Foo) Bar()`
- **Getter prefix**: `func (u *User) GetName()` to `func (u *User) Name()`
- **Stuttering**: `func HttpGet()` in `package http` to `func Get()`
- **Constants**: `MAX_BUFFER_SIZE` to `MaxBufferSize`
- **Error variables**: `var notFound = errors.New(...)` to `var ErrNotFound = errors.New(...)`

## Getting Started

For help getting started with the Moderne CLI, check out our [getting started guide](https://docs.moderne.io/user-documentation/moderne-cli/getting-started/cli-intro). Or, if you'd like to try running these recipes in the Moderne Platform, check out the [Moderne Platform quickstart guide](https://docs.moderne.io/user-documentation/moderne-platform/getting-started/running-your-first-recipe).

## Building and Testing

```bash
cd recipes-code-quality
go test ./... -count=1              # Run all tests
go test ./tests/simplification/    # Run a specific category
./gradlew check                     # Full build via Gradle
```

## Releasing

Releases are cut by pushing a semver tag from `main`:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The `publish` workflow (`.github/workflows/publish.yml`) reacts to tags matching
`vX.Y.Z` or `vX.Y.Z-rc.N` and delegates to the shared
`openrewrite/gh-automation` `publish-gradle.yml` workflow, which publishes the
recipe-library Maven artifact (catalog metadata) to Maven Central via OSSRH.

The Go module itself is consumed by the Moderne CLI directly from the Go module
proxy (`proxy.golang.org`). No active push is needed for the Go side — once the
tag exists on GitHub, `mod config recipes go install
github.com/moderneinc/recipes-go/recipes-code-quality@vX.Y.Z` resolves it.

## Licensing

This project is licensed under the Moderne Proprietary License. Only for use by Moderne customers under the terms of a commercial contract.
