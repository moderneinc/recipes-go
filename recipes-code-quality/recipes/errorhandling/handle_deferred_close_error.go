/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *handleDeferredCloseErrorVisitor) VisitDefer(d *golang.Defer, p any) java.J {
	d = v.GoVisitor.VisitDefer(d, p).(*golang.Defer)

	mi, ok := d.Expr.(*java.MethodInvocation)
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
	closeCall := mi.WithPrefix(java.EmptySpace)

	// Step 2: _ = f.Close()
	assignment := &java.Assignment{
		Prefix:   java.SingleSpace,
		Variable: &java.Identifier{Name: "_"},
		Value:    java.LeftPadded[java.Expression]{Before: java.SingleSpace, Element: closeCall},
	}

	// Step 3: func() { _ = f.Close() }  — a MethodDeclaration with empty name
	funcLiteral := &java.MethodDeclaration{
		Prefix:     java.SingleSpace,
		Name:       &java.Identifier{Name: ""},
		Parameters: java.Container[java.Statement]{},
		Body: &java.Block{
			Prefix: java.SingleSpace,
			Statements: []java.RightPadded[java.Statement]{
				{Element: assignment},
			},
			End: java.SingleSpace,
		},
	}

	// Step 4: func() { _ = f.Close() }()  — call the literal
	outerCall := &java.MethodInvocation{
		Prefix:    java.EmptySpace,
		Select:    &java.RightPadded[java.Expression]{Element: funcLiteral},
		Name:      &java.Identifier{Name: ""},
		Arguments: java.Container[java.Expression]{},
	}

	// Step 5: Keep the defer, replace its expression
	return &golang.Defer{
		ID:      d.ID,
		Prefix:  d.Prefix,
		Markers: d.Markers,
		Expr:    outerCall,
	}
}
