/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveEmptyFunction removes free functions with empty bodies and no return
// type. These are dead code -- they do nothing when called. Methods (functions
// with receivers) are left alone because they may satisfy an interface.
// Functions with return types are left alone because removing them would break
// callers.
type RemoveEmptyFunction struct {
	recipe.Base
}

func (r *RemoveEmptyFunction) Name() string {
	return "org.openrewrite.golang.codequality.RemoveEmptyFunction"
}
func (r *RemoveEmptyFunction) DisplayName() string { return "Remove empty functions" }
func (r *RemoveEmptyFunction) Description() string {
	return "Remove free functions with empty bodies and no return type. Methods with receivers are preserved because they may implement an interface."
}
func (r *RemoveEmptyFunction) Tags() []string { return []string{"style", "lint"} }

func (r *RemoveEmptyFunction) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeEmptyFunctionVisitor{})
}

type removeEmptyFunctionVisitor struct {
	visitor.GoVisitor
}

func (v *removeEmptyFunctionVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil || md.Body == nil {
		return md
	}

	// Skip methods (have a receiver) -- they may implement an interface.
	if md.Receiver != nil {
		return md
	}

	// Skip functions with return types -- removing them would break callers.
	if md.ReturnType != nil {
		return md
	}

	// Check if the body has any real statements (not just Empty sentinels).
	for _, stmt := range md.Body.Statements {
		if _, isEmpty := stmt.Element.(*tree.Empty); !isEmpty {
			return md
		}
	}

	// Remove the empty function.
	return &tree.Empty{}
}
