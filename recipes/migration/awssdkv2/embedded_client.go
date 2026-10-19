/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"path"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// Go names an embedded field after its type, so a struct embedding v1's
// *s3.S3 has a field S3, which v2's *s3.Client renames Client. The methods it
// promotes keep their names; a literal's key and a selector naming the field
// follow the rename. Nothing in the parse-time type says a field is embedded,
// so the scan records the structs declaring one, package by package.

// recordEmbeddedClients notes the structs cu declares that embed a v1 client.
func (a *migrateAcc) recordEmbeddedClients(cu *golang.CompilationUnit, scan *fileScan) {
	dir := path.Dir(cu.SourcePath)
	for _, rp := range cu.Statements {
		decl, isType := rp.Element.(*golang.TypeDecl)
		if !isType || decl.Name == nil {
			continue
		}
		st, isStruct := decl.Definition.(*golang.StructType)
		if !isStruct || st.Body == nil {
			continue
		}
		for _, field := range st.Body.Statements {
			vd, isDecl := field.Element.(*java.VariableDeclarations)
			if !isDecl || namesParameter(vd) {
				continue
			}
			t := vd.TypeExpr
			if ptr, isPointer := t.(*golang.PointerType); isPointer {
				t = ptr.Elem
			}
			fa, isField := t.(*java.FieldAccess)
			if !isField || fa.Name.Element == nil {
				continue
			}
			local, isIdent := fa.Target.(*java.Identifier)
			if !isIdent {
				continue
			}
			service, isService := scan.services[local.Name]
			if !isService || awsmanifest.Place(service, fa.Name.Element.Name) != awsmanifest.IsClient {
				continue
			}
			if a.embeddedClients[dir] == nil {
				a.embeddedClients[dir] = map[string]string{}
			}
			a.embeddedClients[dir][decl.Name.Name] = fa.Name.Element.Name
		}
	}
}

// embeddedClientField reports whether name is, on a value of type t, a field
// embedding a v1 client.
func (v *migrateVisitor) embeddedClientField(t java.JavaType, name string) bool {
	if v.acc == nil {
		return false
	}
	class := matcher.AsClass(t)
	if class == nil {
		return false
	}
	fqn := class.FullyQualifiedName
	typeName := fqn[strings.LastIndex(fqn, ".")+1:]
	return v.acc.embeddedClients[v.dir][typeName] == name
}

// renamedEmbeddedKeys renames a literal's keys naming an embedded client,
// realigning the values gofmt aligned.
func (v *migrateVisitor) renamedEmbeddedKeys(comp *golang.Composite, t java.JavaType) *golang.Composite {
	var elements []java.RightPadded[java.Expression]
	for i, rp := range comp.Elements.Elements {
		kv, isKeyValue := rp.Element.(*golang.KeyValue)
		if !isKeyValue {
			continue
		}
		key, isIdent := kv.Key.(*java.Identifier)
		if !isIdent || !v.embeddedClientField(t, key.Name) {
			continue
		}
		if elements == nil {
			elements = make([]java.RightPadded[java.Expression], len(comp.Elements.Elements))
			copy(elements, comp.Elements.Elements)
		}
		renamed := *kv
		renamed.Key = &java.Identifier{Prefix: key.Prefix, Name: "Client"}
		elements[i].Element = &renamed
	}
	if elements == nil {
		return comp
	}
	realign(elements)
	c := *comp
	c.Elements.Elements = elements
	return &c
}

// realign pads each value to the longest key, as gofmt aligns a literal laid
// out one key to a line, when the values were padded with spaces alone.
func realign(elements []java.RightPadded[java.Expression]) {
	longest := 0
	for _, rp := range elements {
		kv, isKeyValue := rp.Element.(*golang.KeyValue)
		if !isKeyValue {
			return
		}
		key, isIdent := kv.Key.(*java.Identifier)
		if !isIdent || strings.Trim(kv.Value.Element.GetPrefix().Whitespace(), " ") != "" || !strings.Contains(kv.Prefix.Whitespace(), "\n") {
			return
		}
		longest = max(longest, len(key.Name))
	}
	for i, rp := range elements {
		kv := rp.Element.(*golang.KeyValue)
		padded := *kv
		pad := longest - len(kv.Key.(*java.Identifier).Name) + 1
		padded.Value.Element = lstutil.SetExprPrefix(kv.Value.Element, java.MakeSpace(nil, strings.Repeat(" ", pad)))
		elements[i].Element = &padded
	}
}
