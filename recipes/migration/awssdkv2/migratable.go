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
	// shapeAddress records whether each shape variable holds a pointer or a
	// value.
	shapeAddress map[string]byte
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
	// ec2RoleCredsPkg is the local name of the instance-role credentials package.
	ec2RoleCredsPkg string
	// stscredsPkg is the local name of the assume-role credentials package.
	stscredsPkg string
	// expressionPkg is the local name of the DynamoDB expression builder.
	expressionPkg string
	// gureguPkg is the local name of github.com/guregu/dynamo.
	gureguPkg string
	// restPkg is the local name of v1's private REST protocol helpers.
	restPkg string
	// rdsutilsPkg is the local name of the RDS IAM token signer.
	rdsutilsPkg string
	// ioPkg, ioutilPkg, bytesPkg and stringsPkg are the names those standard
	// library packages are imported under.
	ioPkg, ioutilPkg, bytesPkg, stringsPkg string
	// stsNameTaken records that the file binds the name an added STS import
	// would take.
	stsNameTaken bool
	// managerFieldsUnproven are the manager fields this file sets from
	// something not proven to hold S3's own client, and managerClientReads the
	// ones it calls through the S3 of.
	managerFieldsUnproven map[string]bool
	managerClientReads    map[string]bool
	// managerNameTaken records that the file binds the name an added transfer
	// manager import would take.
	managerNameTaken bool
	// callsPartitions records that the file lists v1's partitions.
	callsPartitions bool
	// endpointsPkg is the local name of v1's endpoints package, and clientPkg
	// of its client plumbing.
	endpointsPkg string
	clientPkg    string
	// retryers are the retryer types the file declares, consumedClientRefs the
	// client references they and their literals take, and retryerValues the
	// Retryer values the rewrite carries over.
	retryers           map[string]*retryerType
	consumedClientRefs map[*java.FieldAccess]bool
	retryerValues      map[java.Expression]bool
	// configVariadics are the parameters declared `...*aws.Config`.
	configVariadics map[string]bool
	// requestPkg is the local name of v1's request package, and handlerReason
	// what held back a handler added through it. The consumed refs are the
	// `sess.Handlers` reaches and request references the handler rewrite takes.
	requestPkg          string
	handlerReason       string
	consumedHandlerRefs map[*java.FieldAccess]bool
	consumedRequestRefs map[*java.FieldAccess]bool
	// consumedRequestCalls are the request calls the handler rewrite takes.
	consumedRequestCalls map[*java.MethodInvocation]bool
	// retryerStatements are the locals a statement `request.WithRetryer(cfg, r)`
	// sets the retryer on, which folds into the local's options.
	retryerStatements map[string]bool
	// managerVars maps a name holding a transfer manager to which one it holds.
	managerVars map[string]string
	services    map[string]string
	// adjunct marks a file that names no SDK package but reads values of its
	// shapes, where only what attribution says about a value can guide the
	// rewrite.
	adjunct bool
	// configVars maps a local holding an aws.Config literal to the region it
	// sets, with configMutated recording the ones something else wrote to and
	// credentialsVars are the names declared as a *credentials.Credentials,
	// and consumedAssumeRoleRefs the options types assume-role calls take.
	credentialsVars        map[string]bool
	consumedAssumeRoleRefs map[*java.FieldAccess]bool
	// requestCalls are the OperationRequest calls the request stand-in takes,
	// and requests the names they bind.
	requestCalls map[*java.MethodInvocation]bool
	requests     map[string]bool
	// optionsValues are the names holding a session.Options value carried over
	// as config.LoadOptions, and optionsValuesOK that every such name is.
	optionsValues   map[string]bool
	optionsValuesOK bool
	// bindsStscreds records that the file binds the name stscreds to something
	// other than the package.
	bindsStscreds bool
	// builderVars are the config locals built up through setters, which become
	// the generated builder.
	builderVars map[string]bool
	// literalBuilderVars are those of them declared as a config literal.
	literalBuilderVars map[string]bool
	// configUses how many times each name appears.
	configVars map[string][]configOption
	// configLocals is what the scan learned about each, and configRestructured
	// and configPathStyle what it decided: the declaration takes over the load,
	// and the addressing style is hoisted into a local of its own.
	configLocals       map[string]*configLocal
	configRestructured map[string]bool
	configPathStyle    map[string]bool
	configMutated      map[string]bool
	// configKept are the locals that stay as v2 configs holding settings the
	// generated loader options read back.
	configKept map[string]bool
	configUses map[string]int
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
	// credentialValues are the names bound to credentials a config's provider
	// answered.
	credentialValues map[string]bool
	// awserrNames are the names bound to awserr values.
	awserrNames map[string]bool
	// requestFailureNames are those bound to an awserr.RequestFailure.
	requestFailureNames map[string]bool
	// consumedAwserrRefs are the awserr.Error references the rewrite replaces.
	consumedAwserrRefs map[*java.FieldAccess]bool
	// consumedOptionRefs are the session references inside an Options literal
	// handed straight to NewSessionWithOptions, which the rewrite folds into
	// loader options. Any other reference outlives the session import.
	consumedOptionRefs map[*java.FieldAccess]bool
	// configLocal is the name the v2 config import binds, which is the package's
	// own unless the file already uses it for something else.
	configLocal string
	// noCredentialsChecked records that an awserr code is compared against
	// NoCredentialProviders, so the file's matches go through the generated
	// helper that raises it.
	noCredentialsChecked bool
	// errorsLocal is the name the standard library's errors package binds. A
	// file on github.com/pkg/errors already spells `errors`, so the one this
	// recipe adds is aliased alongside it.
	errorsLocal string
	// allReasons, when the sweep sets it, collects every construct holding the
	// file back rather than stopping at the first.
	allReasons map[string]bool
	// ifacePaths are the v1 iface packages the file imports, whose replacement
	// the migration generates.
	ifacePaths []string
	// attributePort is set by an import of dynamodbattribute, which the port
	// generated into the module stands in for.
	attributePort bool

	// shapes names the modelled shape each variable holds, inferred from the
	// operation calls and shape literals in the file. Both the guard and the
	// rewrite read it, so they agree on what a field access refers to.
	shapes map[string]shapeRef
	// sliceShapes names the shape a variable holds a slice of, so a range over
	// it binds its elements.
	sliceShapes map[string]shapeRef
	// clientConfigs are the locals a client takes as trailing configs.
	clientConfigs map[string]bool
	// clients maps a variable holding a service client to that client's service,
	// so its operation calls can gain a context and name the v2 type they land
	// on.
	clients map[string]string
}

// scanFile decides whether cu migrates as a whole. v2 changed enough that a
// half-migrated file does not compile, so anything unhandled blocks all of it.
func scanFile(cu *golang.CompilationUnit) *fileScan {
	return scanFileReasons(cu, false)
}

// scanFileReasons is scanFile, optionally gathering every blocker.
func scanFileReasons(cu *golang.CompilationUnit, all bool) *fileScan {
	s := &fileScan{
		awsPkg:                 pathswap.Qualifier(cu, v1Aws),
		sessionPkg:             pathswap.Qualifier(cu, v1Session),
		credentialsPkg:         pathswap.Qualifier(cu, v1Credentials),
		awserrPkg:              pathswap.Qualifier(cu, v1Awserr),
		managerPkg:             pathswap.Qualifier(cu, v1S3Manager),
		imdsPkg:                pathswap.Qualifier(cu, v1Ec2Metadata),
		imdsVars:               map[string]bool{},
		managerVars:            map[string]string{},
		stscredsPkg:            pathswap.Qualifier(cu, v1Stscreds),
		ec2RoleCredsPkg:        pathswap.Qualifier(cu, v1Ec2RoleCreds),
		expressionPkg:          pathswap.Qualifier(cu, v1DynamoExpression),
		rdsutilsPkg:            pathswap.Qualifier(cu, v1Rdsutils),
		restPkg:                pathswap.Qualifier(cu, v1Rest),
		gureguPkg:              pathswap.Qualifier(cu, gureguV1),
		requestPkg:             pathswap.Qualifier(cu, v1Request),
		endpointsPkg:           pathswap.Qualifier(cu, v1Endpoints),
		clientPkg:              pathswap.Qualifier(cu, v1Client),
		configVariadics:        map[string]bool{},
		consumedClientRefs:     map[*java.FieldAccess]bool{},
		retryerValues:          map[java.Expression]bool{},
		managerFieldsUnproven:  map[string]bool{},
		managerClientReads:     map[string]bool{},
		consumedHandlerRefs:    map[*java.FieldAccess]bool{},
		consumedRequestRefs:    map[*java.FieldAccess]bool{},
		consumedRequestCalls:   map[*java.MethodInvocation]bool{},
		retryerStatements:      map[string]bool{},
		literalBuilderVars:     map[string]bool{},
		consumedAwserrRefs:     map[*java.FieldAccess]bool{},
		services:               serviceImports(cu),
		clients:                map[string]string{},
		configVars:             map[string][]configOption{},
		configLocals:           map[string]*configLocal{},
		configRestructured:     map[string]bool{},
		configPathStyle:        map[string]bool{},
		configMutated:          map[string]bool{},
		configKept:             map[string]bool{},
		configUses:             map[string]int{},
		requestFailureNames:    map[string]bool{},
		requestCalls:           map[*java.MethodInvocation]bool{},
		credentialsVars:        map[string]bool{},
		consumedAssumeRoleRefs: map[*java.FieldAccess]bool{},
		requests:               map[string]bool{},
		configLocal:            configPkgName,
		errorsLocal:            errorsPkg,
		sessionVars:            map[string]bool{},
		credentialsConsumed:    map[java.Expression]bool{},
	}
	if all {
		s.allReasons = map[string]bool{}
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
	// A v2 path already in the file would collide with the one the swap moves a
	// v1 import onto. A file part-way through by hand is otherwise fine.
	for _, rp := range cu.Imports.Elements {
		if moved, ok := MapPath(rp.Element.Path()); ok && pathswap.Imports(cu, moved) {
			s.reason = "imports both " + rp.Element.Path() + " and the v2 path it moves to"
			return s
		}
	}
	// The standard library's errors package is what the smithy and missing-region
	// rewrites call into, and a file already on another errors package binds the
	// name to that one.
	for _, rp := range cu.Imports.Elements {
		if pathswap.LocalName(rp.Element) == errorsPkg && rp.Element.Path() != errorsPkg {
			s.errorsLocal = "std" + errorsPkg
		}
	}
	if bindsLocalName(cu, errorsPkg) {
		s.errorsLocal = "std" + errorsPkg
	}

	// v2's loader lives in a package called config, a name Go code often gives
	// a local of its own. Where the file does, the import is aliased rather than
	// left to be shadowed.
	// An import of another package called config is aliased around the same way.
	if s.sessionPkg != "" && (bindsLocalName(cu, configPkgName) || bindsName(cu, configPkgName)) {
		s.configLocal = configPkgName + "aws"
		if bindsLocalName(cu, s.configLocal) || bindsName(cu, s.configLocal) {
			s.reason = "both config and " + s.configLocal + " already bound, leaving v2's config package no name"
			return s
		}
	}
	for _, rp := range cu.Imports.Elements {
		path := rp.Element.Path()
		if _, blocked := v1OnlyPackages[path]; blocked {
			s.reason = "v1-only package " + path
			return s
		}
		if reason, builtOnV1 := v1Libraries[path]; builtOnV1 {
			s.reason = path + ", " + reason
			return s
		}
		if _, _, isIface := ifacePackageName(path); isIface {
			continue
		}
		if path == v1DynamodbAttribute {
			if attributeMarshalers(cu, pathswap.LocalName(rp.Element)) {
				s.reason = attributeMarshalerReason
				return s
			}
			s.attributePort = true
			continue
		}
		if underV1(path) && path != v1Awserr && path != v1Request && path != v1Endpoints && path != v1Client {
			// awserr and request have no v2 path at all; their uses are
			// rewritten and the import is dropped rather than repointed.
			if _, ok := MapPath(path); !ok {
				s.reason = "no v2 path for " + path
				return s
			}
		}
	}

	for _, rp := range cu.Imports.Elements {
		path := rp.Element.Path()
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

	if reason := s.pageOverrideReason(cu); reason != "" {
		s.reason = reason
		return s
	}

	s.ioQualifiers(cu)
	if s.endpointsPkg != "" {
		visitor.Init(&partitionsCallScan{scan: s}).Visit(cu, nil)
	}
	s.stsNameTaken = bindsName(cu, stsService) || bindsLocalName(cu, stsService)
	s.managerNameTaken = s.managerPkg == "" && (bindsName(cu, "manager") || bindsLocalName(cu, "manager"))

	sessions := visitor.Init(&sessionVarScan{scan: s})
	sessions.Visit(cu, nil)

	visitor.Init(&awserrRefScan{scan: s}).Visit(cu, nil)
	s.retryers = s.retryerTypes(cu)
	s.consumeRetryers(cu)
	if len(s.retryerValues) > 0 && (bindsName(cu, "retry") || bindsLocalName(cu, "retry") || bindsName(cu, "ratelimit") || bindsLocalName(cu, "ratelimit")) {
		s.reason = "a retryer whose v2 packages' names the file already binds"
		return s
	}
	s.awserrNames = awserrBoundNames(cu, s)
	if reason := requestFailureReason(cu, s); reason != "" {
		s.reason = reason
		return s
	}
	s.credentialValues = map[string]bool{}

	handlers := visitor.Init(&handlerScan{scan: s})
	handlers.Visit(cu, nil)
	if s.handlerReason != "" {
		s.reason = s.handlerReason
		return s
	}

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
	for name := range s.retryerStatements {
		if !s.droppableConfigVar(name) {
			s.reason = "request.WithRetryer on a config variable that is used beyond making the session"
			return s
		}
	}
	s.builderVars = s.configBuilderLocals(cu)
	s.literalBuilders()
	s.bindsStscreds = s.stscredsPkg == "" && (bindsName(cu, "stscreds") || bindsLocalName(cu, "stscreds"))

	s.optionsValues, s.optionsValuesOK = s.optionsValueNames(cu)
	if s.optionsValuesOK {
		visitor.Init(&optionsLiteralRefScan{scan: s}).Visit(cu, nil)
	}

	// A literal's Config may name a local, which needs the locals classified.
	options := visitor.Init(&sessionOptionsScan{scan: s})
	options.Visit(cu, nil)

	if s.sessionNilCompared {
		s.reason = "a session compared to nil, which the config replacing it cannot be"
		return s
	}

	collect := visitor.Init(&clientVarScan{scan: s})
	collect.Visit(cu, nil)
	s.clientConfigs = s.clientConfigLocals(cu)
	s.requestLocals(cu)
	visitor.Init(&credentialsVarScan{scan: s}).Visit(cu, nil)
	visitor.Init(&credentialValueScan{scan: s}).Visit(cu, nil)
	visitor.Init(&managerFieldScan{scan: s}).Visit(cu, nil)
	visitor.Init(&configVariadicScan{scan: s}).Visit(cu, nil)

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

	// A waiter's options are read off its input's shape, known by now.
	visitor.Init(&waiterOptionScan{scan: s}).Visit(cu, nil)
	check := visitor.Init(&blockerScan{scan: s})
	check.Visit(cu, nil)
	s.migratable = !check.blocked && check.reason == ""
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

// topLevelNames returns the names cu declares at the package level, which every
// file of the package shares.
func topLevelNames(cu *golang.CompilationUnit) map[string]bool {
	names := map[string]bool{}
	for _, rp := range cu.Statements {
		switch d := rp.Element.(type) {
		case *java.VariableDeclarations:
			for _, v := range d.Variables {
				if v.Element != nil && v.Element.Name != nil {
					names[v.Element.Name.Name] = true
				}
			}
		case *java.MethodDeclaration:
			if d.Name != nil {
				names[d.Name.Name] = true
			}
		}
	}
	return names
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

	if reason := v.scan.expressionBlockReason(mi, v.Cursor()); reason != "" {
		v.block(reason)
		return mi
	}
	if reason := v.scan.mockExpectationReason(mi); reason != "" {
		v.block(reason)
		return mi
	}
	if call, ok := qualifiedCall(mi, v.scan.stscredsPkg); ok {
		if _, _, _, isAssume := v.scan.assumeRoleCredentials(mi); !isAssume {
			v.block("stscreds." + call + " other than NewCredentials with a session and a role, which v2 has no counterpart for")
		} else if !v.scan.usesService(stsService) && v.scan.stsNameTaken {
			v.block("stscreds.NewCredentials, whose v2 form needs the sts name the file already binds")
		}
		return mi
	}
	if fa, _, ok := v.scan.configCredentialsCall(mi); fa != nil && !ok && !v.scan.configExpiresAtCall(mi) {
		v.block("Credentials." + name + ", which v2's provider has no counterpart for")
		return mi
	}
	if call, ok := qualifiedCall(mi, v.scan.ec2RoleCredsPkg); ok {
		v.block("ec2rolecreds." + call + ", which v2 has no counterpart for")
		return mi
	}
	if call, ok := qualifiedCall(mi, v.scan.requestPkg); ok {
		if !v.scan.consumedRequestCalls[mi] {
			v.block("request." + call + ": " + requestReason)
		}
		return mi
	}
	if call, ok := qualifiedCall(mi, v.scan.clientPkg); ok {
		v.block("client." + call + ": " + clientReason)
		return mi
	}
	if call, ok := qualifiedCall(mi, v.scan.endpointsPkg); ok {
		if !v.scan.partitionsCall(mi) {
			v.block("endpoints." + call + ": " + endpointsReason)
		}
		return mi
	}
	if reason := v.scan.partitionMethodReason(mi); reason != "" {
		v.block(reason)
		return mi
	}
	if call, ok := qualifiedCall(mi, v.scan.awserrPkg); ok {
		if _, _, _, isNew := v.scan.awserrNew(mi); !isNew && !v.scan.awserrRequestFailure(mi) {
			v.block("awserr." + call + ", which v2 has no counterpart for")
		}
		return mi
	}
	if reason := v.scan.rdsutilsBlockReason(mi); reason != "" {
		v.block(reason)
		return mi
	}

	// The aws package's helpers survive only where v2 kept or renamed them.
	// NewConfig survives as the head of the WithRegion chain a client
	// constructor's guard has already checked.
	if helper, ok := qualifiedCall(mi, v.scan.awsPkg); ok {
		outer, isCall := parentTree(v.Cursor()).(*java.MethodInvocation)
		inChain := isCall && outer.Select != nil && outer.Select.Element == java.Expression(mi) || v.scan.configChainConsumed(v.Cursor())
		if _, unwraps := v.scan.readSeekCloserOperand(mi); unwraps {
			return mi
		}
		if managerRelocations[helper] && v.scan.managerNameTaken {
			v.block("aws." + helper + ", whose v2 package's name the file already binds")
		} else if !awsPackageSurvivors[helper] && valueHelperRenames[helper] == "" && !managerRelocations[helper] && !(helper == "NewConfig" && inChain) {
			v.block("aws." + helper + ", which v2 has no counterpart for")
		}
		return mi
	}

	// v2's config is a plain struct with no fluent setters. Only a chain off
	// aws.NewConfig() handed to a constructor has a rewrite; one on a config
	// held in a variable or a field does not.
	if _, _, isSessionSetter := v.scan.sessionConfigSetter(mi); isSessionSetter && isStatement(v.Cursor()) {
		return mi
	}
	if strings.HasPrefix(name, "With") && mi.Select != nil && matcher.DeclaringTypeFQN(mi) == v1Aws+".Config" {
		if root, ok := chainOnto(mi); ok {
			if _, isBuilder := v.scan.builderVar(root); isBuilder {
				return mi
			}
		}
		if _, chained := mi.Select.Element.(*java.MethodInvocation); !chained {
			v.block("aws.Config." + name + " on a config held in a variable, which v2 has no setter for")
			return mi
		}
	}

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
			(call == "NewStaticCredentialsFromCreds" && len(realArgs(mi)) == 1) ||
			(call == "NewSharedCredentials" && len(realArgs(mi)) == 2) ||
			(call == "NewChainCredentials" && len(realArgs(mi)) == 1)
		if !handled {
			v.block("credentials." + call)
		}
		return mi
	}

	// A call on a service package the manifest cannot account for — a v1-only
	// helper, say — has nowhere to land in v2.
	if recv, isIdent := selectIdentifier(mi); isIdent && !shadowsPackage(recv) {
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

	// v1 shapes validated themselves; v2 validates only inside the operation,
	// and checks the required fields alone, not their lengths. A generated copy
	// of v1's stands in where the rules carry over and the receiver is known
	// to be a pointer or a value.
	if name == "Validate" && mi.Select != nil && len(realArgs(mi)) == 0 {
		if owner, isShape := v.scan.receiverShape(mi.Select.Element); isShape {
			if _, _, ok := v.scan.shapeValidateCall(mi); !ok {
				v.block(owner.shape + ".Validate, which v2 shapes do not have")
			}
			return mi
		}
	}

	// A page iterator, on a client or not. v2 hands out a paginator the caller
	// drives, which the manifest has to name.
	if isPages(name) {
		if _, _, _, _, _, _, ok := v.scan.pagesCall(mi); !ok {
			v.block("paginator " + name)
		}
		return mi
	}

	// A waiter anywhere in the file, on a client or not. v2 makes it a type
	// built from the client, which the manifest has to name.
	if strings.HasPrefix(name, waiterPrefix) && name != waiterPrefix && v.scan.sdkWaiterCall(mi) {
		if _, _, ok := v.scan.waiterType(mi); !ok {
			v.block("waiter " + name)
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

	if v.scan.requestCalls[mi] {
		return mi
	}
	if reason := v.scan.gureguReason(mi); reason != "" {
		v.block(reason)
		return mi
	}
	// Of a held provider's methods only Get has a stand-in.
	if configRead, _, _ := v.scan.configCredentialsCall(mi); configRead == nil && mi.Select != nil && v.scan.credentialsReceiver(mi.Select.Element) && name != "Get" {
		v.block("Credentials." + name + " on a provider the file holds, which v2's has no counterpart for")
		return mi
	}
	// A call on a v1 client the rewrite did not recognise above would keep its
	// v1 shape against a v2 client.
	if mi.Select != nil {
		if service, isClient := v.scan.attributedClient(mi.Select.Element); isClient {
			if _, recognised := v.scan.operationInputService(mi); !recognised {
				v.block("call " + name + " on a " + service + " client the rewrite does not recognise")
			}
			return mi
		}
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
		// Relocated already, the shape may carry the name v2 gave it.
		if target.Name == awsmanifest.TypesAlias(svc) {
			return svc, awsmanifest.V1ShapeName(svc, fa.Name.Element.Name), true
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
	var elem java.Expression
	switch t := enclosing.TypeExpr.(type) {
	case *java.ArrayType:
		elem = t.ElementType
	case *golang.MapType:
		elem = t.Value
	}
	if elem == nil {
		return "", "", false
	}
	ref, isShape := s.declaredShape(elem)
	if !isShape {
		return "", "", false
	}
	return ref.service, ref.shape, true
}

// operationInputService reports whether mi takes a single AWS operation input,
// returning the service that input belongs to. Every v1 operation has this
// shape, whatever it is called on.
func (s *fileScan) operationInputService(mi *java.MethodInvocation) (string, bool) {
	if mi.Name == nil {
		return "", false
	}
	args := realArgs(mi)
	operation := mi.Name.Name
	// The WithContext form takes the context first and the input after it.
	if trimmed := strings.TrimSuffix(operation, "WithContext"); trimmed != operation && trimmed != "" && len(args) == 2 {
		operation, args = trimmed, args[1:]
	}
	if len(args) != 1 {
		return "", false
	}
	// The input is as often held in a variable as written inline, which the
	// shape inference has already named. The name has to be the operation that
	// input belongs to: a call taking a shape is not an operation on it, as
	// `json.Marshal(input)` is not.
	service, shape, isShape := s.inputShape(args[0])
	if !isShape {
		return "", false
	}
	input, _, isOperation := awsmanifest.OperationShapes(service, operation)
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
	if v.scan.attributeComposite(comp, cursor) {
		member, value, ok := attributeMember(comp)
		if !ok {
			v.block("an AttributeValue setting other than one of the S, N, B, BOOL and NULL members")
			return comp
		}
		if _, ok := v.scan.memberValue(member, value, cursor); !ok {
			v.block("an AttributeValue whose " + member + " may be nil, which v1 sent as an empty value the service rejected")
			return comp
		}
		return v.GoVisitor.VisitComposite(comp, p)
	}
	// An aws.Config the session does not consume keeps its settings, so each one
	// has to be a setting v2 still holds on the config.
	if v.scan.awsConfigLiteral(comp) && !consumedAsSessionConfig(v.scan, cursor) {
		for _, rp := range comp.Elements.Elements {
			kv, isKeyValue := rp.Element.(*golang.KeyValue)
			if !isKeyValue {
				continue
			}
			if key, isIdent := kv.Key.(*java.Identifier); isIdent {
				if _, mapped := configTargets[key.Name]; !mapped && key.Name != s3ForcePathStyle && !(key.Name == retryerField && v.scan.retryerValues[kv.Value.Element]) {
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
		// A narrowed pointer converts from a value the file names; a pointer
		// it is handed may be nil, which v2's narrower pointer would lose.
		if hasScalarChange(service, shape, key.Name) && !v.scan.narrowableScalar(kv.Value.Element) {
			v.block("field " + shape + "." + key.Name + ", which v2 narrowed, set from a pointer that may be nil")
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

// A code v1 raised itself has no counterpart for a v2 error to carry.
func (v *blockerScan) VisitBinary(bin *java.Binary, p any) java.J {
	// v2's shapes may not be comparable; only a test against the empty
	// literal has a form.
	if bin.Operator.Element == java.Equal || bin.Operator.Element == java.NotEqual {
		if _, _, isZero := v.scan.zeroComparison(bin); !isZero && (v.scan.shapeValue(bin.Left) || v.scan.shapeValue(bin.Right)) {
			v.block("two shapes compared, which v2's may hold lists Go cannot compare")
		}
	}
	if bin.Operator.Element == java.Equal || bin.Operator.Element == java.NotEqual {
		for _, pair := range [][2]java.Expression{{bin.Left, bin.Right}, {bin.Right, bin.Left}} {
			if code, isClient := clientSideCode(pair[1]); isClient && v.scan.awserrCode(pair[0]) {
				if code == noCredentialProviders {
					v.scan.noCredentialsChecked = true
					continue
				}
				v.block("a comparison against " + code + ", a code v1 raised itself and v2 reports as an ordinary error")
			}
		}
	}
	return v.GoVisitor.VisitBinary(bin, p)
}

func (v *blockerScan) VisitSwitch(sw *java.Switch, p any) java.J {
	if sw.Selector != nil && v.scan.awserrCode(sw.Selector.Tree.Element) {
		visitor.Init(&clientCodeCaseScan{block: v.block, scan: v.scan}).Visit(sw.Body, nil)
	}
	return v.GoVisitor.VisitSwitch(sw, p)
}

// clientCodeCaseScan holds the file back on a case naming a client-side code.
type clientCodeCaseScan struct {
	visitor.GoVisitor
	block func(string)
	scan  *fileScan
}

func (v *clientCodeCaseScan) VisitLiteral(lit *java.Literal, p any) java.J {
	if code, isClient := clientSideCode(lit); isClient {
		if code == noCredentialProviders {
			v.scan.noCredentialsChecked = true
			return lit
		}
		v.block("a case on " + code + ", a code v1 raised itself and v2 reports as an ordinary error")
	}
	return lit
}

// A write into a list or map v2 holds by value has to land in v2's own.
func (v *blockerScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if name, aliased := v.scan.appendAliased(a, v.Cursor()); aliased {
		v.block("a pointer " + name + " appended to a list v2 holds by value and used after, which v2's copy would not see")
	}
	if _, isWrite, ok := v.scan.valueElementWrite(a); isWrite && !ok {
		v.block("a write into a list or map v2 holds by value, of a pointer that may be nil")
	}
	if v.scan.valueElementDerefWrite(a.Variable) {
		v.block("a write through a pointer to an element of a list or map v2 holds by value")
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

func (v *blockerScan) VisitAssignmentOperation(a *java.AssignmentOperation, p any) java.J {
	if v.scan.valueElementDerefWrite(a.Variable) {
		v.block("a write through a pointer to an element of a list or map v2 holds by value")
	}
	return v.GoVisitor.VisitAssignmentOperation(a, p)
}

// configChainConsumed reports whether the NewConfig chain the cursor heads is
// turned into loader options, leaving nothing of it behind: a session
// constructor's argument, a session's options Config, or the value of a local
// the load folds in.
func (s *fileScan) configChainConsumed(cursor *visitor.Cursor) bool {
	c := cursor
	for {
		parent := c.Parent()
		for parent != nil && parent.Value() == nil {
			parent = parent.Parent()
		}
		if parent == nil {
			return false
		}
		call, isCall := parent.Value().(*java.MethodInvocation)
		if !isCall {
			break
		}
		// request.WithRetryer takes the chain as its first argument.
		if name, isRequest := qualifiedCall(call, s.requestPkg); isRequest && name == "WithRetryer" {
			if args := realArgs(call); len(args) == 2 && java.Tree(args[0]) == c.Value() {
				c = parent
				continue
			}
		}
		if call.Select == nil || java.Tree(call.Select.Element) != c.Value() {
			break
		}
		c = parent
	}
	expr, isExpr := c.Value().(java.Expression)
	if !isExpr {
		return false
	}
	if _, ok := s.configChainOptions(expr); !ok {
		return false
	}
	switch p := parentTree(c).(type) {
	case *golang.Unary:
		kv, isKeyValue := parentTree(cursorOf(c, p).Parent()).(*golang.KeyValue)
		if !isKeyValue {
			return false
		}
		key, isIdent := kv.Key.(*java.Identifier)
		return isIdent && key.Name == sessionConfig
	case *java.MethodInvocation:
		call, isSession := qualifiedCall(p, s.sessionPkg)
		return isSession && (call == "NewSession" || call == "New")
	case *java.Assignment:
		target, isIdent := p.Variable.(*java.Identifier)
		return isIdent && (s.droppableConfigVar(target.Name) || s.builderVars[target.Name])
	}
	return false
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
	if v.scan.allReasons != nil {
		// The sweep wants every construct, so the walk carries on.
		v.scan.allReasons[reason] = true
		if v.reason == "" {
			v.reason = reason
		}
		return
	}
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
		// Any other value the v1 field took was a *string, which v2 reads
		// through aws.ToString, nil becoming the empty enum v2 leaves unset.
		return true
	}
	fa, ok := inner.(*java.FieldAccess)
	if !ok || fa.Name.Element == nil {
		// Computed at runtime, which converts cleanly.
		return true
	}
	// Of this enum, or of another of the service's, whose string a conversion
	// carries as v1 sent it.
	return true
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
		if _, isPointer := parentTree(v.Cursor()).(*golang.PointerType); isPointer && name == sessionOptions && v.scan.optionsValuesOK {
			return fa
		}
		if name != "Session" && !(sessionOptionNames[name] && !v.scan.sessionOptionsBlocked && v.scan.consumedOptionRefs[fa]) {
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
		// The provider interface is v2's, and a provider literal one the chain
		// has a stand-in for.
		comp, isLiteral := parentTree(v.Cursor()).(*golang.Composite)
		provider := isLiteral && comp.TypeExpr == java.Expression(fa)
		if provider {
			_, provider = v.scan.credentialProviderLiteral(comp)
		}
		_, inPointer := parentTree(v.Cursor()).(*golang.PointerType)
		if name != "Provider" && !provider && name != "Value" && !(name == "Credentials" && inPointer) {
			v.block("credentials." + name + " as a type")
		}
		return fa
	}
	if name, ok := qualifiedRef(fa, v.scan.ec2RoleCredsPkg); ok {
		comp, isLiteral := parentTree(v.Cursor()).(*golang.Composite)
		if !isLiteral || comp.TypeExpr != java.Expression(fa) {
			v.block("ec2rolecreds." + name + " other than an EC2RoleProvider literal")
		} else if _, ok := v.scan.ec2RoleProviderClient(comp); !ok {
			v.block("ec2rolecreds.EC2RoleProvider setting fields v2's provider has none for")
		}
		return fa
	}

	if name, ok := qualifiedRef(fa, v.scan.awserrPkg); ok {
		// Only the error interface has a smithy counterpart the rewrite emits;
		// the constructors and batched shapes have none.
		if !awserrSupported[name] {
			v.block("awserr." + name)
		} else if !v.scan.consumedAwserrRefs[fa] {
			v.block("awserr." + name + " other than asserted to or declared, which v2 has no counterpart for")
		}
		return fa
	}

	if field, calls := v.scan.managerClientCall(fa, v.Cursor()); calls && !v.scan.managerFieldsUnproven[field] {
		// Proven within this file; the rest of the package is checked before
		// anything is edited.
		v.scan.managerClientReads[field] = true
		return fa
	}
	if fa.Name.Element != nil && v.scan.managerPkg != "" && v.scan.managerValue(fa.Target) && !managerFields[fa.Name.Element.Name] {
		v.block("s3manager field " + fa.Name.Element.Name + ", which v2 holds as a different type")
		return fa
	}

	// Only the regional STS setting a NewConfig chain asks for carries over, by
	// going away; the resolvers and the partition data have no v2 counterpart.
	if name, ok := qualifiedRef(fa, v.scan.endpointsPkg); ok {
		outer, isCall := parentTree(v.Cursor()).(*java.MethodInvocation)
		regional := name == "RegionalSTSEndpoint" && isCall && outer.Name != nil && outer.Name.Name == "WithSTSRegionalEndpoint"
		if !regional && partitionTypes[name] == "" {
			v.block("endpoints." + name + ": " + endpointsReason)
		}
		return fa
	}
	if name, ok := qualifiedRef(fa, v.scan.clientPkg); ok {
		if name != configProvider && !v.scan.consumedClientRefs[fa] {
			v.block("client." + name + ": " + clientReason)
		}
		return fa
	}
	if _, isBuilder := v.scan.literalBuilderField(fa); isBuilder {
		return fa
	}
	// A config local stays v2's config, which holds only the settings it kept.
	if fa.Name.Element != nil {
		holder, isIdent := fa.Target.(*java.Identifier)
		local := isIdent && v.scan.configLocals[holder.Name] != nil
		if local || v.scan.configValue(fa.Target) {
			field := fa.Name.Element.Name
			target, mapped := configTargets[field]
			switch {
			case field == s3ForcePathStyle && !assignedTo(v.Cursor()):
				v.block("aws.Config.S3ForcePathStyle read, which v2 holds on each S3 client")
				return fa
			case !mapped && field != s3ForcePathStyle:
				v.block("aws.Config." + field + " on a config the file keeps, which v2 configures elsewhere")
				return fa
			case mapped && target.depointered && !local && !dereferenced(v.Cursor()) && !assignedTo(v.Cursor()):
				v.block("aws.Config." + field + " read as the pointer v1 held it in")
				return fa
			}
		}
		// A provider name read off anything but credentials has nothing to become.
		if fa.Name.Element.Name == providerNameField && !v.scan.providerNameRead(fa) {
			v.block("ProviderName read off something not known to be credentials")
			return fa
		}
	}
	if name, ok := qualifiedRef(fa, v.scan.stscredsPkg); ok {
		if name != "StdinTokenProvider" && !v.scan.consumedAssumeRoleRefs[fa] {
			v.block("stscreds." + name + ", which v2 holds in a different shape")
		}
		return fa
	}
	if name, ok := qualifiedRef(fa, v.scan.requestPkg); ok {
		if !v.scan.consumedRequestRefs[fa] {
			v.block("request." + name + ": " + requestReason)
		}
		return fa
	}
	// The handlers a session carried are rewritten where they are added; any
	// other reach for them, or for a client's, has nothing to land on.
	if holder, isIdent := fa.Target.(*java.Identifier); isIdent && fa.Name.Element != nil && fa.Name.Element.Name == handlersField &&
		(v.scan.sessionVars[holder.Name] || v.scan.clients[holder.Name] != "") && !v.scan.consumedHandlerRefs[fa] {
		v.block("request handlers reached other than by adding one to a session: " + requestReason)
		return fa
	}

	if name, ok := qualifiedRef(fa, v.scan.restPkg); ok && name != "EscapePath" {
		v.block("rest." + name + ", which v2 keeps internal")
		return fa
	}
	if name, ok := qualifiedRef(fa, v.scan.rdsutilsPkg); ok {
		v.block("rdsutils." + name + " " + rdsConnectionBuilderReason)
		return fa
	}

	if name, ok := qualifiedRef(fa, v.scan.awsPkg); ok {
		// aws.Config survives only as the session argument this recipe unpacks,
		// which it has already consumed by the time a bare reference is seen.
		if name == "ErrMissingEndpoint" {
			// A stand-in carries the code and message, but is not the sentinel a
			// comparison would test for.
			if bin, isBinary := parentTree(v.Cursor()).(*java.Binary); isBinary && (bin.Operator.Element == java.Equal || bin.Operator.Element == java.NotEqual) {
				v.block("aws.ErrMissingEndpoint compared to, which v2 has no sentinel for")
			}
			return fa
		}
		if name == errMissingRegion {
			// Only a comparison against the sentinel has a v2 form; a reference
			// the rewrite would leave behind does not.
			if v.scan.missingRegionBlocked {
				v.block("aws." + errMissingRegion)
			}
			return fa
		}
		if managerRelocations[name] && v.scan.managerNameTaken {
			v.block("aws." + name + ", whose v2 package's name the file already binds")
		} else if !awsPackageSurvivors[name] && valueHelperRenames[name] == "" && !managerRelocations[name] && name != "Config" && name != awsContextType {
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
		if service == dynamodbService && name == attributeValue {
			if reason := v.scan.attributeValueBlockReason(fa, v.Cursor()); reason != "" {
				v.block(reason)
			}
		} else if awsmanifest.IsUnion(service, name) {
			comp, isLiteral := parentTree(v.Cursor()).(*golang.Composite)
			// A pointer to the shape names v2's interface, which takes none.
			if _, inPointer := parentTree(v.Cursor()).(*golang.PointerType); inPointer {
				return fa
			}
			if !isLiteral || comp.TypeExpr != java.Expression(fa) {
				v.block("union shape " + service + "." + name)
			} else if _, _, _, ok := v.scan.unionLiteral(comp); !ok {
				v.block("union shape " + service + "." + name + " set to other than one member's literal")
			}
		}
		return fa
	}
	// A union's value reaches a file through an output without its type ever
	// being named, so a read of one of its fields holds the file back too.
	if fa.Name.Element != nil {
		if v.scan.attributeRead(fa) {
			if assignedTo(v.Cursor()) || addressTaken(v.Cursor()) {
				v.block("AttributeValue member ." + fa.Name.Element.Name + " written in place, which v2's member types do not allow")
			}
			return fa
		}
		// A receiver typed as something outside the SDK is not a union, whatever
		// its fields are called: in4it/ecs-deploy's own struct has an S.
		if fqn := matcher.GetFullyQualifiedName(matcher.TypeOfExpression(fa.Target)); fqn != "" && !strings.HasPrefix(fqn, v1Module) {
			return fa
		}
		for _, service := range v.scan.services {
			if awsmanifest.IsUnionVariant(service, fa.Name.Element.Name) {
				v.block("union member read ." + fa.Name.Element.Name + " on a " + service + " value")
				break
			}
		}
	}
	return fa
}

// shapeSetter reports whether mi is one of v1's fluent setters on a shape the
// scan resolved: `input.SetNextToken(tok)`.
func (s *fileScan) shapeSetter(mi *java.MethodInvocation) bool {
	if mi.Name == nil || !strings.HasPrefix(mi.Name.Name, "Set") || mi.Name.Name == "Set" {
		return false
	}
	if mi.Select == nil {
		return false
	}
	_, isShape := s.receiverShape(mi.Select.Element)
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

// addressTaken reports whether the node the cursor points at is the operand of
// an `&`.
func addressTaken(cursor *visitor.Cursor) bool {
	unary, isUnary := parentTree(cursor).(*golang.Unary)
	return isUnary && unary.Operator.Element == golang.AddressOf
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
			v.scan.noteShapeAddress(name.Name, addressOfShapeExpr(a.Value.Element))
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
				v.scan.noteShapeAddress(name.Name, addressOfShapeExpr(ma.Values[0].Element))
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
		_, isPointer := vd.TypeExpr.(*golang.PointerType)
		for _, decl := range vd.Variables {
			if d := decl.Element; d != nil && d.Name != nil {
				v.scan.shapes[d.Name.Name] = ref
				if isPointer {
					v.scan.noteShapeAddress(d.Name.Name, shapePointer)
				} else {
					v.scan.noteShapeAddress(d.Name.Name, shapeValue)
				}
			}
		}
	}
	return vd
}

func (v *shapeScan) VisitForEachLoop(loop *java.ForEachLoop, p any) java.J {
	if name, ref, ok := v.scan.rangeElementShape(loop); ok {
		v.scan.shapes[name] = ref
		// v2 holds the list by value, so the element ranged is a copy.
		v.scan.noteShapeAddress(name, shapeValue)
	}
	return v.GoVisitor.VisitForEachLoop(loop, p)
}

// receiverShape names the shape an expression holds: a name the scan bound, or a
// read of a field that holds another shape, which is what carries the inference
// through `i.State.Name`.
func (s *fileScan) receiverShape(expr java.Expression) (shapeRef, bool) {
	switch target := expr.(type) {
	case *java.Identifier:
		// The parse-time type is the value's own, where the scan's inference is
		// keyed by name: dregsy binds output to two services' outputs in two
		// branches of one function.
		if owner, typed := s.attributedShape(target); typed && awsmanifest.Known(owner.service) {
			return owner, true
		}
		if owner, known := s.shapes[target.Name]; known {
			return owner, true
		}
		return shapeRef{}, false
	case *java.FieldAccess:
		if target.Name.Element == nil {
			return shapeRef{}, false
		}
		outer, known := s.receiverShape(target.Target)
		if !known {
			// A shape held in a field of the program's own struct is named by
			// the field's parse-time type.
			return s.attributedShape(target)
		}
		elem := awsmanifest.FieldShape(outer.service, outer.shape, target.Name.Element.Name)
		if elem == "" {
			return shapeRef{}, false
		}
		return shapeRef{service: outer.service, shape: elem}, true
	case *java.ArrayAccess:
		// An element of a list of shapes, which the editor may already have
		// wrapped to hand back v1's pointers.
		indexed := target.Indexed
		if mi, wrapped := indexed.(*java.MethodInvocation); wrapped && mi.Name != nil && mi.Name.Name == compatPtrs {
			if args := realArgs(mi); len(args) == 1 {
				indexed = args[0]
			}
		}
		switch list := indexed.(type) {
		case *java.Identifier:
			elem, held := s.sliceShapes[list.Name]
			return elem, held
		case *java.FieldAccess:
			owner, known := s.receiverShape(list.Target)
			if !known || list.Name.Element == nil {
				return shapeRef{}, false
			}
			field := list.Name.Element.Name
			elem := awsmanifest.ValueSliceElement(owner.service, owner.shape, field)
			if elem == "" || elem == "-" {
				elem = awsmanifest.FieldShape(owner.service, owner.shape, field)
			}
			if elem != "" && elem != "-" {
				return shapeRef{service: owner.service, shape: elem}, true
			}
		}
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
		service := strings.TrimSuffix(pkg, "/types")
		if prefix == v2ServicePkg {
			service = awsmanifest.V1Name(service)
		}
		return shapeRef{service: service, shape: name}, true
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
		if value, ok := newOfValue(e); ok {
			return value, true
		}
	case *golang.Unary:
		if e.Operator.Element == golang.AddressOf {
			return e.Expression, true
		}
	}
	return nil, false
}

// newOfValue reads Go 1.26's new(v), which points to a copy of a value as
// aws.String does, as opposed to new(T) of a type.
func newOfValue(mi *java.MethodInvocation) (java.Expression, bool) {
	if mi.Select != nil || mi.Name == nil || mi.Name.Name != "new" || len(realArgs(mi)) != 1 {
		return nil, false
	}
	switch arg := realArgs(mi)[0].(type) {
	case *java.Literal, *java.Binary, *golang.Unary, *java.Unary, *java.MethodInvocation:
		return arg, true
	case *java.Identifier:
		// A variable carries its own type; a type's name carries none.
		if arg.Name == "true" || arg.Name == "false" || arg.FieldType != nil {
			return arg, true
		}
	case *java.FieldAccess:
		if arg.Name.Element != nil && arg.Name.Element.FieldType != nil {
			return arg, true
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

// narrowedFieldOf names the v1 pointee type of a depointered field v2 also
// narrowed, or "".
func (s *fileScan) narrowedFieldOf(fa *java.FieldAccess) string {
	if fa.Name.Element == nil {
		return ""
	}
	owner, known := s.receiverShape(fa.Target)
	if !known {
		return ""
	}
	return awsmanifest.NarrowedFrom(owner.service, owner.shape, fa.Name.Element.Name)
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
	// A receiver whose shape is known settles the field: in4it/ecs-deploy's
	// ECS Task has a Group that is no enum, as EC2's permissions' are.
	if _, known := s.receiverShape(fa.Target); known {
		return false
	}
	if fqn := matcher.GetFullyQualifiedName(matcher.TypeOfExpression(fa.Target)); fqn != "" && !strings.HasPrefix(fqn, v1Module) {
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
	case *java.FieldAccess:
		// A client held in a struct field: what attribution says it holds, or
		// the field's declared type.
		if service, isClient := s.attributedClient(recv); isClient {
			return service, true
		}
		if recv.Name.Element != nil {
			service, isClient := s.clients[recv.Name.Element.Name]
			return service, isClient
		}
	case *java.MethodInvocation:
		return s.clientConstructorService(recv)
	}
	return "", false
}

// attributedClient reads a v1 service client off an expression's parse-time
// type.
func (s *fileScan) attributedClient(expr java.Expression) (string, bool) {
	fqn := matcher.GetFullyQualifiedName(matcher.TypeOfExpression(expr))
	rest, under := strings.CutPrefix(fqn, v1ServicePkg)
	if !under {
		return "", false
	}
	service, name, named := strings.Cut(rest, ".")
	if !named || strings.Contains(service, "/") || awsmanifest.Place(service, name) != awsmanifest.IsClient {
		return "", false
	}
	return service, true
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
	// A local the declaration takes the load over for is already the config, and
	// one that stays is read back as loader options.
	if _, isBuilder := s.builderVar(args[0]); isBuilder {
		return true
	}
	_, restructured := s.restructuredConfigVar(args[0])
	return restructured || s.keptConfigVar(args[0])
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
	if !isVar || s.configRestructured[name] || s.configKept[name] {
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

// isStatement reports whether the cursor's node stands as a statement of a
// block, its value discarded.
func isStatement(cursor *visitor.Cursor) bool {
	_, inBlock := parentTree(cursor).(*java.Block)
	return inBlock
}
