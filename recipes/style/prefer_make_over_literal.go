/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// PreferMakeForEmptyMap replaces `map[K]V{}` with `make(map[K]V)` for
// empty map initialization. This is the idiomatic Go style when the map
// will be populated later.
type PreferMakeForEmptyMap struct {
	recipe.Base
}

func (r *PreferMakeForEmptyMap) Name() string {
	return "org.openrewrite.golang.codequality.PreferMakeForEmptyMap"
}
func (r *PreferMakeForEmptyMap) DisplayName() string { return "Prefer make() for empty maps" }
func (r *PreferMakeForEmptyMap) Description() string {
	return "Replace empty map literal `map[K]V{}` with `make(map[K]V)` for clarity."
}
func (r *PreferMakeForEmptyMap) Tags() []string { return []string{"style"} }

func (r *PreferMakeForEmptyMap) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferMakeForEmptyMapVisitor{})
}

type preferMakeForEmptyMapVisitor struct {
	visitor.GoVisitor
}

func (v *preferMakeForEmptyMapVisitor) VisitComposite(comp *golang.Composite, p any) java.J {
	comp = v.GoVisitor.VisitComposite(comp, p).(*golang.Composite)

	mapType, ok := comp.TypeExpr.(*golang.MapType)
	if !ok {
		return comp
	}

	// A `{}` holding only comments carries them as elements, and make() has
	// nowhere to put them.
	if len(comp.Elements.Elements) > 0 {
		return comp
	}

	// `&map[K]V{}` is legal Go; `&make(map[K]V)` is not, since make is a call.
	if unary, ok := v.Cursor().Parent().Value().(*golang.Unary); ok && unary.Operator.Element == golang.AddressOf {
		return comp
	}

	return &java.MethodInvocation{
		ID:     uuid.New(),
		Prefix: comp.Prefix,
		Name:   &java.Identifier{ID: uuid.New(), Name: "make"},
		Arguments: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{Element: mapType.WithPrefix(java.EmptySpace)},
			},
		},
	}
}

// --- Template-expressible recipes ---

var (
	newS1 = template.Expr("s")
	newN1 = template.Expr("n")
)

// PreferStringsEqualFold replaces `strings.ToLower(s) == strings.ToLower(n)`
// with `strings.EqualFold(s, n)` for case-insensitive comparison.
// Staticcheck: SA6005
type PreferStringsEqualFold struct {
	recipe.Base
}

func (r *PreferStringsEqualFold) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsEqualFold"
}
func (r *PreferStringsEqualFold) DisplayName() string { return "Prefer strings.EqualFold" }
func (r *PreferStringsEqualFold) Description() string {
	return "Replace `strings.ToLower(s) == strings.ToLower(n)` with `strings.EqualFold(s, n)`."
}
func (r *PreferStringsEqualFold) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferStringsEqualFold) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA6005", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferEqualFoldLower = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsEqualFold$Lower"),
	template.WithDisplayName("ToLower == ToLower → EqualFold"),
	template.WithBefore(
		fmt.Sprintf(`strings.ToLower(%s) == strings.ToLower(%s)`, newS1, newN1),
		template.Imports("strings"),
	),
	template.WithAfter(
		fmt.Sprintf(`strings.EqualFold(%s, %s)`, newS1, newN1),
		template.Imports("strings"),
	),
	template.WithCaptures(newS1, newN1),
)

var preferEqualFoldUpper = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsEqualFold$Upper"),
	template.WithDisplayName("ToUpper == ToUpper → EqualFold"),
	template.WithBefore(
		fmt.Sprintf(`strings.ToUpper(%s) == strings.ToUpper(%s)`, newS1, newN1),
		template.Imports("strings"),
	),
	template.WithAfter(
		fmt.Sprintf(`strings.EqualFold(%s, %s)`, newS1, newN1),
		template.Imports("strings"),
	),
	template.WithCaptures(newS1, newN1),
)

func (r *PreferStringsEqualFold) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferEqualFoldLower, preferEqualFoldUpper}
}
