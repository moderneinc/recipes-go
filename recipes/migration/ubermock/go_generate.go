/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package ubermock

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Repoints the mockgen invocation in a //go:generate directive at the fork.
type UpdateMockgenGoGenerateDirectives struct {
	recipe.Base
}

func (r *UpdateMockgenGoGenerateDirectives) Name() string {
	return "org.openrewrite.golang.migration.UpdateMockgenGoGenerateDirectives"
}
func (r *UpdateMockgenGoGenerateDirectives) DisplayName() string {
	return "Point `//go:generate mockgen` at `go.uber.org/mock`"
}
func (r *UpdateMockgenGoGenerateDirectives) Description() string {
	return "Rewrite `github.com/golang/mock` to `go.uber.org/mock` inside `//go:generate` directives, so `go generate` runs the maintained mockgen. Covers both the `go run github.com/golang/mock/mockgen` form and the vendored `go run ./vendor/github.com/golang/mock/mockgen` form."
}
func (r *UpdateMockgenGoGenerateDirectives) Tags() []string {
	return []string{"migration", "mock", "testing"}
}

func (r *UpdateMockgenGoGenerateDirectives) Editor() recipe.TreeVisitor {
	return visitor.Init(&goGenerateVisitor{})
}

// The Go parser models a directive comment as a java.Annotation whose type names
// the directive and whose sole argument is the rest of the line as written, so
// the rewrite is a node edit rather than comment-text surgery.
type goGenerateVisitor struct {
	visitor.GoVisitor
}

func (v *goGenerateVisitor) VisitAnnotation(ann *java.Annotation, p any) java.J {
	ann = v.GoVisitor.VisitAnnotation(ann, p).(*java.Annotation)
	name, ok := ann.AnnotationType.(*java.Identifier)
	if !ok || name.Name != "go:generate" || ann.Arguments == nil {
		return ann
	}

	elements := make([]java.RightPadded[java.Expression], len(ann.Arguments.Elements))
	copy(elements, ann.Arguments.Elements)
	changed := false
	for i, rp := range elements {
		lit, ok := rp.Element.(*java.Literal)
		if !ok || !strings.Contains(lit.Source, oldModule) {
			continue
		}
		rewritten := *lit
		rewritten.Source = strings.ReplaceAll(lit.Source, oldModule, newModule)
		rewritten.Value = rewritten.Source
		elements[i].Element = &rewritten
		changed = true
	}
	if !changed {
		return ann
	}
	args := *ann.Arguments
	args.Elements = elements
	c := *ann
	c.Arguments = &args
	return &c
}
