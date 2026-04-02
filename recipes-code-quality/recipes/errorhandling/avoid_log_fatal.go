/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidLogFatal replaces `log.Fatal` / `log.Fatalf` / `log.Fatalln` with
// `log.Println` / `log.Printf` / `log.Println`. The Fatal variants call
// os.Exit(1) and do not run deferred functions.
type AvoidLogFatal struct {
	recipe.Base
}

func (r *AvoidLogFatal) Name() string {
	return "org.openrewrite.golang.codequality.AvoidLogFatal"
}
func (r *AvoidLogFatal) DisplayName() string { return "Avoid log.Fatal" }
func (r *AvoidLogFatal) Description() string {
	return "Replace `log.Fatal`, `log.Fatalf`, and `log.Fatalln` with their non-exiting equivalents (`log.Println`, `log.Printf`, `log.Println`)."
}
func (r *AvoidLogFatal) Tags() []string { return []string{"error-handling", "lint"} }

func (r *AvoidLogFatal) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidLogFatalVisitor{})
}

type avoidLogFatalVisitor struct {
	visitor.GoVisitor
}

// fatalReplacements maps log.Fatal* method names to their safe equivalents.
var fatalReplacements = map[string]string{
	"Fatal":   "Println",
	"Fatalf":  "Printf",
	"Fatalln": "Println",
}

func (v *avoidLogFatalVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "log" {
		return mi
	}

	replacement, found := fatalReplacements[mi.Name.Name]
	if !found {
		return mi
	}

	// Replace the method name: Fatal→Println, Fatalf→Printf, Fatalln→Println
	mi = mi.WithName(mi.Name.WithName(replacement))
	return mi
}
