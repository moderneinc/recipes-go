/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// credentialKeywords lists substrings that indicate a variable may hold a credential.
var credentialKeywords = []string{"password", "secret", "token", "apikey", "api_key"}

// FindHardcodedCredentials finds string literals assigned to variables whose
// names suggest they hold credentials (password, secret, token, apikey, api_key).
// Hardcoded credentials are a security risk and should be loaded from
// configuration or a secrets manager instead.
type FindHardcodedCredentials struct {
	recipe.Base
}

func (r *FindHardcodedCredentials) Name() string {
	return "org.openrewrite.golang.codequality.FindHardcodedCredentials"
}
func (r *FindHardcodedCredentials) DisplayName() string { return "Find hardcoded credentials" }
func (r *FindHardcodedCredentials) Description() string {
	return "Find string literals assigned to variables whose names suggest they hold credentials (password, secret, token, apikey, api_key)."
}
func (r *FindHardcodedCredentials) Tags() []string { return []string{"security"} }

func (r *FindHardcodedCredentials) Editor() recipe.TreeVisitor {
	return visitor.Init(&findHardcodedCredentialsVisitor{})
}

type findHardcodedCredentialsVisitor struct {
	visitor.GoVisitor
}

func (v *findHardcodedCredentialsVisitor) VisitVariableDeclarator(vd *tree.VariableDeclarator, p any) tree.J {
	vd = v.GoVisitor.VisitVariableDeclarator(vd, p).(*tree.VariableDeclarator)

	if vd.Initializer == nil {
		return vd
	}

	lit, ok := vd.Initializer.Element.(*tree.Literal)
	if !ok || lit.Kind != tree.StringLiteral {
		return vd
	}

	varName := strings.ToLower(vd.Name.Name)
	for _, keyword := range credentialKeywords {
		if strings.Contains(varName, keyword) {
			vd = vd.WithMarkers(tree.FoundSearchResult(vd.Markers, "hardcoded credential"))
			return vd
		}
	}

	return vd
}
