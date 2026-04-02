/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// EnforceTlsVerification replaces `InsecureSkipVerify: true` with
// `InsecureSkipVerify: false` in TLS configuration. Setting this field to
// true disables certificate verification, making the connection vulnerable
// to man-in-the-middle attacks.
type EnforceTlsVerification struct {
	recipe.Base
}

func (r *EnforceTlsVerification) Name() string {
	return "org.openrewrite.golang.codequality.EnforceTlsVerification"
}
func (r *EnforceTlsVerification) DisplayName() string {
	return "Enforce TLS verification"
}
func (r *EnforceTlsVerification) Description() string {
	return "Replace `InsecureSkipVerify: true` with `false` in TLS config. Disabling certificate verification makes connections vulnerable to man-in-the-middle attacks."
}
func (r *EnforceTlsVerification) Tags() []string { return []string{"style", "security"} }

func (r *EnforceTlsVerification) Editor() recipe.TreeVisitor {
	return visitor.Init(&enforceTlsVerificationVisitor{})
}

type enforceTlsVerificationVisitor struct {
	visitor.GoVisitor
}

func (v *enforceTlsVerificationVisitor) VisitComposite(c *tree.Composite, p any) tree.J {
	c = v.GoVisitor.VisitComposite(c, p).(*tree.Composite)
	newElems := make([]tree.RightPadded[tree.Expression], len(c.Elements.Elements))
	for i, rp := range c.Elements.Elements {
		visited := v.GoVisitor.Visit(rp.Element, p)
		if visited != nil {
			rp.Element = visited.(tree.Expression)
		}
		newElems[i] = rp
	}
	c.Elements.Elements = newElems
	return c
}

func (v *enforceTlsVerificationVisitor) VisitKeyValue(kv *tree.KeyValue, p any) tree.J {
	kv = v.GoVisitor.VisitKeyValue(kv, p).(*tree.KeyValue)

	keyIdent, ok := kv.Key.(*tree.Identifier)
	if !ok || keyIdent.Name != "InsecureSkipVerify" {
		return kv
	}

	valIdent, ok := kv.Value.Element.(*tree.Identifier)
	if !ok || valIdent.Name != "true" {
		return kv
	}

	newVal := valIdent.WithName("false")
	return &tree.KeyValue{
		ID:      kv.ID,
		Prefix:  kv.Prefix,
		Markers: kv.Markers,
		Key:     kv.Key,
		Value:   tree.LeftPadded[tree.Expression]{Before: kv.Value.Before, Element: newVal},
	}
}
