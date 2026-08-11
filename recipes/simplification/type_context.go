/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Reports the source text of the type the value at the cursor must locally
// satisfy from a direct `return` or a `var x T = call` declaration, or
// ("", false) when the type is inferred or otherwise not locally knowable.
func requiredResultType(c *visitor.Cursor) (string, bool) {
	parent := c.Parent()
	if parent == nil {
		return "", false
	}
	switch parent.Value().(type) {
	case *java.Return, *golang.Return:
		return enclosingFirstResultType(c)
	case *java.VariableDeclarator:
		return explicitDeclType(parent)
	}
	return "", false
}

// Returns the source text of the first result type of the function enclosing
// the cursor.
func enclosingFirstResultType(c *visitor.Cursor) (string, bool) {
	md, ok := visitor.FirstEnclosing[*java.MethodDeclaration](c)
	if !ok {
		return "", false
	}
	types := functionResultTypes(md)
	if len(types) == 0 {
		return "", false
	}
	return types[0], true
}

// Returns the source text of each of a function's result types in order, read
// from the signature rather than the LST's resolved scalar types which do not
// reliably distinguish int from int64.
func functionResultTypes(md *java.MethodDeclaration) []string {
	if md == nil || md.ReturnType == nil {
		return nil
	}
	tl, ok := md.ReturnType.(*golang.TypeList)
	if !ok {
		// Single unnamed result, e.g. func f() error.
		return []string{strings.TrimSpace(printer.Print(md.ReturnType))}
	}
	out := make([]string, 0, len(tl.Types.Elements))
	for _, e := range tl.Types.Elements {
		var typeExpr java.J = e.Element
		if vd, ok := e.Element.(*java.VariableDeclarations); ok {
			typeExpr = vd.TypeExpr
		}
		if typeExpr == nil {
			out = append(out, "")
			continue
		}
		out = append(out, strings.TrimSpace(printer.Print(typeExpr)))
	}
	return out
}

// Returns the declared type of a `var x T = ...` declaration, or ("", false)
// for inferred declarations that carry no explicit type.
func explicitDeclType(declarator *visitor.Cursor) (string, bool) {
	gp := declarator.Parent()
	if gp == nil {
		return "", false
	}
	vd, ok := gp.Value().(*java.VariableDeclarations)
	if !ok || vd.TypeExpr == nil {
		return "", false
	}
	return strings.TrimSpace(printer.Print(vd.TypeExpr)), true
}
