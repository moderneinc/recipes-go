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

// UseShortReceiverName renames method receivers longer than 2 characters to the
// first lowercase letter of the type name. Go convention is to use short,
// one-letter receiver names derived from the type.
type UseShortReceiverName struct {
	recipe.Base
}

func (r *UseShortReceiverName) Name() string {
	return "org.openrewrite.golang.codequality.UseShortReceiverName"
}
func (r *UseShortReceiverName) DisplayName() string { return "Use short receiver name" }
func (r *UseShortReceiverName) Description() string {
	return "Rename method receivers longer than 2 characters to the first lowercase letter of the type name."
}
func (r *UseShortReceiverName) Tags() []string { return []string{"naming"} }

func (r *UseShortReceiverName) Editor() recipe.TreeVisitor {
	return visitor.Init(&useShortReceiverNameVisitor{})
}

type useShortReceiverNameVisitor struct {
	visitor.GoVisitor
}

func (v *useShortReceiverNameVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)

	if md.Receiver == nil {
		return md
	}

	// Get receiver param.
	params := md.Receiver.Elements
	if len(params) == 0 {
		return md
	}
	vd, ok := params[0].Element.(*java.VariableDeclarations)
	if !ok || len(vd.Variables) == 0 {
		return md
	}
	nameIdent := vd.Variables[0].Element.Name
	if nameIdent == nil {
		return md
	}
	oldName := nameIdent.Name
	if len(oldName) <= 2 {
		return md
	}

	// Get type name for short name.
	typeName := extractTypeName(vd.TypeExpr)
	if typeName == "" {
		return md
	}
	newName := strings.ToLower(string([]rune(typeName)[0:1]))

	// Rename receiver param.
	newNameIdent := nameIdent.WithName(newName)
	newVarDecl := vd.Variables[0].Element.WithName(newNameIdent)
	newVars := []java.RightPadded[*java.VariableDeclarator]{
		{Element: newVarDecl, After: vd.Variables[0].After, Markers: vd.Variables[0].Markers},
	}
	newVd := *vd
	newVd.Variables = newVars
	newParams := []java.RightPadded[java.Statement]{
		{Element: &newVd, After: params[0].After, Markers: params[0].Markers},
	}
	newReceiver := *md.Receiver
	newReceiver.Elements = newParams
	c := *md
	c.Receiver = &newReceiver

	// Rename usages in body.
	if c.Body != nil {
		renamer := visitor.Init(&receiverRenameVisitor{oldName: oldName, newName: newName})
		c.Body = renamer.Visit(c.Body, p).(*java.Block)
	}

	return &c
}

// extractTypeName returns the simple type name from a type expression,
// unwrapping pointer types (Unary with Deref operator).
func extractTypeName(expr java.Expression) string {
	if expr == nil {
		return ""
	}
	// Pointer type: *Foo may be PointerType or Unary(Deref).
	if pt, ok := expr.(*golang.PointerType); ok {
		return extractTypeName(pt.Elem)
	}
	if u, ok := expr.(*java.Unary); ok {
		return extractTypeName(u.Operand)
	}
	if ident, ok := expr.(*java.Identifier); ok {
		return ident.Name
	}
	return ""
}

// receiverRenameVisitor renames identifiers matching the old receiver name.
type receiverRenameVisitor struct {
	visitor.GoVisitor
	oldName string
	newName string
}

func (v *receiverRenameVisitor) VisitIdentifier(ident *java.Identifier, p any) java.J {
	ident = v.GoVisitor.VisitIdentifier(ident, p).(*java.Identifier)
	if ident.Name == v.oldName {
		return ident.WithName(v.newName)
	}
	return ident
}
