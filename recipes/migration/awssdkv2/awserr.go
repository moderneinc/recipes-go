/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

const (
	v1Awserr     = v1Aws + "/awserr"
	smithyGo     = "github.com/aws/smithy-go"
	errorsPkg    = "errors"
	smithyAPIErr = "APIError"
)

// awserrMethodRenames map the v1 error interface's accessors to smithy's.
var awserrMethodRenames = map[string]string{
	"Code":    "ErrorCode",
	"Message": "ErrorMessage",
}

// awserrSupported are the awserr names with a mechanical v2 form. The
// constructors and the batched shapes have none, and block their file.
var awserrSupported = map[string]bool{"Error": true, requestFailure: true}

// A RequestFailure is v1's error for a failed response, which v2 wraps every
// such failure in as an *awshttp.ResponseError, answering the same status and
// request ID under other names. Its code and message live on the APIError it
// wraps, which a separate match reaches.
const (
	requestFailure = "RequestFailure"
	v2AwsHTTP      = v2Module + "/aws/transport/http"
	awsHTTPLocal   = "awshttp"
)

var requestFailureRenames = map[string]string{
	"StatusCode": "HTTPStatusCode",
	"RequestID":  "ServiceRequestID",
}

// responseErrorType names *awshttp.ResponseError.
func responseErrorType() java.Expression {
	return &golang.PointerType{
		Prefix: java.SingleSpace,
		Elem: &java.FieldAccess{
			Target: &java.Identifier{Name: awsHTTPLocal, Type: lstutil.NamedType(v2AwsHTTP)},
			Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: "ResponseError"}},
			Type:   lstutil.NamedType(v2AwsHTTP + ".ResponseError"),
		},
	}
}

// awserrTargetType is the type errors.As fills for an awserr interface.
func awserrTargetType(ref *java.FieldAccess) java.Expression {
	if isRequestFailure(ref) {
		return responseErrorType()
	}
	return smithyAPIErrorType()
}

func isRequestFailure(ref *java.FieldAccess) bool {
	return ref != nil && ref.Name.Element != nil && ref.Name.Element.Name == requestFailure
}

// awserrAssertion reports whether an if's init asserts an error to
// awserr.Error, returning the bound name, the ok name and the error expression.
// This is the shape every caller writes: `if aerr, ok := err.(awserr.Error); …`.
func (s *fileScan) awserrAssertion(swi *golang.StatementWithInit) (bound, okName *java.Identifier, errExpr java.Expression, ref *java.FieldAccess, ok bool) {
	assignment, isMulti := swi.Init.Element.(*golang.MultiAssignment)
	if !isMulti {
		return nil, nil, nil, nil, false
	}
	return s.awserrAssertionOf(assignment)
}

// awserrAssertionOf reads `bound, ok := err.(awserr.Error)`, returning the
// awserr.Error reference along with the names and the error.
func (s *fileScan) awserrAssertionOf(assignment *golang.MultiAssignment) (bound, okName *java.Identifier, errExpr java.Expression, ref *java.FieldAccess, ok bool) {
	if s.awserrPkg == "" || len(assignment.Variables) != 2 || len(assignment.Values) != 1 {
		return nil, nil, nil, nil, false
	}
	bound, isIdent := assignment.Variables[0].Element.(*java.Identifier)
	if !isIdent {
		return nil, nil, nil, nil, false
	}
	okName, isIdent = assignment.Variables[1].Element.(*java.Identifier)
	if !isIdent {
		return nil, nil, nil, nil, false
	}
	assertion, isAssertion := assignment.Values[0].Element.(*golang.TypeAssertion)
	if !isAssertion || assertion.AssertedType == nil {
		return nil, nil, nil, nil, false
	}
	fa, isField := assertion.AssertedType.Tree.Element.(*java.FieldAccess)
	if !isField || !s.awserrError(fa) {
		return nil, nil, nil, nil, false
	}
	return bound, okName, assertion.Left.Element, fa, true
}

// awserrError reports whether fa names awserr.Error, or the RequestFailure
// that extends it.
func (s *fileScan) awserrError(fa *java.FieldAccess) bool {
	name, isAwserr := qualifiedRef(fa, s.awserrPkg)
	return isAwserr && (name == "Error" || name == requestFailure)
}

// awserrDeclared reports whether a declaration's type is awserr.Error, the
// target an errors.As call fills.
func (s *fileScan) awserrDeclared(vd *java.VariableDeclarations) (*java.FieldAccess, bool) {
	fa, isField := vd.TypeExpr.(*java.FieldAccess)
	return fa, isField && s.awserrError(fa)
}

// awserrNew reports whether mi is awserr.New, returning its code, message and
// the error it wraps.
func (s *fileScan) awserrNew(mi *java.MethodInvocation) (code, message, orig java.Expression, ok bool) {
	call, isAwserr := qualifiedCall(mi, s.awserrPkg)
	args := realArgs(mi)
	if !isAwserr || call != "New" || len(args) != 3 {
		return nil, nil, nil, false
	}
	return args[0], args[1], args[2], true
}

// awserrRequestFailure reports whether mi is awserr.NewRequestFailure around an
// error the rewrite also carries over.
func (s *fileScan) awserrRequestFailure(mi *java.MethodInvocation) bool {
	call, isAwserr := qualifiedCall(mi, s.awserrPkg)
	args := realArgs(mi)
	if !isAwserr || call != "NewRequestFailure" || len(args) != 3 {
		return false
	}
	switch inner := args[0].(type) {
	case *java.MethodInvocation:
		_, _, _, ok := s.awserrNew(inner)
		return ok
	case *java.FieldAccess:
		return s.missingEndpoint(inner)
	}
	return false
}

// missingEndpoint reports whether fa is aws.ErrMissingEndpoint.
func (s *fileScan) missingEndpoint(fa *java.FieldAccess) bool {
	name, isAws := qualifiedRef(fa, s.awsPkg)
	return isAws && name == "ErrMissingEndpoint"
}

// v1's awserr.New built an error answering its code and message, reading
// "Code: message" and the wrapped error after; NewRequestFailure added the
// response's status and request ID. smithy's own errors read differently, so
// generated ones stand in, answering smithy's APIError as v1's answered
// awserr.Error. Neither unwraps, as v1's did not.
const (
	errorHelper          = "awsCompatNewError"
	requestFailureHelper = "awsCompatRequestFailure"
	errorHelperFile      = "zz_awssdkv2_errors.go"
)

const errorHelperSource = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import (
	"fmt"

	"github.com/aws/smithy-go"
)

// awsCompatError stands in for an error aws-sdk-go v1's awserr.New built. It
// answers smithy's APIError as v1's answered awserr.Error, and reads as v1's
// did.
type awsCompatError struct {
	code, message string
	orig          error
}

func (e *awsCompatError) Error() string {
	msg := fmt.Sprintf("%%s: %%s", e.code, e.message)
	if e.orig != nil {
		msg = fmt.Sprintf("%%s\ncaused by: %%s", msg, e.orig.Error())
	}
	return msg
}
func (e *awsCompatError) ErrorCode() string             { return e.code }
func (e *awsCompatError) ErrorMessage() string          { return e.message }
func (e *awsCompatError) ErrorFault() smithy.ErrorFault { return smithy.FaultUnknown }

func awsCompatNewError(code, message string, orig error) smithy.APIError {
	return &awsCompatError{code: code, message: message, orig: orig}
}

// awsCompatRequestFailureError stands in for aws-sdk-go v1's RequestFailure,
// adding the response's status and request ID to the error it wraps.
type awsCompatRequestFailureError struct {
	*awsCompatError
	status    int
	requestID string
}

func (e *awsCompatRequestFailureError) Error() string {
	msg := fmt.Sprintf("%%s: %%s\n\tstatus code: %%d, request id: %%s", e.code, e.message, e.status, e.requestID)
	if e.orig != nil {
		msg = fmt.Sprintf("%%s\ncaused by: %%s", msg, e.orig.Error())
	}
	return msg
}
func (e *awsCompatRequestFailureError) HTTPStatusCode() int      { return e.status }
func (e *awsCompatRequestFailureError) ServiceRequestID() string { return e.requestID }

func awsCompatRequestFailure(err smithy.APIError, status int, requestID string) smithy.APIError {
	inner, ok := err.(*awsCompatError)
	if !ok {
		inner = &awsCompatError{code: err.ErrorCode(), message: err.ErrorMessage()}
	}
	return &awsCompatRequestFailureError{awsCompatError: inner, status: status, requestID: requestID}
}
`

// helperCall builds a call of a generated helper.
func helperCall(prefix java.Space, helper string, args ...java.Expression) java.Expression {
	elements := make([]java.RightPadded[java.Expression], len(args))
	for i, arg := range args {
		p := java.SingleSpace
		if i == 0 {
			p = java.EmptySpace
		}
		elements[i] = java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(arg, p)}
	}
	return &java.MethodInvocation{
		ID:         uuid.New(),
		Prefix:     prefix,
		Name:       &java.Identifier{ID: uuid.New(), Name: helper},
		Arguments:  java.Container[java.Expression]{Elements: elements},
		MethodType: lstutil.FuncType("", helper, nil),
	}
}

// missingEndpointError builds the stand-in for aws.ErrMissingEndpoint.
func missingEndpointError(prefix java.Space) java.Expression {
	text := func(s string) java.Expression {
		return &java.Literal{ID: uuid.New(), Value: s, Source: strconv.Quote(s), Type: lstutil.NamedType("string")}
	}
	return helperCall(prefix, errorHelper, text("MissingEndpoint"), text("'Endpoint' configuration is required for this service"), &java.Identifier{Name: "nil", Type: lstutil.NamedType("untyped nil")})
}

// awserrRefScan records the awserr.Error references the rewrite replaces: an
// assertion in an if's init or standing as a statement, and a declaration.
// Anywhere else — a type switch case, a parameter — the reference would
// outlive the import.
type awserrRefScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *awserrRefScan) VisitStatementWithInit(swi *golang.StatementWithInit, p any) java.J {
	if _, isIf := swi.Statement.(*java.If); isIf {
		if assignment, isMulti := swi.Init.Element.(*golang.MultiAssignment); isMulti {
			if _, _, _, ref, ok := v.scan.awserrAssertionOf(assignment); ok {
				v.scan.consumedAwserrRefs[ref] = true
			}
		}
	}
	return v.GoVisitor.VisitStatementWithInit(swi, p)
}

func (v *awserrRefScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	if _, inBlock := parentTree(v.Cursor()).(*java.Block); inBlock && java.HasMarker[golang.ShortVarDecl](ma.Markers) {
		if _, _, _, ref, ok := v.scan.awserrAssertionOf(ma); ok {
			v.scan.consumedAwserrRefs[ref] = true
		}
	}
	return v.GoVisitor.VisitMultiAssignment(ma, p)
}

func (v *awserrRefScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	if ref, ok := v.scan.awserrDeclared(vd); ok {
		v.scan.consumedAwserrRefs[ref] = true
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

// expandAwserrStatement turns `bound, ok := err.(awserr.Error)` standing as a
// statement into smithy's errors.As form, which leaves bound nil and ok false
// on a mismatch as the assertion did.
func expandAwserrStatement(ma *golang.MultiAssignment, bound, okName *java.Identifier, errExpr java.Expression, ref *java.FieldAccess, errorsLocal string) []java.Statement {
	declaration := awserrDeclaration(bound, ref, ma.Prefix)
	matched := lstutil.SetExprPrefix(errorsAsCall(errExpr, bound, errorsLocal), java.SingleSpace)
	var markers []java.Marker
	// A blank cannot be declared, only assigned to.
	if okName.Name != "_" {
		markers = []java.Marker{golang.ShortVarDecl{Ident: uuid.New()}}
	}
	check := &java.Assignment{
		ID:       uuid.New(),
		Prefix:   java.MakeSpace(nil, "\n"+lstutil.BaseIndent(ma.Prefix)),
		Markers:  java.MakeMarkers(uuid.New(), markers),
		Variable: &java.Identifier{ID: uuid.New(), Name: okName.Name, Type: lstutil.NamedType("bool")},
		Value:    java.LeftPadded[java.Expression]{Before: java.SingleSpace, Element: matched},
	}
	return []java.Statement{declaration, check}
}

// awserrDeclaration declares bound as the smithy interface errors.As fills.
func awserrDeclaration(bound *java.Identifier, ref *java.FieldAccess, prefix java.Space) *java.VariableDeclarations {
	declaration := &java.VariableDeclarations{
		ID:       uuid.New(),
		Prefix:   prefix,
		TypeExpr: awserrTargetType(ref),
		Variables: []java.RightPadded[*java.VariableDeclarator]{
			{Element: &java.VariableDeclarator{
				ID:   uuid.New(),
				Name: &java.Identifier{Prefix: java.SingleSpace, Name: bound.Name, Type: lstutil.NamedType(smithyGo + "." + smithyAPIErr)},
			}},
		},
	}
	declaration.Markers = java.MakeMarkers(uuid.New(), []java.Marker{golang.VarKeyword{Ident: uuid.New()}})
	return declaration
}

// expandAwserrAssertion turns the v1 assertion into smithy's errors.As form:
// the interface is declared first, then matched, since errors.As takes a
// pointer to it.
func expandAwserrAssertion(cursor *visitor.Cursor, swi *golang.StatementWithInit, bound, okName *java.Identifier, errExpr java.Expression, ref *java.FieldAccess, errorsLocal string) ([]java.Statement, bool) {
	iff, isIf := swi.Statement.(*java.If)
	if !isIf || iff.Condition == nil {
		return nil, false
	}

	declaration := awserrDeclaration(bound, ref, swi.Prefix)

	// The `ok` the assertion produced becomes the errors.As call itself.
	matched := errorsAsCall(errExpr, bound, errorsLocal)
	condition := replaceIdentifier(iff.Condition.Tree.Element, okName.Name, matched)

	rewrittenIf := *iff
	// The declaration carries the comments that led the statement.
	rewrittenIf.Prefix = java.MakeSpace(nil, "\n"+lstutil.BaseIndent(swi.Prefix))
	control := *iff.Condition
	control.Tree = java.RightPadded[java.Expression]{Element: condition, After: iff.Condition.Tree.After}
	rewrittenIf.Condition = &control

	return []java.Statement{declaration, &rewrittenIf}, true
}

// smithyAPIErrorType names the interface v2 reports an API failure through.
func smithyAPIErrorType() java.Expression {
	return &java.FieldAccess{
		ID:     uuid.New(),
		Prefix: java.SingleSpace,
		Target: &java.Identifier{ID: uuid.New(), Name: "smithy", Type: lstutil.NamedType(smithyGo)},
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{ID: uuid.New(), Name: smithyAPIErr}},
		Type:   lstutil.NamedType(smithyGo + "." + smithyAPIErr),
	}
}

// errorsAsCall builds `errors.As(err, &bound)`, or the generated helper that
// also raises NoCredentialProviders where local is empty.
func errorsAsCall(errExpr java.Expression, bound *java.Identifier, local string) java.Expression {
	address := &golang.Unary{
		ID:         uuid.New(),
		Operator:   java.LeftPadded[golang.UnaryOperator]{Element: golang.AddressOf},
		Expression: &java.Identifier{ID: uuid.New(), Name: bound.Name},
	}
	if local == "" {
		return &java.MethodInvocation{
			ID:   uuid.New(),
			Name: &java.Identifier{ID: uuid.New(), Name: asAPIErrorHelper},
			Arguments: java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{
				{Element: lstutil.SetExprPrefix(errExpr, java.EmptySpace)},
				{Element: lstutil.SetExprPrefix(address, java.SingleSpace)},
			}},
			MethodType: lstutil.FuncType("", asAPIErrorHelper, nil),
		}
	}
	return &java.MethodInvocation{
		ID:     uuid.New(),
		Select: &java.RightPadded[java.Expression]{Element: &java.Identifier{ID: uuid.New(), Name: local, Type: lstutil.NamedType(errorsPkg)}},
		Name:   &java.Identifier{ID: uuid.New(), Name: "As"},
		Arguments: java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{
			{Element: lstutil.SetExprPrefix(errExpr, java.EmptySpace)},
			{Element: lstutil.SetExprPrefix(address, java.SingleSpace)},
		}},
		MethodType: lstutil.FuncType(errorsPkg, "As", nil),
	}
}

// replaceIdentifier swaps every reference to name in expr for replacement,
// which is how the assertion's `ok` becomes the errors.As call that now decides
// the branch.
func replaceIdentifier(expr java.Expression, name string, replacement java.Expression) java.Expression {
	v := visitor.Init(&identifierSwap{name: name, replacement: replacement})
	if out, ok := v.Visit(expr, nil).(java.Expression); ok {
		return out
	}
	return expr
}

type identifierSwap struct {
	visitor.GoVisitor
	name        string
	replacement java.Expression
}

func (v *identifierSwap) VisitIdentifier(id *java.Identifier, p any) java.J {
	if id.Name != v.name {
		return v.GoVisitor.VisitIdentifier(id, p)
	}
	return lstutil.SetExprPrefix(v.replacement, id.Prefix)
}

// errCodeLiteral rewrites a v1 ErrCode constant to the wire string it held,
// which is what a v2 caller compares smithy's ErrorCode against.
func errCodeLiteral(service, name string, prefix java.Space) (java.Expression, bool) {
	value, ok := awsmanifest.ErrCodeValue(service, name)
	if !ok {
		return nil, false
	}
	return &java.Literal{
		ID:      uuid.New(),
		Prefix:  prefix,
		Value:   value,
		Source:  `"` + value + `"`,
		Type:    &java.JavaTypePrimitive{Keyword: "String"},
		Markers: java.MakeMarkers(uuid.New(), nil),
	}, true
}

// awserrBoundNames collects the names an awserr.Error assertion binds anywhere
// in the file. The accessors on them are renamed while visiting an if body the
// expansion only reaches on the way out, so the names are gathered up front.
func awserrBoundNames(cu *golang.CompilationUnit, scan *fileScan) map[string]bool {
	found := map[string]bool{}
	if scan.awserrPkg == "" {
		return found
	}
	collect := visitor.Init(&awserrBoundScan{scan: scan, found: found})
	collect.Visit(cu, nil)
	return found
}

type awserrBoundScan struct {
	visitor.GoVisitor
	scan  *fileScan
	found map[string]bool
}

func (v *awserrBoundScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	if bound, _, _, ref, ok := v.scan.awserrAssertionOf(ma); ok {
		v.found[bound.Name] = true
		if isRequestFailure(ref) {
			v.scan.requestFailureNames[bound.Name] = true
		}
	}
	return v.GoVisitor.VisitMultiAssignment(ma, p)
}

func (v *awserrBoundScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	if ref, ok := v.scan.awserrDeclared(vd); ok {
		for _, rp := range vd.Variables {
			if d := rp.Element; d != nil && d.Name != nil {
				v.found[d.Name.Name] = true
				if isRequestFailure(ref) {
					v.scan.requestFailureNames[d.Name.Name] = true
				}
			}
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

func (v *awserrBoundScan) VisitStatementWithInit(swi *golang.StatementWithInit, p any) java.J {
	if bound, _, _, ref, ok := v.scan.awserrAssertion(swi); ok {
		v.found[bound.Name] = true
		if isRequestFailure(ref) {
			v.scan.requestFailureNames[bound.Name] = true
		}
	}
	return v.GoVisitor.VisitStatementWithInit(swi, p)
}

// v1 raised these codes itself, before or instead of a response: the
// credential chain, the request plumbing, the parameter validation. v2 reports
// those failures as ordinary errors, not as an APIError carrying a code, so a
// comparison against one would quietly never match.
var clientSideCodes = map[string]bool{
	"NoCredentialProviders": true, "EnvAccessKeyNotFound": true, "EnvSecretNotFound": true,
	"SharedCredsLoad": true, "SharedCredsAccessKey": true, "SharedCredsSecret": true,
	"EmptyStaticCreds": true, "ProviderNotExpirer": true, "AssumeRoleTokenNotAvailable": true,
	"CredentialsEndpointError": true, "UserHomeNotFound": true, "SharedConfigErr": true,
	"WebIdentityErr": true, "SSOProviderInvalidToken": true,
	"ProcessProviderExecutionError": true, "ProcessProviderParseError": true,
	"ProcessProviderRequiredError": true, "ProcessProviderVersionError": true,
	"RequestCanceled": true, "RequestError": true, "ReadError": true, "ResponseTimeout": true,
	"SerializationError": true, "InvalidPresignExpireError": true, "UnknownError": true,
	"MissingRegion": true, "MissingEndpoint": true, "InvalidEndpointURL": true, "UnknownEndpointError": true,
	"InvalidParameter": true, "ParamRequiredError": true, "ParamMinLenError": true,
	"ParamMaxLenError": true, "ParamMinValueError": true, "ParamFormatInvalidError": true,
	"LoadCustomCABundleError": true, "LoadClientTLSCertError": true, "MinimumHTTPProtocolError": true,
	"ResourceNotReady": true, "EventStreamInputWriterCloseError": true,
}

// awserrCode reports whether expr reads the code of an awserr value the file
// bound, as `aerr.Code()`.
func (s *fileScan) awserrCode(expr java.Expression) bool {
	mi, isCall := expr.(*java.MethodInvocation)
	if !isCall || mi.Name == nil || mi.Name.Name != "Code" || len(realArgs(mi)) != 0 {
		return false
	}
	recv, isIdent := selectIdentifier(mi)
	return isIdent && s.awserrNames[recv.Name]
}

// clientSideCode returns the code a string literal names, when v1 raised it
// itself.
func clientSideCode(expr java.Expression) (string, bool) {
	lit, isLiteral := expr.(*java.Literal)
	if !isLiteral {
		return "", false
	}
	code, err := strconv.Unquote(lit.Source)
	return code, err == nil && clientSideCodes[code]
}

// requestFailureReason names an accessor called on a bound RequestFailure that
// v2's ResponseError does not answer.
func requestFailureReason(cu *golang.CompilationUnit, scan *fileScan) string {
	if len(scan.requestFailureNames) == 0 {
		return ""
	}
	check := visitor.Init(&requestFailureCallScan{scan: scan})
	check.Visit(cu, nil)
	return check.reason
}

type requestFailureCallScan struct {
	visitor.GoVisitor
	scan   *fileScan
	reason string
}

func (v *requestFailureCallScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if recv, isIdent := selectIdentifier(mi); isIdent && v.scan.requestFailureNames[recv.Name] && mi.Name != nil && v.reason == "" {
		if _, renamed := requestFailureRenames[mi.Name.Name]; !renamed && mi.Name.Name != "Error" {
			v.reason = "awserr.RequestFailure." + mi.Name.Name + ", which v2's ResponseError does not answer"
		}
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}
