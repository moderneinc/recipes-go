---
name: writing-go-recipes
description: Use when authoring OpenRewrite Go recipes (.go recipe files, GoVisitor, GoPattern/GoTemplate, template.NewRecipe, MethodMatcher, RewriteTest). Covers recipe structure, visitor patterns, template matching, testing, and Go-specific AST gotchas.
---

# Authoring OpenRewrite Go Recipes

## When NOT to Use This Skill

- Authoring OpenRewrite recipes in **Java** — use `writing-openrewrite-recipes`
- Authoring OpenRewrite recipes in **C#** — use `writing-csharp-recipes`
- General Go programming questions unrelated to OpenRewrite
- Running existing recipes or build configuration

## Recipe Approaches

There are two approaches, choose based on complexity:

### 1. Template Recipe (declarative — preferred for simple before→after patterns)

```go
package simplification

import (
    "fmt"
    "github.com/moderneinc/recipes-go/diagnostic"
    "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
    "github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var myArg = template.Expr("argName")

var myRecipeImpl = template.NewRecipe(
    template.RecipeName("org.openrewrite.golang.codequality.MyRecipe$Impl"),
    template.WithDisplayName("My Recipe"),
    template.WithBefore(fmt.Sprintf(`old(%s)`, myArg), template.Imports("oldpkg")),
    template.WithAfter(fmt.Sprintf(`new(%s)`, myArg), template.Imports("newpkg")),
    template.WithCaptures(myArg),
)

type MyRecipe struct { recipe.Base }

func (r *MyRecipe) Name() string        { return "org.openrewrite.golang.codequality.MyRecipe" }
func (r *MyRecipe) DisplayName() string  { return "My Recipe" }
func (r *MyRecipe) Description() string  { return "Replace old(x) with new(x)." }
func (r *MyRecipe) Tags() []string       { return []string{"cleanup", "simplification"} }

func (r *MyRecipe) DiagnosticMappings() []diagnostic.Mapping {
    return []diagnostic.Mapping{
        {DiagnosticID: "S1000", Tool: diagnostic.Staticcheck, HasFix: true},
    }
}

func (r *MyRecipe) RecipeList() []recipe.Recipe {
    return []recipe.Recipe{myRecipeImpl}
}
```

### 2. Manual Visitor (for complex logic, search-only, or multi-node transforms)

```go
package redundancy

import (
    "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
    "github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
    "github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

type MyRecipe struct { recipe.Base }

func (r *MyRecipe) Name() string        { return "org.openrewrite.golang.codequality.MyRecipe" }
func (r *MyRecipe) DisplayName() string  { return "My Recipe" }
func (r *MyRecipe) Description() string  { return "..." }
func (r *MyRecipe) Tags() []string       { return []string{"cleanup"} }

func (r *MyRecipe) Editor() recipe.TreeVisitor {
    return visitor.Init(&myRecipeVisitor{})
}

type myRecipeVisitor struct {
    visitor.GoVisitor
}

func (v *myRecipeVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
    mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
    // Always call base visitor first to recurse into children
    // Then apply transformation logic
    return mi
}
```

## Key Conventions

- **License**: `Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.`
- **Imports**: Always from `github.com/openrewrite/rewrite/rewrite-go/pkg/...`
- **Recipe naming**: `org.openrewrite.golang.codequality.<RecipeName>`
- **Visitor**: Always call `v.GoVisitor.VisitX(node, p).(*java.X)` first to recurse
- **`visitor.Init()`**: Required to wire up `Self` for virtual dispatch
- **Categories**: `golang → codeQuality → subcategory` (Simplification, Redundancy, Style, Error handling, Performance, Naming)

## Template API Details

### Captures

```go
expr := template.Expr("name")     // expression position
stmt := template.Stmt("name")     // statement position
typ := template.TypeExpr("name")  // type position
id := template.Ident("name")      // identifier position
```

- Capture names must be **globally unique** within a Go package — use distinctive prefixes (e.g., `hpS`, `beA`)
- `capture.String()` returns `__plh_name__` — use with `fmt.Sprintf` to embed in template strings
- A capture matches any subtree at its syntactic position, unless it declares a type

```go
e := template.Expr("e").WithType("error")   // matches only an error-assignable operand
```

`WithType` filters the **match** path: `Matches` and `Apply` refuse a candidate of the wrong type, and an interface is satisfied structurally from the candidate's method set. It does not filter `Bind` — binding an int literal to a capture declared `string` instantiates `errors.New(42)` and says nothing, which `RewriteRun` will not catch either, since it compares printed source. Only an expression capture may declare a type; `WithType` panics on any other kind.

### Multiple Before Patterns (Refaster anyOf)

```go
var impl = template.NewRecipe(
    template.WithBefore(fmt.Sprintf(`%s == true`, x)),
    template.WithBefore(fmt.Sprintf(`true == %s`, x)),   // also matches
    template.WithAfter(fmt.Sprintf(`%s`, x)),             // single after
    template.WithCaptures(x),
)
```

First matching before wins. All befores share the same captures and after.

### Negation in Templates

```go
template.WithAfter(fmt.Sprintf(`!%s`, x))  // produces Unary(Not, x)
```

### Scaffold Kind Detection

Templates auto-detect whether code is an expression, statement, or top-level declaration. Override with:
```go
template.AsExpression()   // force expression
template.AsStatement()    // force statement
```

### Building a node from a template by hand

`template.NewRecipe` covers before→after rewrites. When the recipe decides for itself what to emit, build the template directly and bind the captures — this is how a rewritten node gets attributed by go/types instead of by the recipe:

```go
values := template.NewMatchResult().
    Bind(name, &java.Identifier{Name: varName}).
    BindList(args, template.Elems([]java.Expression{recv}, rest))
```

`Bind` takes one subtree, `BindList` a run of them, and `Elems` widens (Go converts no `[]java.Expression` to `[]java.J`). A bound subtree keeps its own attribution across the splice; a runtime-computed literal or name goes in as a bare `&java.Literal{Source: ...}` / `&java.Identifier{Name: ...}`.

Then either:

- `Apply(v.Cursor(), values)` — replacing the node being visited. Takes that node's prefix, parenthesizes where the surrounding expression binds tighter, and formats.
- `Instantiate(values)` — inserting somewhere new. Returns a detached node carrying no prefix, which the caller positions.

Both return nil when a capture is unbound, a variadic run falls outside its declared bounds, or a bound node has no slot in the list it lands in. Check before using the result.

Templates are `ExpressionTemplate` / `StatementTemplate` / `TopLevelTemplate`, matching the `Expression` / `StatementPattern` / `TopLevel` pattern builders.

### Attributing a non-stdlib import

A template type-checks against `importer.Default()`, which resolves the stdlib and nothing else, so `Imports("github.com/pkg/errors")` alone yields a call with no signature. Ship the package's compiler export data and pass it:

```bash
go run github.com/openrewrite/rewrite/rewrite-go/cmd/goexportdata -o <dir> <import paths>
```

```go
template.ExpressionTemplate(code).Imports(path).ExportData(mydata.FS).Build()
```

`recipes/migration/testify/testifyexportdata` is the worked example. Losing the blob is silent, so a module shipping one asserts `exportdata.Verify(FS, Paths...)` in a test.

## MethodMatcher

Pattern-based method invocation matching using AspectJ-style syntax:

```go
import "github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"

var fmtSprintf = matcher.NewMethodMatcher("fmt Sprintf(..)")

func (v *myVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
    mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
    if fmtSprintf.Matches(mi) {
        // matched
    }
    return mi
}
```

**Prefer this to comparing an identifier's name.** A qualifier read as text matches anything spelled that way, including a local that shadows the package:

```go
ident, ok := mi.Select.Element.(*java.Identifier)
if !ok || ident.Name != "context" { return mi }   // also matches:
                                                  //   context := fake{}
                                                  //   context.Background()
```

The matcher resolves the receiver through the type system, so it answers no there. Recipes here predating the matcher still do the name check; a new one should not.

**Pattern format**: `"DeclaringType MethodName(ArgType1, ..)"` where:
- `*` matches any single name
- `*..*` matches any type in any package
- `..` in args matches zero or more arguments

## TypeUtils

```go
import "github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"

matcher.GetFullyQualifiedName(t)         // extract FQN
matcher.IsOfClassType(t, "time.Time")    // exact match
matcher.IsAssignableTo(t, "error")       // hierarchy walk
matcher.IsError(t)                        // is error interface
matcher.IsString(t)                       // is string type
matcher.TypeOfExpression(expr)            // get type from expression
matcher.DeclaringTypeFQN(mi)             // get declaring type of method call
matcher.AsClass(t)                        // safe cast to *JavaTypeClass
matcher.IsSameGoType(a, b)                // same Go type: folds byte/uint8 and rune/int32,
                                          // and refuses a literal, whose keyword names a class
                                          // of Go types rather than one
matcher.IsAssignableToType(from, to)      // usable where `to` is wanted; an interface is
                                          // answered from the method set, so it misses a
                                          // method promoted from an embedded field
```

Compare two attributed types with `IsSameGoType` rather than their names: the names distinguish `byte` from `uint8`, which are one type.

## Markup Levels

For search-only recipes, use the appropriate severity:

```go
// Warnings — definite issues (SQL injection, panic, unreachable code)
node = node.WithMarkers(java.MarkupWarn(node.Markers, "message"))

// Info — suggestions/awareness (consider X, ensure Y)
node = node.WithMarkers(java.MarkupInfo(node.Markers, "message"))

// Error — critical issues
node = node.WithMarkers(java.MarkupError(node.Markers, "message"))
```

**Never use `java.FoundSearchResult` for new recipes** — use `MarkupWarn` or `MarkupInfo` instead.

## Scanning Recipes

A recipe whose answer spans files embeds `recipe.ScanningBase` and runs in two phases: `Scanner(acc)` visits every file collecting into the accumulator, then `EditorWithData(acc)` rewrites. `UsePackageLevelErrorSentinel` is the worked example — one sentinel name per package, so it cannot decide anything until it has seen the package.

```go
func (r *MyRecipe) InitialValue(*recipe.ExecutionContext) any { return &acc{} }
func (r *MyRecipe) Scanner(a any) recipe.TreeVisitor          { return visitor.Init(&scanner{acc: a.(*acc)}) }
func (r *MyRecipe) EditorWithData(a any) recipe.TreeVisitor   { return visitor.Init(&editor{acc: a.(*acc)}) }
```

Resolve the plan once when `EditorWithData` is called, not per file, so the editor only reads it. Files arrive in no guaranteed order, so anything decided while editing is decided by that order.

## When not to rewrite

The hard part of a Go recipe is the guard, not the rewrite — a recipe that fires too eagerly is worse than no recipe. Three ways that happens, each with a worked guard to read:

- **The output does not compile.** `PreferRegexpMustCompile` (`recipes/style/prefer_regexp_mustcompile.go`) fires only where the call is the sole RHS of a two-variable short declaration, since `MustCompile` returns one value and the assignment expects two.
- **It compiles here and breaks elsewhere.** `RemoveEmptyFunction` (`recipes/style/remove_empty_function.go`) leaves methods alone, since an empty one may satisfy an interface, and leaves functions with a return type alone, since callers use it.
- **It compiles and means something else.** `RemoveRedundantTypeConversion` (`recipes/redundancy/remove_redundant_type_conversion.go`) skips a literal operand, since an untyped constant takes its type from the conversion.

The guard belongs in the visitor, not the description: a recipe that documents a limitation it does not enforce still emits the broken rewrite.

## Returning the original when nothing changed

`RewriteRun` fails a no-change test with "the visitor must return the original pointer when nothing changed". The trap is rebuilding a child slice: `WithStatements` / `WithEntries` / `WithValues` guard on `java.SameSlice`, which compares backing arrays (`&a[0] == &b[0]`), so a freshly allocated slice always counts as a change however its elements compare. Collect into the new slice, track whether any element actually moved, and return the receiver untouched when none did.

## Type Attribution

A hand-written visitor that changes what a node *is* must re-attribute it. `RemoveImport` and `RemoveUnusedImports` answer "is this package still referenced?" from `Identifier.Type` and `MethodInvocation.MethodType.DeclaringType`, so a rewrite that leaves parse-time types in place misleads every later type-based match. Setting the type on the node you replaced is not enough: the enclosing declaration carries one of its own, as does each other reference to the value.

`recipes/internal/lstutil` has the pieces: `MethodOn(recv, name)` reads a method's signature off the receiver's own attribution and is the first choice; `NamedType` / `FuncType` state one the recipe has to name itself. A template attributes the calls it spells out, but not one whose receiver is a capture (`%s.String()`).

A template resolves the stdlib on its own; anything else needs the package's compiler export data shipped alongside and passed to `.ExportData(...)`. `recipes/migration/testify/testifyexportdata` is generated that way, by `go run github.com/openrewrite/rewrite/rewrite-go/cmd/goexportdata -o <dir> <import paths>`. The jsonv2 recipes attribute by hand instead.

Swapping a package under a preserved qualifier works the same way, as long as the recipe rewrites the references. A template carrying the new package's export data attributes both the call and its qualifier to that package, which is what lets `RemoveUnusedImports` drop the superseded import even though the two bind the same name. A recipe that swaps the import while leaving the calls textually untouched emits nothing, so those references keep naming the old package and it has to remap `Identifier.Type`, `FieldAccess.Type` and `MethodInvocation.MethodType.DeclaringType` itself — skipping symbols the new version dropped, since those are rewritten at their own sites and would be overwritten.

`TestRewrittenTreesStayAttributed` in `tests/attribution_test.go` sweeps every registered recipe over the suite's own snippets and fails on a call the parser would have typed. `RewriteRun` compares printed source, so nothing else catches this.


## Go LST Gotchas

### `select` is its own node
`*golang.Select`, a statement whose `Body` holds `golang.CommClause`. A `java.Switch` is never a select.

### A conversion is a `java.TypeCast`
`string(s)` is `*java.TypeCast` with `Clazz` (the target type, in `ControlParentheses`) and `Expr` (the operand) — not a call with no receiver.

### A basic type is a `JavaTypeClass` named for the Go type
`string` attributes as `JavaTypeClass{string}`, not a `JavaTypePrimitive` keyword, and `int`, `int32`, `byte` and `int8` are each distinct. A `J.Literal` still carries a primitive keyword, so a literal's type and a variable's are different kinds.

### `Literal.Value` carries the type the source wrote
`int64` for an integer or rune, `float64` for a float, and `*big.Int` past int64. Code assuming `int64` panics on a wide constant.


### `true`, `false`, `nil` are Identifiers, not Literals
```go
// WRONG
lit, ok := expr.(*java.Literal)

// RIGHT
ident, ok := expr.(*java.Identifier)
if ok && ident.Name == "true" { ... }
```

### A node's leading whitespace is its own `Prefix`
The parser attaches inter-element whitespace to the outermost element, compound nodes included. At `x := a + b` the `java.Binary` has prefix `" "` and its `Left` has `""`:
```go
prefix := bin.Prefix // not bin.Left's
```

### MethodInvocation structure
```go
mi.Select     // *RightPadded[Expression] — receiver/package (nil for builtins)
mi.Name       // *Identifier — method name
mi.Arguments  // Container[Expression] — args (may contain Empty sentinels)
mi.MethodType // *JavaTypeMethod — type info (nullable)
```

### Empty sentinels in argument lists
```go
// Count real arguments (skip Empty sentinels)
var count int
for _, a := range mi.Arguments.Elements {
    if _, isEmpty := a.Element.(*java.Empty); !isEmpty {
        count++
    }
}
```

### Short variable declarations
`:=` is an `*java.Assignment` with a `ShortVarDecl` marker, NOT a separate node type.

### `FieldAccess.Name` is `LeftPadded[*Identifier]`
```go
fa.Name.Element.Name  // get the field name string
```

### Prefix preservation for replacements
Carry the replaced node's own prefix onto the replacement. `template.Apply(v.Cursor(), values)` does this for you, along with parenthesization and formatting, and is the reason to prefer it over hand-building:
```go
expr = lstutil.SetExprPrefix(expr, prefix) // and SetStmtPrefix for a statement
```

Both wrap `format.WithPrefix`, which reaches the `Prefix` field reflectively. A hand-rolled type switch over node kinds silently leaves the whitespace unapplied for a kind it does not list.

## Testing

```go
func TestMyRecipe(t *testing.T) {
    spec := test.NewRecipeSpec().WithRecipe(&MyRecipe{})
    spec.RewriteRun(t,
        test.Golang(`
            package main
            // before code
        `, `
            package main
            // expected after code
        `),
    )
}
```

- **Omit second arg** for no-change tests (search-only or no match)
- **Parse-print idempotence** is automatically validated
- **Space validation** is automatically validated (catches parser bugs)
- **Composite recipes** (RecipeList) are supported — sub-recipes run in sequence

## Diagnostic Mapping

```go
import "github.com/moderneinc/recipes-go/diagnostic"

func (r *MyRecipe) DiagnosticMappings() []diagnostic.Mapping {
    return []diagnostic.Mapping{
        {DiagnosticID: "S1012", Tool: diagnostic.Staticcheck, HasFix: true},
    }
}
```

Tools: `diagnostic.Staticcheck`, `diagnostic.GoVet`, `diagnostic.GolangciLint`

## Registration

Every recipe must be registered in `recipes/activate.go`:

```go
r.Register(&simplification.MyRecipe{}, golang, codeQuality, simplify)
```

Categories: `simplify`, `redundant`, `styleCategory`, `errCategory`, `perfCategory`, `namingCategory`

And added to `tests/validation_test.go` `allRecipes()` for real-repo validation.

## Common Visitor Methods

| Method | Node Type | Common Use |
|--------|-----------|------------|
| `VisitMethodInvocation` | Function/method calls | Match API patterns |
| `VisitBinary` | `a + b`, `a == b` | Simplify expressions |
| `VisitIf` | If statements | Control flow patterns |
| `VisitReturn` | Return statements | Error handling |
| `VisitAssignment` | `x = expr` | Assignment patterns |
| `VisitForLoop` | For loops | Loop patterns |
| `VisitForEachLoop` | For-range loops | Range patterns |
| `VisitGoStmt` | `go expr` | Concurrency |
| `VisitDefer` | `defer expr` | Resource cleanup |
| `VisitSwitch` | Switch/select | Control flow |
| `VisitCase` | Case clauses | Switch cases |
| `VisitBlock` | `{ stmts }` | Block-level transforms |
| `VisitMethodDeclaration` | Function declarations | Function-level analysis |
| `VisitIdentifier` | Names | Identifier patterns |
| `VisitLiteral` | Literals | Value patterns |
| `VisitTypeCast` | `x.(T)` | Type assertions |
| `VisitUnary` | `!x`, `*x`, `&x` | Unary operations |
| `VisitCompilationUnit` | Whole file | File-level analysis |

## Loop Depth Tracking Pattern

For recipes that detect patterns inside loops:

```go
type myVisitor struct {
    visitor.GoVisitor
    insideLoop int
}

func (v *myVisitor) VisitForLoop(forLoop *java.ForLoop, p any) java.J {
    v.insideLoop++
    result := v.GoVisitor.VisitForLoop(forLoop, p)
    v.insideLoop--
    return result
}

func (v *myVisitor) VisitForEachLoop(forEach *java.ForEachLoop, p any) java.J {
    v.insideLoop++
    result := v.GoVisitor.VisitForEachLoop(forEach, p)
    v.insideLoop--
    return result
}

func (v *myVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
    mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
    if v.insideLoop > 0 && someCondition(mi) {
        mi = mi.WithMarkers(java.MarkupWarn(mi.Markers, "found in loop"))
    }
    return mi
}
```
