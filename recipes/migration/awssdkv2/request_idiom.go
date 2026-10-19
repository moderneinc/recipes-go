/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v1's OperationRequest built a request that Send ran, after which the request
// held the raw HTTP response — present for a failed call as for a successful
// one — and the error; hypnoglow/helm-s3 reads a bucket's region off a HEAD
// that fails. v2 has no request objects. A generated stand-in runs the v2
// operation with a step recording the raw response, and answers Send,
// HTTPResponse and Error as v1's request did. Anything else read off the
// request holds the file back.

const (
	requestHelper = "awsCompatNewRequest"
	requestFile   = "zz_awssdkv2_request.go"
)

// requestLocal is a request an OperationRequest call returned.
type requestLocal struct {
	service, operation, output string
}

// requestCall reads `req, _ := client.OpRequest(input)`.
func (s *fileScan) requestCall(ma *golang.MultiAssignment) (*java.Identifier, *java.MethodInvocation, requestLocal, bool) {
	if !java.HasMarker[golang.ShortVarDecl](ma.Markers) || len(ma.Variables) != 2 || len(ma.Values) != 1 {
		return nil, nil, requestLocal{}, false
	}
	req, isIdent := ma.Variables[0].Element.(*java.Identifier)
	out, isOut := ma.Variables[1].Element.(*java.Identifier)
	// The output is filled by Send in v1; only a discarded one is followed.
	if !isIdent || !isOut || out.Name != "_" {
		return nil, nil, requestLocal{}, false
	}
	mi, isCall := ma.Values[0].Element.(*java.MethodInvocation)
	if !isCall || mi.Name == nil || len(realArgs(mi)) != 1 {
		return nil, nil, requestLocal{}, false
	}
	operation, isRequest := strings.CutSuffix(mi.Name.Name, "Request")
	if !isRequest || operation == "" {
		return nil, nil, requestLocal{}, false
	}
	service, isClient := s.clientOperationService(mi)
	if !isClient {
		service, isClient = s.attributedClient(mi.Select.Element)
	}
	if !isClient {
		return nil, nil, requestLocal{}, false
	}
	input, output, isOperation := awsmanifest.OperationShapes(service, operation)
	if !isOperation {
		return nil, nil, requestLocal{}, false
	}
	if svc, shape, isShape := s.inputShape(realArgs(mi)[0]); isShape && (svc != service || shape != input) {
		return nil, nil, requestLocal{}, false
	}
	return req, mi, requestLocal{service: service, operation: operation, output: output}, true
}

// requestLocals finds the requests whose every use the stand-in answers.
func (s *fileScan) requestLocals(cu *golang.CompilationUnit) {
	scan := visitor.Init(&requestScan{scan: s, accepted: map[string]int{}, found: map[string]*java.MethodInvocation{}})
	scan.Visit(cu, nil)
	for name, call := range scan.found {
		if scan.accepted[name] == s.configUses[name] {
			s.requestCalls[call] = true
			s.requests[name] = true
		}
	}
}

type requestScan struct {
	visitor.GoVisitor
	scan     *fileScan
	accepted map[string]int
	found    map[string]*java.MethodInvocation
}

func (v *requestScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	if req, call, _, ok := v.scan.requestCall(ma); ok {
		v.found[req.Name] = call
		v.accepted[req.Name]++
	}
	return v.GoVisitor.VisitMultiAssignment(ma, p)
}

func (v *requestScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if recv, isIdent := selectIdentifier(mi); isIdent && mi.Name != nil && mi.Name.Name == "Send" && len(realArgs(mi)) == 0 {
		v.accepted[recv.Name]++
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

func (v *requestScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if target, isIdent := fa.Target.(*java.Identifier); isIdent && fa.Name.Element != nil && !assignedTo(v.Cursor()) {
		switch fa.Name.Element.Name {
		case "HTTPResponse", "Error":
			v.accepted[target.Name]++
		}
	}
	return v.GoVisitor.VisitFieldAccess(fa, p)
}

// requestStandIn builds the stand-in for an OperationRequest call, running
// the v2 operation with the step recording its response added to its stack.
func (v *migrateVisitor) requestStandIn(mi *java.MethodInvocation, local requestLocal) (java.Expression, bool) {
	ctx := v.contextExpr()
	if ctx == nil || mi.Select == nil {
		return nil, false
	}
	serviceLocal := v.scan.serviceLocal(local.service)
	source := fmt.Sprintf(`package p

var _ = %s(awsCtx, func(ctx context.Context, capture awsCompatCapture) (*%s.%s, error) {
	return awsClient.%s(ctx, awsInput, func(o *%s.Options) { o.APIOptions = append(o.APIOptions, capture) })
})
`, requestHelper, serviceLocal, local.output, local.operation, serviceLocal)
	cu, err := parser.NewGoParser().Parse("request.go", source)
	if err != nil {
		return nil, false
	}
	lift := visitor.Init(&callLift{name: requestHelper})
	lift.Visit(cu, nil)
	if lift.call == nil {
		return nil, false
	}
	helper := *lift.call
	helper.MethodType = lstutil.FuncType("", requestHelper, nil)
	var out java.Expression = visitor.Init(&methodTyping{name: local.operation, declaring: v2ServicePath(local.service) + ".Client"}).Visit(&helper, nil).(java.Expression)
	for name, value := range map[string]java.Expression{"awsCtx": ctx, "awsClient": mi.Select.Element, "awsInput": realArgs(mi)[0]} {
		bound, ok := visitor.Init(&placeholderBind{name: name, value: value}).Visit(out, nil).(java.Expression)
		if !ok {
			return nil, false
		}
		out = bound
	}
	v.needsRequest, v.needsContext = true, true
	v.serviceUsed[local.service] = true
	return lstutil.SetExprPrefix(out, mi.Prefix), true
}

const requestSource = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import (
	"context"
	"net/http"

	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// awsCompatCapture adds to an operation's stack the step recording its raw
// response.
type awsCompatCapture = func(*middleware.Stack) error

// awsCompatRequest stands in for the aws-sdk-go v1 request an operation's
// Request method returned. Send runs the operation; HTTPResponse then holds
// the raw response, present for a failed call too, and Error the error.
type awsCompatRequest[T any] struct {
	HTTPResponse *http.Response
	Error        error
	ctx          context.Context
	run          func(context.Context, awsCompatCapture) (*T, error)
	data         *T
}

// awsCompatNewRequest returns the request and the output Send fills, as v1's
// OperationRequest did.
func awsCompatNewRequest[T any](ctx context.Context, run func(context.Context, awsCompatCapture) (*T, error)) (*awsCompatRequest[T], *T) {
	r := &awsCompatRequest[T]{ctx: ctx, run: run, data: new(T)}
	return r, r.data
}

func (r *awsCompatRequest[T]) Send() error {
	capture := func(stack *middleware.Stack) error {
		return stack.Deserialize.Add(middleware.DeserializeMiddlewareFunc("awsCompatCapture", func(ctx context.Context, in middleware.DeserializeInput, next middleware.DeserializeHandler) (middleware.DeserializeOutput, middleware.Metadata, error) {
			out, metadata, err := next.HandleDeserialize(ctx, in)
			if resp, ok := out.RawResponse.(*smithyhttp.Response); ok {
				r.HTTPResponse = resp.Response
			}
			return out, metadata, err
		}), middleware.After)
	}
	out, err := r.run(r.ctx, capture)
	if out != nil {
		*r.data = *out
	}
	r.Error = err
	return err
}
`

// serviceLocal is the name the file binds a service's package to.
func (s *fileScan) serviceLocal(service string) string {
	for local, svc := range s.services {
		if svc == service {
			return local
		}
	}
	return service
}

// requestRewrite replaces an OperationRequest call with the stand-in.
func (v *migrateVisitor) requestRewrite(original, visited *golang.MultiAssignment) (*golang.MultiAssignment, bool) {
	req, _, local, ok := v.scan.requestCall(original)
	if !ok || !v.scan.requests[req.Name] {
		return nil, false
	}
	call, isCall := visited.Values[0].Element.(*java.MethodInvocation)
	if !isCall {
		return nil, false
	}
	args := realArgs(call)
	if len(args) == 0 {
		return nil, false
	}
	// The input is the call's last argument, whatever context went before it.
	c := *call
	c.Arguments.Elements = []java.RightPadded[java.Expression]{{Element: args[len(args)-1]}}
	standIn, built := v.requestStandIn(&c, local)
	if !built {
		return nil, false
	}
	out := *visited
	values := make([]java.RightPadded[java.Expression], len(visited.Values))
	copy(values, visited.Values)
	values[0].Element = standIn
	out.Values = values
	return &out, true
}

// methodTyping attributes the calls named name to declaring.
type methodTyping struct {
	visitor.GoVisitor
	name, declaring string
}

func (v *methodTyping) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Name == nil || mi.Name.Name != v.name {
		return mi
	}
	c := *mi
	c.MethodType = lstutil.FuncType(v.declaring, v.name, nil)
	return &c
}
