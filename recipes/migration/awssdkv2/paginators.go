/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"strings"
	"sync"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v1 iterated pages by handing the client a callback; v2 gives out a paginator
// the caller drives. The two are not the same statement, so rather than turn a
// callback into a loop — which would mean rewriting every `return` in its body,
// and getting `break` wrong inside a switch — the loop is wrapped around the
// callback, unchanged, in a function literal called on the spot. That keeps the
// expression an expression, so it drops into whatever the v1 call site was.
const paginateSource = `func() error {
	awsFn := %s
	awsPager := %s.New%sPaginator(%s, %s)
	for awsPager.HasMorePages() {
		awsPage, awsErr := awsPager.NextPage(%s)
		if awsErr != nil {
			return awsErr
		}
		if !awsFn(awsPage, !awsPager.HasMorePages()) {
			return nil
		}
	}
	return nil
}()`

var (
	awsFn     = template.Expr("awsFn")
	awsClient = template.Expr("awsClient")
	awsInput  = template.Expr("awsInput")
)

var (
	paginateMu    sync.Mutex
	paginateCache = map[string]*template.GoTemplate{}
)

// paginateTemplate builds the wrapper for one service and operation. The
// paginator's name is part of the source, so there is one per operation.
func paginateTemplate(serviceLocal, operation string) (*template.GoTemplate, bool) {
	key := serviceLocal + "." + operation

	paginateMu.Lock()
	defer paginateMu.Unlock()
	if t, ok := paginateCache[key]; ok {
		return t, t != nil
	}
	paginateCache[key] = nil

	source := fmt.Sprintf(paginateSource, awsFn, serviceLocal, operation, awsClient, awsInput, awsCtx)
	t := template.ExpressionTemplate(source).
		Captures(awsFn, awsClient, awsInput, awsCtx).
		Imports(contextPkg).
		Build()
	if t == nil {
		return nil, false
	}
	paginateCache[key] = t
	return t, true
}

// pagesCall reports whether mi is a v1 page iterator this recipe rewrites,
// returning what the v2 form is built from. The operation is read off the input
// the call takes rather than the receiver, since the receiver is as often a
// struct field or an interface as it is a client value.
func (s *fileScan) pagesCall(mi *java.MethodInvocation) (serviceLocal, operation string, client, input, fn, ctx java.Expression, ok bool) {
	if mi.Name == nil || mi.Select == nil {
		return "", "", nil, nil, nil, nil, false
	}
	name := mi.Name.Name
	withContext := strings.HasSuffix(name, "PagesWithContext")
	switch {
	case withContext && name != "PagesWithContext":
		operation = strings.TrimSuffix(name, "PagesWithContext")
	case strings.HasSuffix(name, "Pages") && name != "Pages":
		operation = strings.TrimSuffix(name, "Pages")
	default:
		return "", "", nil, nil, nil, nil, false
	}

	args := realArgs(mi)
	// v1 took an optional trailing request.Option, which v2 configures through
	// the client's own options instead.
	want := 2
	if withContext {
		want = 3
		if len(args) > 0 {
			ctx = args[0]
			args = args[1:]
		}
	}
	if len(args)+boolToInt(withContext) != want {
		return "", "", nil, nil, nil, nil, false
	}
	input, fn = args[0], args[1]

	service, shape, isShape := s.inputShape(input)
	if !isShape {
		return "", "", nil, nil, nil, nil, false
	}
	wantInput, _, isOperation := awsmanifest.OperationShapes(service, operation)
	if !isOperation || shape != wantInput {
		return "", "", nil, nil, nil, nil, false
	}
	if awsmanifest.Place(service, "New"+operation+"Paginator") != awsmanifest.InService {
		return "", "", nil, nil, nil, nil, false
	}
	// The callback has to stay a callback for the wrapper to call it.
	if _, isLiteral := funcLiteral(fn); !isLiteral {
		if _, isName := fn.(*java.Identifier); !isName {
			return "", "", nil, nil, nil, nil, false
		}
	}
	return s.qualifierFor(service), operation, mi.Select.Element, input, fn, ctx, true
}

// isPages reports whether a method name is one of v1's page iterators.
func isPages(name string) bool {
	for _, suffix := range blockedCallSuffixes {
		if strings.HasSuffix(name, suffix) && name != suffix {
			return true
		}
	}
	return false
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// funcLiteral unwraps the statement the parser wraps a function literal in when
// it stands where an expression is expected.
func funcLiteral(expr java.Expression) (*java.MethodDeclaration, bool) {
	if se, isStatement := expr.(*golang.StatementExpression); isStatement {
		md, isFunc := se.Statement.(*java.MethodDeclaration)
		return md, isFunc
	}
	md, isFunc := expr.(*java.MethodDeclaration)
	return md, isFunc
}

// inputShape names the service and operation-input shape an expression carries,
// whether written as a literal or held in a variable the scan recognised.
func (s *fileScan) inputShape(expr java.Expression) (service, shape string, ok bool) {
	if unary, isUnary := expr.(*golang.Unary); isUnary {
		expr = unary.Expression
	}
	if comp, isComposite := expr.(*golang.Composite); isComposite {
		service, shape, ok = s.compositeShape(comp)
		return service, shape, ok && strings.HasSuffix(shape, "Input")
	}
	id, isIdent := expr.(*java.Identifier)
	if !isIdent {
		return "", "", false
	}
	ref, held := s.shapes[id.Name]
	if !held || !strings.HasSuffix(ref.shape, "Input") {
		return "", "", false
	}
	return ref.service, ref.shape, true
}

// paginate rewrites a v1 page iterator as the loop v2 wants around the callback
// it was already given.
func (v *migrateVisitor) paginate(mi *java.MethodInvocation) (java.J, bool) {
	serviceLocal, operation, client, input, fn, ctx, ok := v.scan.pagesCall(mi)
	if !ok {
		return nil, false
	}
	t, ok := paginateTemplate(serviceLocal, operation)
	if !ok {
		return nil, false
	}
	if ctx == nil {
		if ctx = v.contextExpr(); ctx == nil {
			return nil, false
		}
	}
	v.serviceUsed[v.scan.services[serviceLocal]] = true

	values := template.NewMatchResult().
		Bind(awsFn, lstutil.SetExprPrefix(fn, java.EmptySpace)).
		Bind(awsClient, lstutil.SetExprPrefix(client, java.EmptySpace)).
		Bind(awsInput, lstutil.SetExprPrefix(input, java.EmptySpace)).
		Bind(awsCtx, lstutil.SetExprPrefix(ctx, java.EmptySpace))
	applied := t.Apply(v.Cursor(), values)
	if applied == nil {
		return nil, false
	}
	service := v.scan.services[serviceLocal]
	_, output, _ := awsmanifest.OperationShapes(service, operation)
	return attributePaginator(applied, service, serviceLocal, operation, output), true
}

// attributePaginator types the names the template introduced. The v2 service
// packages are not in the export data a template attributes from — there are
// twenty-nine of them, and the blobs would dwarf the recipes — so the paginator,
// its methods and the page they yield are named here instead.
func attributePaginator(tree java.J, service, serviceLocal, operation, output string) java.J {
	attributed := visitor.Init(&paginatorAttrib{
		servicePath:  v2ServicePkg + service,
		serviceLocal: serviceLocal,
		constructor:  "New" + operation + "Paginator",
		pagerType:    v2ServicePkg + service + "." + operation + "Paginator",
		outputType:   v2ServicePkg + service + "." + output,
	}).Visit(tree, nil)
	if out, ok := attributed.(java.J); ok {
		return out
	}
	return tree
}

type paginatorAttrib struct {
	visitor.GoVisitor
	servicePath  string
	serviceLocal string
	constructor  string
	pagerType    string
	outputType   string
}

func (v *paginatorAttrib) VisitIdentifier(id *java.Identifier, p any) java.J {
	var named string
	switch id.Name {
	case "awsPager":
		named = v.pagerType
	case "awsPage":
		named = v.outputType
	default:
		return v.GoVisitor.VisitIdentifier(id, p)
	}
	if id.Type != nil {
		return v.GoVisitor.VisitIdentifier(id, p)
	}
	c := *id
	c.Type = lstutil.NamedType(named)
	return &c
}

func (v *paginatorAttrib) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Name == nil || mi.MethodType != nil {
		return mi
	}
	c := *mi
	if mi.Select == nil {
		if mi.Name.Name == "awsFn" {
			c.MethodType = lstutil.FuncType("", "awsFn", nil)
			return &c
		}
		return mi
	}
	recv, isIdent := mi.Select.Element.(*java.Identifier)
	if !isIdent {
		return mi
	}
	switch {
	case recv.Name == "awsPager":
		c.MethodType = lstutil.FuncType(v.pagerType, mi.Name.Name, nil)
	case recv.Name == v.serviceLocal && mi.Name.Name == v.constructor:
		c.MethodType = lstutil.FuncType(v.servicePath, mi.Name.Name, nil)
		c.Select = &java.RightPadded[java.Expression]{
			Element: &java.Identifier{Prefix: recv.Prefix, Name: recv.Name, Type: lstutil.NamedType(v.servicePath)},
			After:   mi.Select.After,
		}
	default:
		return mi
	}
	return &c
}
