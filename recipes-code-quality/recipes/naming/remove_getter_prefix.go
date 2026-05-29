/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
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

func (v *removeGetterPrefixVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)

	if md.Name == nil {
		return md
	}

	// Only check methods (with a receiver), not free functions.
	if md.Receiver == nil {
		return md
	}

	funcName := md.Name.Name
	if len(funcName) <= 3 {
		return md
	}

	if !strings.HasPrefix(funcName, "Get") {
		return md
	}

	// Strip "Get" prefix from the method name.
	newName := strings.TrimPrefix(funcName, "Get")
	md = md.WithName(md.Name.WithName(newName).WithMarkers(
		java.MarkupInfo(md.Name.Markers, "callers of "+funcName+" must be updated to use "+newName),
	))
	return md
}
