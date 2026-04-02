/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidDotImport removes the dot alias from `import . "pkg"`, converting it to
// a normal `import "pkg"`. Dot imports pollute the local namespace and make it
// harder to understand where identifiers come from. Code using unqualified names
// from the dot-imported package will need to be updated to use qualified names.
// golangci-lint: revive (dot-imports)
type AvoidDotImport struct {
	recipe.Base
}

func (r *AvoidDotImport) Name() string {
	return "org.openrewrite.golang.codequality.AvoidDotImport"
}
func (r *AvoidDotImport) DisplayName() string { return "Avoid dot imports" }
func (r *AvoidDotImport) Description() string {
	return "Remove the dot alias from `import . \"pkg\"`, converting to a normal import."
}
func (r *AvoidDotImport) Tags() []string { return []string{"style", "lint"} }

func (r *AvoidDotImport) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "dot-imports", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *AvoidDotImport) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidDotImportVisitor{})
}

type avoidDotImportVisitor struct {
	visitor.GoVisitor
}

func (v *avoidDotImportVisitor) VisitImport(imp *tree.Import, p any) tree.J {
	imp = v.GoVisitor.VisitImport(imp, p).(*tree.Import)

	if imp.Alias == nil {
		return imp
	}

	aliasIdent := imp.Alias.Element
	if aliasIdent.Name != "." {
		return imp
	}

	// Remove the dot alias, converting `import . "pkg"` to `import "pkg"`.
	// The qualid's prefix was the space between "." and the path string.
	// With the alias gone, the import prefix covers the space between
	// "import" and the path, so we clear the qualid's leading whitespace.
	c := *imp
	c.Alias = nil
	if lit, ok := c.Qualid.(*tree.Literal); ok {
		c.Qualid = lit.WithPrefix(tree.EmptySpace)
	} else if ident, ok := c.Qualid.(*tree.Identifier); ok {
		c.Qualid = ident.WithPrefix(tree.EmptySpace)
	}
	return &c
}
