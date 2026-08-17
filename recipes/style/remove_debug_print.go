/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (r *RemoveDebugPrint) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "forbidigo", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *RemoveDebugPrint) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeDebugPrintVisitor{})
}

type removeDebugPrintVisitor struct {
	visitor.GoVisitor
	changed bool
}

func (v *removeDebugPrintVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)
	if v.changed {
		v.DoAfterVisit((&recipegolang.RemoveUnusedImports{}).Editor())
	}
	return cu
}

func (v *removeDebugPrintVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// Match built-in println/print (no Select)
	if mi.Select == nil {
		if mi.Name.Name == "println" || mi.Name.Name == "print" {
			return &java.Empty{}
		}
		return mi
	}

	// Match fmt.Println, fmt.Printf, fmt.Print
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "fmt" {
		return mi
	}

	switch mi.Name.Name {
	case "Println", "Printf", "Print":
		v.changed = true
		return &java.Empty{}
	}

	return mi
}
