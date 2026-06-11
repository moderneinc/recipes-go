/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveGetterPrefix removes the "Get" prefix from method names. In Go, getters
// should not have the "Get" prefix — `GetName()` should be `Name()`.
// golangci-lint: revive (exported)
type RemoveGetterPrefix struct {
	recipe.Base
}

func (r *RemoveGetterPrefix) Name() string {
	return "org.openrewrite.golang.codequality.RemoveGetterPrefix"
}
func (r *RemoveGetterPrefix) DisplayName() string { return "Remove getter prefix" }
func (r *RemoveGetterPrefix) Description() string {
	return "Remove the \"Get\" prefix from method names. Go convention is that getters should not have the \"Get\" prefix. Callers of this method will need to be updated separately."
}
func (r *RemoveGetterPrefix) Tags() []string { return []string{"naming"} }

func (r *RemoveGetterPrefix) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeGetterPrefixVisitor{})
}

type removeGetterPrefixVisitor struct {
	visitor.GoVisitor
}

// Only methods (which carry a receiver) are wrapped in golang.MethodDeclaration;
// free functions stay as a bare java.MethodDeclaration. Visiting the wrapper
// restricts this recipe to methods.
func (v *removeGetterPrefixVisitor) VisitGoMethodDeclaration(md *golang.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitGoMethodDeclaration(md, p).(*golang.MethodDeclaration)

	decl := md.Declaration
	if decl == nil || decl.Name == nil {
		return md
	}

	funcName := decl.Name.Name
	if len(funcName) <= 3 {
		return md
	}

	if !strings.HasPrefix(funcName, "Get") {
		return md
	}

	// Strip "Get" prefix from the method name.
	newName := strings.TrimPrefix(funcName, "Get")
	c := *md
	c.Declaration = decl.WithName(decl.Name.WithName(newName).WithMarkers(
		java.MarkupInfo(decl.Name.Markers, "callers of "+funcName+" must be updated to use "+newName),
	))
	return &c
}
