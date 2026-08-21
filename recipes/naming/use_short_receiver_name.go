/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseShortReceiverName renames method receivers longer than 2 characters to the
// first lowercase letter of the type name, leaving alone any whose short name is
// already bound in the method. Go convention is to use short, one-letter receiver
// names derived from the type.
type UseShortReceiverName struct {
	recipe.Base
}

func (r *UseShortReceiverName) Name() string {
	return "org.openrewrite.golang.codequality.UseShortReceiverName"
}
func (r *UseShortReceiverName) DisplayName() string { return "Use short receiver name" }
func (r *UseShortReceiverName) Description() string {
	return "Rename method receivers longer than 2 characters to the first lowercase letter of the type name, unless that name is already bound in the method."
}
func (r *UseShortReceiverName) Tags() []string { return []string{"naming"} }

func (r *UseShortReceiverName) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "ST1006", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *UseShortReceiverName) Editor() recipe.TreeVisitor {
	return visitor.Init(&useShortReceiverNameVisitor{})
}

type useShortReceiverNameVisitor struct {
	visitor.GoVisitor
}

func (v *useShortReceiverNameVisitor) VisitGoMethodDeclaration(md *golang.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitGoMethodDeclaration(md, p).(*golang.MethodDeclaration)

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

	// Renaming onto a name the method already binds would either fail to compile
	// or silently rebind the body's references to it, so leave the receiver alone.
	if boundNames(md, typeName)[newName] {
		return md
	}

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
	c := *md
	c.Receiver.Elements = newParams

	// Rename usages in body.
	if c.Declaration != nil && c.Declaration.Body != nil {
		renamer := visitor.Init(&receiverRenameVisitor{oldName: oldName, newName: newName})
		newDecl := *c.Declaration
		newDecl.Body = renamer.Visit(newDecl.Body, p).(*java.Block)
		c.Declaration = &newDecl
	}

	return &c
}

// extractTypeName returns the simple type name from a type expression,
// unwrapping pointer types (Unary with Deref operator).
func extractTypeName(expr java.Expression) string {
	if expr == nil {
		return ""
	}
	// Pointer type: *Foo may be PointerType or Unary(Indirection).
	if pt, ok := expr.(*golang.PointerType); ok {
		return extractTypeName(pt.Elem)
	}
	if u, ok := expr.(*golang.Unary); ok {
		return extractTypeName(u.Expression)
	}
	if u, ok := expr.(*java.Unary); ok {
		return extractTypeName(u.Operand)
	}
	if ident, ok := expr.(*java.Identifier); ok {
		return ident.Name
	}
	return ""
}

// boundNames returns every name bound in md's scope other than the receiver
// itself: the receiver's type name, parameters, named results, and locals
// declared anywhere in the body.
func boundNames(md *golang.MethodDeclaration, typeName string) map[string]bool {
	names := map[string]bool{typeName: true}
	decl := md.Declaration
	if decl == nil {
		return names
	}
	collector := visitor.Init(&boundNameVisitor{names: names})
	for _, param := range decl.Parameters.Elements {
		collector.Visit(param.Element, nil)
	}
	if decl.ReturnType != nil {
		collector.Visit(decl.ReturnType, nil)
	}
	if decl.Body != nil {
		collector.Visit(decl.Body, nil)
	}
	return names
}

type boundNameVisitor struct {
	visitor.GoVisitor
	names map[string]bool
}

// Parameters, named results and `var` declarations all share this node.
func (v *boundNameVisitor) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	for _, variable := range vd.Variables {
		if name := variable.Element.Name; name != nil {
			v.names[name.Name] = true
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

// An assignment target is recorded whether it declares (`:=`) or merely writes
// (`=`), since a plain write names something the receiver would shadow too.
func (v *boundNameVisitor) VisitAssignment(assign *java.Assignment, p any) java.J {
	v.recordIdentifier(assign.Variable)
	return v.GoVisitor.VisitAssignment(assign, p)
}

func (v *boundNameVisitor) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	for _, variable := range ma.Variables {
		v.recordIdentifier(variable.Element)
	}
	return v.GoVisitor.VisitMultiAssignment(ma, p)
}

func (v *boundNameVisitor) recordIdentifier(expr java.Expression) {
	if ident, ok := expr.(*java.Identifier); ok {
		v.names[ident.Name] = true
	}
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
