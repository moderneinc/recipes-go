/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindDebugPrint finds calls to `fmt.Println`, `fmt.Printf`, `fmt.Print`,
// `println`, and `print`. These are often left as debug statements and should
// be replaced with structured logging.
type FindDebugPrint struct {
	recipe.Base
}

func (r *FindDebugPrint) Name() string {
	return "org.openrewrite.golang.codequality.FindDebugPrint"
}
func (r *FindDebugPrint) DisplayName() string { return "Find debug print statements" }
func (r *FindDebugPrint) Description() string {
	return "Find calls to `fmt.Println`, `fmt.Printf`, `fmt.Print`, `println`, and `print`. These are often left as debug statements."
}
func (r *FindDebugPrint) Tags() []string { return []string{"style"} }

func (r *FindDebugPrint) Editor() recipe.TreeVisitor {
	return visitor.Init(&findDebugPrintVisitor{})
}

type findDebugPrintVisitor struct {
	visitor.GoVisitor
}

func (v *findDebugPrintVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match built-in println/print (no Select)
	if mi.Select == nil {
		if mi.Name.Name == "println" || mi.Name.Name == "print" {
			mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "debug print statement; consider using structured logging"))
		}
		return mi
	}

	// Match fmt.Println, fmt.Printf, fmt.Print
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "fmt" {
		return mi
	}

	switch mi.Name.Name {
	case "Println", "Printf", "Print":
		mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "debug print statement; consider using structured logging"))
	}

	return mi
}
