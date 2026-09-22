/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"strings"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// fileScan is what the migration needs to know about a file before touching it:
// whether every v1 construct in it has a faithful v2 form, and the names it
// bound along the way.
type fileScan struct {
	migratable bool
	// reason names what held the file back, for the recipe's own reporting and
	// for steering which construct to cover next.
	reason string

	awsPkg         string
	sessionPkg     string
	credentialsPkg string
	awserrPkg      string
	// managerPkg is the local name of the v1 transfer-manager package.
	managerPkg string
	// imdsPkg is the local name of the v1 instance-metadata package, and
	// imdsVars the names holding one of its clients.
	imdsPkg  string
	imdsVars map[string]bool
	// stscredsPkg is the local name of the assume-role credentials package.
	stscredsPkg string
	// managerVars maps a name holding a transfer manager to which one it holds.
	managerVars map[string]string
	services    map[string]string
	// adjunct marks a file that names no SDK package but reads values of its
	// shapes, where only what attribution says about a value can guide the
	// rewrite.
	adjunct bool
	// configVars maps a local holding an aws.Config literal to the region it
	// sets, with configMutated recording the ones something else wrote to and
	// configUses how many times each name appears.
	configVars map[string][]configOption
	// configLocals is what the scan learned about each, and configRestructured
	// and configPathStyle what it decided: the declaration takes over the load,
	// and the addressing style is hoisted into a local of its own.
	configLocals       map[string]*configLocal
	configRestructured map[string]bool
	configPathStyle    map[string]bool
	configMutated      map[string]bool
	configUses         map[string]int
	// missingRegionBlocked is set by a reference to v1's missing-region sentinel
	// outside a comparison, which has no v2 form.
	missingRegionBlocked bool
	// credentialsConsumed are the credential constructors an aws.Config literal
	// takes over, which the loader options replace.
	credentialsConsumed map[java.Expression]bool
	// configAccessBlocked names a reach through a v1 Config field that v2 leaves
	// nothing in place of.
	configAccessBlocked string
	// sessionVars are the names declared as a *session.Session, and
	// sessionNilCompared records that one was tested against nil — which the
	// config value replacing it cannot be.
	sessionVars        map[string]bool
	sessionNilCompared bool
	// sessionOptionsBlocked is set by a session.Options literal carrying
	// something a v2 config load has no option for.
	sessionOptionsBlocked bool
	// configLocal is the name the v2 config import binds, which is the package's
	// own unless the file already uses it for something else.
	configLocal string
	// errorsLocal is the name the standard library's errors package binds. A
	// file on github.com/pkg/errors already spells `errors`, so the one this
	// recipe adds is aliased alongside it.
	errorsLocal string
	// ifacePaths are the v1 iface packages the file imports, whose replacement
	// the migration generates.
	ifacePaths []string

	// shapes names the modelled shape each variable holds, inferred from the
	// operation calls and shape literals in the file. Both the guard and the
	// rewrite read it, so they agree on what a field access refers to.
	shapes map[string]shapeRef
	// sliceShapes names the shape a variable holds a slice of, so a range over
	// it binds its elements.
	sliceShapes map[string]shapeRef
	// clients maps a variable holding a service client to that client's service,
	// so its operation calls can gain a context and name the v2 type they land
	// on.
	clients map[string]string
}

// scanFile decides whether cu migrates as a whole. v2 changed enough that a
// half-migrated file does not compile, so anything unhandled blocks all of it.
func scanFile(cu *golang.CompilationUnit) *fileScan {
	s := &fileScan{
		awsPkg:              pathswap.Qualifier(cu, v1Aws),
		sessionPkg:          pathswap.Qualifier(cu, v1Session),
		credentialsPkg:      pathswap.Qualifier(cu, v1Credentials),
		awserrPkg:           pathswap.Qualifier(cu, v1Awserr),
		managerPkg:          pathswap.Qualifier(cu, v1S3Manager),
		imdsPkg:             pathswap.Qualifier(cu, v1Ec2Metadata),
		imdsVars:            map[string]bool{},
		managerVars:         map[string]string{},
		stscredsPkg:         pathswap.Qualifier(cu, v1Stscreds),
		services:            serviceImports(cu),
		clients:             map[string]string{},
		configVars:          map[string][]configOption{},
		configLocals:        map[string]*configLocal{},
		configRestructured:  map[string]bool{},
		configPathStyle:     map[string]bool{},
		configMutated:       map[string]bool{},
		configUses:          map[string]int{},
		configLocal:         configPkgName,
		errorsLocal:         errorsPkg,
		sessionVars:         map[string]bool{},
		credentialsConsumed: map[java.Expression]bool{},
	}
	if !importsV1(cu) {
		// A file can hold SDK values without naming the SDK — a formatter handed
		// a list of shapes by the file that fetched them. It has no imports to
		// swap and no construct that could hold it back, but the fields it reads
		// changed shape all the same, so the inference still runs over it.
		s.adjunct = true
		s.migratable = true
		s.shapes = map[string]shapeRef{}
		s.sliceShapes = map[string]shapeRef{}
		visitor.Init(&shapeScan{scan: s}).Visit(cu, nil)
		return s
	}
	// A v2 path already in the file would collide with the one the swap
	// introduces, and the config import replaces the session's own name.
	for _, name := range []string{v2Aws, v2Config, v2Credentials} {
		if pathswap.Imports(cu, name) {
			return s
		}
	}
	if s.sessionPkg != "" && bindsName(cu, configPkgName) {
		return s
	}
	// The standard library's errors package is what the smithy and missing-region
	// rewrites call into, and a file already on another errors package binds the
	// name to that one.
	for _, rp := range cu.Imports.Elements {
		if pathswap.LocalName(rp.Element) == errorsPkg && pathswap.Path(rp.Element) != errorsPkg {
			s.errorsLocal = "std" + errorsPkg
		}
	}
	if bindsLocalName(cu, errorsPkg) {
		s.errorsLocal = "std" + errorsPkg
	}

	// v2's loader lives in a package called config, a name Go code often gives
	// a local of its own. Where the file does, the import is aliased rather than
	// left to be shadowed.
	if s.sessionPkg != "" && bindsLocalName(cu, configPkgName) {
		s.configLocal = configPkgName + "aws"
	}
	for _, rp := range cu.Imports.Elements {
		path := pathswap.Path(rp.Element)
		if _, blocked := v1OnlyPackages[path]; blocked {
			s.reason = "v1-only package " + path
			return s
		}
		if _, _, isIface := ifacePackageName(path); isIface {
			continue
		}
		if underV1(path) && path != v1Awserr {
			// awserr has no v2 path at all; its uses become smithy and the
			// import is dropped rather than repointed.
			if _, ok := MapPath(path); !ok {
				s.reason = "no v2 path for " + path
				return s
			}
		}
	}

	for _, rp := range cu.Imports.Elements {
		path := pathswap.Path(rp.Element)
		service, _, isIface := ifacePackageName(path)
		if !isIface {
			continue
		}
		// The replacement package is rendered from the manifest, so a service
		// missing from it leaves the import pointing at nothing.
		if _, ok := ifaceSource(service); !ok {
			s.reason = "no manifest for iface package " + path
			return s
		}
		s.ifacePaths = append(s.ifacePaths, path)
	}

	// A service with no manifest cannot be classified at all.
	for _, service := range s.services {
		if !awsmanifest.Known(service) {
			s.reason = "no manifest for service " + service
			return s
		}
	}

	options := visitor.Init(&sessionOptionsScan{scan: s})
	options.Visit(cu, nil)

	sessions := visitor.Init(&sessionVarScan{scan: s})
	sessions.Visit(cu, nil)

	creds := visitor.Init(&credentialsScan{scan: s, consumed: s.credentialsConsumed})
	creds.Visit(cu, nil)

	region := visitor.Init(&missingRegionScan{scan: s})
	region.Visit(cu, nil)
	s.missingRegionBlocked = region.references != region.compared

	metadata := visitor.Init(&imdsVarScan{scan: s})
	metadata.Visit(cu, nil)

	managers := visitor.Init(&managerVarScan{scan: s})
	managers.Visit(cu, nil)

	configs := visitor.Init(&configVarScan{scan: s})
	configs.Visit(cu, nil)
	s.classifyConfigLocals()

	if s.sessionNilCompared {
		s.reason = "a session compared to nil, which the config replacing it cannot be"
		return s
	}

	collect := visitor.Init(&clientVarScan{scan: s})
	collect.Visit(cu, nil)

	// After the clients, since a reach through a Config field is only
	// recognisable once its receiver is known to be one.
	access := visitor.Init(&configAccessScan{scan: s})
	access.Visit(cu, nil)
	if s.configAccessBlocked != "" {
		s.reason = s.configAccessBlocked
		return s
	}

	s.shapes = map[string]shapeRef{}
	s.sliceShapes = map[string]shapeRef{}
	shapes := visitor.Init(&shapeScan{scan: s})
	shapes.Visit(cu, nil)

	check := visitor.Init(&blockerScan{scan: s})
	check.Visit(cu, nil)
	s.migratable = !check.blocked
	s.reason = check.reason
	return s
}

// bindsLocalName reports whether anything in cu declares name — a parameter, a
// variable, a field or a function — which an added import of the same name
// would collide with.
func bindsLocalName(cu *golang.CompilationUnit, name string) bool {
	scan := visitor.Init(&localNameScan{name: name})
	scan.Visit(cu, nil)
	return scan.found
}

type localNameScan struct {
	visitor.GoVisitor
	name  string
	found bool
}

func (v *localNameScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	for _, rp := range vd.Variables {
		if d := rp.Element; d != nil && d.Name != nil && d.Name.Name == v.name {
			v.found = true
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

func (v *localNameScan) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	if md.Name != nil && md.Name.Name == v.name {
		v.found = true
	}
	return v.GoVisitor.VisitMethodDeclaration(md, p)
}

func (v *localNameScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if java.HasMarker[golang.ShortVarDecl](a.Markers) {
		if id, ok := a.Variable.(*java.Identifier); ok && id.Name == v.name {
			v.found = true
		}
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

func (v *localNameScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	if java.HasMarker[golang.ShortVarDecl](ma.Markers) {
		for _, rp := range ma.Variables {
			if id, ok := rp.Element.(*java.Identifier); ok && id.Name == v.name {
				v.found = true
			}
		}
	}
	return v.GoVisitor.VisitMultiAssignment(ma, p)
}

// bindsName reports whether cu already binds name through an import.
func bindsName(cu *golang.CompilationUnit, name string) bool {
	if cu == nil || cu.Imports == nil {
		return false
	}
	for _, rp := range cu.Imports.Elements {
		if pathswap.LocalName(rp.Element) == name {
			return true
		}
	}
	return false
}

// clientVarScan records the variables assigned from a service client
// constructor, so their operation calls can be told apart from any other method
// call in the file.
type clientVarScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *clientVarScan) VisitAssignment(a *java.Assignment, p any) java.J {
	a = v.GoVisitor.VisitAssignment(a, p).(*java.Assignment)
	name, ok := a.Variable.(*java.Identifier)
	if !ok {
		return a
	}
	if mi, ok := a.Value.Element.(*java.MethodInvocation); ok {
		if service, ok := v.scan.clientConstructorService(mi); ok {
			v.scan.clients[name.Name] = service
		}
	}
	return a
}

// A client also arrives already built — as a parameter, or a variable declared
// with the client type. The manifest names that type, and the relocation
// rewrites the declaration to *<svc>.Client, so its operations take a context
// like any other.
func (v *clientVarScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	vd = v.GoVisitor.VisitVariableDeclarations(vd, p).(*java.VariableDeclarations)
	service, ok := v.scan.declaredClientService(vd.TypeExpr)
	if !ok {
		return vd
	}
	for _, decl := range vd.Variables {
		if d := decl.Element; d != nil && d.Name != nil {
			v.scan.clients[d.Name.Name] = service
		}
	}
	return vd
}

// declaredClientService reports whether a type expression names a v1 service
// client, through a pointer or directly.
func (s *fileScan) declaredClientService(expr java.Expression) (string, bool) {
	if ptr, ok := expr.(*golang.PointerType); ok {
		expr = ptr.Elem
	}
	fa, ok := expr.(*java.FieldAccess)
	if !ok || fa.Name.Element == nil {
		return "", false
	}
	target, ok := fa.Target.(*java.Identifier)
	if !ok {
		return "", false
	}
	service, isService := s.services[target.Name]
	if !isService {
		return "", false
	}
	if awsmanifest.Place(service, fa.Name.Element.Name) != awsmanifest.IsClient {
		return "", false
	}
	return service, true
}

// nonClientReceiver returns the name a method's receiver binds, when the type it
// binds it to is not a service client. The Go LST keeps the receiver outside the
// java.MethodDeclaration the shadow scan walks, so without this a method on a
// type called `c` inherits the client-ness of a `c` declared elsewhere.
func (s *fileScan) nonClientReceiver(md *golang.MethodDeclaration) string {
	for _, rp := range md.Receiver.Elements {
		vd, isDecl := rp.Element.(*java.VariableDeclarations)
		if !isDecl {
			continue
		}
		if _, isClient := s.declaredClientService(vd.TypeExpr); isClient {
			continue
		}
		for _, decl := range vd.Variables {
			if d := decl.Element; d != nil && d.Name != nil {
				return d.Name.Name
			}
		}
	}
	return ""
}

// clientConstructorService reports whether mi is `<service>.New(…)`, returning
// the service's short name.
func (s *fileScan) clientConstructorService(mi *java.MethodInvocation) (string, bool) {
	if mi.Name == nil || mi.Select == nil {
		return "", false
	}
	// The editor visits a call's children before the call itself, so a chained
	// constructor has already become NewFromConfig by the time the operation
	// around it asks what its receiver is.
	if mi.Name.Name != "New" && mi.Name.Name != "NewFromConfig" {
		return "", false
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	if !ok {
		return "", false
	}
	service, isService := s.services[recv.Name]
	return service, isService
}

// blockerScan walks a file for any v1 construct the rewrite cannot carry over.
type blockerScan struct {
	visitor.GoVisitor
	scan    *fileScan
	blocked bool
	reason  string
	// shadowed are the names this function declares with a non-client type.
	shadowed map[string]bool
	// receiver is the enclosing method's receiver name, when it binds something
	// other than a client.
	receiver string
	// ctxName is the context.Context in scope, if any. A function without one
	// still migrates — v2 takes a context everywhere v1 took none, and
	// context.TODO() is what stands in until a real one is plumbed through.
	ctxName string
}

func (v *blockerScan) VisitGoMethodDeclaration(md *golang.MethodDeclaration, p any) java.J {
	outer := v.receiver
	v.receiver = v.scan.nonClientReceiver(md)
	md = v.GoVisitor.VisitGoMethodDeclaration(md, p).(*golang.MethodDeclaration)
	v.receiver = outer
	return md
}

func (v *blockerScan) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	outer := v.scan.scopeTo(md)
	defer v.scan.restore(outer)
	outerCtx, outerShadowed := v.ctxName, v.shadowed
	if name := contextParamName(md); name != "" {
		v.ctxName = name
	}
	v.shadowed = shadowedClients(md, v.scan)
	if v.receiver != "" {
		v.shadowed[v.receiver] = true
	}
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)
	v.ctxName, v.shadowed = outerCtx, outerShadowed
	return md
}

func (v *blockerScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if v.blocked {
		return mi
	}
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Name == nil {
		return mi
	}
	name := mi.Name.Name

	if call, ok := qualifiedCall(mi, v.scan.sessionPkg); ok {
		// NewSession has a faithful config form, and Must expands into the load
		// plus the panic it stood for.
		switch call {
		case "NewSession":
			if !v.scan.convertibleSessionArgs(mi) {
				v.block(v.scan.sessionArgReason(mi, "session.NewSession"))
			}
		case "New":
			// session.New is NewSession without the error return, which the
			// generated loader stands in for.
			if !v.scan.convertibleSessionArgs(mi) {
				v.block(v.scan.sessionArgReason(mi, "session.New"))
			}
		case "NewSessionWithOptions":
			if v.scan.sessionOptionsBlocked || len(realArgs(mi)) != 1 {
				v.block("session.NewSessionWithOptions with options beyond a profile or a region")
			}
		case "Must":
			if !v.scan.mustWrapsNewSession(mi, v.scan.sessionPkg) {
				v.block("session.Must wrapping something other than NewSession")
			}
		default:
			v.block("session." + call)
		}
		return mi
	}

	if call, ok := qualifiedCall(mi, v.scan.imdsPkg); ok {
		if !imdsNames[call] {
			v.block("ec2metadata." + call)
		}
		return mi
	}
	if method, isMetadata := v.scan.imdsMethod(mi); isMetadata {
		// Every method answers with an output struct in v2 where v1 returned
		// the value; only the ones with a helper carry over.
		if !imdsMethods[method] {
			v.block("ec2metadata client method " + method)
		}
		return mi
	}

	if method, isManager := v.scan.managerMethod(mi); isManager {
		if !managerMethods[strings.TrimSuffix(method, "WithContext")] {
			v.block("s3manager " + method)
		}
		return mi
	}

	if call, ok := qualifiedCall(mi, v.scan.managerPkg); ok {
		if reason := v.scan.managerBlockReason(call); reason != "" {
			v.block(reason)
		}
		return mi
	}

	if call, ok := qualifiedCall(mi, v.scan.credentialsPkg); ok {
		// A shared-credentials provider the loader options do not reach has a
		// generated stand-in, which needs the file and the profile it named.
		handled := call == "NewStaticCredentials" || v.scan.consumesCredentials(mi) ||
			(call == "NewSharedCredentials" && len(realArgs(mi)) == 2)
		if !handled {
			v.block("credentials." + call)
		}
		return mi
	}

	// A call on a service package the manifest cannot account for — a v1-only
	// helper, say — has nowhere to land in v2.
	if recv, isIdent := selectIdentifier(mi); isIdent {
		if service, isService := v.scan.services[recv.Name]; isService {
			if _, hasHelper := serviceHelpers[service][name]; hasHelper {
				return mi
			}
			if awsmanifest.Place(service, name) == awsmanifest.Unknown {
				v.block("unknown name " + service + "." + name)
				return mi
			}
		}
	}

	// v1 gave every shape fluent setters; v2 dropped them, and the field they
	// set is typed differently often enough that the assignment is not a
	// rewrite this recipe makes.
	if v.scan.shapeSetter(mi) {
		if _, _, _, ok := v.scan.shapeSetterTarget(mi); !ok {
			v.block("fluent setter " + name)
		}
		return mi
	}

	// A page iterator, on a client or not. v2 hands out a paginator the caller
	// drives, which the manifest has to name.
	if isPages(name) {
		if _, _, _, _, _, _, ok := v.scan.pagesCall(mi); !ok {
			v.block("paginator " + name)
		}
		return mi
	}

	// An operation on a client value.
	if _, isClient := v.scan.clientOperationService(mi); isClient && !v.shadowedReceiver(mi) {
		if !convertibleOperation(mi, name) {
			v.block(operationBlockReason(name))
		}
		return mi
	}

	// A client constructor taking a trailing config: v2 takes functional
	// options over the client's own Options instead.
	if _, isConstructor := v.scan.clientConstructorService(mi); isConstructor {
		if !v.scan.convertibleClientConstructor(mi) {
			v.block("client constructor with options beyond a region")
		}
		return mi
	}

	// A waiter anywhere in the file, on a client or not. v2 makes it a type
	// built from the client, which the manifest has to name.
	if strings.HasPrefix(name, waiterPrefix) && name != waiterPrefix {
		if _, _, ok := v.scan.waiterType(mi); !ok {
			v.block("waiter " + name)
		}
		return mi
	}

	// An operation reached through anything else — a struct field, an interface
	// the file declares, a value passed in — is still an operation, recognised
	// by the input shape it takes. Its receiver's declared type gains the
	// context along with it, since the declaration is retyped too.
	return mi
}

// compositeShape names the service and shape a composite literal constructs,
// whether it still names the service package or has already been relocated to
// the types alias.
func (s *fileScan) compositeShape(comp *golang.Composite) (service, shape string, ok bool) {
	fa, isField := comp.TypeExpr.(*java.FieldAccess)
	if !isField || fa.Name.Element == nil {
		return "", "", false
	}
	target, isIdent := fa.Target.(*java.Identifier)
	if !isIdent {
		return "", "", false
	}
	if service, shape, isManager := s.managerShapeOf(comp); isManager {
		return service, shape, true
	}
	if svc, isService := s.services[target.Name]; isService {
		return svc, fa.Name.Element.Name, true
	}
	for _, svc := range s.services {
		if target.Name == awsmanifest.TypesAlias(svc) {
			return svc, fa.Name.Element.Name, true
		}
	}
	return "", "", false
}

// compositeShapeAt names the shape a composite constructs, including one that
// left its type out because the composite around it already said what its
// elements are — the shape of a test fixture's `[]s3.ListObjectsOutput{{…}}`.
func (s *fileScan) compositeShapeAt(comp *golang.Composite, cursor *visitor.Cursor) (service, shape string, ok bool) {
	if service, shape, named := s.compositeShape(comp); named {
		return service, shape, true
	}
	if comp.TypeExpr != nil || cursor == nil {
		return "", "", false
	}
	// A map's values sit under a key-value pair; a slice's elements do not.
	parent := cursor.Parent()
	if parent == nil {
		return "", "", false
	}
	if _, isKeyValue := parent.Value().(*golang.KeyValue); isKeyValue {
		parent = parent.Parent()
	}
	if parent == nil {
		return "", "", false
	}
	enclosing, isComposite := parent.Value().(*golang.Composite)
	if !isComposite {
		return "", "", false
	}
	arr, isArray := enclosing.TypeExpr.(*java.ArrayType)
	if !isArray || arr.ElementType == nil {
		return "", "", false
	}
	ref, isShape := s.declaredShape(arr.ElementType)
	if !isShape {
		return "", "", false
	}
	return ref.service, ref.shape, true
}

// operationInputService reports whether mi takes a single AWS operation input,
// returning the service that input belongs to. Every v1 operation has this
// shape, whatever it is called on.
func (s *fileScan) operationInputService(mi *java.MethodInvocation) (string, bool) {
	args := realArgs(mi)
	if len(args) != 1 {
		return "", false
	}
	// The input is as often held in a variable as written inline, which the
	// shape inference has already named. The name has to be the operation that
	// input belongs to: a call taking a shape is not an operation on it, as
	// `json.Marshal(input)` is not.
	service, shape, isShape := s.inputShape(args[0])
	if !isShape || mi.Name == nil {
		return "", false
	}
	input, _, isOperation := awsmanifest.OperationShapes(service, mi.Name.Name)
	if !isOperation || input != shape {
		return "", false
	}
	return service, true
}

// An enum field changed type between v1 and v2, so the value assigned to it has
// to be one the rewrite can retype. Anything else leaves the file alone.
func (v *blockerScan) VisitComposite(comp *golang.Composite, p any) java.J {
	if v.blocked {
		return comp
	}
	cursor := v.Cursor()
	// An aws.Config the session does not consume keeps its settings, so each one
	// has to be a setting v2 still holds on the config.
	if v.scan.awsConfigLiteral(comp) && !consumedAsSessionConfig(v.scan, cursor) {
		for _, rp := range comp.Elements.Elements {
			kv, isKeyValue := rp.Element.(*golang.KeyValue)
			if !isKeyValue {
				continue
			}
			if key, isIdent := kv.Key.(*java.Identifier); isIdent {
				if _, mapped := configTargets[key.Name]; !mapped {
					v.block("aws.Config." + key.Name + ", which v2 configures elsewhere")
					return comp
				}
			}
		}
		return v.GoVisitor.VisitComposite(comp, p)
	}
	comp = v.GoVisitor.VisitComposite(comp, p).(*golang.Composite)
	service, shape, ok := v.scan.compositeShapeAt(comp, cursor)
	if !ok {
		return comp
	}
	for _, rp := range comp.Elements.Elements {
		kv, ok := rp.Element.(*golang.KeyValue)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*java.Identifier)
		if !ok {
			continue
		}
		if awsmanifest.IsMapOfValueSlices(service, shape, key.Name) {
			v.block("map field " + shape + "." + key.Name + ", whose slice values v2 holds by value")
			continue
		}
		enum := awsmanifest.FieldEnum(service, shape, key.Name)
		if enum == "" {
			continue
		}
		if enumSliceOf(service, shape, key.Name) != "" {
			if !v.scan.retypableEnumSlice(kv.Value.Element, service, enum) {
				v.block("enum list " + shape + "." + key.Name + " set from something unconvertible")
			}
			continue
		}
		if !v.scan.retypableEnumValue(kv.Value.Element, service, enum) {
			v.block("enum field " + shape + "." + key.Name + " set from something unconvertible")
			return comp
		}
	}
	return comp
}

// operationBlockReason names why an operation shape has no single-call v2 form.
func operationBlockReason(name string) string {
	for _, suffix := range blockedCallSuffixes {
		if strings.HasSuffix(name, suffix) && name != suffix {
			return "paginator " + name
		}
	}
	return "operation " + name + " with options beyond its input"
}

// block records the first reason a file was held back.
func (v *blockerScan) block(reason string) {
	v.blocked = true
	if v.reason == "" {
		v.reason = reason
	}
}

// shadowedReceiver reports whether a call's receiver is a name this function
// declared as something other than a client.
func (v *blockerScan) shadowedReceiver(mi *java.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	return ok && v.shadowed[recv.Name]
}

// retypableEnumValue reports whether an enum field's value can be carried over:
// an aws.String around a constant of that same enum loses its wrapper, and one
// around a computed value becomes a conversion. A constant belonging to some
// other enum fitted v1's *string field and fits nothing in v2, and a bare
// pointer needs a dereference this recipe does not write.
func (s *fileScan) retypableEnumValue(expr java.Expression, service, enum string) bool {
	// A read off another shape's field of the same enum needs nothing: v2 lines
	// both ends up.
	if read, isRead := expr.(*java.FieldAccess); isRead {
		if held, isEnum := s.enumFieldOf(read); isEnum {
			return held == enum
		}
	}
	inner, ok := s.enumValueSource(expr)
	if !ok {
		return false
	}
	fa, ok := inner.(*java.FieldAccess)
	if !ok || fa.Name.Element == nil {
		// Computed at runtime, which converts cleanly.
		return true
	}
	target, ok := fa.Target.(*java.Identifier)
	if !ok || s.services[target.Name] != service {
		return true
	}
	return awsmanifest.EnumOf(service, fa.Name.Element.Name) == enum
}

// enumValueSource returns the string expression a v1 enum field's value carried.
// The field was a *string, so it was written either as aws.String around a value
// or as the address of one.
func (s *fileScan) enumValueSource(expr java.Expression) (java.Expression, bool) {
	switch e := expr.(type) {
	case *java.MethodInvocation:
		if helper, isAws := qualifiedCall(e, s.awsPkg); isAws && helper == "String" {
			if args := realArgs(e); len(args) == 1 {
				return args[0], true
			}
		}
	case *golang.Unary:
		if e.Operator.Element == golang.AddressOf {
			return e.Expression, true
		}
	}
	return nil, false
}

func (v *blockerScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if v.blocked {
		return fa
	}
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)

	// The session package has no v2 counterpart at all, and v2's credentials
	// package exports different types, so a reference to either dangles once
	// the import moves. Only the calls this recipe rewrites are handled.
	if name, ok := qualifiedRef(fa, v.scan.sessionPkg); ok {
		// A session passed between functions becomes the config that replaced
		// it; anything else the package exported has no counterpart.
		if name != "Session" && !(sessionOptionNames[name] && !v.scan.sessionOptionsBlocked) {
			v.block("session." + name + " as a type")
		}
		return fa
	}
	if name, ok := qualifiedRef(fa, v.scan.imdsPkg); ok {
		if !imdsNames[name] {
			v.block("ec2metadata." + name + " as a type")
		}
		return fa
	}
	if name, ok := qualifiedRef(fa, v.scan.credentialsPkg); ok {
		v.block("credentials." + name + " as a type")
		return fa
	}

	if name, ok := qualifiedRef(fa, v.scan.awserrPkg); ok {
		// Only the error interface has a smithy counterpart the rewrite emits;
		// the constructors and batched shapes have none.
		if !awserrSupported[name] {
			v.block("awserr." + name)
		}
		return fa
	}

	if name, ok := qualifiedRef(fa, v.scan.awsPkg); ok {
		// aws.Config survives only as the session argument this recipe unpacks,
		// which it has already consumed by the time a bare reference is seen.
		if name == errMissingRegion {
			// Only a comparison against the sentinel has a v2 form; a reference
			// the rewrite would leave behind does not.
			if v.scan.missingRegionBlocked {
				v.block("aws." + errMissingRegion)
			}
			return fa
		}
		if !awsPackageSurvivors[name] && valueHelperRenames[name] == "" && name != "Config" {
			v.block("aws." + name)
		}
		return fa
	}
	if target, isIdent := fa.Target.(*java.Identifier); isIdent && fa.Name.Element != nil {
		// v1's client carried its configuration on itself; v2's carries none of
		// it, so a read off a client value has nothing to resolve to.
		_, isPackage := v.scan.services[target.Name]
		// Config is the one reach with a v2 counterpart, which the config-access
		// scan has already checked in full.
		if _, isClient := v.scan.clients[target.Name]; isClient && !isPackage && !v.shadowed[target.Name] &&
			fa.Name.Element.Name != configField {
			v.block("client field " + fa.Name.Element.Name)
			return fa
		}
	}
	for local, service := range v.scan.services {
		name, ok := qualifiedRef(fa, local)
		if !ok {
			continue
		}
		// The manifest says where v2 put the name. Anything it cannot account
		// for — an ErrCode constant that became a typed error, a service with no
		// manifest — leaves the file alone rather than being guessed at.
		if awsmanifest.Place(service, name) == awsmanifest.Unknown {
			if _, isErrCode := awsmanifest.ErrCodeValue(service, name); !isErrCode {
				v.block("unknown name " + service + "." + name)
			}
		}
		return fa
	}
	return fa
}

// shapeSetter reports whether mi is one of v1's fluent setters on a shape the
// scan resolved: `input.SetNextToken(tok)`.
func (s *fileScan) shapeSetter(mi *java.MethodInvocation) bool {
	if mi.Name == nil || !strings.HasPrefix(mi.Name.Name, "Set") || mi.Name.Name == "Set" {
		return false
	}
	recv, isIdent := selectIdentifier(mi)
	if !isIdent {
		return false
	}
	_, isShape := s.shapes[recv.Name]
	return isShape
}

// selectIdentifier returns a call's receiver when it is a bare name.
func selectIdentifier(mi *java.MethodInvocation) (*java.Identifier, bool) {
	if mi.Select == nil {
		return nil, false
	}
	id, ok := mi.Select.Element.(*java.Identifier)
	return id, ok
}

// depointeredFieldOf2 reports whether v2 holds the field by value where v1 held
// a pointer to it.
func (s *fileScan) depointeredFieldOf2(fa *java.FieldAccess) bool {
	_, ok := s.depointeredFieldOf(fa)
	return ok
}

// dereferenced reports whether the node the cursor points at is the operand of
// a `*`.
func dereferenced(cursor *visitor.Cursor) bool {
	parent := cursor.Parent()
	if parent == nil {
		return false
	}
	unary, isUnary := parent.Value().(*golang.Unary)
	return isUnary && unary.Operator.Element == golang.Indirection
}

// usesService reports whether the file imported the named v1 service package.
func (s *fileScan) usesService(service string) bool {
	for _, imported := range s.services {
		if imported == service {
			return true
		}
	}
	return false
}

// shapeScan names the shape each variable holds, from the same seeds the
// rewrite uses: an operation's output, a shape literal, a declared type, and a
// range over a value-slice field.
type shapeScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *shapeScan) VisitAssignment(a *java.Assignment, p any) java.J {
	a = v.GoVisitor.VisitAssignment(a, p).(*java.Assignment)
	if name, ok := a.Variable.(*java.Identifier); ok {
		if ref, ok := v.scan.shapeOfExpression(a.Value.Element); ok {
			v.scan.shapes[name.Name] = ref
		}
	}
	return a
}

func (v *shapeScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	ma = v.GoVisitor.VisitMultiAssignment(ma, p).(*golang.MultiAssignment)
	if len(ma.Values) == 1 && len(ma.Variables) > 0 {
		if name, ok := ma.Variables[0].Element.(*java.Identifier); ok {
			if ref, ok := v.scan.shapeOfExpression(ma.Values[0].Element); ok {
				v.scan.shapes[name.Name] = ref
			}
		}
	}
	return ma
}

func (v *shapeScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	vd = v.GoVisitor.VisitVariableDeclarations(vd, p).(*java.VariableDeclarations)
	if elem, ok := v.scan.declaredSliceShape(vd.TypeExpr); ok {
		for _, decl := range vd.Variables {
			if d := decl.Element; d != nil && d.Name != nil {
				v.scan.sliceShapes[d.Name.Name] = elem
			}
		}
	}
	if ref, ok := v.scan.declaredShape(vd.TypeExpr); ok {
		for _, decl := range vd.Variables {
			if d := decl.Element; d != nil && d.Name != nil {
				v.scan.shapes[d.Name.Name] = ref
			}
		}
	}
	return vd
}

func (v *shapeScan) VisitForEachLoop(loop *java.ForEachLoop, p any) java.J {
	if name, ref, ok := v.scan.rangeElementShape(loop); ok {
		v.scan.shapes[name] = ref
	}
	return v.GoVisitor.VisitForEachLoop(loop, p)
}

// receiverShape names the shape an expression holds: a name the scan bound, or a
// read of a field that holds another shape, which is what carries the inference
// through `i.State.Name`.
func (s *fileScan) receiverShape(expr java.Expression) (shapeRef, bool) {
	switch target := expr.(type) {
	case *java.Identifier:
		if owner, known := s.shapes[target.Name]; known {
			return owner, true
		}
		// The scan seeds itself from what one function can see; a value that
		// came from another file is named only by the parse-time type.
		return s.attributedShape(target)
	case *java.FieldAccess:
		if target.Name.Element == nil {
			return shapeRef{}, false
		}
		outer, known := s.receiverShape(target.Target)
		if !known {
			return shapeRef{}, false
		}
		elem := awsmanifest.FieldShape(outer.service, outer.shape, target.Name.Element.Name)
		if elem == "" {
			return shapeRef{}, false
		}
		return shapeRef{service: outer.service, shape: elem}, true
	}
	return shapeRef{}, false
}

// attributedShape reads a shape off an expression's parse-time type, for a value
// the scan's own seeds do not reach. There is no attribution for a package the
// parser could not resolve, so this adds to the inference rather than replacing
// it.
func (s *fileScan) attributedShape(expr java.Expression) (shapeRef, bool) {
	fqn := matcher.GetFullyQualifiedName(matcher.TypeOfExpression(expr))
	if fqn == "" {
		return shapeRef{}, false
	}
	for _, prefix := range []string{v1ServicePkg, v2ServicePkg} {
		rest, under := strings.CutPrefix(fqn, prefix)
		if !under {
			continue
		}
		pkg, name, named := strings.Cut(rest, ".")
		if !named {
			continue
		}
		// The path itself is the guard: a name the manifest knows cannot be read
		// off something unrelated once the type says which SDK package it came
		// from. The file need not import that package — the value as often
		// arrives from one that does.
		return shapeRef{service: strings.TrimSuffix(pkg, "/types"), shape: name}, true
	}
	return shapeRef{}, false
}

// enumFieldOf names the enum a dereferenced field carries, when the receiver's
// shape is known.
func (s *fileScan) enumFieldOf(fa *java.FieldAccess) (string, bool) {
	if fa.Name.Element == nil {
		return "", false
	}
	owner, known := s.receiverShape(fa.Target)
	if !known {
		return "", false
	}
	enum := awsmanifest.FieldEnum(owner.service, owner.shape, fa.Name.Element.Name)
	return enum, enum != ""
}

// pointerHelpers are the aws package's pointer constructors, which a v1 call
// site wrapped a value in to fill a pointer field.
var pointerHelpers = map[string]bool{
	"Bool": true, "Float64": true, "Int": true, "Int32": true, "Int64": true,
	"String": true, "Time": true, "Uint": true, "Uint64": true,
}

// pointerValueSource returns the value a v1 call site wrapped to fill a pointer
// field, written either as an aws helper or as the address of a local.
func (s *fileScan) pointerValueSource(expr java.Expression) (java.Expression, bool) {
	switch e := expr.(type) {
	case *java.MethodInvocation:
		if helper, isAws := qualifiedCall(e, s.awsPkg); isAws && pointerHelpers[helper] {
			if args := realArgs(e); len(args) == 1 {
				return args[0], true
			}
		}
	case *golang.Unary:
		if e.Operator.Element == golang.AddressOf {
			return e.Expression, true
		}
	}
	return nil, false
}

// depointeredFieldOf names the basic type v2 holds a field by value where v1
// held a pointer to it, for a receiver whose shape the scan resolved.
func (s *fileScan) depointeredFieldOf(fa *java.FieldAccess) (string, bool) {
	if fa.Name.Element == nil {
		return "", false
	}
	owner, known := s.receiverShape(fa.Target)
	if !known {
		return "", false
	}
	basic := awsmanifest.Depointered(owner.service, owner.shape, fa.Name.Element.Name)
	return basic, basic != ""
}

// unresolvedEnumField reports whether fa reads a field the imported services
// only ever use for an enum, where the receiver's own shape is not known. The
// name alone settles the rewrite: whatever shape it belongs to, v2 holds it by
// value. A receiver that is not an SDK shape at all would take the conversion
// wrongly, but it fails to compile rather than passing silently, and standing
// off would hold back every file in the module.
func (s *fileScan) unresolvedEnumField(fa *java.FieldAccess) bool {
	if fa.Name.Element == nil {
		return false
	}
	if _, resolved := s.enumFieldOf(fa); resolved {
		return false
	}
	for _, service := range s.services {
		if awsmanifest.IsOnlyEnumFieldNamed(service, fa.Name.Element.Name) {
			return true
		}
	}
	return false
}

// scope is what a function's own bindings replace while it is being visited.
// The maps are keyed by name, and two functions readily use the same name for
// different things — one for an EC2 client, the next for an SSM one; one ranging
// SDK instances, the next ranging a struct of its own.
type scope struct {
	shapes  map[string]shapeRef
	slices  map[string]shapeRef
	clients map[string]string
}

// scopeTo re-derives the name-keyed inference for one function: what the file
// declares outside it, less every name it binds, plus what those bindings turn
// out to hold. A function literal sees the scope around it, so it is left alone.
func (s *fileScan) scopeTo(md *java.MethodDeclaration) scope {
	outer := scope{shapes: s.shapes, slices: s.sliceShapes, clients: s.clients}
	if md.Name == nil || md.Name.Name == "" {
		return outer
	}
	bound := boundNames(md)
	s.shapes = without(outer.shapes, bound)
	s.sliceShapes = without(outer.slices, bound)
	s.clients = without(outer.clients, bound)
	visitor.Init(&clientVarScan{scan: s}).Visit(md, nil)
	visitor.Init(&shapeScan{scan: s}).Visit(md, nil)
	return outer
}

// restore puts back what scopeTo replaced.
func (s *fileScan) restore(outer scope) {
	s.shapes, s.sliceShapes, s.clients = outer.shapes, outer.slices, outer.clients
}

func without[V any](m map[string]V, names map[string]bool) map[string]V {
	out := make(map[string]V, len(m))
	for k, v := range m {
		if !names[k] {
			out[k] = v
		}
	}
	return out
}

// boundNames returns every name a function binds, whatever it binds it to.
func boundNames(md *java.MethodDeclaration) map[string]bool {
	names := map[string]bool{}
	visitor.Init(&bindingScan{names: names}).Visit(md, nil)
	return names
}

type bindingScan struct {
	visitor.GoVisitor
	names map[string]bool
}

func (v *bindingScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	for _, rp := range vd.Variables {
		if d := rp.Element; d != nil && d.Name != nil {
			v.names[d.Name.Name] = true
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

func (v *bindingScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if java.HasMarker[golang.ShortVarDecl](a.Markers) {
		if id, isIdent := a.Variable.(*java.Identifier); isIdent {
			v.names[id.Name] = true
		}
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

func (v *bindingScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	if java.HasMarker[golang.ShortVarDecl](ma.Markers) {
		for _, rp := range ma.Variables {
			if id, isIdent := rp.Element.(*java.Identifier); isIdent {
				v.names[id.Name] = true
			}
		}
	}
	return v.GoVisitor.VisitMultiAssignment(ma, p)
}

// shadowedClients returns the names a function declares with a type that is not
// a service client, though some other function in the file uses the same name
// for one. The maps are keyed by name, so without this a parameter called `cw`
// holding an interface would inherit the client-ness of a `cw` elsewhere.
func shadowedClients(md *java.MethodDeclaration, scan *fileScan) map[string]bool {
	shadowed := map[string]bool{}
	collect := visitor.Init(&shadowScan{scan: scan, shadowed: shadowed})
	collect.Visit(md, nil)
	return shadowed
}

type shadowScan struct {
	visitor.GoVisitor
	scan     *fileScan
	shadowed map[string]bool
}

func (v *shadowScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	vd = v.GoVisitor.VisitVariableDeclarations(vd, p).(*java.VariableDeclarations)
	if _, isClient := v.scan.declaredClientService(vd.TypeExpr); isClient {
		return vd
	}
	for _, decl := range vd.Variables {
		if d := decl.Element; d != nil && d.Name != nil && v.scan.clients[d.Name.Name] != "" {
			v.shadowed[d.Name.Name] = true
		}
	}
	return vd
}

// clientOperationService reports whether mi is a call on a service client,
// returning that client's service. The client is either a variable the file
// assigned from a constructor, or a constructor called inline —
// `sts.New(sess).GetCallerIdentity(…)` is one call chained onto another.
func (s *fileScan) clientOperationService(mi *java.MethodInvocation) (string, bool) {
	if mi.Select == nil {
		return "", false
	}
	switch recv := mi.Select.Element.(type) {
	case *java.Identifier:
		// A name the file imports a service under is the package, whatever a
		// struct field of the same name holds; that field is reached through
		// its own receiver.
		if _, isService := s.services[recv.Name]; isService {
			return "", false
		}
		service, isClient := s.clients[recv.Name]
		return service, isClient
	case *java.MethodInvocation:
		return s.clientConstructorService(recv)
	}
	return "", false
}

// convertibleOperation reports whether a client call has a v2 form this recipe
// emits: the plain one-argument shape, or the WithContext shape with nothing
// after the input. A trailing request.Option has no v2 counterpart.
func convertibleOperation(mi *java.MethodInvocation, name string) bool {
	for _, suffix := range blockedCallSuffixes {
		if strings.HasSuffix(name, suffix) && name != suffix {
			return false
		}
	}
	args := realArgs(mi)
	if strings.HasSuffix(name, "WithContext") && name != "WithContext" {
		return len(args) == 2
	}
	return len(args) == 1
}

// mustWrapsNewSession reports whether a session.Must wraps exactly the
// constructor the expansion knows how to load.
func (s *fileScan) mustWrapsNewSession(mi *java.MethodInvocation, sessionPkg string) bool {
	args := realArgs(mi)
	if len(args) != 1 {
		return false
	}
	inner, ok := args[0].(*java.MethodInvocation)
	if !ok {
		return false
	}
	call, isSession := qualifiedCall(inner, sessionPkg)
	if !isSession {
		return false
	}
	switch call {
	case "NewSession":
		return s.convertibleSessionArgs(inner)
	case "NewSessionWithOptions":
		// The options literal is checked in full by its own scan; Must only has
		// to know that the load it wraps is one the rewrite writes.
		return !s.sessionOptionsBlocked && len(realArgs(inner)) == 1
	}
	return false
}

// convertibleSessionArgs reports whether a session.NewSession call is one the
// rewrite can express as config.LoadDefaultConfig: either no argument at all, or
// a single &aws.Config literal setting nothing but the region.
func (s *fileScan) convertibleSessionArgs(mi *java.MethodInvocation) bool {
	args := realArgs(mi)
	if len(args) == 0 {
		return true
	}
	if len(args) != 1 {
		return false
	}
	if _, ok := s.sessionArgOptions(args[0]); ok {
		return true
	}
	// A local the declaration takes the load over for is already the config.
	_, restructured := s.restructuredConfigVar(args[0])
	return restructured
}

// sessionArgReason says why a session constructor's argument does not carry
// over, naming the config local where that is what went wrong.
func (s *fileScan) sessionArgReason(mi *java.MethodInvocation, call string) string {
	if args := realArgs(mi); len(args) == 1 {
		if reason := s.mutatedConfigReason(args[0]); reason != "" {
			return reason
		}
	}
	return call + " with options the v2 loader has none for"
}

// mutatedConfigReason names a config local the rewrite cannot lift options out
// of, for a clearer answer than the session call that consumed it.
func (s *fileScan) mutatedConfigReason(expr java.Expression) string {
	name, isVar := s.sessionConfigVar(expr)
	if !isVar || s.configRestructured[name] {
		return ""
	}
	if s.configMutated[name] {
		return "an aws.Config built field by field, which the v2 loader takes as options up front"
	}
	if !s.droppableConfigVar(name) {
		return "an aws.Config local something else reads"
	}
	return ""
}

// contextParamName returns the name of md's context.Context parameter, or "".
func contextParamName(md *java.MethodDeclaration) string {
	for _, rp := range md.Parameters.Elements {
		vd, ok := rp.Element.(*java.VariableDeclarations)
		if !ok || !isContextType(vd.TypeExpr) {
			continue
		}
		for _, decl := range vd.Variables {
			if d := decl.Element; d != nil && d.Name != nil && d.Name.Name != "_" {
				return d.Name.Name
			}
		}
	}
	return ""
}

func isContextType(expr java.Expression) bool {
	if matcher.GetFullyQualifiedName(matcher.TypeOfExpression(expr)) == "context.Context" {
		return true
	}
	fa, ok := expr.(*java.FieldAccess)
	if !ok || fa.Name.Element == nil || fa.Name.Element.Name != "Context" {
		return false
	}
	target, ok := fa.Target.(*java.Identifier)
	return ok && target.Name == contextPkg
}
