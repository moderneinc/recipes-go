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
    "github.com/moderneinc/recipes-go/code-quality/diagnostic"
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

func (v *myRecipeVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
    mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)
    // Always call base visitor first to recurse into children
    // Then apply transformation logic
    return mi
}
```

## Key Conventions

- **License**: `Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.`
- **Imports**: Always from `github.com/openrewrite/rewrite/rewrite-go/pkg/...`
- **Recipe naming**: `org.openrewrite.golang.codequality.<RecipeName>`
- **Visitor**: Always call `v.GoVisitor.VisitX(node, p).(*tree.X)` first to recurse
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
- Captures match any subtree at their syntactic position

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

## MethodMatcher

Pattern-based method invocation matching using AspectJ-style syntax:

```go
import "github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"

var fmtSprintf = matcher.NewMethodMatcher("fmt Sprintf(..)")

func (v *myVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
    mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)
    if fmtSprintf.Matches(mi) {
        // matched
    }
    return mi
}
```

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
```

## Markup Levels

For search-only recipes, use the appropriate severity:

```go
// Warnings — definite issues (SQL injection, panic, unreachable code)
node = node.WithMarkers(tree.MarkupWarn(node.Markers, "message"))

// Info — suggestions/awareness (consider X, ensure Y)
node = node.WithMarkers(tree.MarkupInfo(node.Markers, "message"))

// Error — critical issues
node = node.WithMarkers(tree.MarkupError(node.Markers, "message"))
```

**Never use `tree.FoundSearchResult` for new recipes** — use `MarkupWarn` or `MarkupInfo` instead.

## Go LST Gotchas

### `true`, `false`, `nil` are Identifiers, not Literals
```go
// WRONG
lit, ok := expr.(*tree.Literal)

// RIGHT
ident, ok := expr.(*tree.Identifier)
if ok && ident.Name == "true" { ... }
```

### Binary.Prefix is typically empty
The leading whitespace lives on `Binary.Left`, not on the Binary itself:
```go
// Get effective prefix of a binary expression:
prefix := exprPrefix(bin.Left)  // NOT bin.Prefix
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
    if _, isEmpty := a.Element.(*tree.Empty); !isEmpty {
        count++
    }
}
```

### Short variable declarations
`:=` is an `*tree.Assignment` with a `ShortVarDecl` marker, NOT a separate node type.

### `FieldAccess.Name` is `LeftPadded[*Identifier]`
```go
fa.Name.Element.Name  // get the field name string
```

### Prefix preservation for replacements
Use `setLeadingPrefix` (from the template package) which walks to the leftmost leaf:
```go
// For compound nodes (MethodInvocation, Binary, FieldAccess):
// setPrefix() sets the ROOT prefix — often wrong
// setLeadingPrefix() sets the FIRST LEAF prefix — correct
```

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
import "github.com/moderneinc/recipes-go/code-quality/diagnostic"

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

func (v *myVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
    v.insideLoop++
    result := v.GoVisitor.VisitForLoop(forLoop, p)
    v.insideLoop--
    return result
}

func (v *myVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
    v.insideLoop++
    result := v.GoVisitor.VisitForEachLoop(forEach, p)
    v.insideLoop--
    return result
}

func (v *myVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
    mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)
    if v.insideLoop > 0 && someCondition(mi) {
        mi = mi.WithMarkers(tree.MarkupWarn(mi.Markers, "found in loop"))
    }
    return mi
}
```
