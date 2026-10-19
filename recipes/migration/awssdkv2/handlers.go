/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	gofmt "go/format"
	"sort"
	"strconv"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/format"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v1 let a session carry request handlers, run at fixed points of every request
// its clients made. v2 has smithy middleware instead, added to the config's
// APIOptions. v1's Send and UnmarshalError handlers ran once per attempt around
// the HTTP round trip, before the response was unmarshalled; v2 retries in its
// Finalize step, so the innermost Deserialize middleware sees the same attempts
// and the same span. A generated adapter keeps both lists there, in v1's order
// and with v1's Push and Set semantics, and hands each handler the few request
// fields v2 can fill faithfully, leaving its body as written.
const (
	v1Request     = v1Aws + "/request"
	handlersField = "Handlers"
	requestType   = "Request"
	namedHandler  = "NamedHandler"
	compatRequest = "awsCompatRequest"
	compatList    = "awsCompatList"
	apiOptions    = "APIOptions"
	handlerFile   = "zz_awssdkv2_handlers.go"

	requestReason = "the v1 request plumbing has no v2 counterpart beyond a session's Send and UnmarshalError handlers"
)

// handlerLists maps a v1 handler list to the generated value standing in for it.
var handlerLists = map[string]string{
	"Send":           "awsCompatSend",
	"UnmarshalError": "awsCompatUnmarshalError",
}

// handlerMethods are the HandlerList methods a handler is added through, which
// the generated lists share, each recording whether it takes a NamedHandler.
var handlerMethods = map[string]bool{
	"PushBack":       false,
	"PushFront":      false,
	"PushBackNamed":  true,
	"PushFrontNamed": true,
	"SetBackNamed":   true,
	"SetFrontNamed":  true,
}

// handlerUse is a handler added to a session's list, as the rewrite needs it.
type handlerUse struct {
	session *java.Identifier
	list    string
	method  string
	// name is the NamedHandler's, or "" for an unnamed one.
	name string
	// fn is the function literal, still wrapped as the argument it was.
	fn     java.Expression
	lambda *java.MethodDeclaration
	// handlersRef is `sess.Handlers`, and requestRefs the request package
	// references the rewrite consumes.
	handlersRef *java.FieldAccess
	requestRefs []*java.FieldAccess
	// userAgent is the request call adding to the User-Agent, for a Build
	// handler that only does that.
	userAgent *java.MethodInvocation
}

// v1's request package built Build handlers adding a product token to the
// User-Agent; v2's middleware package adds the same token from an API option.
var userAgentHandlers = map[string]struct {
	option string
	args   int
}{
	"MakeAddToUserAgentHandler":         {"AddUserAgentKeyValue", 2},
	"MakeAddToUserAgentFreeFormHandler": {"AddUserAgentKey", 1},
}

// userAgentHandler reports whether expr builds one of those handlers, with no
// extra comments, which v2 has no slot for.
func (s *fileScan) userAgentHandler(expr java.Expression) (*java.MethodInvocation, string, bool) {
	mi, isCall := expr.(*java.MethodInvocation)
	if !isCall {
		return nil, "", false
	}
	call, isRequest := qualifiedCall(mi, s.requestPkg)
	handler, known := userAgentHandlers[call]
	if !isRequest || !known || len(realArgs(mi)) != handler.args {
		return nil, "", false
	}
	return mi, handler.option, true
}

// handlerCall reads `sess.Handlers.<List>.<Method>(…)`, or reports why such a
// call has no v2 form. ok is false for any other call.
func (s *fileScan) handlerCall(mi *java.MethodInvocation) (use handlerUse, reason string, ok bool) {
	if mi.Select == nil || mi.Name == nil {
		return handlerUse{}, "", false
	}
	listRef, isField := mi.Select.Element.(*java.FieldAccess)
	if !isField || listRef.Name.Element == nil {
		return handlerUse{}, "", false
	}
	handlers, isField := listRef.Target.(*java.FieldAccess)
	if !isField || handlers.Name.Element == nil || handlers.Name.Element.Name != handlersField {
		return handlerUse{}, "", false
	}
	session, isIdent := handlers.Target.(*java.Identifier)
	if !isIdent || !s.sessionVars[session.Name] {
		return handlerUse{}, "", false
	}
	use = handlerUse{session: session, list: listRef.Name.Element.Name, handlersRef: handlers}
	if use.list == "Build" && (mi.Name.Name == "PushBack" || mi.Name.Name == "PushFront") {
		if args := realArgs(mi); len(args) == 1 {
			if call, _, isUserAgent := s.userAgentHandler(args[0]); isUserAgent {
				use.method = mi.Name.Name
				use.userAgent = call
				return use, "", true
			}
		}
	}
	if _, placed := handlerLists[use.list]; !placed {
		return use, "the v1 " + use.list + " handler list, which runs at a point v2's middleware has no counterpart for", true
	}
	named, known := handlerMethods[mi.Name.Name]
	if !known {
		return use, "HandlerList." + mi.Name.Name + ", which has no v2 counterpart", true
	}
	use.method = mi.Name.Name
	args := realArgs(mi)
	if len(args) != 1 {
		return use, "HandlerList." + mi.Name.Name + " without a single handler", true
	}
	fnArg := args[0]
	if named {
		comp, isComposite := fnArg.(*golang.Composite)
		if !isComposite {
			return use, "a NamedHandler built elsewhere", true
		}
		fa, isField := comp.TypeExpr.(*java.FieldAccess)
		if name, isRequest := "", false; isField {
			name, isRequest = qualifiedRef(fa, s.requestPkg)
			isField = isRequest && name == namedHandler
		}
		if !isField {
			return use, "a named handler that is not a request.NamedHandler literal", true
		}
		use.requestRefs = append(use.requestRefs, fa)
		fnArg = nil
		for _, rp := range comp.Elements.Elements {
			kv, isKeyValue := rp.Element.(*golang.KeyValue)
			if !isKeyValue {
				return use, "a NamedHandler literal without field names", true
			}
			key, _ := kv.Key.(*java.Identifier)
			switch {
			case key != nil && key.Name == "Name":
				lit, isLiteral := kv.Value.Element.(*java.Literal)
				if !isLiteral {
					return use, "a handler named by something other than a string literal", true
				}
				name, err := strconv.Unquote(lit.Source)
				if err != nil {
					return use, "a handler named by something other than a string literal", true
				}
				// v1's own handlers have no v2 counterpart to replace or sit beside.
				if strings.HasPrefix(name, "core.") || strings.HasPrefix(name, "awssdk.") {
					return use, "a handler replacing v1's own " + name, true
				}
				use.name = name
			case key != nil && key.Name == "Fn":
				fnArg = kv.Value.Element
			default:
				return use, "a NamedHandler field the rewrite does not know", true
			}
		}
		if use.name == "" || fnArg == nil {
			return use, "a NamedHandler without a name and a function", true
		}
	}
	use.fn = fnArg
	wrapped, isWrapped := fnArg.(*golang.StatementExpression)
	if !isWrapped {
		return use, "a handler function declared elsewhere, whose body the rewrite cannot see", true
	}
	lambda, isLambda := wrapped.Statement.(*java.MethodDeclaration)
	if !isLambda || lambda.Body == nil {
		return use, "a handler function declared elsewhere, whose body the rewrite cannot see", true
	}
	use.lambda = lambda
	param, ref, isParam := s.requestParam(lambda)
	if !isParam {
		return use, "a handler not taking a single *request.Request", true
	}
	use.requestRefs = append(use.requestRefs, ref)
	if reason := requestReadsReason(lambda.Body, param); reason != "" {
		return use, reason, true
	}
	return use, "", true
}

// requestParam returns the name a handler binds its request to, and the
// `request.Request` naming its type.
func (s *fileScan) requestParam(lambda *java.MethodDeclaration) (string, *java.FieldAccess, bool) {
	params := lambda.Parameters.Elements
	if len(params) != 1 {
		return "", nil, false
	}
	vd, isDecl := params[0].Element.(*java.VariableDeclarations)
	if !isDecl || len(vd.Variables) != 1 || vd.Variables[0].Element == nil || vd.Variables[0].Element.Name == nil {
		return "", nil, false
	}
	ptr, isPointer := vd.TypeExpr.(*golang.PointerType)
	if !isPointer {
		return "", nil, false
	}
	fa, isField := ptr.Elem.(*java.FieldAccess)
	if !isField {
		return "", nil, false
	}
	if name, isRequest := qualifiedRef(fa, s.requestPkg); !isRequest || name != requestType {
		return "", nil, false
	}
	return vd.Variables[0].Element.Name.Name, fa, true
}

// requestReadsReason names a use of the request a handler makes that the
// adapter's request cannot answer the way v1's did, or "".
func requestReadsReason(body *java.Block, param string) string {
	scan := visitor.Init(&requestReadScan{param: param})
	scan.Visit(body, nil)
	if scan.reason != "" {
		return scan.reason
	}
	if scan.uses != scan.reads {
		return "a handler passing its request on, which has no v2 counterpart"
	}
	return ""
}

// requestReadScan checks each read of a handler's request against what the
// adapter fills: the service and operation names, the attempt's start, and —
// where they are only handed to a formatter — the operation and its input.
type requestReadScan struct {
	visitor.GoVisitor
	param  string
	uses   int
	reads  int
	reason string
}

func (v *requestReadScan) VisitIdentifier(id *java.Identifier, p any) java.J {
	if id.Name == v.param {
		if fa, isName := parentTree(v.Cursor()).(*java.FieldAccess); !isName || fa.Name.Element != id {
			v.uses++
		}
	}
	return id
}

func (v *requestReadScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	target, isIdent := fa.Target.(*java.Identifier)
	if !isIdent || target.Name != v.param || fa.Name.Element == nil {
		return fa
	}
	field := fa.Name.Element.Name
	parent := parentTree(v.Cursor())
	outer, _ := parent.(*java.FieldAccess)
	selected := func(name string) bool {
		return outer != nil && outer.Name.Element != nil && outer.Name.Element.Name == name
	}
	call, isCall := parent.(*java.MethodInvocation)
	argument := isCall && (call.Select == nil || call.Select.Element != java.Expression(fa))
	switch {
	case field == "AttemptTime",
		field == "ClientInfo" && selected("ServiceName"),
		field == "Operation" && (selected("Name") || argument),
		field == "Params" && argument:
		v.reads++
	case v.reason == "":
		v.reason = "a handler reading the request's " + field + ", which v2's middleware has no faithful counterpart for"
	}
	return fa
}

// parentTree returns the nearest tree above the cursor's own.
func parentTree(cursor *visitor.Cursor) java.Tree {
	for c := cursor.Parent(); c != nil; c = c.Parent() {
		if c.Value() != nil {
			return c.Value()
		}
	}
	return nil
}

// handlerScan checks every handler a file adds: one on a list v2 has a
// placement for, added to a session the function built. A session handed in
// becomes a config value, which the caller's copy would not see added to.
type handlerScan struct {
	visitor.GoVisitor
	scan *fileScan
	// built are the sessions the enclosing function constructs.
	built map[string]bool
}

func (v *handlerScan) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	// A function literal shares the sessions of the function around it.
	if md.Name == nil || md.Name.Name == "" {
		return v.GoVisitor.VisitMethodDeclaration(md, p)
	}
	outer := v.built
	v.built = map[string]bool{}
	if md.Body != nil {
		visitor.Init(&builtSessionScan{scan: v.scan, built: v.built}).Visit(md.Body, nil)
	}
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)
	v.built = outer
	return md
}

// builtSessionScan collects the names a function assigns a new session to.
type builtSessionScan struct {
	visitor.GoVisitor
	scan  *fileScan
	built map[string]bool
}

func (v *builtSessionScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if name, isIdent := a.Variable.(*java.Identifier); isIdent && v.scan.constructsSession(a.Value.Element) {
		v.built[name.Name] = true
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

func (v *builtSessionScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	if len(ma.Variables) == 2 && len(ma.Values) == 1 && v.scan.constructsSession(ma.Values[0].Element) {
		if name, isIdent := ma.Variables[0].Element.(*java.Identifier); isIdent {
			v.built[name.Name] = true
		}
	}
	return v.GoVisitor.VisitMultiAssignment(ma, p)
}

func (v *builtSessionScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	for _, rp := range vd.Variables {
		if d := rp.Element; d != nil && d.Name != nil && d.Initializer != nil && v.scan.constructsSession(d.Initializer.Element) {
			v.built[d.Name.Name] = true
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

func (v *handlerScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	use, reason, isHandler := v.scan.handlerCall(mi)
	if !isHandler {
		return v.GoVisitor.VisitMethodInvocation(mi, p)
	}
	_, isStatement := parentTree(v.Cursor()).(*java.Block)
	switch {
	case reason != "":
	case !isStatement:
		reason = "a handler list call used as a value"
	case !v.built[use.session.Name]:
		reason = "a handler added to a session the function did not build, which becomes a config value the caller's copy would not see added to"
	}
	if reason != "" {
		if v.scan.handlerReason == "" {
			v.scan.handlerReason = reason
		}
		return mi
	}
	v.scan.consumedHandlerRefs[use.handlersRef] = true
	for _, ref := range use.requestRefs {
		v.scan.consumedRequestRefs[ref] = true
	}
	if use.userAgent != nil {
		v.scan.consumedRequestCalls[use.userAgent] = true
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

// handlerStatement rewrites a handler added to a session as the adapter
// appended to the config's APIOptions.
func (v *migrateVisitor) handlerStatement(mi *java.MethodInvocation, p any) (java.Statement, bool) {
	use, reason, isHandler := v.scan.handlerCall(mi)
	if !isHandler || reason != "" {
		return nil, false
	}
	if use.userAgent != nil {
		return v.userAgentStatement(mi, use), true
	}
	list := handlerLists[use.list]

	// The handler keeps its body; only the request it is handed changes type.
	lambda := *use.lambda
	params := lambda.Parameters
	params.Elements = append([]java.RightPadded[java.Statement](nil), params.Elements...)
	vd := *params.Elements[0].Element.(*java.VariableDeclarations)
	ptr := *vd.TypeExpr.(*golang.PointerType)
	ptr.Elem = &java.Identifier{Prefix: ptr.Elem.GetPrefix(), Name: compatRequest, Type: lstutil.NamedType(compatRequest)}
	vd.TypeExpr = &ptr
	params.Elements[0].Element = &vd
	lambda.Parameters = params
	wrapped := *use.fn.(*golang.StatementExpression)
	wrapped.Statement = &lambda

	optionsRef := func() java.Expression {
		return &java.FieldAccess{
			Target: &java.Identifier{Name: use.session.Name, Type: lstutil.NamedType(v2Aws + ".Config")},
			Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: apiOptions}},
		}
	}
	args := []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(&wrapped, java.EmptySpace)}}
	if use.name != "" {
		args = []java.RightPadded[java.Expression]{
			{Element: &java.Literal{Value: use.name, Source: strconv.Quote(use.name), Type: lstutil.NamedType("string")}},
			{Element: lstutil.SetExprPrefix(&wrapped, java.SingleSpace)},
		}
	}
	listType := lstutil.NamedType(compatList)
	adapterCall := &java.MethodInvocation{
		Prefix:     java.SingleSpace,
		Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: list, Type: listType}},
		Name:       &java.Identifier{Name: use.method},
		Arguments:  java.Container[java.Expression]{Elements: args},
		MethodType: lstutil.FuncType(compatList, use.method, nil),
	}
	appended := &java.MethodInvocation{
		Prefix: java.SingleSpace,
		Name:   &java.Identifier{Name: "append"},
		Arguments: java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{
			{Element: optionsRef()},
			{Element: adapterCall},
		}},
	}
	stmt := &java.Assignment{
		Prefix:   mi.Prefix,
		Variable: optionsRef(),
		Value:    java.LeftPadded[java.Expression]{Before: java.SingleSpace, Element: appended},
	}
	v.needsHandlers = true
	// The literal moves out of the NamedHandler it sat in, a level shallower.
	formatted, _ := format.AutoFormat(stmt, p, nil, v.Cursor()).(java.Statement)
	if formatted == nil {
		return stmt, true
	}
	return formatted, true
}

// userAgentStatement appends v2's user-agent option to the config's APIOptions.
func (v *migrateVisitor) userAgentStatement(mi *java.MethodInvocation, use handlerUse) java.Statement {
	_, option, _ := v.scan.userAgentHandler(use.userAgent)
	v.needsUserAgent = true
	args := make([]java.RightPadded[java.Expression], 0, 2)
	for i, arg := range realArgs(use.userAgent) {
		prefix := java.EmptySpace
		if i > 0 {
			prefix = java.SingleSpace
		}
		args = append(args, java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(arg, prefix)})
	}
	optionsRef := func() java.Expression {
		return &java.FieldAccess{
			Target: &java.Identifier{Name: use.session.Name, Type: lstutil.NamedType(v2Aws + ".Config")},
			Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: apiOptions}},
		}
	}
	return &java.Assignment{
		Prefix:   mi.Prefix,
		Variable: optionsRef(),
		Value: java.LeftPadded[java.Expression]{Before: java.SingleSpace, Element: &java.MethodInvocation{
			Prefix: java.SingleSpace,
			Name:   &java.Identifier{Name: "append"},
			Arguments: java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{
				{Element: optionsRef()},
				{Element: &java.MethodInvocation{
					Prefix:     java.SingleSpace,
					Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: awsMiddlewareAlias, Type: lstutil.NamedType(v2Aws + "/middleware")}},
					Name:       &java.Identifier{Name: option},
					Arguments:  java.Container[java.Expression]{Elements: args},
					MethodType: lstutil.FuncType(v2Aws+"/middleware", option, nil),
				}},
			}},
		}},
	}
}

// awsMiddlewareAlias binds v2's aws middleware package apart from smithy's.
const awsMiddlewareAlias = "awsmiddleware"

const handlerSource = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import (
	"context"
	"time"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// awsCompatRequest is what a migrated aws-sdk-go v1 request handler reads off
// its request. Operation carries only the name and Params holds the v2 input,
// so a log line printing either reads differently from v1's.
type awsCompatRequest struct {
	ClientInfo  awsCompatClientInfo
	Operation   *awsCompatOperation
	Params      interface{}
	AttemptTime time.Time
}

type awsCompatClientInfo struct{ ServiceName string }

type awsCompatOperation struct{ Name string }

// awsCompatServiceNames maps v2's service ID to the name v1 reported, so a
// value labelled with it keeps its label.
var awsCompatServiceNames = map[string]string{
%s}

type awsCompatParamsKey struct{}

// awsCompatHandler is a handler on one of the lists, a nil fn standing for the
// round trip v1's own Send handler made.
type awsCompatHandler struct {
	name string
	fn   func(*awsCompatRequest)
}

// awsCompatList stands in for one of v1's handler lists. Each method returns
// the API option adding a handler to it, as the v1 method of the same name did.
type awsCompatList int

const (
	awsCompatSend awsCompatList = iota
	awsCompatUnmarshalError
)

func (l awsCompatList) PushBack(fn func(*awsCompatRequest)) func(*middleware.Stack) error {
	return l.add(awsCompatHandler{fn: fn}, false, false)
}

func (l awsCompatList) PushFront(fn func(*awsCompatRequest)) func(*middleware.Stack) error {
	return l.add(awsCompatHandler{fn: fn}, true, false)
}

func (l awsCompatList) PushBackNamed(name string, fn func(*awsCompatRequest)) func(*middleware.Stack) error {
	return l.add(awsCompatHandler{name: name, fn: fn}, false, false)
}

func (l awsCompatList) PushFrontNamed(name string, fn func(*awsCompatRequest)) func(*middleware.Stack) error {
	return l.add(awsCompatHandler{name: name, fn: fn}, true, false)
}

// SetBackNamed replaces every handler of the same name, or adds it at the back.
func (l awsCompatList) SetBackNamed(name string, fn func(*awsCompatRequest)) func(*middleware.Stack) error {
	return l.add(awsCompatHandler{name: name, fn: fn}, false, true)
}

// SetFrontNamed replaces every handler of the same name, or adds it at the front.
func (l awsCompatList) SetFrontNamed(name string, fn func(*awsCompatRequest)) func(*middleware.Stack) error {
	return l.add(awsCompatHandler{name: name, fn: fn}, true, true)
}

func (l awsCompatList) add(h awsCompatHandler, front, replace bool) func(*middleware.Stack) error {
	return func(s *middleware.Stack) error {
		m, err := awsCompatHandlersOn(s)
		if err != nil {
			return err
		}
		list := &m.lists[l]
		if replace {
			swapped := false
			for i := range *list {
				if (*list)[i].name == h.name {
					(*list)[i], swapped = h, true
				}
			}
			if swapped {
				return nil
			}
		}
		if front {
			*list = append([]awsCompatHandler{h}, *list...)
		} else {
			*list = append(*list, h)
		}
		return nil
	}
}

// awsCompatHandlers runs both lists for one operation. v1 ran them once per
// attempt around the HTTP round trip, before the response was unmarshalled; v2
// retries in its Finalize step, so the innermost Deserialize middleware sees the
// same attempts and the same span.
type awsCompatHandlers struct {
	lists [2][]awsCompatHandler
}

func (*awsCompatHandlers) ID() string { return "awsCompatHandlers" }

func (m *awsCompatHandlers) HandleDeserialize(ctx context.Context, in middleware.DeserializeInput, next middleware.DeserializeHandler) (middleware.DeserializeOutput, middleware.Metadata, error) {
	service := awsmiddleware.GetServiceID(ctx)
	if name, ok := awsCompatServiceNames[service]; ok {
		service = name
	}
	r := &awsCompatRequest{
		ClientInfo:  awsCompatClientInfo{ServiceName: service},
		Operation:   &awsCompatOperation{Name: awsmiddleware.GetOperationName(ctx)},
		Params:      middleware.GetStackValue(ctx, awsCompatParamsKey{}),
		AttemptTime: time.Now(),
	}
	var (
		out      middleware.DeserializeOutput
		metadata middleware.Metadata
		err      error
	)
	for _, h := range m.lists[awsCompatSend] {
		if h.fn == nil {
			out, metadata, err = next.HandleDeserialize(ctx, in)
			continue
		}
		h.fn(r)
	}
	// v1 ran these for a response with an error status, not a failed round trip.
	if resp, _ := out.RawResponse.(*smithyhttp.Response); resp != nil && resp.StatusCode >= 300 {
		for _, h := range m.lists[awsCompatUnmarshalError] {
			h.fn(r)
		}
	}
	return out, metadata, err
}

// awsCompatHandlersOn returns the stack's handler lists, adding them on first
// use along with the middleware that keeps the operation's input for them.
func awsCompatHandlersOn(s *middleware.Stack) (*awsCompatHandlers, error) {
	if existing, ok := s.Deserialize.Get("awsCompatHandlers"); ok {
		if m, ok := existing.(*awsCompatHandlers); ok {
			return m, nil
		}
	}
	err := s.Initialize.Add(middleware.InitializeMiddlewareFunc("awsCompatParams", func(ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
		return next.HandleInitialize(middleware.WithStackValue(ctx, awsCompatParamsKey{}, in.Parameters), in)
	}), middleware.Before)
	if err != nil {
		return nil, err
	}
	m := &awsCompatHandlers{}
	m.lists[awsCompatSend] = []awsCompatHandler{{name: "core.SendHandler"}}
	return m, s.Deserialize.Add(m, middleware.After)
}
`

// handlerFileSource renders the adapter for a package.
func handlerFileSource(pkg string) string {
	ids := make([]string, 0, len(v1ServiceNames))
	for id := range v1ServiceNames {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var names strings.Builder
	for _, id := range ids {
		fmt.Fprintf(&names, "\t%q: %q,\n", id, v1ServiceNames[id])
	}
	src := fmt.Sprintf(handlerSource, pkg, names.String())
	if formatted, err := gofmt.Source([]byte(src)); err == nil {
		return string(formatted)
	}
	return src
}
