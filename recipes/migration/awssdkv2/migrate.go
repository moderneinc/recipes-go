/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsexportdata"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/format"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var (
	awsCtx    = template.Expr("awsCtx")
	awsRegion = template.Expr("awsRegion")

	loadConfigTemplate = template.ExpressionTemplate(
		fmt.Sprintf(`config.LoadDefaultConfig(%s)`, awsCtx),
	).Captures(awsCtx).Imports(v2Config).ExportData(awsexportdata.FS).Build()

	loadConfigWithRegionTemplate = template.ExpressionTemplate(
		fmt.Sprintf(`config.LoadDefaultConfig(%s, config.WithRegion(%s))`, awsCtx, awsRegion),
	).Captures(awsCtx, awsRegion).Imports(v2Config).ExportData(awsexportdata.FS).Build()

	// v1 took no context anywhere v2 takes one, so a function that has none to
	// give gets the placeholder the standard library provides for exactly this.
	todoContextTemplate = template.ExpressionTemplate(`context.TODO()`).Imports(contextPkg).Build()
)

// AwsSdkGoV1BlockerRow is one row of the table naming the files the migration
// left alone.
type AwsSdkGoV1BlockerRow struct {
	SourcePath string
	Reason     string
}

var awsBlockersTable = recipe.NewDataTable[AwsSdkGoV1BlockerRow](
	"org.openrewrite.golang.migration.MigrateAwsSdkGoToV2$Blockers",
	"Files left on aws-sdk-go v1",
	"The files holding a v1 construct with no faithful v2 form, and what held each back. Their callers may have migrated around them, so a module is not expected to build until these are seen to by hand.",
	[]recipe.ColumnDescriptor{
		{Name: "sourcePath", DisplayName: "Source path", Description: "The file left on v1.", Type: "String"},
		{Name: "reason", DisplayName: "Reason", Description: "The construct with no faithful v2 form.", Type: "String"},
	},
)

// Migrates a file from aws-sdk-go v1 to aws-sdk-go-v2.
type MigrateAwsSdkGoToV2 struct {
	recipe.ScanningBase
}

// migrateAcc carries what the edit phase needs that one file cannot answer: the
// module's Go release, and which packages need the slice helpers written into
// them.
type migrateAcc struct {
	moduleAcc
	// compatPackages maps a directory to the package declared in it.
	compatPackages map[string]string
	// credentialsPackages does the same for the credentials chain stand-ins.
	credentialsPackages map[string]string
	// pathStylePackages does the same for S3's path-style service option.
	pathStylePackages map[string]string
	// partitionPackages does the same for v1's partition data.
	partitionPackages map[string]string
	// sslPackages does the same for the DisableSSL stand-ins, and serviceDirs
	// the directories building clients of each service, whose resolvers the
	// stand-in wraps.
	sslPackages map[string]string
	serviceDirs map[string]map[string]bool
	// noCredentialsPackages does the same for the NoCredentialProviders match.
	noCredentialsPackages map[string]string
	// builderPackages does the same for the config builder, and
	// expiresAtPackages for the credentials expiry stand-in.
	builderPackages          map[string]string
	expiresAtPackages        map[string]string
	loadOptionsPackages      map[string]string
	requestPackages          map[string]string
	credentialsValuePackages map[string]string
	enumListPackages         map[string]string
	zeroPackages             map[string]string
	pagePackages             map[string]string
	// dirImports maps a package to the import paths its files name, and sdkAPI
	// marks the packages whose exported API names a v1 type.
	dirImports map[string]map[string]bool
	sdkAPI     map[string]bool
	// adjuncts maps a file naming no SDK package that the rewrite changes, for
	// the SDK values it reads, to the module's packages it imports.
	adjuncts map[string][]string
	// embeddedClients maps a directory to its structs embedding a v1 client,
	// and each to the field's v1 name.
	embeddedClients map[string]map[string]string
	// prettifyPackages does the same for the shape printer, and
	// prettifyServices the services whose shapes each package prints.
	prettifyPackages map[string]string
	prettifyServices map[string]map[string]bool
	// validatePackages does the same for the shape validators, and
	// validateShapes the `service.Shape` each package validates.
	validatePackages map[string]string
	validateShapes   map[string]map[string]bool
	// errorPackages does the same for the awserr.New stand-in.
	errorPackages map[string]string
	// attributePackages does the same for the AttributeValue getters.
	attributePackages map[string]string
	// handlerPackages does the same for the request-handler adapter.
	handlerPackages map[string]string
	// configPackages does the same for the session.New replacement, which is a
	// separate file because it carries imports the slice helpers do not.
	configPackages map[string]string
	// waitPackages does the same for the waiter bound.
	waitPackages map[string]string
	// imdsPackages does the same for the metadata client's Region.
	imdsPackages map[string]string
	// helperPackages does the same for the v1 service functions v2 dropped.
	helperPackages map[string]string
	// ifaceServices are the services whose v1 iface package the module imports
	// and the migration regenerates.
	ifaceServices map[string][]string
	// attributePortUsers are the directories importing dynamodbattribute.
	attributePortUsers []string
	// modulePath is the go.mod `module` path, which the regenerated iface
	// packages are imported through.
	modulePath string
	// usage is what the go.mod requires are decided from, gathered here rather
	// than by a second scanning recipe — the Moderne CLI hangs on a recipe list
	// holding more than one.
	usage awsUsageAcc
	// deferred are the files whose probe waits for the go.mod, since one
	// importing an iface package migrates only once the module path is known.
	deferred []*golang.CompilationUnit
	// blockers names the files the migration left alone, and what held each
	// back. A client, a shape and an interface over them cross file boundaries,
	// so the callers that did migrate may no longer agree with them — which is a
	// type error the compiler points at, and this table is the list to work
	// through. Holding the whole module back instead would mean one file's
	// unmigratable instrumentation costing every other file its migration.
	blockers map[string]string
	// heldBack maps a directory to the first file in it left on v1. A package
	// compiles as a unit, so the rest of it stays on v1 too rather than calling
	// across a boundary that no longer type-checks.
	heldBack map[string]string
	// ifaceFiles are the files importing an iface package, which stay on v1
	// when the go.mod names no module path.
	ifaceFiles []string
	// ifaceMocks are the MockGen mocks of iface interfaces, by path, which are
	// rendered afresh against the regenerated interface.
	ifaceMocks map[string]ifaceMock
	// testifyMocks are mockery's mocks of iface interfaces, by path, and
	// testifyCalls the calls each file makes naming a method of a testify mock.
	testifyMocks map[string]testifyMock
	testifyCalls map[string][]testifyCall
	// blockedTests maps a test file the rewrite cannot carry over to why, until
	// the package it tests is known to migrate; disabledTests are those
	// commented out because it did. fileDecls and fileIdents are each file's
	// top-level names and the names it uses, which decide whether commenting a
	// test out leaves another file without a helper.
	blockedTests  map[string]string
	disabledTests map[string]string
	fileDecls     map[string]map[string]bool
	fileIdents    map[string]map[string]bool
	// packageNames maps a package's directory to the names its files declare at
	// the top level, which an import added to any one of them would collide
	// with.
	packageNames map[string]map[string]bool
	// managerReaders maps a package's directory and a manager field to the
	// files calling through its S3, and managerUnproven to the fields some file
	// there sets from something not proven to hold S3's own client.
	managerReaders  map[string][]string
	managerUnproven map[string]bool
}

// holdBack records that file stays on v1, and with it the package it is in.
func (a *migrateAcc) holdBack(file, reason string) {
	a.blockerReason(file, reason)
	a.usage.held[path.Dir(file)] = true
	if _, held := a.heldBack[path.Dir(file)]; !held {
		a.heldBack[path.Dir(file)] = file
	}
}

// heldBackBy names the file keeping cu's package on v1, when it is another one.
func (a *migrateAcc) heldBackBy(cu *golang.CompilationUnit) (string, bool) {
	by, held := a.heldBack[path.Dir(cu.SourcePath)]
	return by, held && by != cu.SourcePath
}

// blockerReason records why a file was left alone, keeping the first answer.
func (a *migrateAcc) blockerReason(path, reason string) {
	if _, seen := a.blockers[path]; seen {
		return
	}
	if reason == "" {
		reason = "a v1 construct with no faithful v2 form"
	}
	a.blockers[path] = reason
}

// ifaceUnreachable reports whether the generated iface packages have nowhere to
// live. They are imported through the module's own path, which the go.mod names;
// without one the files importing them stay on v1. Derived rather than set
// during the scan, since a go.mod may be visited after them.
func (a *migrateAcc) ifaceUnreachable() bool {
	return (len(a.ifaceServices) > 0 || len(a.attributePortUsers) > 0) && a.modulePath == ""
}

func (r *MigrateAwsSdkGoToV2) InitialValue(*recipe.ExecutionContext) any {
	return &migrateAcc{
		blockers:                 map[string]string{},
		heldBack:                 map[string]string{},
		compatPackages:           map[string]string{},
		configPackages:           map[string]string{},
		waitPackages:             map[string]string{},
		imdsPackages:             map[string]string{},
		helperPackages:           map[string]string{},
		handlerPackages:          map[string]string{},
		attributePackages:        map[string]string{},
		errorPackages:            map[string]string{},
		partitionPackages:        map[string]string{},
		prettifyPackages:         map[string]string{},
		validatePackages:         map[string]string{},
		validateShapes:           map[string]map[string]bool{},
		sslPackages:              map[string]string{},
		noCredentialsPackages:    map[string]string{},
		builderPackages:          map[string]string{},
		expiresAtPackages:        map[string]string{},
		loadOptionsPackages:      map[string]string{},
		requestPackages:          map[string]string{},
		credentialsValuePackages: map[string]string{},
		enumListPackages:         map[string]string{},
		zeroPackages:             map[string]string{},
		pagePackages:             map[string]string{},
		embeddedClients:          map[string]map[string]string{},
		adjuncts:                 map[string][]string{},
		dirImports:               map[string]map[string]bool{},
		sdkAPI:                   map[string]bool{},
		serviceDirs:              map[string]map[string]bool{},
		ifaceMocks:               map[string]ifaceMock{},
		testifyMocks:             map[string]testifyMock{},
		testifyCalls:             map[string][]testifyCall{},
		prettifyServices:         map[string]map[string]bool{},
		pathStylePackages:        map[string]string{},
		credentialsPackages:      map[string]string{},
		ifaceServices:            map[string][]string{},
		managerReaders:           map[string][]string{},
		packageNames:             map[string]map[string]bool{},
		blockedTests:             map[string]string{},
		disabledTests:            map[string]string{},
		fileDecls:                map[string]map[string]bool{},
		fileIdents:               map[string]map[string]bool{},
		managerUnproven:          map[string]bool{},
		usage:                    newAwsUsageAcc(),
	}
}

func (r *MigrateAwsSdkGoToV2) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&compatScanner{acc: acc.(*migrateAcc)})
}

func (r *MigrateAwsSdkGoToV2) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&migrateVisitor{acc: acc.(*migrateAcc), editing: true})
}

// Generate writes the slice helpers once into each package that needs them.
func (r *MigrateAwsSdkGoToV2) Generate(acc any, ctx *recipe.ExecutionContext) []java.Tree {
	a := acc.(*migrateAcc)
	for _, cu := range a.deferred {
		a.probe(cu)
	}
	a.deferred = nil
	if a.ifaceUnreachable() {
		for _, file := range a.ifaceFiles {
			a.holdBack(file, ifaceUnreachableReason)
		}
	}
	a.decideTestifyCalls()
	a.decideBlockedTests()
	// Holding back across one boundary can strand an adjunct, and the
	// reverse.
	for before := -1; before != len(a.heldBack); {
		before = len(a.heldBack)
		a.holdBackAcrossBoundaries()
		a.holdBackStrandedAdjuncts()
	}
	for key, readers := range a.managerReaders {
		if a.managerUnproven[key] {
			for _, file := range readers {
				a.holdBack(file, managerFieldReason)
			}
		}
	}
	// A package held back on v1 calls none of the helpers.
	for _, packages := range []map[string]string{a.partitionPackages, a.prettifyPackages, a.validatePackages, a.sslPackages, a.noCredentialsPackages, a.builderPackages, a.expiresAtPackages, a.loadOptionsPackages, a.requestPackages, a.credentialsValuePackages, a.enumListPackages, a.zeroPackages, a.pagePackages, a.credentialsPackages, a.pathStylePackages, a.errorPackages, a.attributePackages, a.handlerPackages, a.helperPackages, a.imdsPackages, a.waitPackages, a.configPackages, a.compatPackages} {
		for dir := range packages {
			if _, held := a.heldBack[strings.TrimSuffix(dir, externalTestUnit)]; held {
				delete(packages, dir)
			}
		}
	}
	for _, path := range sortedKeys(a.blockers) {
		awsBlockersTable.InsertRow(ctx, AwsSdkGoV1BlockerRow{SourcePath: path, Reason: a.blockers[path]})
	}
	var out []java.Tree
	// v2 deleted the iface packages outright, so a replacement is written under
	// the v2 path its importers now name. Their importers stay on v1 when there
	// is no module path, but the files around them still need their helpers.
	for service, importers := range a.ifaceServices {
		if a.ifaceUnreachable() {
			break
		}
		// Only a migrating importer names the regenerated package; one held
		// back keeps importing v1's, and nothing would require v2's service.
		if !slices.ContainsFunc(importers, func(dir string) bool { _, held := a.heldBack[dir]; return !held }) {
			continue
		}
		source, ok := ifaceSource(service)
		if !ok {
			continue
		}
		cu, err := parser.NewGoParser().Parse(path.Join(ifaceDir(service), ifaceFileName), source)
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	// Only a migrating importer names the port.
	if !a.ifaceUnreachable() && slices.ContainsFunc(a.attributePortUsers, func(dir string) bool { _, held := a.heldBack[dir]; return !held }) {
		out = append(out, attributePortFiles()...)
	}
	for dir, pkg := range a.pagePackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, pageReplayFile), fmt.Sprintf(pageReplayHelperSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.zeroPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, zeroFile), fmt.Sprintf(zeroSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.enumListPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, enumListFile), fmt.Sprintf(enumListSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.credentialsValuePackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, credentialsValueFile), fmt.Sprintf(credentialsValueSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.requestPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, requestFile), fmt.Sprintf(requestSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.loadOptionsPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, loadOptionsFile), fmt.Sprintf(loadOptionsSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.expiresAtPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, expiresAtFile), fmt.Sprintf(expiresAtSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.builderPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, builderFile), fmt.Sprintf(builderSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.noCredentialsPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, noCredentialsFile), fmt.Sprintf(noCredentialsSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.sslPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, disableSSLFile), disableSSLSource(pkg, a.migratedServices()))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.prettifyPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, prettifyFile), prettifySource(pkg, a.prettifyServices[dir]))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.validatePackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, validateFile), validateSource(pkg, a.validateShapes[dir]))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.partitionPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, partitionsHelperFile), partitionsSource(pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.credentialsPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, credentialsChainFile), fmt.Sprintf(credentialsChainSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.pathStylePackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, pathStyleHelperFile), fmt.Sprintf(pathStyleHelperSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.errorPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, errorHelperFile), fmt.Sprintf(errorHelperSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.attributePackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, attributeValueFile), fmt.Sprintf(attributeValueSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.handlerPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, handlerFile), handlerFileSource(pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.helperPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, serviceHelperFileName), fmt.Sprintf(serviceHelperSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.imdsPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, imdsFileName), fmt.Sprintf(imdsSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.waitPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, waitFileName), fmt.Sprintf(waitSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.configPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, configFileName), fmt.Sprintf(configHelperSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.compatPackages {
		cu, err := parser.NewGoParser().Parse(helperPath(dir, compatFileName), fmt.Sprintf(compatSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	return out
}

// compatScanner asks of each file whether migrating it would emit a slice
// conversion, by running the edit and looking. The helpers are unexported, so
// one definition per package is what the generated file provides.
type compatScanner struct {
	visitor.GoVisitor
	acc *migrateAcc
}

func (v *compatScanner) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	recordBuildList(&v.acc.usage, gm)
	return v.GoVisitor.VisitGoMod(gm, p)
}

func (v *compatScanner) VisitGoModDirective(d *golang.GoModDirective, p any) java.Tree {
	if len(d.Values) == 1 {
		switch d.Keyword {
		case "go":
			v.acc.goVersion = d.Values[0].Text
		case "module":
			v.acc.modulePath = d.Values[0].Text
		}
	}
	return d
}

func (v *compatScanner) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	recordAwsUsage(&v.acc.usage, cu)
	dir := path.Dir(cu.SourcePath)
	if v.acc.packageNames[dir] == nil {
		v.acc.packageNames[dir] = map[string]bool{}
	}
	for name := range topLevelNames(cu) {
		v.acc.packageNames[dir][name] = true
	}
	v.acc.fileDecls[cu.SourcePath] = topLevelNames(cu)
	v.acc.fileIdents[cu.SourcePath] = usedNames(cu)
	if vendored(cu) {
		return cu
	}
	if m, isMock := ifaceMockOf(cu); isMock {
		v.acc.ifaceMocks[cu.SourcePath] = m
		v.acc.ifaceFiles = append(v.acc.ifaceFiles, cu.SourcePath)
		v.acc.usage.imported[v2ServicePath(m.service)] = true
		v.acc.recordBoundaries(cu)
		return cu
	}
	if m, isMock := testifyMockOf(cu); isMock {
		v.acc.testifyMocks[cu.SourcePath] = m
		v.acc.ifaceFiles = append(v.acc.ifaceFiles, cu.SourcePath)
		v.acc.usage.imported[v2ServicePath(m.service)] = true
		v.acc.recordBoundaries(cu)
		return cu
	}
	calls := visitor.Init(&testifyCallScan{stubs: map[*java.MethodInvocation]string{}})
	calls.Visit(cu, nil)
	if len(calls.calls) > 0 {
		v.acc.testifyCalls[cu.SourcePath] = calls.calls
	}
	scan := scanFile(cu)
	v.acc.recordEmbeddedClients(cu, scan)
	v.acc.recordBoundaries(cu)
	if importsV1(cu) && !scan.migratable {
		// A test the rewrite cannot carry over holds its package back only if
		// the rest of the package has to stay too, which is known once every
		// file has been seen.
		if strings.HasSuffix(cu.SourcePath, "_test.go") {
			v.acc.blockedTests[cu.SourcePath] = scan.reason
		} else {
			v.acc.holdBack(cu.SourcePath, scan.reason)
		}
	}
	if scan.migratable && aliasedSession(cu) {
		v.acc.holdBack(cu.SourcePath, aliasedSessionReason)
	}
	if scan.needsModulePath() {
		v.acc.ifaceFiles = append(v.acc.ifaceFiles, cu.SourcePath)
	}
	if scan.attributePort {
		v.acc.attributePortUsers = append(v.acc.attributePortUsers, path.Dir(cu.SourcePath))
		v.acc.usage.imported[v2ServicePath(dynamodbService)] = true
	}
	for field := range scan.managerClientReads {
		v.acc.managerReaders[dir+"#"+field] = append(v.acc.managerReaders[dir+"#"+field], cu.SourcePath)
	}
	for field := range scan.managerFieldsUnproven {
		v.acc.managerUnproven[dir+"#"+field] = true
	}
	for _, ifacePath := range scan.ifacePaths {
		if service, _, ok := ifacePackageName(ifacePath); ok {
			v.acc.ifaceServices[service] = append(v.acc.ifaceServices[service], path.Dir(cu.SourcePath))
		}
	}
	if cu.PackageDecl == nil || cu.PackageDecl.Element == nil {
		return cu
	}
	// Files arrive in no fixed order, so a go.mod not yet seen would make the
	// probe skip a file the edit later migrates, and its helpers go unwritten.
	if v.acc.modulePath == "" && scan.needsModulePath() {
		v.acc.deferred = append(v.acc.deferred, cu)
		return cu
	}
	v.acc.probe(cu)
	return cu
}

// externalTestUnit marks a directory's external test package, whose helpers
// go in a _test.go file of their own: Go reads a non-test file of another
// package in the directory as a build error.
const externalTestUnit = "#external-test"

// helperUnit names the package cu's generated helpers are declared in.
func helperUnit(cu *golang.CompilationUnit) string {
	dir := path.Dir(cu.SourcePath)
	if strings.HasSuffix(cu.PackageDecl.Element.Name, "_test") && strings.HasSuffix(cu.SourcePath, "_test.go") {
		return dir + externalTestUnit
	}
	return dir
}

// helperPath is where unit's generated file named file is written.
func helperPath(unit, file string) string {
	if dir, ok := strings.CutSuffix(unit, externalTestUnit); ok {
		return path.Join(dir, strings.TrimSuffix(file, ".go")+"_test.go")
	}
	return path.Join(unit, file)
}

// probe runs the edit over cu and records the generated files it would need.
func (a *migrateAcc) probe(cu *golang.CompilationUnit) {
	probe := visitor.Init(&migrateVisitor{acc: a})
	if probed := probe.Visit(cu, recipe.NewExecutionContext()); probed != java.Tree(cu) && probe.scan != nil && probe.scan.adjunct {
		a.recordAdjunct(cu)
	}
	dir, pkg := helperUnit(cu), cu.PackageDecl.Element.Name
	if probe.needsServiceHelpers {
		a.helperPackages[dir] = pkg
	}
	if probe.needsImdsHelper {
		a.imdsPackages[dir] = pkg
	}
	if probe.needsWaitDuration {
		a.waitPackages[dir] = pkg
	}
	if probe.needsConfigHelper {
		a.configPackages[dir] = pkg
		// The generated loader names both packages whether or not the source it
		// stands in for did.
		a.usage.imported[v2Module] = true
		a.usage.imported[v2Config] = true
	}
	if probe.needsCompat {
		a.compatPackages[dir] = pkg
	}
	if probe.needsCredentialsChain {
		a.credentialsPackages[dir] = pkg
		a.usage.imported[v2Module] = true
		a.usage.imported[v2Config] = true
	}
	if probe.needsPathStyle {
		a.pathStylePackages[dir] = pkg
		a.usage.imported[v2ServicePkg+s3Service] = true
	}
	for service := range probe.serviceUsed {
		if a.serviceDirs[service] == nil {
			a.serviceDirs[service] = map[string]bool{}
		}
		a.serviceDirs[service][path.Dir(cu.SourcePath)] = true
	}
	if probe.needsPageReplay {
		a.pagePackages[dir] = pkg
	}
	if probe.needsZero {
		a.zeroPackages[dir] = pkg
	}
	if probe.needsEnumList {
		a.enumListPackages[dir] = pkg
	}
	if probe.needsCredentialsValue {
		a.credentialsValuePackages[dir] = pkg
		a.usage.imported[v2Module] = true
	}
	if probe.needsRequest {
		a.requestPackages[dir] = pkg
		a.usage.imported[smithyGo] = true
	}
	if probe.needsLoadOptions {
		a.loadOptionsPackages[dir] = pkg
		a.usage.imported[v2Config] = true
		a.usage.imported[v2Stscreds] = true
	}
	if probe.needsExpiresAt {
		a.expiresAtPackages[dir] = pkg
		a.usage.imported[v2Module] = true
	}
	if probe.needsConfigBuilder {
		a.builderPackages[dir] = pkg
		a.usage.imported[v2Module] = true
		a.usage.imported[v2Config] = true
	}
	if probe.needsNoCredentials {
		a.noCredentialsPackages[dir] = pkg
		a.usage.imported[smithyGo] = true
	}
	if probe.needsDisableSSL {
		a.sslPackages[dir] = pkg
		a.usage.imported[smithyGo] = true
	}
	if len(probe.printedServices) > 0 {
		a.prettifyPackages[dir] = pkg
		if a.prettifyServices[dir] == nil {
			a.prettifyServices[dir] = map[string]bool{}
		}
		for service := range probe.printedServices {
			a.prettifyServices[dir][service] = true
		}
	}
	if len(probe.validatedShapes) > 0 {
		a.validatePackages[dir] = pkg
		if a.validateShapes[dir] == nil {
			a.validateShapes[dir] = map[string]bool{}
		}
		for key := range probe.validatedShapes {
			a.validateShapes[dir][key] = true
		}
	}
	if probe.needsPartitions {
		a.partitionPackages[dir] = pkg
	}
	if probe.needsErrorHelper {
		a.errorPackages[dir] = pkg
	}
	if probe.needsAttributeHelpers {
		a.attributePackages[dir] = pkg
	}
	if probe.needsSTS {
		a.usage.imported[v2ServicePkg+stsService] = true
	}
	if probe.needsManager {
		a.usage.imported[v2S3Manager] = true
	}
	if probe.needsHandlers {
		a.handlerPackages[dir] = pkg
		a.usage.imported[v2Module] = true
		a.usage.imported[smithyGo] = true
	}
}

// singleImportDecl reports whether cu's imports sit in one declaration, which
// any later declaration would mark as the start of its own block.
// TODO: remove with its one caller once rewrite-go is past v0.0.38, whose
// OrderImports moves imports between declarations and prints source that does
// not compile; openrewrite/rewrite@d1fa9de773 sorts each declaration on its own.
func singleImportDecl(cu *golang.CompilationUnit) bool {
	if cu.Imports == nil {
		return true
	}
	for _, rp := range cu.Imports.Elements {
		if java.HasMarker[golang.ImportBlock](rp.Element.Markers) {
			return false
		}
	}
	return true
}

const (
	managerFieldReason     = "a call through a transfer manager's S3 where some file in the package sets the manager from something not proven to hold S3's own client"
	ifaceUnreachableReason = "an iface package with no module path to regenerate it under"
	aliasedSessionReason   = "an aliased session import, which the config loader replacing it cannot keep"
)

// aliasedSession reports whether cu binds the session package to a name of its
// own, which the swap would keep while the loader this recipe emits is spelled
// `config`.
func aliasedSession(cu *golang.CompilationUnit) bool {
	imp := pathswap.Find(cu, v1Session)
	return imp != nil && pathswap.Alias(imp) != ""
}

func (r *MigrateAwsSdkGoToV2) Name() string {
	return "org.openrewrite.golang.migration.MigrateAwsSdkGoToV2"
}
func (r *MigrateAwsSdkGoToV2) DisplayName() string {
	return "Migrate `aws-sdk-go` to `aws-sdk-go-v2`"
}
func (r *MigrateAwsSdkGoToV2) Description() string {
	return "Migrate `github.com/aws/aws-sdk-go`, whose support AWS ended in July 2025, to `github.com/aws/aws-sdk-go-v2`. The go directive rises to the Go 1.24 the v2 modules require; the session becomes a config loaded through a context, with each `aws.Config` field it carried — the region, the endpoint, the retry count, a static or shared credentials provider — becoming the loader option that replaces it; each client is built with `NewFromConfig`, a per-client region override — or a config local that is nil or sets only the region — becoming a functional option; and every operation takes a context. A config built field by field instead of in one literal keeps its shape: the load moves to the local's declaration and the writes that follow retarget onto v2's own config, with the S3 addressing style — which v2 holds on the client's options rather than the config — hoisted into a local the constructor reads. Shapes and enums follow the manifest to the `types` sub-package, an enum field losing the `aws.String` its `*string` needed and an enum list changing element type with it; v1's fluent setters become assignments; and a field v2 holds by value loses the dereference that read it, or regains the pointer where it was passed on — v1 could report such a field as nil and v2 cannot, so review a nil test downstream of one. The packages v2 relocated follow too: `s3manager` becomes `feature/s3/manager`, `ec2metadata` becomes `feature/ec2/imds`, the DynamoDB `expression` builder becomes `feature/dynamodb/expression` — short of an operand v2's encoder would store differently — `rdsutils.BuildAuthToken` becomes `feature/rds/auth`'s, taking a context and the config's credentials provider, and `stscreds` moves up beside `credentials`, each bound back to the name the file already spells, and the per-service `iface` packages v2 deleted are regenerated into the module under `internal/awsiface`, so the interfaces mocks embed still exist; a mockery mock of one is regenerated against v2's signatures, the expectations set on it gaining the context v2 passes first, and a hand-written stub overriding a page iterator gains the operation v2's paginator calls instead, serving the pages its override hands out in turn. `dynamodbattribute` is ported onto v2's `AttributeValue` and generated under `internal/awscompat`, since v2's `attributevalue` ignores json tags and stores an empty value as itself where v1 stored NULL, so items written before the migration still read back. A shape's `Validate` becomes a generated function checking v1's rules and reporting them in v1's words. Where the go directive rises past 1.24, a printf-style call whose format is not a constant gains a `\"%s\"` format, which that release's vet otherwise rejects. Where v2 restructured rather than renamed, a wrapper keeps the v1 call site's shape: a page iterator's callback is driven by the v2 paginator in a function literal called on the spot, a waiter becomes v2's waiter type bounded by a generated constant, an inline `session.New` becomes a generated loader that panics as `session.Must` did, as does a `session.Must` anywhere but an `x :=` statement, which expands into the load and its panic, a handler a function adds to its session's `Send` or `UnmarshalError` list becomes a generated stand-in for that list on the config's `APIOptions`, run per attempt around the round trip in v1's order and handed the service and operation names, the attempt's start and the input — so metrics and logs it feeds keep their meaning, though the operation and input print differently — `EC2Metadata.Region` and `Available` go through generated helpers, a DynamoDB `AttributeValue` setting one scalar member becomes that member's v2 type and a read of one goes through a generated getter answering nil where v1's field was unset, `awserr.New` with nothing wrapped becomes a generated error reading as v1's did, `aws.WriteAtBuffer` moves to the transfer manager, an endpoint gains the https scheme v1 added to one written without, and a list or map v2 holds by value is converted at the API boundary by generated helpers so the code around it keeps its v1 shape. v1 took no context anywhere v2 takes one, so a function with none in scope gets `context.TODO()` — review those and plumb a real context through. A package migrates whole or not at all, since a half-migrated one does not compile, but the module does not: a file holding a construct with no faithful v2 form — the rest of the v1 request plumbing, such as a handler reading the request's error or response, a session compared to nil, a shared-credentials provider assigned to a config field, a v1-only helper the manifest cannot account for, a shape v2 turned into a union interface such as DynamoDB's `AttributeValue`, a sub-package v2 moved or dropped such as `s3crypto`, a type implementing `dynamodbattribute`'s Marshaler, which takes v1's `AttributeValue`, a stub overriding a WithContext, Request or waiter method v2's client would never call — is left as it is along with the rest of its package, each listed in the blockers data table, and the rest of the module moves around it. Both requires then sit in the go.mod side by side. A client and a shape cross package boundaries, so callers of what stayed behind will not compile until it is migrated by hand; that table is the list to work through, and `FindAwsSdkGoV1Usage` marks the constructs within each file. go.mod requires each v2 module the migrated source imports, along with the indirect requirements they bring, at versions the regenerated iface packages compile against — a module the build already selects higher keeps its version — and go.sum gains their hashes, so the result builds under `-mod=readonly` as it stands. Then run `go mod tidy`: the build does not need it, but it drops what the migration leaves behind — the requirements only v1 needed, the go.sum lines nothing reads any more, and an `// indirect` marker on a module the source now imports directly."
}
func (r *MigrateAwsSdkGoToV2) DataTables() []recipe.DataTableDescriptor {
	return []recipe.DataTableDescriptor{awsBlockersTable.Descriptor()}
}
func (r *MigrateAwsSdkGoToV2) Tags() []string {
	return []string{"migration", "aws"}
}

type migrateVisitor struct {
	visitor.GoVisitor
	acc     *migrateAcc
	scan    *fileScan
	ctxName string
	// needsContext records that context.TODO() was emitted, so the file gains
	// the import it reads.
	needsContext bool
	// needsTypes holds the services whose types sub-package was referenced, and
	// typesAliases the local names those imports bind.
	needsTypes   map[string]bool
	typesAliases map[string]bool
	// shadowed are the names this function declares with a non-client type.
	shadowed map[string]bool
	// awserrVars are the names bound by an awserr.Error assertion, whose
	// accessors smithy spells differently.
	awserrVars map[string]bool
	// needsAws records that an aws package reference was emitted where the file
	// may not have imported it.
	needsAws bool
	// needsSmithy records that the smithy error interface was emitted, along
	// with the errors.As matching it, and needsSmithyType a smithy type alone.
	needsSmithy     bool
	needsSmithyType bool
	// needsErrorHelper records that awserr.New became the generated error.
	needsErrorHelper bool
	// awsUsed records that an aws package reference survived the rewrite.
	awsUsed bool
	// pathStyleHoisted records that this function carries the S3 addressing
	// style in a local of its own, which the S3 clients it builds read.
	pathStyleHoisted bool
	// needsConfigHelper records that session.New was rewritten as the generated
	// loader, which the package then needs a definition of.
	needsConfigHelper bool
	// carriedConfigs counts the pass-throughs a carried config became, less the
	// ones a Must unwrapped again, which need no helper.
	carriedConfigs int
	// receiver is the enclosing method's receiver name, when it binds something
	// other than a client.
	receiver string
	// needsErrors records that errors.As was emitted outside the awserr
	// expansion, which brings its own imports.
	needsErrors bool
	// needsServiceHelpers records that a v1 service function was rewritten as the
	// generated stand-in for it.
	needsServiceHelpers bool
	// needsImdsHelper records that a metadata client's Region was rewritten.
	needsImdsHelper bool
	// needsWaitDuration records that a waiter was rewritten, which needs the
	// generated bound.
	needsWaitDuration bool
	// sessionResults are the positions of the enclosing function's results
	// declared as a session.
	sessionResults map[int]bool
	// needsS3 records that the s3 package was named by a rewrite of the
	// transfer manager, whose input moved there.
	needsS3 bool
	// needsManager records that the transfer manager was named by a relocated
	// aws reference in a file that did not import it.
	needsManager bool
	// needsSTS records that an assume-role provider named the STS package in a
	// file that did not import it.
	needsSTS bool
	// serviceUsed records the services still referenced through their own
	// package after the relocation, so the ones emptied out lose their import.
	serviceUsed map[string]bool
	// needsCompat records that a slice conversion was emitted, so the package
	// gains the helper file that defines it.
	needsCompat bool
	// needsHandlers records that a request handler was rewritten onto the
	// generated adapter.
	needsHandlers bool
	// needsUserAgent records that a user-agent handler became v2's option.
	needsUserAgent bool
	// needsPathStyle records that S3's path-style setting became the generated
	// service option.
	needsPathStyle bool
	// needsCredentialsChain records that a v1 credentials chain or provider
	// became a generated stand-in.
	needsCredentialsChain bool
	// reformat records that a value parsed apart was placed in a literal.
	reformat bool
	// needsPartitions records that v1's partition data was reached for.
	needsPartitions bool
	// needsDisableSSL records that a DisableSSL setting became the generated
	// service option.
	needsDisableSSL bool
	// dir is the directory of the file being visited.
	dir string
	// needsPageReplay records that a page iterator override gained the
	// operation, which calls the replay helper.
	needsPageReplay bool
	// needsZero records that a shape's comparison with its empty literal became
	// the generated zero check.
	needsZero bool
	// needsEnumList records that a list built at runtime became v2's enum list
	// through the generated conversion.
	needsEnumList bool
	// needsCredentialsValue records that a credentials.Value or a held
	// provider's Get became the stand-in keeping v1's shape.
	needsCredentialsValue bool
	// needsRequest records that an OperationRequest call became the request
	// stand-in.
	needsRequest bool
	// needsLoadOptions records that a session.Options value became the
	// LoadOptions the generated helpers load and add to.
	needsLoadOptions bool
	// needsExpiresAt records that a credentials ExpiresAt became the stand-in.
	needsExpiresAt bool
	// needsAwsHTTP records that a RequestFailure became awshttp's ResponseError.
	needsAwsHTTP bool
	// needsStscreds records that assume-role options were written in a file
	// not importing stscreds.
	needsStscreds bool
	// needsConfigBuilder records that a config local built up through setters
	// became the generated builder.
	needsConfigBuilder bool
	// needsNoCredentials records that an awserr match became the generated
	// helper raising NoCredentialProviders.
	needsNoCredentials bool
	// printedServices are the services whose shapes a printing call was
	// handed, which the generated printer masks the sensitive fields of.
	printedServices map[string]bool
	// editing is unset in the probe the scan runs, which must not settle what
	// the module migrates before every file is seen.
	editing bool
	// validatedShapes are the `service.Shape` whose Validate the file calls.
	validatedShapes map[string]bool
	// needsRetry records that a v1 retryer became v2's standard one, and
	// needsTime that what it wrote names the time package.
	needsRetry bool
	needsTime  bool
	// needsAttributeHelpers records that an AttributeValue member read became
	// the generated getter.
	needsAttributeHelpers bool
	// gomockLocal is the name the file binds gomock to, and gomockImport the
	// path to import where an expectation needs a matcher and it binds none.
	gomockLocal  string
	gomockImport string
}

func (v *migrateVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	out := v.migrateCompilationUnit(cu, p)
	if migrated, isCU := out.(*golang.CompilationUnit); isCU && v.editing && v.acc.rewritesTestifyCalls(migrated) {
		out = v.acc.rewriteTestifyCalls(migrated)
	}
	if v.editing && v.acc.raisesPastPrintfCheck(cu.SourcePath) && !vendored(cu) {
		out = fixFormatStrings(out)
	}
	return out
}

func (v *migrateVisitor) migrateCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.dir = path.Dir(cu.SourcePath)
	// The iface replacements are imported through the module's own path, so a
	// file needing one stays on v1 when there is no go.mod to name it.
	if v.acc != nil && v.acc.ifaceUnreachable() && scanFile(cu).needsModulePath() {
		v.acc.blockerReason(cu.SourcePath, ifaceUnreachableReason)
		return cu
	}
	if vendored(cu) {
		return cu
	}
	if v.acc != nil {
		if reason, disabled := v.acc.disabledTests[cu.SourcePath]; disabled {
			return commentedOutTest(cu, reason)
		}
		// A file held back by a decision spanning the package, rather than by
		// its own scan, is listed already.
		if _, listed := v.acc.blockers[cu.SourcePath]; listed {
			return cu
		}
		if by, held := v.acc.heldBackBy(cu); held {
			// A file that never names the SDK still reads its values, typed as the
			// v1 shapes the rest of the package keeps.
			if importsV1(cu) {
				v.acc.blockerReason(cu.SourcePath, "in the same package as "+by+", which stays on v1")
			}
			return cu
		}
	}
	if v.acc != nil {
		if m, isMock := v.acc.ifaceMocks[cu.SourcePath]; isMock {
			if regenerated, ok := v.acc.regeneratedMock(cu, m); ok {
				return regenerated
			}
			return cu
		}
		if m, isMock := v.acc.testifyMocks[cu.SourcePath]; isMock {
			if regenerated, ok := regeneratedMockery(cu, m); ok {
				return regenerated
			}
			return cu
		}
	}
	v.scan = scanFile(cu)
	if !v.scan.migratable || aliasedSession(cu) {
		return cu
	}
	// A name another file of the package declares at the top level is as
	// taken as one this file binds.
	if v.acc != nil && v.scan.sessionPkg != "" && v.scan.configLocal == configPkgName && v.acc.packageNames[path.Dir(cu.SourcePath)][configPkgName] {
		v.scan.configLocal = configPkgName + "aws"
	}

	v.ctxName = ""
	v.needsContext = false
	v.needsTypes = map[string]bool{}
	v.typesAliases = map[string]bool{}
	v.serviceUsed = map[string]bool{}
	v.awsUsed = false
	// The accessors are renamed inside an if body the expansion only reaches
	// afterwards, so the names it binds are collected first.
	v.awserrVars = awserrBoundNames(cu, v.scan)
	v.needsSmithy = false
	v.needsSmithyType = false
	v.needsErrorHelper = false
	v.needsAws = false
	v.needsCompat = false
	v.needsS3 = false
	v.needsManager = false
	v.needsSTS = false
	v.needsConfigHelper = false
	v.carriedConfigs = 0
	v.needsWaitDuration = false
	v.needsImdsHelper = false
	v.needsServiceHelpers = false
	v.needsErrors = false
	v.needsHandlers = false
	v.needsUserAgent = false
	v.needsPathStyle = false
	v.needsCredentialsChain = false
	v.needsRetry = false
	v.needsPartitions = false
	v.needsDisableSSL = false
	v.needsNoCredentials = false
	v.needsConfigBuilder = false
	v.needsStscreds = false
	v.needsAwsHTTP = false
	v.needsExpiresAt = false
	v.needsLoadOptions = false
	v.needsRequest = false
	v.needsCredentialsValue = false
	v.needsEnumList = false
	v.needsZero = false
	v.needsPageReplay = false
	v.printedServices = map[string]bool{}
	v.validatedShapes = nil
	v.needsTime = false
	v.needsAttributeHelpers = false
	v.gomockLocal = gomockLocal(cu)
	v.gomockImport = ""
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)
	cu = v.rewriteRetryerDeclarations(cu)
	cu = v.addPageReplays(cu)
	if v.carriedConfigs > 0 {
		v.needsConfigHelper = true
	}
	// golang.org/x/net/context bound to the name serves as well: its Context
	// is the standard library's, and it has TODO.
	if v.needsContext && !importsAs(cu, "golang.org/x/net/context", contextPkg) {
		recipegolang.MaybeAddImport(v, contextPkg, nil, false)
	}
	// A file naming no SDK package gets the aws import only where the rewrite
	// left a reference to it.
	if v.needsAws && (!v.scan.adjunct || referencesQualifier(cu, v.scan.awsQualifier())) {
		recipegolang.MaybeAddImport(v, v2Aws, nil, false)
	}
	if v.needsSmithy || v.needsErrors {
		recipegolang.MaybeAddImport(v, errorsPkg, v.errorsAlias(), false)
	}
	if v.gomockImport != "" {
		recipegolang.MaybeAddImport(v, v.gomockImport, nil, false)
	}
	if v.needsSmithy || v.needsSmithyType {
		recipegolang.MaybeAddImport(v, smithyGo, nil, false)
	}
	if v.needsManager {
		recipegolang.MaybeAddImport(v, v2S3Manager, nil, false)
	}
	if v.needsStscreds {
		recipegolang.MaybeAddImport(v, v2Stscreds, nil, false)
	}
	if v.needsAwsHTTP {
		alias := awsHTTPLocal
		recipegolang.MaybeAddImport(v, v2AwsHTTP, &alias, false)
	}
	if v.needsRetry {
		recipegolang.MaybeAddImport(v, v2Retry, nil, false)
		recipegolang.MaybeAddImport(v, v2RateLimit, nil, false)
	}
	if v.needsTime {
		recipegolang.MaybeAddImport(v, "time", nil, false)
	}
	if v.needsUserAgent {
		alias := awsMiddlewareAlias
		recipegolang.MaybeAddImport(v, v2Aws+"/middleware", &alias, false)
	}
	if v.needsSTS {
		recipegolang.MaybeAddImport(v, v2ServicePkg+stsService, nil, false)
	}
	if v.needsS3 && !pathswap.Imports(cu, v1ServicePkg+s3Service) {
		recipegolang.MaybeAddImport(v, v2ServicePkg+s3Service, nil, false)
	}
	for service := range v.needsTypes {
		alias := awsmanifest.TypesAlias(service)
		recipegolang.MaybeAddImport(v, v2ServicePath(service)+"/types", &alias, false)
	}
	if v.scan.adjunct {
		// Nothing to swap: the file never named the SDK, and what it reads off
		// an SDK value has already been seen to.
		return cu
	}
	return v.swapImports(cu, p)
}

// dropEmptiedServiceImports removes the service imports nothing references any
// more, which is every service whose names all moved to its types sub-package.
// RemoveUnusedImports answers that from attribution, and there is none for a
// package the parser could not resolve, so the references were counted during
// the visit instead.
func (v *migrateVisitor) dropEmptiedServiceImports(cu *golang.CompilationUnit) *golang.CompilationUnit {
	if cu.Imports == nil {
		return cu
	}
	// A dropped statement took its references with it, which the flags set
	// during the visit cannot unlearn, and a type such as *aws.Config sets none,
	// so the aws package is re-counted here.
	v.awsUsed = referencesQualifier(cu, v.scan.awsPkg)
	var kept []java.RightPadded[*java.Import]
	dropped := false
	// The blank line that separates import groups is the prefix of the first
	// entry in a group, so dropping that entry would take the separation with
	// it. It is carried to whichever entry takes its place.
	var carried *java.Space
	for _, rp := range cu.Imports.Elements {
		if carried != nil {
			rp.Element = rp.Element.WithPrefix(*carried)
			carried = nil
		}
		path := rp.Element.Path()
		service, isService := strings.CutPrefix(path, v2ServicePkg)
		service = awsmanifest.V1Name(service)
		if isService && !strings.Contains(service, "/") && v.scan.usesService(service) && !v.serviceUsed[service] {
			dropped = true
			carried = groupSeparator(rp.Element.Prefix, carried)
			continue
		}
		// The aws package goes the same way when its every reference was
		// consumed — a Config literal unpacked into config.WithRegion leaves
		// nothing behind.
		// The session import became the config package, which is dead when the
		// file only ever named the session as a type.
		if path == v2Config && v.scan.sessionPkg != "" && !referencesQualifier(cu, v.scan.configLocal) {
			dropped = true
			carried = groupSeparator(rp.Element.Prefix, carried)
			continue
		}
		// awserr has no v2 package; its every use became smithy or a literal.
		// The request package goes once its handlers were rewritten.
		if path == v1Awserr || path == v1Request && !referencesQualifier(cu, v.scan.requestPkg) ||
			path == v1Endpoints && !referencesQualifier(cu, v.scan.endpointsPkg) ||
			path == v1Client && !referencesQualifier(cu, v.scan.clientPkg) {
			dropped = true
			carried = groupSeparator(rp.Element.Prefix, carried)
			continue
		}
		// The transfer manager goes when its input was the only thing named from
		// it, that input having moved to the s3 package.
		if path == v2S3Manager && v.scan.managerPkg != "" && !referencesQualifier(cu, v.scan.managerPkg) {
			dropped = true
			carried = groupSeparator(rp.Element.Prefix, carried)
			continue
		}
		// The credentials package goes when its every constructor was folded into
		// a loader option.
		if path == v2Credentials && v.scan.credentialsPkg != "" && !referencesQualifier(cu, v.scan.credentialsPkg) {
			dropped = true
			carried = groupSeparator(rp.Element.Prefix, carried)
			continue
		}
		if path == v2Aws && v.scan.awsPkg != "" && !v.awsUsed {
			dropped = true
			carried = groupSeparator(rp.Element.Prefix, carried)
			continue
		}
		kept = append(kept, rp)
	}
	if !dropped {
		return cu
	}
	// An import added afterwards starts a declaration of its own rather than
	// inheriting an empty one's missing spacing.
	if len(kept) == 0 {
		c := *cu
		c.Imports = nil
		return &c
	}
	// The whitespace before the block's closing paren belongs to the last entry,
	// so dropping that entry would put the paren on the same line as its
	// neighbour.
	if last := cu.Imports.Elements[len(cu.Imports.Elements)-1]; len(kept) > 0 {
		if kept[len(kept)-1].Element.Path() != last.Element.Path() {
			kept[len(kept)-1].After = last.After
		}
	}
	imports := *cu.Imports
	imports.Elements = kept
	c := *cu
	c.Imports = &imports
	return &c
}

// groupSeparator keeps a blank-line prefix alive across a dropped import, so the
// entry that follows still opens its group.
func groupSeparator(prefix java.Space, carried *java.Space) *java.Space {
	if carried != nil {
		return carried
	}
	if strings.Contains(prefix.Whitespace(), "\n\n") {
		return &prefix
	}
	return nil
}

// The requires follow the same scan, so the module lands on v2 in one pass.
func (v *migrateVisitor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	return applyAwsRequires(gm, &v.acc.usage)
}

func (v *migrateVisitor) VisitGoSum(gs *golang.GoSum, p any) java.Tree {
	return applyAwsSums(gs, &v.acc.usage)
}

// swapImports repoints every v1 import at its v2 path and re-attributes the
// references, so RemoveUnusedImports does not read the new imports as unused.
func (v *migrateVisitor) swapImports(cu *golang.CompilationUnit, p any) *golang.CompilationUnit {
	var rules []pathswap.Rule
	// awserr sits under the aws package, so the rule moving that package would
	// carry it to a v2 path that does not exist. Mapping it to itself keeps it
	// put for dropEmptiedServiceImports to remove, its every use having become
	// smithy or a literal.
	if pathswap.Imports(cu, v1Awserr) {
		rules = append(rules, pathswap.Rule{Old: v1Awserr, New: v1Awserr})
	}
	for _, kept := range []string{v1Request, v1Endpoints, v1Client} {
		if pathswap.Imports(cu, kept) {
			rules = append(rules, pathswap.Rule{Old: kept, New: kept})
		}
	}
	for _, rp := range cu.Imports.Elements {
		path := rp.Element.Path()
		if service, _, isIface := ifacePackageName(path); isIface {
			rules = append(rules, pathswap.Rule{Old: path, New: ifaceImportPath(v.acc.modulePath, service)})
			continue
		}
		if path == v1DynamodbAttribute {
			rules = append(rules, pathswap.Rule{Old: path, New: attributePortImportPath(v.acc.modulePath)})
			continue
		}
		if newPath, ok := MapPath(path); ok {
			// The transfer manager's package was renamed along with its move, so
			// it is bound back to the name the file already spells.
			rule := pathswap.Rule{Old: path, New: newPath, KeepName: path == v1S3Manager || path == v1Ec2Metadata || path == v1Rdsutils || path == v1Rest || renamedService(path)}
			if newPath == v2Config && v.scan.configLocal != configPkgName {
				rule.Alias = v.scan.configLocal
			}
			rules = append(rules, rule)
		}
	}
	if len(rules) == 0 {
		return cu
	}
	// A pathswap rule matches a parent path as well as an exact one, and the
	// session sits under the aws package it is listed beside. Longest first so
	// the specific rule wins: `.../aws/session` becomes the config package
	// rather than `.../v2/aws/session`.
	sort.SliceStable(rules, func(i, j int) bool { return len(rules[i].Old) > len(rules[j].Old) })
	swapped := pathswap.RewriteImports(cu, rules)
	if retyped, ok := pathswap.Retype(rules).Visit(swapped, p).(*golang.CompilationUnit); ok {
		swapped = retyped
	}
	// A file whose every reference moved to the types sub-package no longer uses
	// the service package the swap pointed at. RemoveUnusedImports answers that
	// from attribution, which is absent for a package it cannot resolve, so the
	// references are counted here instead.
	swapped = v.dropEmptiedServiceImports(swapped)
	// The v2 paths sort differently from the v1 ones they replace — the session
	// becoming the config package moves furthest — so the block is re-ordered
	// rather than left in an order gofmt would not produce. A file with more
	// than one declaration is skipped until singleImportDecl can go.
	if singleImportDecl(swapped) {
		v.DoAfterVisit((&recipegolang.OrderImports{}).Editor())
	}
	if drained, ok := visitor.DrainAfterVisits(v, swapped, p).(*golang.CompilationUnit); ok {
		return drained
	}
	return swapped
}

// The context every v2 operation takes comes from the enclosing function, and a
// closure inherits its enclosing one.
func (v *migrateVisitor) VisitGoMethodDeclaration(md *golang.MethodDeclaration, p any) java.J {
	outer := v.receiver
	v.receiver = v.scan.nonClientReceiver(md)
	md = v.GoVisitor.VisitGoMethodDeclaration(md, p).(*golang.MethodDeclaration)
	v.receiver = outer
	return md
}

func (v *migrateVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	managerName, managerCtx, managerMethod := v.scan.managerMethodDeclaration(md)
	if managerMethod {
		defer func() { v.needsContext = v.needsContext || managerCtx }()
	}
	outer := v.scan.scopeTo(md)
	defer v.scan.restore(outer)
	outerPathStyle := v.pathStyleHoisted
	v.pathStyleHoisted = hoistsPathStyle(md, v.scan)
	defer func() { v.pathStyleHoisted = outerPathStyle }()
	outerCtx, outerShadowed, outerResults := v.ctxName, v.shadowed, v.sessionResults
	if name := contextParamName(md); name != "" {
		v.ctxName = name
	}
	v.shadowed = shadowedClients(md, v.scan)
	if v.receiver != "" {
		v.shadowed[v.receiver] = true
	}
	v.sessionResults = v.scan.sessionResults(md)
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)
	v.ctxName, v.shadowed, v.sessionResults = outerCtx, outerShadowed, outerResults
	if managerMethod {
		return retypedManagerMethod(md, managerName, managerCtx)
	}
	return v.retypeOperationDeclaration(md)
}

// An interface over the SDK, or a mock overriding one of its methods, declares
// the v1 signature. Every v2 operation takes a context and functional options,
// so the declaration is brought to that shape — without it the v2 client no
// longer satisfies the interface the file wrote.
func (v *migrateVisitor) retypeOperationDeclaration(md *java.MethodDeclaration) *java.MethodDeclaration {
	serviceLocal, input, output, inputName, ok := v.scan.awsOperationDeclaration(md)
	if !ok {
		return md
	}
	params, ok := operationParams(serviceLocal, md.Name.Name, input, output, inputName)
	if !ok {
		return md
	}
	// The signature names context.Context whether or not it names the parameter,
	// so an interface's method needs the import just as a concrete one does.
	v.needsContext = true
	c := *md
	c.Parameters = md.Parameters
	c.Parameters.Elements = params
	return &c
}

// mergedClientConfig builds the client from the session's config with its
// trailing configs merged over it, as v1 did.
func (v *migrateVisitor) mergedClientConfig(renamed *java.MethodInvocation, args []java.Expression) java.J {
	for _, arg := range args[1:] {
		if !v.scan.mergedConfig(arg) {
			return renamed
		}
	}
	elements := []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(args[0], java.EmptySpace)}}
	reformat := false
	for _, arg := range args[1:] {
		if options, isChain := v.scan.configChainOptions(arg); isChain {
			if literal, built := v.scan.keptChainLiteral(options); built {
				v.flagOptions(options)
				v.awsUsed, v.needsAws = true, true
				arg = literal
				reformat = true
			}
		}
		elements = append(elements, java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(arg, java.SingleSpace)})
	}
	v.needsConfigHelper = true
	merged := &java.MethodInvocation{
		Name:       &java.Identifier{Name: mergeConfigHelper},
		Arguments:  java.Container[java.Expression]{Elements: elements},
		MethodType: lstutil.FuncType("", mergeConfigHelper, nil),
	}
	c := *renamed
	c.Arguments = renamed.Arguments
	c.Arguments.Elements = []java.RightPadded[java.Expression]{{Element: merged}}
	// A literal parsed apart takes the call's indentation.
	if reformat {
		if formatted, ok := format.AutoFormat(&c, nil, nil, v.Cursor().Parent()).(java.Expression); ok {
			return formatted
		}
	}
	return &c
}

// An operation's result is a shape the manifest names, which seeds the
// inference that locates every value slice read off it.
func (v *migrateVisitor) VisitAssignment(a *java.Assignment, p any) java.J {
	if target, isField := a.Variable.(*java.FieldAccess); isField && v.scan.managerPartsLimit(target) {
		visited := v.GoVisitor.VisitAssignment(a, p).(*java.Assignment)
		if _, isLiteral := visited.Value.Element.(*java.Literal); isLiteral {
			return visited
		}
		c := *visited
		c.Value.Element = lstutil.SetExprPrefix(basicConversion(lstutil.SetExprPrefix(visited.Value.Element, java.EmptySpace), "int32"), visited.Value.Element.GetPrefix())
		return &c
	}
	if target, isIdent := a.Variable.(*java.Identifier); isIdent && v.scan.configKept[target.Name] && v.scan.configLocals[target.Name] != nil && v.scan.configLocals[target.Name].chain && java.HasMarker[golang.ShortVarDecl](a.Markers) {
		visited := v.GoVisitor.VisitAssignment(a, p).(*java.Assignment)
		if rewritten, ok := v.keptChainDeclaration(visited); ok {
			if formatted, ok := format.AutoFormat(rewritten, p, nil, v.Cursor().Parent()).(*java.Assignment); ok {
				return formatted
			}
			return rewritten
		}
		return visited
	}
	if v.scan.optionsAssignment(a) {
		visited := v.GoVisitor.VisitAssignment(a, p).(*java.Assignment)
		if rewritten, ok := v.optionsLocalLiteral(visited); ok {
			if formatted, ok := format.AutoFormat(rewritten, p, nil, v.Cursor().Parent()).(*java.Assignment); ok {
				return formatted
			}
			return rewritten
		}
		if rewritten, ok := v.optionsWrite(visited); ok {
			if formatted, ok := format.AutoFormat(rewritten, p, nil, v.Cursor().Parent()).(*java.Assignment); ok {
				return formatted
			}
			return rewritten
		}
		return visited
	}
	if _, isWrite, ok := v.scan.valueElementWrite(a); isWrite && ok {
		visited := v.GoVisitor.VisitAssignment(a, p).(*java.Assignment)
		if value, ok := v.scan.pointerValueSource(visited.Value.Element); ok {
			return unwrappedElementWrite(visited, value)
		}
		return visited
	}
	// Writing a setting v2 holds by value is the read's mirror: the reach
	// changes and the aws helper the pointer needed comes off the value. Taken
	// before the children are visited, since the rewrite no longer looks like
	// what this recognises.
	// An endpoint written to a config takes the scheme v1 would have added.
	if rewritten, target, ok := v.rewriteConfigAccess(a.Variable); ok && target.name == "BaseEndpoint" {
		if call, isCall := a.Value.Element.(*java.MethodInvocation); isCall {
			if helper, isAws := qualifiedCall(call, v.scan.awsPkg); isAws && helper == "String" && len(realArgs(call)) == 1 {
				value := *call
				value.Arguments.Elements = []java.RightPadded[java.Expression]{{Element: endpointCall(realArgs(call)[0])}}
				v.needsConfigHelper = true
				c := *a
				c.Variable = rewritten
				c.Value = java.LeftPadded[java.Expression]{Before: a.Value.Before, Element: lstutil.SetExprPrefix(&value, java.SingleSpace), Markers: a.Value.Markers}
				return &c
			}
		}
	}
	if rewritten, target, ok := v.rewriteConfigAccess(a.Variable); ok && target.depointered {
		if value, unwrapped := v.scan.pointerValueSource(a.Value.Element); unwrapped {
			c := *a
			c.Variable = rewritten
			c.Value = java.LeftPadded[java.Expression]{
				Before:  a.Value.Before,
				Element: lstutil.SetExprPrefix(shifted(value, target.offset), java.SingleSpace),
				Markers: a.Value.Markers,
			}
			return &c
		}
	}
	if rewritten, ok := v.pathStyleServiceOption(a); ok {
		return rewritten
	}
	if value, isPathStyle := v.pathStyleWrite(a); isPathStyle {
		c := *a
		c.Variable = &java.Identifier{Prefix: a.Variable.GetPrefix(), Name: pathStyleVar, Type: lstutil.NamedType("bool")}
		c.Value = java.LeftPadded[java.Expression]{
			Before:  a.Value.Before,
			Element: lstutil.SetExprPrefix(value, java.SingleSpace),
			Markers: a.Value.Markers,
		}
		return &c
	}
	a = v.GoVisitor.VisitAssignment(a, p).(*java.Assignment)
	if service, shape, field, isShapeField := v.assignedShapeField(a.Variable); isShapeField {
		if converted, rewrote := v.convertedFieldValue(service, shape, field, a.Value.Element); rewrote {
			c := *a
			c.Value = java.LeftPadded[java.Expression]{
				Before:  a.Value.Before,
				Element: converted,
				Markers: a.Value.Markers,
			}
			a = &c
		}
	}
	v.recordShapeOf(a.Variable, a.Value.Element)
	return a
}

func (v *migrateVisitor) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	original := ma
	ma = v.GoVisitor.VisitMultiAssignment(ma, p).(*golang.MultiAssignment)
	if rewritten, ok := v.requestRewrite(original, ma); ok {
		if formatted, ok := format.AutoFormat(rewritten, p, nil, v.Cursor().Parent()).(*golang.MultiAssignment); ok {
			return formatted
		}
		return rewritten
	}
	if len(ma.Values) == 1 && len(ma.Variables) > 0 {
		v.recordShapeOf(ma.Variables[0].Element, ma.Values[0].Element)
	}
	return ma
}

// recordShapeOf names the shape a variable takes on from what it was assigned:
// an operation's output, or a shape literal.
func (v *migrateVisitor) recordShapeOf(target, value java.Expression) {
	name, ok := target.(*java.Identifier)
	if !ok {
		return
	}
	if ref, ok := v.scan.shapeOfExpression(value); ok {
		v.scan.shapes[name.Name] = ref
	}
}

// shadowedReceiver reports whether a call's receiver is a name this function
// declared as something other than a client.
func (v *migrateVisitor) shadowedReceiver(mi *java.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	return ok && v.shadowed[recv.Name]
}

// shapeOfExpression names the shape an expression produces.
func (s *fileScan) shapeOfExpression(expr java.Expression) (shapeRef, bool) {
	if unary, ok := expr.(*golang.Unary); ok {
		expr = unary.Expression
	}
	if comp, ok := expr.(*golang.Composite); ok {
		if service, shape, ok := s.compositeShape(comp); ok {
			return shapeRef{service: service, shape: shape}, true
		}
		return shapeRef{}, false
	}
	mi, ok := expr.(*java.MethodInvocation)
	if !ok || mi.Name == nil {
		return shapeRef{}, false
	}
	service, isClient := s.clientOperationService(mi)
	if !isClient {
		// The operation may be reached through a struct field or an interface,
		// which the input it takes identifies just as well.
		if service, isClient = s.operationInputService(mi); !isClient {
			return shapeRef{}, false
		}
	}
	if _, output, ok := awsmanifest.OperationShapes(service, strings.TrimSuffix(mi.Name.Name, "WithContext")); ok {
		return shapeRef{service: service, shape: output}, true
	}
	return shapeRef{}, false
}

// A parameter or local declared as a shape is known just as directly.
func (v *migrateVisitor) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	// The target an errors.As call fills is smithy's interface in v2.
	if ref, ok := v.scan.awserrDeclared(vd); ok {
		c := *vd
		c.TypeExpr = lstutil.SetExprPrefix(awserrTargetType(ref), ref.Prefix)
		if isRequestFailure(ref) {
			v.needsAwsHTTP = true
		} else {
			v.needsSmithyType = true
		}
		return &c
	}
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

// declaredShape names the shape a type expression refers to, through a pointer
// or directly, whether it still names the service package or the types alias.
func (s *fileScan) declaredShape(expr java.Expression) (shapeRef, bool) {
	if arr, isArray := expr.(*java.ArrayType); isArray {
		// A slice of shapes is not itself one; what it binds is its elements,
		// which a range over it picks up.
		_ = arr
		return shapeRef{}, false
	}
	if ptr, ok := expr.(*golang.PointerType); ok {
		expr = ptr.Elem
	}
	fa, ok := expr.(*java.FieldAccess)
	if !ok || fa.Name.Element == nil {
		return shapeRef{}, false
	}
	target, ok := fa.Target.(*java.Identifier)
	if !ok {
		return shapeRef{}, false
	}
	for local, service := range s.services {
		if target.Name == local || target.Name == awsmanifest.TypesAlias(service) {
			return shapeRef{service: service, shape: fa.Name.Element.Name}, true
		}
	}
	return shapeRef{}, false
}

// declaredSliceShape names the shape a `[]*svc.Shape` declaration holds
// elements of, which a range over it binds.
func (s *fileScan) declaredSliceShape(expr java.Expression) (shapeRef, bool) {
	arr, isArray := expr.(*java.ArrayType)
	if !isArray || arr.ElementType == nil {
		return shapeRef{}, false
	}
	return s.declaredShape(arr.ElementType)
}

// Ranging a value slice binds its elements, which carries the inference into
// the loop body.
func (v *migrateVisitor) VisitForEachLoop(loop *java.ForEachLoop, p any) java.J {
	// Recorded before the body is visited, since the body is what reads off the
	// element.
	if name, ref, ok := v.scan.rangeElementShape(loop); ok {
		v.scan.shapes[name] = ref
	}
	return v.GoVisitor.VisitForEachLoop(loop, p)
}

// v1's missing-region sentinel became a typed error in v2, so the comparison
// against it becomes an errors.As.
func (v *migrateVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	enumNilCheck := v.scan.enumNilCheck(bin)
	_, zeroEqual, zeroCompare := v.scan.zeroComparison(bin)
	zeroOnLeft := zeroCompare && v.scan.emptyShapeLiteral(bin.Left)
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)
	if zeroCompare {
		value := bin.Left
		if zeroOnLeft {
			value = bin.Right
		}
		return v.zeroCheck(bin, value, zeroEqual)
	}
	if rewritten, ok := v.missingRegionRewrite(bin); ok {
		return v.renamedErrorsQualifier(rewritten)
	}
	if enumNilCheck {
		return enumUnsetCheck(bin)
	}
	return bin
}

// enumNilCheck reports whether bin compares an enum field to nil. v2 holds an
// enum by value, and since no enum has the empty string as a value, empty is
// what v1's nil was.
func (s *fileScan) enumNilCheck(bin *java.Binary) bool {
	if bin.Operator.Element != java.Equal && bin.Operator.Element != java.NotEqual {
		return false
	}
	for _, pair := range [][2]java.Expression{{bin.Left, bin.Right}, {bin.Right, bin.Left}} {
		fa, isField := pair[0].(*java.FieldAccess)
		id, isIdent := pair[1].(*java.Identifier)
		if isField && isIdent && id.Name == "nil" {
			_, isEnum := s.enumFieldOf(fa)
			return isEnum
		}
	}
	return false
}

// enumUnsetCheck compares the enum the visit re-pointered to the empty string
// instead of nil.
func enumUnsetCheck(bin *java.Binary) *java.Binary {
	unwrap := func(e java.Expression) (java.Expression, bool) {
		call, isCall := e.(*java.MethodInvocation)
		if !isCall || call.Name == nil || call.Name.Name != "String" || len(realArgs(call)) != 1 {
			return nil, false
		}
		cast, isCast := realArgs(call)[0].(*java.TypeCast)
		if !isCast {
			return nil, false
		}
		return lstutil.SetExprPrefix(cast.Expr, e.GetPrefix()), true
	}
	empty := func(prefix java.Space) java.Expression {
		return &java.Literal{Prefix: prefix, Value: "", Source: `""`, Type: &java.JavaTypePrimitive{Keyword: "String"}}
	}
	c := *bin
	if enum, ok := unwrap(bin.Left); ok {
		c.Left, c.Right = enum, empty(bin.Right.GetPrefix())
		return &c
	}
	if enum, ok := unwrap(bin.Right); ok {
		c.Left, c.Right = empty(bin.Left.GetPrefix()), enum
		return &c
	}
	return bin
}

// feedsMatchingKeyValue reads the cursor's parent and grandparent for the
// composite a read is being written into.
func (v *migrateVisitor) feedsMatchingKeyValue(fa *java.FieldAccess) bool {
	parent := v.Cursor().Parent()
	if parent == nil || parent.Parent() == nil {
		return false
	}
	return v.scan.feedsMatchingField(fa, parent.Value(), parent.Parent().Value())
}

func (v *migrateVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	// The error a request failure wraps is rewritten on the way in, so the
	// failure is recognised beforehand.
	heldGet := false
	if configRead, _, _ := v.scan.configCredentialsCall(mi); configRead == nil && mi.Name != nil && mi.Name.Name == "Get" && mi.Select != nil && len(realArgs(mi)) == 0 {
		heldGet = v.scan.credentialsReceiver(mi.Select.Element)
	}
	if heldGet {
		visited := v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
		if ctx := v.contextExpr(); ctx != nil {
			v.needsCredentialsValue = true
			return helperCallOf(getCredentialsHelper, lstutil.SetExprPrefix(ctx, visited.Prefix), visited.Select.Element)
		}
		return visited
	}
	if call, isCredentials := qualifiedCall(mi, v.scan.credentialsPkg); isCredentials && call == "NewStaticCredentialsFromCreds" && len(realArgs(mi)) == 1 {
		visited := v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
		v.needsCredentialsValue = true
		return helperCallOf(staticCredentialHelper, lstutil.SetExprPrefix(realArgs(visited)[0], visited.Prefix))
	}
	if helper, isAws := qualifiedCall(mi, v.scan.awsPkg); isAws && helper == "NewConfig" && v.scan.builderHead(v.Cursor()) {
		return builderHeadCall(mi.Prefix)
	}
	requestFailure := v.scan.awserrRequestFailure(mi)
	// Read before the receiver is rewritten: `sess.Config.Credentials` loses
	// the Config that identifies it.
	_, credentialsCtx, credentialsCall := v.scan.configCredentialsCall(mi)
	expiresAt := v.scan.configExpiresAtCall(mi)
	// Arguments lose their v1 attribution as they are rewritten.
	printed := v.scan.printedShapeArgs(mi)
	beforeVisit := mi
	narrowedFrom, narrowedTo, narrowedByValue, narrowedRead := v.scan.narrowedValueRead(mi)
	enumValueRead := v.scan.enumValueRead(mi)
	_, stringRef, shapeString := v.scan.shapeStringCall(mi)
	validated, validatedByValue, validates := v.scan.shapeValidateCall(mi)
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	mi = v.wrapPrinted(mi, printed)
	if rewritten, ok := v.mockExpectationCall(mi); ok {
		return rewritten
	}
	if rewritten, ok := v.mockStub(mi); ok {
		return rewritten
	}
	if rewritten, ok := v.gureguCall(beforeVisit, mi); ok {
		return rewritten
	}
	if validates {
		return v.validateCall(mi, validated, validatedByValue)
	}
	if shapeString {
		if v.printedServices == nil {
			v.printedServices = map[string]bool{}
		}
		v.printedServices[stringRef.service] = true
		return lstutil.SetExprPrefix(wrapCompat(prettifyHelper, mi.Select.Element), mi.Prefix)
	}
	if mi.Name == nil {
		return mi
	}

	if inner, ok := v.scan.readSeekCloserOperand(mi); ok {
		return lstutil.SetExprPrefix(inner, mi.Prefix)
	}
	if helper, ok := qualifiedCall(mi, v.scan.awsPkg); ok {
		if managerRelocations[helper] {
			c := renameCall(mi, helper, v2S3Manager).(*java.MethodInvocation)
			sel := *c.Select
			sel.Element = v.relocatedToManager(mi.Select.Element.(*java.Identifier))
			c.Select = &sel
			return c
		}
		v.awsUsed = true
		if renamed, isHelper := valueHelperRenames[helper]; isHelper {
			// An enum v2 holds by value is the string v1's helper answered.
			if enumValueRead {
				read := realArgs(mi)[0]
				if inner, ok := v.scan.pointerValueSource(read); ok {
					return lstutil.SetExprPrefix(inner, mi.Prefix)
				}
				return lstutil.SetExprPrefix(basicConversion(lstutil.SetExprPrefix(read, java.EmptySpace), "string"), mi.Prefix)
			}
			// A field v2 narrowed is read at its own width and widened back to
			// the one v1 answered with.
			if narrowedRead {
				// A field v2 holds by value needs no helper to read.
				var read java.Expression = renameCall(mi, "To"+awsHelperFor(narrowedTo), v2Aws).(*java.MethodInvocation)
				if narrowedByValue {
					read = realArgs(mi)[0]
					// The read was handed the pointer v1 passed on, around the
					// value already widened.
					if inner, ok := v.scan.pointerValueSource(read); ok {
						return lstutil.SetExprPrefix(inner, mi.Prefix)
					}
				}
				return lstutil.SetExprPrefix(basicConversion(lstutil.SetExprPrefix(read, java.EmptySpace), narrowedFrom), mi.Prefix)
			}
			return renameCall(mi, renamed, v2Aws)
		}
		return mi
	}
	if expiresAt {
		if ctx := v.contextExpr(); ctx != nil {
			v.needsExpiresAt = true
			return helperCallOf(expiresAtHelper, lstutil.SetExprPrefix(ctx, mi.Prefix), mi.Select.Element)
		}
	}
	if ctx := credentialsCtx; credentialsCall {
		if ctx == nil {
			ctx = v.contextExpr()
		}
		if ctx != nil {
			c := renameCall(mi, "Retrieve", v2Aws+".CredentialsProvider").(*java.MethodInvocation)
			c.Arguments.Elements = []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(ctx, java.EmptySpace)}}
			return c
		}
	}
	if v.scan.partitionsCall(mi) {
		v.needsPartitions = true
		return partitionsHelperCall(mi.Prefix)
	}
	if rewritten, ok := v.chainCredentials(mi); ok {
		return rewritten
	}
	if rewritten, ok := v.assumeRoleCall(mi); ok {
		return rewritten
	}
	if code, message, orig, ok := v.scan.awserrNew(mi); ok {
		v.needsErrorHelper = true
		return helperCall(mi.Prefix, errorHelper, code, message, orig)
	}
	if requestFailure {
		v.needsErrorHelper = true
		args := realArgs(mi)
		return helperCall(mi.Prefix, requestFailureHelper, args[0], args[1], args[2])
	}
	if call, ok := qualifiedCall(mi, v.scan.credentialsPkg); ok && call == "NewStaticCredentials" {
		return renameCall(mi, "NewStaticCredentialsProvider", v2Credentials)
	}
	if call, ok := qualifiedCall(mi, v.scan.sessionPkg); ok {
		// A restructured local is already the config the session stood for.
		if call == "NewSession" || call == "New" {
			if args := realArgs(mi); len(args) == 1 {
				if name, isRestructured := v.scan.restructuredConfigVar(args[0]); isRestructured {
					return &java.Identifier{Prefix: mi.Prefix, Name: name, Type: lstutil.NamedType(v2Aws + ".Config")}
				}
			}
		}
		// Must around a value rather than a call has nothing left to do.
		if call == "Must" {
			if args := realArgs(mi); len(args) == 1 {
				if id, isIdent := args[0].(*java.Identifier); isIdent {
					return lstutil.SetExprPrefix(id, mi.Prefix)
				}
				if rewritten, ok := v.mustExpression(mi, args[0]); ok {
					return rewritten
				}
			}
		}
		switch call {
		case "NewSession":
			return v.loadConfig(mi)
		case "NewSessionWithOptions":
			return v.loadConfigFromOptions(mi)
		case "New":
			return v.sessionNewConfig(mi)
		}
	}
	if rewritten, ok := v.serviceHelperCall(mi); ok {
		return rewritten
	}
	if rewritten, ok := v.sharedCredentialsCall(mi); ok {
		return rewritten
	}
	if rewritten, ok := v.paginate(mi); ok {
		return rewritten
	}
	if rewritten, ok := v.imdsCall(mi); ok {
		return rewritten
	}
	if call, isMetadata := qualifiedCall(mi, v.scan.imdsPkg); isMetadata && call == "New" {
		return renameCall(mi, "NewFromConfig", v2Imds)
	}
	if rewritten, ok := v.waiterCall(mi); ok {
		return rewritten
	}
	if rewritten, ok := v.managerCall(mi); ok {
		return rewritten
	}
	if rewritten, ok := v.rdsAuthToken(mi); ok {
		return rewritten
	}
	if rewritten, ok := v.managerConstructor(mi); ok {
		return rewritten
	}
	if service, ok := v.scan.clientConstructorService(mi); ok {
		v.serviceUsed[service] = true
		return v.clientConstructor(mi, service)
	}
	if renamed, ok := v.awserrAccessor(mi); ok {
		return renamed
	}
	if service, ok := v.scan.clientOperationService(mi); ok && !v.shadowedReceiver(mi) {
		return v.operationWithContext(mi, v2ServicePath(service)+".Client")
	}
	// An operation reached through an interface or a struct field, recognised by
	// the input it takes. Its declaration is retyped alongside, so the call site
	// passes the context the same way.
	if service, ok := v.scan.operationInputService(mi); ok {
		return v.operationWithContext(mi, v2ServicePath(service)+".Client")
	}
	return mi
}

// contextExpr returns the context to pass: the one the enclosing function
// takes, or context.TODO() where it takes none.
func (v *migrateVisitor) contextExpr() java.Expression {
	if v.ctxName != "" {
		return &java.Identifier{Name: v.ctxName, Type: lstutil.NamedType("context.Context")}
	}
	todo := todoContextTemplate.Instantiate(template.NewMatchResult())
	if expr, ok := todo.(java.Expression); ok {
		v.needsContext = true
		return expr
	}
	return nil
}

// loadConfig replaces session.NewSession with the v2 config loader, which
// returns the same two values so the surrounding assignment is unchanged.
func (v *migrateVisitor) loadConfig(mi *java.MethodInvocation) java.J {
	ctx := v.contextExpr()
	if ctx == nil {
		return mi
	}

	args := realArgs(mi)
	if len(args) == 0 {
		if applied, built := v.loadConfigCall(ctx, nil); built {
			return applied
		}
		return mi
	}
	if local, isBuilder := v.scan.builderVar(args[0]); isBuilder {
		v.needsConfigBuilder, v.needsDisableSSL, v.needsPathStyle = true, true, true
		return builderLoadCall(mi.Prefix, local, ctx)
	}
	if v.scan.keptConfigVar(args[0]) {
		return v.keptConfigLoad(mi, ctx, args[0])
	}
	options, ok := v.scan.sessionArgOptions(args[0])
	if !ok {
		return mi
	}
	if applied, built := v.loadConfigCall(ctx, options); built {
		return applied
	}
	return mi
}

// keptConfigLoad loads the config with the settings a kept local holds as
// options: `config.LoadDefaultConfig(ctx, awsCompatConfigOptions(cfg)...)`.
func (v *migrateVisitor) keptConfigLoad(mi *java.MethodInvocation, ctx, local java.Expression) java.J {
	v.needsConfigHelper = true
	options := &java.MethodInvocation{
		Name:       &java.Identifier{Name: configOptionsHelper},
		Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(local, java.EmptySpace)}}},
		MethodType: lstutil.FuncType("", configOptionsHelper, nil),
	}
	return &java.MethodInvocation{
		Prefix:     mi.Prefix,
		Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: v.scan.configLocal, Type: lstutil.NamedType(v2Config)}},
		Name:       &java.Identifier{Name: "LoadDefaultConfig"},
		MethodType: lstutil.FuncType(v2Config, "LoadDefaultConfig", nil),
		Arguments: java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{
			{Element: lstutil.SetExprPrefix(ctx, java.EmptySpace)},
			{Element: &golang.Variadic{Prefix: java.SingleSpace, Element: options, Postfix: true}},
		}},
	}
}

// clientConstructor renames the v1 constructor and turns a trailing config into
// the functional option v2 takes in its place.
func (v *migrateVisitor) clientConstructor(mi *java.MethodInvocation, service string) java.J {
	renamed := renameCall(mi, "NewFromConfig", v2ServicePath(service)).(*java.MethodInvocation)
	args := realArgs(renamed)
	if v.pathStyleHoisted && service == s3Service && len(args) == 1 {
		if option, built := pathStyleOption(v.scan.qualifierFor(s3Service)); built {
			c := *renamed
			c.Arguments = renamed.Arguments
			c.Arguments.Elements = []java.RightPadded[java.Expression]{
				{Element: args[0]},
				{Element: lstutil.SetExprPrefix(option, java.SingleSpace)},
			}
			return &c
		}
	}
	if len(args) < 2 {
		return renamed
	}
	region, ok := v.scan.clientConfigRegion(args[1])
	if !ok || len(args) != 2 {
		return v.mergedClientConfig(renamed, args)
	}
	option, ok := regionOption(v.scan.qualifierFor(service), lstutil.SetExprPrefix(region, java.EmptySpace))
	if !ok {
		return renamed
	}
	c := *renamed
	c.Arguments = renamed.Arguments
	c.Arguments.Elements = []java.RightPadded[java.Expression]{
		{Element: args[0]},
		{Element: lstutil.SetExprPrefix(option, java.SingleSpace)},
	}
	return &c
}

// operationWithContext gives a client operation the context v2 takes, dropping
// the WithContext suffix where the caller already passed one.
func (v *migrateVisitor) operationWithContext(mi *java.MethodInvocation, declaringFQN string) java.J {
	name := mi.Name.Name
	if strings.HasSuffix(name, "WithContext") && name != "WithContext" {
		// The context is already the first argument; only the name moves.
		return renameCall(mi, strings.TrimSuffix(name, "WithContext"), declaringFQN)
	}

	args := mi.Arguments.Elements
	if len(args) != 1 {
		return mi
	}
	ctxExpr := v.contextExpr()
	if ctxExpr == nil {
		return mi
	}
	ctx := java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(ctxExpr, java.EmptySpace)}
	input := args[0]
	input.Element = lstutil.SetExprPrefix(input.Element, java.SingleSpace)

	c := *mi
	c.Arguments = mi.Arguments
	c.Arguments.Elements = []java.RightPadded[java.Expression]{ctx, input}
	c.MethodType = lstutil.FuncType(declaringFQN, name, nil)
	return &c
}

// A *session.Session handed between functions becomes the aws.Config that
// replaced it — a value, not a pointer, so the indirection goes with the type.
func (v *migrateVisitor) VisitPointerType(ptr *golang.PointerType, p any) java.J {
	// A held *credentials.Credentials is v2's provider interface.
	if v.scan.isCredentialsPointer(ptr) {
		v.awsUsed, v.needsAws = true, true
		return credentialsProviderType(ptr.Prefix, v.scan.awsQualifier())
	}
	ptr = v.GoVisitor.VisitPointerType(ptr, p).(*golang.PointerType)
	// v2's AttributeValue is the interface a pointer to a member satisfies.
	if elem, isField := ptr.Elem.(*java.FieldAccess); isField && v.scan.attributeValueRef(elem) {
		return lstutil.SetExprPrefix(elem, ptr.Prefix)
	}
	// So is any union's.
	if elem, isField := ptr.Elem.(*java.FieldAccess); isField && v.unionType(elem) {
		return lstutil.SetExprPrefix(elem, ptr.Prefix)
	}
	fa, isField := ptr.Elem.(*java.FieldAccess)
	if !isField || fa.Name.Element == nil || fa.Name.Element.Name != "Session" {
		return ptr
	}
	target, isIdent := fa.Target.(*java.Identifier)
	if !isIdent || target.Name != v.scan.sessionPkg {
		return ptr
	}
	v.awsUsed = true
	v.needsAws = true
	return &java.FieldAccess{
		Prefix: ptr.Prefix,
		Target: &java.Identifier{Name: "aws", Type: lstutil.NamedType(v2Aws)},
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: "Config"}},
		Type:   lstutil.NamedType(v2Aws + ".Config"),
	}
}

// v1 made an enum field a pointer and v2 holds it by value, so the dereference
// a reader wrote comes off.
func (v *migrateVisitor) VisitGoUnary(unary *golang.Unary, p any) java.J {
	// A field v2 narrowed is read at its own width and widened back.
	if from, narrowed := v.scan.narrowedDeref(unary); narrowed {
		visited := v.GoVisitor.VisitGoUnary(unary, p).(*golang.Unary)
		return lstutil.SetExprPrefix(basicConversion(lstutil.SetExprPrefix(visited, java.EmptySpace), from), unary.Prefix)
	}
	// v2's instance-role provider is built by a constructor taking its client.
	if unary.Operator.Element == golang.AddressOf {
		if comp, isComposite := unary.Expression.(*golang.Composite); isComposite && v.scan.ec2RoleCredsPkg != "" {
			if _, ok := v.scan.ec2RoleProviderClient(comp); ok {
				visited := v.GoVisitor.VisitGoUnary(unary, p).(*golang.Unary)
				if visitedComp, still := visited.Expression.(*golang.Composite); still {
					if call, built := v.ec2RoleProvider(visitedComp, unary.Prefix); built {
						return call
					}
				}
				return visited
			}
		}
	}
	// Checked before the children are visited, since the rewrite the Config
	// reach becomes no longer looks like the one this recognises.
	if unary.Operator.Element == golang.Indirection {
		if rewritten, target, ok := v.rewriteConfigAccess(unary.Expression); ok && target.depointered {
			read := shifted(rewritten, -target.offset)
			// The read is a subtraction now, which an operator around it binds
			// tighter than.
			if target.offset != 0 {
				switch parentTree(v.Cursor()).(type) {
				case *java.Binary, *java.Unary, *golang.Unary:
					read = parenthesized(read)
				}
			}
			return lstutil.SetExprPrefix(read, unary.Prefix)
		}
	}
	unary = v.GoVisitor.VisitGoUnary(unary, p).(*golang.Unary)
	if unary.Operator.Element != golang.Indirection {
		return unary
	}
	fa, isField := unary.Expression.(*java.FieldAccess)
	if !isField {
		return unary
	}
	if _, isValue := v.scan.depointeredFieldOf(fa); isValue {
		// v2 holds the field by value, so the dereference simply goes; unlike an
		// enum there is no named type to convert into, unless v2 also narrowed it.
		if from := v.scan.narrowedFieldOf(fa); from != "" {
			return lstutil.SetExprPrefix(basicConversion(fa, from), unary.Prefix)
		}
		return lstutil.SetExprPrefix(fa, unary.Prefix)
	}
	if _, isEnum := v.scan.enumFieldOf(fa); !isEnum && !v.scan.unresolvedEnumField(fa) {
		return unary
	}
	// Dereferencing v1's *string yielded a string, and v2's field is a named
	// string type, so the conversion keeps the expression the type its readers
	// were written against.
	return &java.TypeCast{
		Prefix: unary.Prefix,
		Clazz: &java.ControlParentheses{
			Tree: java.RightPadded[java.Expression]{Element: &java.Identifier{Name: "string", Type: lstutil.NamedType("string")}},
		},
		Expr: lstutil.SetExprPrefix(fa, java.EmptySpace),
	}
}

// A v1 service package held the client, the operation shapes, the modelled
// shapes and the enums together. v2 keeps the first two and moved the rest to a
// types sub-package, so each reference is placed by the manifest.
func (v *migrateVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if _, isBuilder := v.scan.literalBuilderField(fa); isBuilder {
		visited := v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
		renamed, _ := v.literalBuilderFieldAccess(visited)
		return renamed
	}
	// A setting v2 keeps a pointer to reaches through the same rewrite without
	// anything else changing around it.
	if rewritten, target, ok := v.rewriteConfigAccess(fa); ok && !target.depointered {
		return rewritten
	}
	_, managerClient := v.scan.managerClientCall(fa, v.Cursor())
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	if managerClient {
		return v.managerClient(fa)
	}
	if v.scan.managerPartsConstant(fa) {
		return &java.Literal{Prefix: fa.Prefix, Value: int64(10000), Source: maxUploadPartsValue, Type: lstutil.NamedType("int")}
	}
	if v.scan.managerPartsLimit(fa) && !assignedTo(v.Cursor()) {
		return lstutil.SetExprPrefix(basicConversion(lstutil.SetExprPrefix(fa, java.EmptySpace), "int"), fa.Prefix)
	}
	if name, isCredentials := qualifiedRef(fa, v.scan.credentialsPkg); isCredentials && name == "Value" {
		v.needsCredentialsValue = true
		return &java.Identifier{Prefix: fa.Prefix, Name: credentialsValueType, Type: lstutil.NamedType(credentialsValueType)}
	}
	if name, isMetadata := qualifiedRef(fa, v.scan.imdsPkg); isMetadata && imdsTypeRenames[name] != "" {
		c := *fa
		c.Name = java.LeftPadded[*java.Identifier]{Before: fa.Name.Before, Element: &java.Identifier{Prefix: fa.Name.Element.Prefix, Name: imdsTypeRenames[name]}}
		c.Type = lstutil.NamedType(v2Imds + "." + imdsTypeRenames[name])
		return &c
	}
	if v.scan.consumedAssumeRoleRefs[fa] {
		c := *fa
		c.Name = java.LeftPadded[*java.Identifier]{Before: fa.Name.Before, Element: &java.Identifier{Prefix: fa.Name.Element.Prefix, Name: "AssumeRoleOptions"}}
		c.Type = lstutil.NamedType(v2Stscreds + ".AssumeRoleOptions")
		return &c
	}
	if name, isSession := qualifiedRef(fa, v.scan.sessionPkg); isSession && name == sessionOptions && v.scan.optionsValuesOK {
		if _, isPointer := parentTree(v.Cursor()).(*golang.PointerType); isPointer {
			return v.scan.loadOptionsTypeRef(fa.Prefix)
		}
	}
	if fa.Name.Element != nil && v.embeddedClientField(matcher.TypeOfExpression(fa.Target), fa.Name.Element.Name) {
		c := *fa
		c.Name = java.LeftPadded[*java.Identifier]{Before: fa.Name.Before, Element: &java.Identifier{Prefix: fa.Name.Element.Prefix, Name: "Client"}}
		return &c
	}
	if name, isAws := qualifiedRef(fa, v.scan.awsPkg); isAws && name == awsContextType {
		v.needsContext = true
		return contextType(fa.Prefix)
	}
	if v.scan.providerNameRead(fa) {
		c := *fa
		c.Name = java.LeftPadded[*java.Identifier]{Before: fa.Name.Before, Element: &java.Identifier{Prefix: fa.Name.Element.Prefix, Name: "Source"}}
		return &c
	}
	if v.scan.attributeRead(fa) && !assignedTo(v.Cursor()) {
		return v.attributeGetterCall(fa)
	}
	if fa.Name.Element == nil {
		return fa
	}
	// A read off a shape comes first: the receiver is a value the file holds,
	// not the service package, so the lookup below would never reach it.
	// A write is the field's own; only a read is handed back the pointers v1
	// modelled it with.
	if assignedTo(v.Cursor()) {
		return fa
	}
	if helper, isMapRead := v.scan.valueMapRead(fa); isMapRead {
		v.needsCompat = true
		return wrapCompat(helper, fa)
	}
	if _, isRead := v.scan.valueSliceRead(fa); isRead {
		// v2 holds this list by value; the reader was written against v1's
		// pointers, so it is handed those back.
		v.needsCompat = true
		return wrapCompat(compatPtrs, fa)
	}
	target, ok := fa.Target.(*java.Identifier)
	if !ok {
		return fa
	}

	if v.scan.managerUploadInput(fa) {
		return v.retargetToS3(fa)
	}

	// A field v2 holds by value, read where v1 handed out a pointer. The
	// dereference case is the enclosing unary's to handle, and a read feeding a
	// field of the same kind needs nothing.
	if helper, conversion, isPointerRead := v.scan.pointerRead(fa); isPointerRead &&
		!dereferenced(v.Cursor()) && !assignedTo(v.Cursor()) && !v.feedsMatchingKeyValue(fa) {
		return v.restorePointer(fa, helper, conversion)
	}

	// v1's credentials provider interface is v2's CredentialsProvider, and the
	// providers a chain is built from have generated stand-ins.
	if name, isCredentials := qualifiedRef(fa, v.scan.credentialsPkg); isCredentials {
		if name == "Provider" {
			v.awsUsed, v.needsAws = true, true
			return &java.FieldAccess{
				Prefix: fa.Prefix,
				Target: &java.Identifier{Prefix: target.Prefix, Name: v.scan.awsQualifier(), Type: lstutil.NamedType(v2Aws)},
				Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: "CredentialsProvider"}},
				Type:   lstutil.NamedType(v2Aws + ".CredentialsProvider"),
			}
		}
		if comp, isLiteral := parentTree(v.Cursor()).(*golang.Composite); isLiteral && java.Tree(comp.TypeExpr) == v.Cursor().Value() {
			if compat, ok := v.scan.credentialProviderLiteral(comp); ok {
				v.needsCredentialsChain = true
				return &java.Identifier{Prefix: fa.Prefix, Name: compat, Type: lstutil.NamedType(compat)}
			}
		}
	}
	// v1's ConfigProvider was the interface a session satisfied; v2's config is
	// what replaced the session.
	if name, isClient := qualifiedRef(fa, v.scan.clientPkg); isClient && name == configProvider {
		v.awsUsed, v.needsAws = true, true
		return &java.FieldAccess{
			Prefix: fa.Prefix,
			Target: &java.Identifier{Prefix: target.Prefix, Name: v.scan.awsQualifier(), Type: lstutil.NamedType(v2Aws)},
			Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: "Config"}},
			Type:   lstutil.NamedType(v2Aws + ".Config"),
		}
	}
	if name, isEndpoints := qualifiedRef(fa, v.scan.endpointsPkg); isEndpoints && partitionTypes[name] != "" {
		v.needsPartitions = true
		return &java.Identifier{Prefix: fa.Prefix, Name: partitionTypes[name], Type: lstutil.NamedType(partitionTypes[name])}
	}
	if v.scan.missingEndpoint(fa) {
		v.needsErrorHelper = true
		return missingEndpointError(fa.Prefix)
	}
	if name, isAws := qualifiedRef(fa, v.scan.awsPkg); isAws && managerRelocations[name] {
		c := *fa
		c.Target = v.relocatedToManager(target)
		c.Type = lstutil.NamedType(v2S3Manager + "." + name)
		return &c
	}

	service, isService := v.scan.services[target.Name]
	if !isService {
		return fa
	}
	name := fa.Name.Element.Name

	if literal, ok := errCodeLiteral(service, name, fa.Prefix); ok {
		return literal
	}

	switch awsmanifest.Place(service, name) {
	case awsmanifest.InService:
		v.serviceUsed[service] = true
		return fa
	case awsmanifest.IsClient:
		v.serviceUsed[service] = true
		// v1 named the client for its service; v2 calls every one Client.
		c := *fa
		c.Name = java.LeftPadded[*java.Identifier]{
			Before:  fa.Name.Before,
			Element: &java.Identifier{Prefix: fa.Name.Element.Prefix, Name: "Client"},
		}
		c.Type = lstutil.NamedType(v2ServicePath(service) + ".Client")
		return &c
	case awsmanifest.InTypes, awsmanifest.InTypesConst:
		alias := awsmanifest.TypesAlias(service)
		v.needsTypes[service] = true
		v.typesAliases[alias] = true
		c := *fa
		c.Target = &java.Identifier{
			Prefix: target.Prefix,
			Name:   alias,
			Type:   lstutil.NamedType(v2ServicePath(service) + "/types"),
		}
		if renamed := awsmanifest.TypesName(service, name); renamed != name {
			c.Name = java.LeftPadded[*java.Identifier]{Before: fa.Name.Before, Element: &java.Identifier{Prefix: fa.Name.Element.Prefix, Name: renamed}}
		}
		c.Type = lstutil.NamedType(v2ServicePath(service) + "/types." + awsmanifest.TypesName(service, name))
		if awsmanifest.Place(service, name) == awsmanifest.InTypesConst && !v.enumSlot(v.Cursor()) {
			return lstutil.SetExprPrefix(basicConversion(&c, "string"), fa.Prefix)
		}
		return &c
	}
	return fa
}

// expandAwserr turns an awserr.Error assertion into smithy's errors.As form.
func (v *migrateVisitor) expandAwserr(stmt java.Statement) ([]java.Statement, bool) {
	if ma, isMulti := stmt.(*golang.MultiAssignment); isMulti && java.HasMarker[golang.ShortVarDecl](ma.Markers) {
		bound, okName, errExpr, ref, ok := v.scan.awserrAssertionOf(ma)
		if !ok {
			return nil, false
		}
		v.markAwserrMatch(ref)
		return expandAwserrStatement(ma, bound, okName, errExpr, ref, v.awserrMatcher(ref)), true
	}
	swi, isInit := stmt.(*golang.StatementWithInit)
	if !isInit {
		return nil, false
	}
	bound, okName, errExpr, ref, ok := v.scan.awserrAssertion(swi)
	if !ok {
		return nil, false
	}
	expanded, ok := expandAwserrAssertion(v.Cursor(), swi, bound, okName, errExpr, ref, v.awserrMatcher(ref))
	if !ok {
		return nil, false
	}
	v.markAwserrMatch(ref)
	return expanded, true
}

// awserrMatcher names the errors package an awserr match calls As on, or ""
// for the generated helper a file checking NoCredentialProviders calls.
func (v *migrateVisitor) awserrMatcher(ref *java.FieldAccess) string {
	if v.scan.noCredentialsChecked && !isRequestFailure(ref) {
		return ""
	}
	return v.scan.errorsLocal
}

func (v *migrateVisitor) markAwserrMatch(ref *java.FieldAccess) {
	if isRequestFailure(ref) {
		v.needsErrors, v.needsAwsHTTP = true, true
		return
	}
	if v.scan.noCredentialsChecked {
		v.needsSmithyType = true
		v.needsNoCredentials = true
		return
	}
	v.needsSmithy = true
}

// awserrAccessor renames the v1 error accessors to smithy's spelling.
func (v *migrateVisitor) awserrAccessor(mi *java.MethodInvocation) (java.J, bool) {
	if mi.Name == nil || mi.Select == nil {
		return nil, false
	}
	recv, isIdent := mi.Select.Element.(*java.Identifier)
	if isIdent && v.scan.requestFailureNames[recv.Name] {
		if renamed, isAccessor := requestFailureRenames[mi.Name.Name]; isAccessor {
			return renameCall(mi, renamed, v2AwsHTTP+".ResponseError"), true
		}
		return nil, false
	}
	renamed, isAccessor := awserrMethodRenames[mi.Name.Name]
	if !isAccessor {
		return nil, false
	}
	if !isIdent || !v.awserrVars[recv.Name] {
		return nil, false
	}
	return renameCall(mi, renamed, smithyGo+"."+smithyAPIErr), true
}

// session.Must becomes two statements, so it is expanded where statements live
// rather than where the call does.
func (v *migrateVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	var out []java.RightPadded[java.Statement]
	changed := false
	// A dropped statement's prefix carries the blank line that separated it
	// from what came before, which the statement taking its place inherits.
	var carried *java.Space
	awserrDeclared := map[string]bool{}
	for i, rp := range block.Statements {
		if carried != nil {
			rp.Element = lstutil.SetStmtPrefix(rp.Element, *carried)
			carried = nil
		}
		// The config local takes over the load, and the addressing style the v2
		// config has no room for is hoisted alongside it.
		if name, options, restructured := v.restructuresConfigDecl(rp.Element); restructured {
			if expanded, built := v.restructuredConfigDecl(rp.Element, name, options); built {
				for i, stmt := range expanded {
					after := java.EmptySpace
					if i == len(expanded)-1 {
						after = rp.After
					}
					out = append(out, java.RightPadded[java.Statement]{Element: stmt, After: after})
				}
				changed = true
				continue
			}
		}
		// A fluent setter is a call in v1 and an assignment in v2, so it is
		// swapped where it stands rather than inside the expression visit.
		if call, isCall := rp.Element.(*java.MethodInvocation); isCall {
			if appended, swapped := v.handlerStatement(call, p); swapped {
				out = append(out, java.RightPadded[java.Statement]{Element: appended, After: rp.After})
				changed = true
				continue
			}
			if assignment, swapped := v.sessionConfigAssignment(call); swapped {
				out = append(out, java.RightPadded[java.Statement]{Element: assignment, After: rp.After})
				changed = true
				continue
			}
			if assignment, swapped := v.setterAssignment(call); swapped {
				out = append(out, java.RightPadded[java.Statement]{Element: assignment, After: rp.After})
				changed = true
				continue
			}
		}
		// The config local the session took was read for its region and has
		// nothing left to hold.
		if v.dropsConfigDecl(rp.Element) || v.scan.retryerStatement(rp.Element) {
			// What opens the block takes the place of the block's first statement.
			if prefix := rp.Element.GetPrefix(); len(out) == 0 && carried == nil {
				carried = &prefix
			} else {
				carried = groupSeparator(prefix, carried)
			}
			changed = true
			continue
		}
		if expanded, ok := v.expandAwserr(rp.Element); ok {
			// Successive checks binding the same name share the first one's
			// declaration, which a second would redeclare.
			if decl, isDecl := expanded[0].(*java.VariableDeclarations); isDecl && len(decl.Variables) == 1 {
				name := decl.Variables[0].Element.Name.Name
				if awserrDeclared[name] {
					expanded = append([]java.Statement{lstutil.SetStmtPrefix(expanded[1], decl.Prefix)}, expanded[2:]...)
				}
				awserrDeclared[name] = true
			}
			for i, stmt := range expanded {
				after := java.EmptySpace
				if i == len(expanded)-1 {
					after = rp.After
				}
				out = append(out, java.RightPadded[java.Statement]{Element: stmt, After: after})
			}
			changed = true
			continue
		}
		name, newSession, isMust := v.mustSession(rp.Element)
		if !isMust {
			out = append(out, rp)
			continue
		}
		expanded, ok := v.expandMustSession(v.Cursor(), rp.Element, name, newSession, mustErrName(block.Statements[i+1:]))
		if !ok {
			out = append(out, rp)
			continue
		}
		for i, stmt := range expanded {
			after := java.EmptySpace
			if i == len(expanded)-1 {
				after = rp.After
			}
			out = append(out, java.RightPadded[java.Statement]{Element: stmt, After: after})
		}
		changed = true
	}
	if !changed {
		return block
	}
	return block.WithStatements(out)
}

// An enum field was a *string in v1 and is a named type in v2, so the aws.String
// the pointer needed comes off. Which enum the field carries is what says
// whether a given constant fits, so the rewrite happens at the composite literal
// that names the shape rather than at the call on its own.
func (v *migrateVisitor) VisitComposite(comp *golang.Composite, p any) java.J {
	cursor := v.Cursor()
	if v.scan.awsConfigLiteral(comp) && v.scan.literalBuilderDecl(cursor) {
		return v.literalBuilderComposite(v.GoVisitor.VisitComposite(comp, p).(*golang.Composite))
	}
	// An aws.Config the session does not consume is a value the file keeps, so
	// its settings are brought to the v2 shape where they are written.
	isConfig := v.scan.awsConfigLiteral(comp) && !consumedAsSessionConfig(v.scan, cursor)
	isAttribute := v.scan.attributeComposite(comp, cursor)
	unionService, unionMember, _, isUnion := v.scan.unionLiteral(comp)
	if isUnion {
		visited := v.GoVisitor.VisitComposite(comp, p).(*golang.Composite)
		return v.unionMemberLiteral(visited, unionService, unionMember)
	}
	literalType := matcher.TypeOfExpression(comp)
	if matcher.AsClass(literalType) == nil {
		literalType = matcher.TypeOfExpression(comp.TypeExpr)
	}
	comp = v.GoVisitor.VisitComposite(comp, p).(*golang.Composite)
	comp = v.renamedEmbeddedKeys(comp, literalType)
	if isAttribute {
		_, addressed := parentTree(cursor).(*golang.Unary)
		if literal, ok := v.attributeLiteral(comp, addressed); ok {
			return literal
		}
		return comp
	}
	if isConfig {
		v.reformat = false
		if retyped, rewrote := v.retypedConfigLiteral(comp); rewrote {
			// A value parsed apart takes the literal's indentation.
			if v.reformat {
				if formatted, ok := format.AutoFormat(retyped, p, nil, cursor.Parent()).(java.Expression); ok {
					return formatted
				}
			}
			return retyped
		}
		return comp
	}
	service, shape, ok := v.scan.compositeShapeAt(comp, cursor)
	if !ok {
		return comp
	}

	elements := make([]java.RightPadded[java.Expression], len(comp.Elements.Elements))
	copy(elements, comp.Elements.Elements)
	changed := false
	for i, rp := range elements {
		kv, ok := rp.Element.(*golang.KeyValue)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*java.Identifier)
		if !ok {
			continue
		}
		rewritten, rewrote := v.convertedFieldValue(service, shape, key.Name, kv.Value.Element)
		if !rewrote {
			continue
		}
		replaced := *kv
		replaced.Value = java.LeftPadded[java.Expression]{
			Before:  kv.Value.Before,
			Element: rewritten,
			Markers: kv.Value.Markers,
		}
		elements[i].Element = &replaced
		changed = true
	}
	if !changed {
		return comp
	}
	c := *comp
	c.Elements = comp.Elements
	c.Elements.Elements = elements
	return &c
}

// depointeredValue unwraps the pointer v1 needed around a field v2 holds by
// value: aws.Bool(b) and &b both carried b.
func (v *migrateVisitor) depointeredValue(expr java.Expression) (java.Expression, bool) {
	inner, wrapped := v.scan.pointerValueSource(expr)
	if !wrapped {
		return nil, false
	}
	return lstutil.SetExprPrefix(inner, expr.GetPrefix()), true
}

// convertedFieldValue puts a value into whatever v2 holds the field in. The
// answer is the same whether the field is filled in a composite literal or
// assigned to afterwards.
func (v *migrateVisitor) convertedFieldValue(service, shape, field string, value java.Expression) (java.Expression, bool) {
	if v.scan.expressionMap(value) {
		return nil, false
	}
	// nil is as good a list or map in v2 as in v1, and no helper could infer
	// the type it converts from it.
	if id, isIdent := value.(*java.Identifier); isIdent && id.Name == "nil" && (awsmanifest.IsValueSlice(service, shape, field) || awsmanifest.IsMapOfValues(service, shape, field) || awsmanifest.IsMapOfValueSlices(service, shape, field)) {
		return value, true
	}
	switch {
	case awsmanifest.Depointered(service, shape, field) != "":
		inner, ok := v.depointeredValue(value)
		if ok && awsmanifest.NarrowedFrom(service, shape, field) != "" {
			inner = narrowedValue(inner, awsmanifest.NarrowedFrom(service, shape, field), awsmanifest.Depointered(service, shape, field))
		}
		return inner, ok
	case enumSliceOf(service, shape, field) != "":
		return v.enumSliceValue(value, service, enumSliceOf(service, shape, field))
	case awsmanifest.IsMapOfValues(service, shape, field):
		v.needsCompat = true
		return wrapCompat(compatValsMap, value), true
	case awsmanifest.IsMapOfValueSlices(service, shape, field):
		// Both the map and the slices inside it changed, which the slice
		// helpers reach only one level of.
		v.needsCompat = true
		return wrapCompat(compatValsSliceMap, value), true
	case awsmanifest.IsValueSlice(service, shape, field):
		return v.valueSliceWrite(value)
	case hasScalarChange(service, shape, field):
		return v.retypedScalar(value, service, shape, field)
	}
	enum := awsmanifest.FieldEnum(service, shape, field)
	if enum == "" {
		return nil, false
	}
	return v.enumFieldValue(value, service, enum)
}

// A shape's field is as often filled after the literal as inside it, and takes
// the same conversion either way.
func (v *migrateVisitor) assignedShapeField(target java.Expression) (service, shape, field string, ok bool) {
	fa, isField := target.(*java.FieldAccess)
	if !isField || fa.Name.Element == nil {
		return "", "", "", false
	}
	owner, known := v.scan.receiverShape(fa.Target)
	if !known {
		return "", "", "", false
	}
	return owner.service, owner.shape, fa.Name.Element.Name, true
}

func hasScalarChange(service, shape, field string) bool {
	_, _, ok := awsmanifest.ScalarChange(service, shape, field)
	return ok
}

// retypedScalar rebuilds the pointer helper around a field v2 narrowed —
// aws.Int64(n) becomes aws.Int32(int32(n)) — unwrapping a v1-width conversion
// the caller wrote rather than nesting one inside the other.
func (v *migrateVisitor) retypedScalar(expr java.Expression, service, shape, field string) (java.Expression, bool) {
	from, to, ok := awsmanifest.ScalarChange(service, shape, field)
	if !ok {
		return nil, false
	}
	// The address of a value, or new of one, is a copy at v2's width;
	// narrowing needs one.
	var copied java.Expression
	if unary, isUnary := expr.(*golang.Unary); isUnary && unary.Operator.Element == golang.AddressOf {
		copied = unary.Expression
	}
	if call, isCall := expr.(*java.MethodInvocation); isCall {
		if value, isNew := newOfValue(call); isNew {
			copied = value
		}
	}
	if copied != nil {
		v.awsUsed, v.needsAws = true, true
		return lstutil.SetExprPrefix(&java.MethodInvocation{
			Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: v.scan.awsQualifier(), Type: lstutil.NamedType(v2Aws)}},
			Name:       &java.Identifier{Name: awsHelperFor(to)},
			Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: narrowedValue(lstutil.SetExprPrefix(copied, java.EmptySpace), from, to)}}},
			MethodType: lstutil.FuncType(v2Aws, awsHelperFor(to), nil),
		}, expr.GetPrefix()), true
	}
	mi, isCall := expr.(*java.MethodInvocation)
	if !isCall {
		return nil, false
	}
	helper, isAws := qualifiedCall(mi, v.scan.awsPkg)
	if !isAws || helper != awsHelperFor(from) {
		return nil, false
	}
	args := realArgs(mi)
	if len(args) != 1 {
		return nil, false
	}

	converted := narrowedValue(args[0], from, to)
	renamed := *mi
	renamed.Name = &java.Identifier{Prefix: mi.Name.Prefix, Name: awsHelperFor(to), Type: mi.Name.Type}
	renamed.MethodType = lstutil.FuncType(v2Aws, awsHelperFor(to), nil)
	renamed.Arguments = mi.Arguments
	renamed.Arguments.Elements = []java.RightPadded[java.Expression]{{Element: converted}}
	return &renamed, true
}

// narrowedValue converts a value of v1's width to the one v2 narrowed the field
// to, unwrapping a v1-width conversion the caller wrote rather than nesting one
// inside the other. An untyped constant already fits, so it is left bare.
func narrowedValue(expr java.Expression, from, to string) java.Expression {
	if cast, isCast := expr.(*java.TypeCast); isCast && castTypeName(cast) == from {
		expr = lstutil.SetExprPrefix(cast.Expr, cast.Prefix)
	}
	if _, isLiteral := expr.(*java.Literal); isLiteral {
		return expr
	}
	return lstutil.SetExprPrefix(basicConversion(expr, to), expr.GetPrefix())
}

// basicConversion wraps expr in a conversion to a basic type.
func basicConversion(expr java.Expression, basic string) java.Expression {
	return &java.TypeCast{
		Clazz: &java.ControlParentheses{
			Tree: java.RightPadded[java.Expression]{Element: &java.Identifier{Name: basic, Type: lstutil.NamedType(basic)}},
		},
		Expr: lstutil.SetExprPrefix(expr, java.EmptySpace),
	}
}

// awsHelperFor names the aws package helper that takes a value of a basic type.
func awsHelperFor(basic string) string {
	switch basic {
	case "int32":
		return "Int32"
	case "int64":
		return "Int64"
	case "float32":
		return "Float32"
	case "float64":
		return "Float64"
	case "string":
		return "String"
	}
	return ""
}

// castTypeName names the type a conversion targets, when it is a bare name.
func castTypeName(cast *java.TypeCast) string {
	if cast.Clazz == nil {
		return ""
	}
	id, ok := cast.Clazz.Tree.Element.(*java.Identifier)
	if !ok {
		return ""
	}
	return id.Name
}

// valueSliceWrite converts a v1-shaped pointer slice back to the value slice v2
// holds. A read this visit already wrapped is unwrapped instead, since
// converting it twice would be a round trip.
func (v *migrateVisitor) valueSliceWrite(expr java.Expression) (java.Expression, bool) {
	if mi, ok := expr.(*java.MethodInvocation); ok && mi.Name != nil && mi.Name.Name == compatPtrs {
		args := realArgs(mi)
		if len(args) == 1 {
			return lstutil.SetExprPrefix(args[0], expr.GetPrefix()), true
		}
	}
	v.needsCompat = true
	return wrapCompat(compatVals, expr), true
}

// enumFieldValue rewrites the value assigned to an enum field: an aws.String
// around a constant of that same enum loses the wrapper, and one around anything
// else becomes a conversion to the enum.
func (v *migrateVisitor) enumFieldValue(expr java.Expression, service, enum string) (java.Expression, bool) {
	inner, ok := v.scan.enumValueSource(expr)
	if !ok {
		// A read of another enum field is already v2's enum.
		if read, isRead := expr.(*java.FieldAccess); isRead {
			if _, isEnum := v.scan.enumFieldOf(read); isEnum {
				return nil, false
			}
		}
		v.awsUsed, v.needsAws = true, true
		toString := &java.MethodInvocation{
			Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: v.scan.awsQualifier(), Type: lstutil.NamedType(v2Aws)}},
			Name:       &java.Identifier{Name: "ToString"},
			Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(expr, java.EmptySpace)}}},
			MethodType: lstutil.FuncType(v2Aws, "ToString", nil),
		}
		return lstutil.SetExprPrefix(v.enumConversion(toString, service, enum), expr.GetPrefix()), true
	}

	// The argument was relocated on the way in, so a constant already names the
	// types alias.
	if fa, ok := inner.(*java.FieldAccess); ok && fa.Name.Element != nil {
		if target, ok := fa.Target.(*java.Identifier); ok && v.typesAliases[target.Name] {
			if awsmanifest.EnumOf(service, fa.Name.Element.Name) == enum {
				return lstutil.SetExprPrefix(fa, expr.GetPrefix()), true
			}
			// A constant of another enum fitted v1's *string field by its
			// string, which converts to v2's typed one.
		}
	}

	return lstutil.SetExprPrefix(v.enumConversion(inner, service, enum), expr.GetPrefix()), true
}

// enumConversion wraps a value computed at runtime in the enum v2 types the
// field with. Go models a conversion as a cast, not as a call on the type's
// name.
func (v *migrateVisitor) enumConversion(expr java.Expression, service, enum string) java.Expression {
	alias := awsmanifest.TypesAlias(service)
	v.needsTypes[service] = true
	v.typesAliases[alias] = true
	enumType := &java.FieldAccess{
		Target: &java.Identifier{Name: alias, Type: lstutil.NamedType(v2ServicePath(service) + "/types")},
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: enum}},
		Type:   lstutil.NamedType(v2ServicePath(service) + "/types." + enum),
	}
	return &java.TypeCast{
		Prefix: expr.GetPrefix(),
		Clazz: &java.ControlParentheses{
			Tree: java.RightPadded[java.Expression]{Element: enumType},
		},
		Expr: lstutil.SetExprPrefix(expr, java.EmptySpace),
	}
}

// renameCall returns mi with its method name replaced, keeping the receiver,
// the arguments and the whitespace. The renamed call is attributed to the v2
// package it now resolves in, so a later type-based match reads it as v2 rather
// than as the v1 name it no longer carries.
func renameCall(mi *java.MethodInvocation, name, declaringFQN string) java.J {
	c := *mi
	c.Name = &java.Identifier{Prefix: mi.Name.Prefix, Name: name, Type: mi.Name.Type}
	c.MethodType = lstutil.FuncType(declaringFQN, name, nil)
	return &c
}
