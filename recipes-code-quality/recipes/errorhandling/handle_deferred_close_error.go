/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// HandleDeferredCloseError transforms `defer x.Close()` into
// `defer func() { _ = x.Close() }()` so that the error return value
// is explicitly handled rather than silently discarded.
type HandleDeferredCloseError struct {
	recipe.Base
}

func (r *HandleDeferredCloseError) Name() string {
	return "org.openrewrite.golang.codequality.HandleDeferredCloseError"
}
func (r *HandleDeferredCloseError) DisplayName() string { return "Handle deferred Close() error" }
func (r *HandleDeferredCloseError) Description() string {
	return "Wrap `defer x.Close()` in a closure to explicitly handle the error: `defer func() { _ = x.Close() }()`."
}
func (r *HandleDeferredCloseError) Tags() []string { return []string{"error-handling"} }

func (r *HandleDeferredCloseError) Editor() recipe.TreeVisitor {
	return visitor.Init(&handleDeferredCloseErrorVisitor{})
}

type handleDeferredCloseErrorVisitor struct {
	visitor.GoVisitor
}

func (v *handleDeferredCloseErrorVisitor) VisitDefer(d *tree.Defer, p any) tree.J {
	d = v.GoVisitor.VisitDefer(d, p).(*tree.Defer)

	mi, ok := d.Expr.(*tree.MethodInvocation)
	if !ok {
		return d
	}

	if mi.Name.Name != "Close" {
		return d
	}

	// Build: defer func() { _ = f.Close() }()
	//
	// Step 1: Move the original Close() call, stripping its leading space
	// so it sits right after "= " in the assignment.
	closeCall := mi.WithPrefix(tree.EmptySpace)

	// Step 2: _ = f.Close()
	assignment := &tree.Assignment{
		Prefix:   tree.SingleSpace,
		Variable: &tree.Identifier{Name: "_"},
		Value:    tree.LeftPadded[tree.Expression]{Before: tree.SingleSpace, Element: closeCall},
	}

	// Step 3: func() { _ = f.Close() }  — a MethodDeclaration with empty name
	funcLiteral := &tree.MethodDeclaration{
		Prefix:     tree.SingleSpace,
		Name:       &tree.Identifier{Name: ""},
		Parameters: tree.Container[tree.Statement]{},
		Body: &tree.Block{
			Prefix: tree.SingleSpace,
			Statements: []tree.RightPadded[tree.Statement]{
				{Element: assignment},
			},
			End: tree.SingleSpace,
		},
	}

	// Step 4: func() { _ = f.Close() }()  — call the literal
	outerCall := &tree.MethodInvocation{
		Prefix:    tree.EmptySpace,
		Select:    &tree.RightPadded[tree.Expression]{Element: funcLiteral},
		Name:      &tree.Identifier{Name: ""},
		Arguments: tree.Container[tree.Expression]{},
	}

	// Step 5: Keep the defer, replace its expression
	return &tree.Defer{
		ID:      d.ID,
		Prefix:  d.Prefix,
		Markers: d.Markers,
		Expr:    outerCall,
	}
}
