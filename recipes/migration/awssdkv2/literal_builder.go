/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// A config literal held in a local and filled in field by field — DisableSSL
// among them, which v2's config has no field for — becomes the generated
// builder too. Its fields hold the pointers v1's did, so the literal and each
// read and write keep their values and only the names change.
var builderFields = map[string]string{
	"Region":         "region",
	"Endpoint":       "endpoint",
	disableSSLField:  "disableSSL",
	s3ForcePathStyle: "pathStyle",
	"MaxRetries":     "maxRetries",
	credentialsField: "credentials",
	"HTTPClient":     "httpClient",
}

// literalBuilders picks the config locals that become builders: declared as
// an &aws.Config literal, reached only through the builder's fields and by the
// sessions made from it, and setting DisableSSL somewhere.
func (s *fileScan) literalBuilders() {
	for name, local := range s.configLocals {
		if !local.pointerLiteral || local.reassigned || local.credentialsRead || local.sessionArgs == 0 || local.singleValue > 0 {
			continue
		}
		if local.uses != 1+local.fieldUses+local.sessionArgs {
			continue
		}
		disables := false
		mapped := true
		for _, field := range append(append([]string{}, local.literalKeys...), local.fields...) {
			if builderFields[field] == "" {
				mapped = false
			}
			disables = disables || field == disableSSLField
		}
		if !mapped || !disables {
			continue
		}
		s.builderVars[name] = true
		s.literalBuilderVars[name] = true
		delete(s.configMutated, name)
		delete(s.configKept, name)
		delete(s.configRestructured, name)
		delete(s.configPathStyle, name)
	}
}

// literalBuilderField reports whether fa reaches a field of a builder local,
// returning the builder's name for it.
func (s *fileScan) literalBuilderField(fa *java.FieldAccess) (string, bool) {
	holder, isIdent := fa.Target.(*java.Identifier)
	if !isIdent || !s.literalBuilderVars[holder.Name] || fa.Name.Element == nil {
		return "", false
	}
	field := builderFields[fa.Name.Element.Name]
	return field, field != ""
}

// literalBuilderDecl reports whether the cursor's composite is the literal a
// builder local is declared from.
func (s *fileScan) literalBuilderDecl(cursor *visitor.Cursor) bool {
	parent := cursor.Parent()
	for parent != nil {
		switch node := parent.Value().(type) {
		case nil:
			parent = parent.Parent()
			continue
		case *golang.Unary:
			parent = parent.Parent()
			continue
		case *java.Assignment:
			target, isIdent := node.Variable.(*java.Identifier)
			return isIdent && s.literalBuilderVars[target.Name]
		}
		return false
	}
	return false
}

// literalBuilderComposite retypes a builder local's literal as the builder,
// its keys as the builder's fields.
func (v *migrateVisitor) literalBuilderComposite(comp *golang.Composite) *golang.Composite {
	c := *comp
	c.TypeExpr = &java.Identifier{Prefix: comp.TypeExpr.GetPrefix(), Name: builderType, Type: lstutil.NamedType(builderType)}
	elements := make([]java.RightPadded[java.Expression], len(comp.Elements.Elements))
	for i, rp := range comp.Elements.Elements {
		if kv, isKeyValue := rp.Element.(*golang.KeyValue); isKeyValue {
			if key, isIdent := kv.Key.(*java.Identifier); isIdent && builderFields[key.Name] != "" {
				k := *kv
				k.Key = &java.Identifier{Prefix: key.Prefix, Name: builderFields[key.Name]}
				rp.Element = &k
			}
		}
		elements[i] = rp
	}
	c.Elements.Elements = elements
	v.needsConfigBuilder, v.needsDisableSSL, v.needsPathStyle = true, true, true
	return &c
}

// literalBuilderFieldAccess renames a reach into a builder local's field.
func (v *migrateVisitor) literalBuilderFieldAccess(fa *java.FieldAccess) (*java.FieldAccess, bool) {
	field, ok := v.scan.literalBuilderField(fa)
	if !ok {
		return fa, false
	}
	c := *fa
	c.Name = java.LeftPadded[*java.Identifier]{Before: fa.Name.Before, Element: &java.Identifier{Prefix: fa.Name.Element.Prefix, Name: field}}
	c.Type = nil
	return &c, true
}
