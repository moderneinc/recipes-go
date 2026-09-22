/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"sync"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Every v2 operation has the same signature shape, so the parameters an
// interface or a mock has to declare are derivable from the operation's name.
// They are produced by parsing a synthetic declaration rather than assembled by
// hand, which is what gets the variadic option parameter right.
const operationParamsSource = `package p

import "context"

type shim interface {
	Op(%s) (*%s.%s, error)
}
`

var (
	operationParamsMu    sync.Mutex
	operationParamsCache = map[string][]java.RightPadded[java.Statement]{}
)

// operationParams returns the parameter list a v2 operation declares. inputName
// is what the declaration calls its input, empty where it names nothing — Go
// refuses a parameter list that mixes named and unnamed entries.
func operationParams(serviceLocal, operation, input, output, inputName string) ([]java.RightPadded[java.Statement], bool) {
	ctxName := "ctx"
	if inputName == ctxName {
		// The declaration already calls its input what the context would be.
		ctxName = "awsCtx"
	}
	key := fmt.Sprintf("%s|%s|%s|%s|%s", serviceLocal, input, output, inputName, ctxName)

	operationParamsMu.Lock()
	defer operationParamsMu.Unlock()
	if params, ok := operationParamsCache[key]; ok {
		return params, params != nil
	}
	operationParamsCache[key] = nil

	// The declaration's own name for its input is kept: the body refers to it,
	// and an interface declares its parameters without names at all.
	list := fmt.Sprintf("context.Context, *%s.%s, ...func(*%s.Options)", serviceLocal, input, serviceLocal)
	if inputName != "" {
		list = fmt.Sprintf("%s context.Context, %s *%s.%s, optFns ...func(*%s.Options)", ctxName, inputName, serviceLocal, input, serviceLocal)
	}
	cu, err := parser.NewGoParser().Parse("shim.go", fmt.Sprintf(operationParamsSource, list, serviceLocal, output))
	if err != nil {
		return nil, false
	}
	lift := visitor.Init(&paramLift{})
	lift.Visit(cu, nil)
	if lift.params == nil {
		return nil, false
	}
	operationParamsCache[key] = lift.params
	return lift.params, true
}

type paramLift struct {
	visitor.GoVisitor
	params []java.RightPadded[java.Statement]
}

func (v *paramLift) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	if md.Name != nil && md.Name.Name == "Op" {
		v.params = md.Parameters.Elements
	}
	return v.GoVisitor.VisitMethodDeclaration(md, p)
}

// awsOperationDeclaration reports whether a method declaration is a v1-shaped
// AWS operation — one input pointer in, the matching output and an error out —
// returning what the v2 form needs. Both an interface's method and a mock's
// override take this shape.
func (s *fileScan) awsOperationDeclaration(md *java.MethodDeclaration) (serviceLocal, input, output, inputName string, ok bool) {
	if md.Name == nil || len(md.Parameters.Elements) != 1 {
		return "", "", "", "", false
	}
	vd, isDecl := md.Parameters.Elements[0].Element.(*java.VariableDeclarations)
	if !isDecl {
		return "", "", "", "", false
	}
	ptr, isPointer := vd.TypeExpr.(*golang.PointerType)
	if !isPointer {
		return "", "", "", "", false
	}
	fa, isField := ptr.Elem.(*java.FieldAccess)
	if !isField || fa.Name.Element == nil {
		return "", "", "", "", false
	}
	local, isIdent := fa.Target.(*java.Identifier)
	if !isIdent {
		return "", "", "", "", false
	}
	service, isService := s.services[local.Name]
	if !isService {
		return "", "", "", "", false
	}
	wantInput, wantOutput, isOperation := awsmanifest.OperationShapes(service, md.Name.Name)
	if !isOperation || fa.Name.Element.Name != wantInput {
		return "", "", "", "", false
	}

	// An interface declares its parameters without names; a concrete method
	// names them, and its body refers to that name, so it is what the
	// replacement keeps.
	for _, decl := range vd.Variables {
		if decl.Element != nil && decl.Element.Name != nil && decl.Element.Name.Name != "" {
			inputName = decl.Element.Name.Name
		}
	}
	return local.Name, wantInput, wantOutput, inputName, true
}
