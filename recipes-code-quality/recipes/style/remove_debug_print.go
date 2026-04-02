/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveDebugPrint removes calls to `fmt.Println`, `fmt.Printf`, `fmt.Print`,
// `println`, and `print`. These are often left as debug statements and should
// be replaced with structured logging.
type RemoveDebugPrint struct {
	recipe.Base
}

func (r *RemoveDebugPrint) Name() string {
	return "org.openrewrite.golang.codequality.RemoveDebugPrint"
}
func (r *RemoveDebugPrint) DisplayName() string { return "Remove debug print statements" }
func (r *RemoveDebugPrint) Description() string {
	return "Remove calls to `fmt.Println`, `fmt.Printf`, `fmt.Print`, `println`, and `print`."
}
func (r *RemoveDebugPrint) Tags() []string { return []string{"style"} }

func (r *RemoveDebugPrint) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeDebugPrintVisitor{})
}

type removeDebugPrintVisitor struct {
	visitor.GoVisitor
}

func (v *removeDebugPrintVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match built-in println/print (no Select)
	if mi.Select == nil {
		if mi.Name.Name == "println" || mi.Name.Name == "print" {
			return &tree.Empty{}
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
		return &tree.Empty{}
	}

	return mi
}
