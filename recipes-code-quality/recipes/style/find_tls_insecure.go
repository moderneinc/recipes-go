/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindTlsInsecureSkipVerify finds `InsecureSkipVerify: true` in TLS
// configuration. Setting this field to true disables certificate
// verification, making the connection vulnerable to man-in-the-middle attacks.
type FindTlsInsecureSkipVerify struct {
	recipe.Base
}

func (r *FindTlsInsecureSkipVerify) Name() string {
	return "org.openrewrite.golang.codequality.FindTlsInsecureSkipVerify"
}
func (r *FindTlsInsecureSkipVerify) DisplayName() string {
	return "Find InsecureSkipVerify: true"
}
func (r *FindTlsInsecureSkipVerify) Description() string {
	return "Find `InsecureSkipVerify: true` in TLS config. Disabling certificate verification makes connections vulnerable to man-in-the-middle attacks."
}
func (r *FindTlsInsecureSkipVerify) Tags() []string { return []string{"style", "security"} }

func (r *FindTlsInsecureSkipVerify) Editor() recipe.TreeVisitor {
	return visitor.Init(&findTlsInsecureSkipVerifyVisitor{})
}

type findTlsInsecureSkipVerifyVisitor struct {
	visitor.GoVisitor
}

func (v *findTlsInsecureSkipVerifyVisitor) VisitKeyValue(kv *tree.KeyValue, p any) tree.J {
	kv = v.GoVisitor.VisitKeyValue(kv, p).(*tree.KeyValue)

	keyIdent, ok := kv.Key.(*tree.Identifier)
	if !ok || keyIdent.Name != "InsecureSkipVerify" {
		return kv
	}

	valIdent, ok := kv.Value.Element.(*tree.Identifier)
	if !ok || valIdent.Name != "true" {
		return kv
	}

	kv = kv.WithMarkers(tree.FoundSearchResult(kv.Markers, "InsecureSkipVerify disables TLS certificate verification"))
	return kv
}
