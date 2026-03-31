# recipes-go

OpenRewrite recipes for Go codebases — code quality, migration, and remediation.

## Project Structure

- `recipes-code-quality/` — Go code quality recipes (simplification, redundancy, style, performance, error handling)
- `recipes-code-quality/diagnostic/` — Diagnostic mapping types for harness comparison
- `recipes-code-quality/recipes/` — Recipe implementations by category
- `recipes-code-quality/tests/` — xunit-style tests by category
- `code-quality-harness/` — Comparison harness (staticcheck/golangci-lint vs OpenRewrite)
- `working-set-code-quality/` — Real Go repos for harness testing

## Building & Testing

```bash
cd recipes-code-quality
go test ./... -count=1       # Run all tests
go test ./tests/redundancy/  # Run specific category
```

## Cross-repo Development with rewrite-go

The recipes depend on the `github.com/openrewrite/rewrite` module. During local development, `go.mod` uses a `replace` directive pointing to the local rewrite-go checkout:

```
replace github.com/openrewrite/rewrite => ../../../openrewrite/rewrite/rewrite-go/rewrite
```

## License

Moderne Proprietary. All source files use the single-line license header:
```
Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
```

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

func (v *myRecipeVisitor) VisitBinary(bin *tree.Binary, p any) tree.J {
    bin = v.GoVisitor.VisitBinary(bin, p).(*tree.Binary) // recurse first
    // ... transformation logic
    return bin
}
```

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

Omit the second argument for no-change tests.

### GoTemplate (upstream in rewrite-go)

For template-based matching and replacement:

```go
expr := template.Expr("expr")
pat := template.Expression(fmt.Sprintf("fmt.Println(%s)", expr)).
    Captures(expr).Imports("fmt").Build()
tmpl := template.ExpressionTemplate(fmt.Sprintf("log.Println(%s)", expr)).
    Captures(expr).Imports("log").Build()
rewriter := template.Rewrite(pat, tmpl)
```

### Go-specific AST Notes

- `true`/`false` are `*tree.Identifier` (predeclared identifiers), not `*tree.Literal`
- `nil` is also `*tree.Identifier`
- Binary.Prefix is often empty — the leading whitespace is on Binary.Left
- Short var decls (`:=`) are `*tree.Assignment` with a `ShortVarDecl` marker
- The `VisitX` method should call `v.GoVisitor.VisitX(...)` first to recurse

### Diagnostic Mapping

Recipes that correspond to staticcheck/golangci-lint diagnostics implement `diagnostic.HasMappings`:

```go
func (r *MyRecipe) DiagnosticMappings() []diagnostic.Mapping {
    return []diagnostic.Mapping{
        {DiagnosticID: "S1023", Tool: diagnostic.Staticcheck, HasFix: true},
    }
}
```
