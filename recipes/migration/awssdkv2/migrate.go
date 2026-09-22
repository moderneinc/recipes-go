/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsexportdata"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
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
	ifaceServices map[string]bool
	// modulePath is the go.mod `module` path, which the regenerated iface
	// packages are imported through.
	modulePath string
	// usage is what the go.mod requires are decided from, gathered here rather
	// than by a second scanning recipe — the Moderne CLI hangs on a recipe list
	// holding more than one.
	usage awsUsageAcc
	// blockers names the files the migration left alone, and what held each
	// back. A client, a shape and an interface over them cross file boundaries,
	// so the callers that did migrate may no longer agree with them — which is a
	// type error the compiler points at, and this table is the list to work
	// through. Holding the whole module back instead would mean one file's
	// unmigratable instrumentation costing every other file its migration.
	blockers map[string]string
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
	return len(a.ifaceServices) > 0 && a.modulePath == ""
}

func (r *MigrateAwsSdkGoToV2) InitialValue(*recipe.ExecutionContext) any {
	return &migrateAcc{
		blockers:       map[string]string{},
		compatPackages: map[string]string{},
		configPackages: map[string]string{},
		waitPackages:   map[string]string{},
		imdsPackages:   map[string]string{},
		helperPackages: map[string]string{},
		ifaceServices:  map[string]bool{},
		usage:          awsUsageAcc{needed: map[string]bool{}},
	}
}

func (r *MigrateAwsSdkGoToV2) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&compatScanner{acc: acc.(*migrateAcc)})
}

func (r *MigrateAwsSdkGoToV2) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&migrateVisitor{acc: acc.(*migrateAcc)})
}

// Generate writes the slice helpers once into each package that needs them.
func (r *MigrateAwsSdkGoToV2) Generate(acc any, ctx *recipe.ExecutionContext) []java.Tree {
	a := acc.(*migrateAcc)
	for _, path := range sortedKeys(a.blockers) {
		awsBlockersTable.InsertRow(ctx, AwsSdkGoV1BlockerRow{SourcePath: path, Reason: a.blockers[path]})
	}
	var out []java.Tree
	if a.ifaceUnreachable() {
		return out
	}
	// v2 deleted the iface packages outright, so a replacement is written under
	// the v2 path its importers now name.
	for service := range a.ifaceServices {
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
	for dir, pkg := range a.helperPackages {
		cu, err := parser.NewGoParser().Parse(path.Join(dir, serviceHelperFileName), fmt.Sprintf(serviceHelperSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.imdsPackages {
		cu, err := parser.NewGoParser().Parse(path.Join(dir, imdsFileName), fmt.Sprintf(imdsSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.waitPackages {
		cu, err := parser.NewGoParser().Parse(path.Join(dir, waitFileName), fmt.Sprintf(waitSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.configPackages {
		cu, err := parser.NewGoParser().Parse(path.Join(dir, configFileName), fmt.Sprintf(configHelperSource, pkg))
		if err != nil {
			continue
		}
		out = append(out, cu)
	}
	for dir, pkg := range a.compatPackages {
		cu, err := parser.NewGoParser().Parse(path.Join(dir, compatFileName), fmt.Sprintf(compatSource, pkg))
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
	if vendored(cu) {
		return cu
	}
	scan := scanFile(cu)
	if importsV1(cu) && !scan.migratable {
		v.acc.blockerReason(cu.SourcePath, scan.reason)
	}
	for _, path := range scan.ifacePaths {
		if service, _, ok := ifacePackageName(path); ok {
			v.acc.ifaceServices[service] = true
		}
	}
	if cu.PackageDecl == nil || cu.PackageDecl.Element == nil {
		return cu
	}
	probe := visitor.Init(&migrateVisitor{acc: v.acc})
	probe.Visit(cu, recipe.NewExecutionContext())
	if probe.needsServiceHelpers {
		v.acc.helperPackages[path.Dir(cu.SourcePath)] = cu.PackageDecl.Element.Name
	}
	if probe.needsImdsHelper {
		v.acc.imdsPackages[path.Dir(cu.SourcePath)] = cu.PackageDecl.Element.Name
	}
	if probe.needsWaitDuration {
		v.acc.waitPackages[path.Dir(cu.SourcePath)] = cu.PackageDecl.Element.Name
	}
	if probe.needsConfigHelper {
		v.acc.configPackages[path.Dir(cu.SourcePath)] = cu.PackageDecl.Element.Name
		// The generated loader names both packages whether or not the source it
		// stands in for did.
		v.acc.usage.needed[v2Module] = true
		v.acc.usage.needed[v2Config] = true
	}
	if probe.needsCompat {
		v.acc.compatPackages[path.Dir(cu.SourcePath)] = cu.PackageDecl.Element.Name
	}
	return cu
}

func (r *MigrateAwsSdkGoToV2) Name() string {
	return "org.openrewrite.golang.migration.MigrateAwsSdkGoToV2"
}
func (r *MigrateAwsSdkGoToV2) DisplayName() string {
	return "Migrate `aws-sdk-go` to `aws-sdk-go-v2`"
}
func (r *MigrateAwsSdkGoToV2) Description() string {
	return "Migrate `github.com/aws/aws-sdk-go`, whose support AWS ended in July 2025, to `github.com/aws/aws-sdk-go-v2`. The go directive rises to the Go 1.24 the v2 modules require; the session becomes a config loaded through a context, with each `aws.Config` field it carried — the region, the endpoint, the retry count, a static or shared credentials provider — becoming the loader option that replaces it; each client is built with `NewFromConfig`, a per-client region override becoming a functional option; and every operation takes a context. A config built field by field instead of in one literal keeps its shape: the load moves to the local's declaration and the writes that follow retarget onto v2's own config, with the S3 addressing style — which v2 holds on the client's options rather than the config — hoisted into a local the constructor reads. Shapes and enums follow the manifest to the `types` sub-package, an enum field losing the `aws.String` its `*string` needed and an enum list changing element type with it; v1's fluent setters become assignments; and a field v2 holds by value loses the dereference that read it, or regains the pointer where it was passed on — v1 could report such a field as nil and v2 cannot, so review a nil test downstream of one. The packages v2 relocated follow too: `s3manager` becomes `feature/s3/manager`, `ec2metadata` becomes `feature/ec2/imds` and `stscreds` moves up beside `credentials`, each bound back to the name the file already spells, and the per-service `iface` packages v2 deleted are regenerated into the module under `internal/awsiface`, so the interfaces mocks embed still exist. Where v2 restructured rather than renamed, a wrapper keeps the v1 call site's shape: a page iterator's callback is driven by the v2 paginator in a function literal called on the spot, a waiter becomes v2's waiter type bounded by a generated constant, an inline `session.New` becomes a generated loader that panics as `session.Must` did, `EC2Metadata.Region` goes through a generated helper, and a list or map v2 holds by value is converted at the API boundary by generated helpers so the code around it keeps its v1 shape. v1 took no context anywhere v2 takes one, so a function with none in scope gets `context.TODO()` — review those and plumb a real context through. A file migrates whole or not at all, since a half-migrated one does not compile, but the module does not: a file holding a construct with no faithful v2 form — the v1 request handler stack, a session compared to nil, a shared-credentials provider assigned to a config field, a v1-only helper the manifest cannot account for — is left as it is and listed in the blockers data table, and the rest of the module moves around it. Both requires then sit in the go.mod side by side. A client and a shape cross file boundaries, so callers of what stayed behind will not compile until it is migrated by hand; that table is the list to work through, and `FindAwsSdkGoV1Usage` marks the constructs within each file. Run `go mod tidy` afterwards to resolve the per-service modules."
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
	// configUsed records that a config load was emitted, which is the only thing
	// the session import becomes.
	configUsed bool
	// needsAws records that an aws package reference was emitted where the file
	// may not have imported it.
	needsAws bool
	// needsSmithy records that the smithy error interface was emitted.
	needsSmithy bool
	// awsUsed records that an aws package reference survived the rewrite.
	awsUsed bool
	// pathStyleHoisted records that this function carries the S3 addressing
	// style in a local of its own, which the S3 clients it builds read.
	pathStyleHoisted bool
	// needsConfigHelper records that session.New was rewritten as the generated
	// loader, which the package then needs a definition of.
	needsConfigHelper bool
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
	// needsS3 records that the s3 package was named by a rewrite of the
	// transfer manager, whose input moved there.
	needsS3 bool
	// serviceUsed records the services still referenced through their own
	// package after the relocation, so the ones emptied out lose their import.
	serviceUsed map[string]bool
	// needsCompat records that a slice conversion was emitted, so the package
	// gains the helper file that defines it.
	needsCompat bool
}

func (v *migrateVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	// The iface replacements are imported through the module's own path, so a
	// file needing one stays on v1 when there is no go.mod to name it.
	if v.acc != nil && v.acc.ifaceUnreachable() && len(scanFile(cu).ifacePaths) > 0 {
		v.acc.blockerReason(cu.SourcePath, "an iface package with no module path to regenerate it under")
		return cu
	}
	if vendored(cu) {
		return cu
	}
	v.scan = scanFile(cu)
	if !v.scan.migratable {
		return cu
	}
	// An aliased session import keeps its own name across the swap, while the
	// loader this recipe emits is spelled `config`.
	if imp := pathswap.Find(cu, v1Session); imp != nil && pathswap.Alias(imp) != "" {
		return cu
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
	v.needsAws = false
	v.configUsed = false
	v.needsCompat = false
	v.needsS3 = false
	v.needsConfigHelper = false
	v.needsWaitDuration = false
	v.needsImdsHelper = false
	v.needsServiceHelpers = false
	v.needsErrors = false
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)
	if v.needsContext {
		recipegolang.MaybeAddImport(v, contextPkg, nil, false)
	}
	if v.needsAws {
		recipegolang.MaybeAddImport(v, v2Aws, nil, false)
	}
	if v.needsSmithy || v.needsErrors {
		recipegolang.MaybeAddImport(v, errorsPkg, v.errorsAlias(), false)
	}
	if v.needsSmithy {
		recipegolang.MaybeAddImport(v, smithyGo, nil, false)
	}
	if v.needsS3 && !pathswap.Imports(cu, v1ServicePkg+s3Service) {
		recipegolang.MaybeAddImport(v, v2ServicePkg+s3Service, nil, false)
	}
	for service := range v.needsTypes {
		alias := awsmanifest.TypesAlias(service)
		recipegolang.MaybeAddImport(v, v2ServicePkg+service+"/types", &alias, false)
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
	// during the visit cannot unlearn, so the aws package is re-counted here.
	if v.awsUsed {
		v.awsUsed = referencesQualifier(cu, v.scan.awsPkg)
	}
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
		path := pathswap.Path(rp.Element)
		service, isService := strings.CutPrefix(path, v2ServicePkg)
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
		if path == v2Config && v.scan.sessionPkg != "" && !v.configUsed {
			dropped = true
			carried = groupSeparator(rp.Element.Prefix, carried)
			continue
		}
		// awserr has no v2 package; its every use became smithy or a literal.
		if path == v1Awserr {
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
	// The whitespace before the block's closing paren belongs to the last entry,
	// so dropping that entry would put the paren on the same line as its
	// neighbour.
	if last := cu.Imports.Elements[len(cu.Imports.Elements)-1]; len(kept) > 0 {
		if pathswap.Path(kept[len(kept)-1].Element) != pathswap.Path(last.Element) {
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
	if strings.Contains(prefix.Whitespace, "\n\n") {
		return &prefix
	}
	return nil
}

// The requires follow the same scan, so the module lands on v2 in one pass.
func (v *migrateVisitor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	return applyAwsRequires(gm, &v.acc.usage)
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
	for _, rp := range cu.Imports.Elements {
		path := pathswap.Path(rp.Element)
		if service, _, isIface := ifacePackageName(path); isIface {
			rules = append(rules, pathswap.Rule{Old: path, New: ifaceImportPath(v.acc.modulePath, service)})
			continue
		}
		if newPath, ok := MapPath(path); ok {
			// The transfer manager's package was renamed along with its move, so
			// it is bound back to the name the file already spells.
			rule := pathswap.Rule{Old: path, New: newPath, KeepName: path == v1S3Manager || path == v1Ec2Metadata}
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
	// rather than left in an order gofmt would not produce.
	v.DoAfterVisit((&recipegolang.OrderImports{}).Editor())
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
	outer := v.scan.scopeTo(md)
	defer v.scan.restore(outer)
	outerPathStyle := v.pathStyleHoisted
	v.pathStyleHoisted = hoistsPathStyle(md, v.scan)
	defer func() { v.pathStyleHoisted = outerPathStyle }()
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

// An operation's result is a shape the manifest names, which seeds the
// inference that locates every value slice read off it.
func (v *migrateVisitor) VisitAssignment(a *java.Assignment, p any) java.J {
	// Writing a setting v2 holds by value is the read's mirror: the reach
	// changes and the aws helper the pointer needed comes off the value. Taken
	// before the children are visited, since the rewrite no longer looks like
	// what this recognises.
	if rewritten, target, ok := v.rewriteConfigAccess(a.Variable); ok && target.depointered {
		if value, unwrapped := v.scan.pointerValueSource(a.Value.Element); unwrapped {
			c := *a
			c.Variable = rewritten
			c.Value = java.LeftPadded[java.Expression]{
				Before:  a.Value.Before,
				Element: lstutil.SetExprPrefix(value, java.SingleSpace),
				Markers: a.Value.Markers,
			}
			return &c
		}
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
	ma = v.GoVisitor.VisitMultiAssignment(ma, p).(*golang.MultiAssignment)
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
	if _, output, ok := awsmanifest.OperationShapes(service, mi.Name.Name); ok {
		return shapeRef{service: service, shape: output}, true
	}
	return shapeRef{}, false
}

// A parameter or local declared as a shape is known just as directly.
func (v *migrateVisitor) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
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
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)
	if rewritten, ok := v.missingRegionRewrite(bin); ok {
		return v.renamedErrorsQualifier(rewritten)
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
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Name == nil {
		return mi
	}

	if helper, ok := qualifiedCall(mi, v.scan.awsPkg); ok {
		v.awsUsed = true
		if renamed, isHelper := valueHelperRenames[helper]; isHelper {
			return renameCall(mi, renamed, v2Aws)
		}
		return mi
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
		return v.operationWithContext(mi, v2ServicePkg+service+".Client")
	}
	// An operation reached through an interface or a struct field, recognised by
	// the input it takes. Its declaration is retyped alongside, so the call site
	// passes the context the same way.
	if service, ok := v.scan.operationInputService(mi); ok {
		return v.operationWithContext(mi, v2ServicePkg+service+".Client")
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
	options, ok := v.scan.sessionArgOptions(args[0])
	if !ok {
		return mi
	}
	if applied, built := v.loadConfigCall(ctx, options); built {
		return applied
	}
	return mi
}

// clientConstructor renames the v1 constructor and turns a trailing config into
// the functional option v2 takes in its place.
func (v *migrateVisitor) clientConstructor(mi *java.MethodInvocation, service string) java.J {
	renamed := renameCall(mi, "NewFromConfig", v2ServicePkg+service).(*java.MethodInvocation)
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
	if len(args) != 2 {
		return renamed
	}
	region, ok := v.scan.clientConfigRegion(args[1])
	if !ok {
		return renamed
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
	ptr = v.GoVisitor.VisitPointerType(ptr, p).(*golang.PointerType)
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
	// Checked before the children are visited, since the rewrite the Config
	// reach becomes no longer looks like the one this recognises.
	if unary.Operator.Element == golang.Indirection {
		if rewritten, target, ok := v.rewriteConfigAccess(unary.Expression); ok && target.depointered {
			return lstutil.SetExprPrefix(rewritten, unary.Prefix)
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
		// enum there is no named type to convert into.
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
	// A setting v2 keeps a pointer to reaches through the same rewrite without
	// anything else changing around it.
	if rewritten, target, ok := v.rewriteConfigAccess(fa); ok && !target.depointered {
		return rewritten
	}
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	target, ok := fa.Target.(*java.Identifier)
	if !ok || fa.Name.Element == nil {
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
		c.Type = lstutil.NamedType(v2ServicePkg + service + ".Client")
		return &c
	case awsmanifest.InTypes, awsmanifest.InTypesConst:
		alias := awsmanifest.TypesAlias(service)
		v.needsTypes[service] = true
		v.typesAliases[alias] = true
		c := *fa
		c.Target = &java.Identifier{
			Prefix: target.Prefix,
			Name:   alias,
			Type:   lstutil.NamedType(v2ServicePkg + service + "/types"),
		}
		c.Type = lstutil.NamedType(v2ServicePkg + service + "/types." + name)
		return &c
	}
	return fa
}

// expandAwserr turns an awserr.Error assertion into smithy's errors.As form.
func (v *migrateVisitor) expandAwserr(stmt java.Statement) ([]java.Statement, bool) {
	swi, isInit := stmt.(*golang.StatementWithInit)
	if !isInit {
		return nil, false
	}
	bound, okName, errExpr, ok := v.scan.awserrAssertion(swi)
	if !ok {
		return nil, false
	}
	expanded, ok := expandAwserrAssertion(v.Cursor(), swi, bound, okName, errExpr, v.scan.errorsLocal)
	if !ok {
		return nil, false
	}
	v.needsSmithy = true
	return expanded, true
}

// awserrAccessor renames the v1 error accessors to smithy's spelling.
func (v *migrateVisitor) awserrAccessor(mi *java.MethodInvocation) (java.J, bool) {
	if mi.Name == nil || mi.Select == nil {
		return nil, false
	}
	renamed, isAccessor := awserrMethodRenames[mi.Name.Name]
	if !isAccessor {
		return nil, false
	}
	recv, isIdent := mi.Select.Element.(*java.Identifier)
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
	for _, rp := range block.Statements {
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
			if assignment, swapped := v.setterAssignment(call); swapped {
				out = append(out, java.RightPadded[java.Statement]{Element: assignment, After: rp.After})
				changed = true
				continue
			}
		}
		// The config local the session took was read for its region and has
		// nothing left to hold.
		if v.dropsConfigDecl(rp.Element) {
			carried = groupSeparator(rp.Element.GetPrefix(), carried)
			changed = true
			continue
		}
		if expanded, ok := v.expandAwserr(rp.Element); ok {
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
		expanded, ok := v.expandMustSession(v.Cursor(), rp.Element, name, newSession)
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
	// An aws.Config the session does not consume is a value the file keeps, so
	// its settings are brought to the v2 shape where they are written.
	isConfig := v.scan.awsConfigLiteral(comp) && !consumedAsSessionConfig(v.scan, cursor)
	comp = v.GoVisitor.VisitComposite(comp, p).(*golang.Composite)
	if isConfig {
		if retyped, rewrote := v.retypedConfigLiteral(comp); rewrote {
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
	switch {
	case awsmanifest.Depointered(service, shape, field) != "":
		return v.depointeredValue(value)
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

	inner := args[0]
	// A conversion to the v1 width is replaced rather than wrapped.
	if cast, isCast := inner.(*java.TypeCast); isCast && castTypeName(cast) == from {
		inner = cast.Expr
	}
	// An untyped constant already fits the narrower type, so it is left bare.
	var converted java.Expression = inner
	if _, isLiteral := inner.(*java.Literal); !isLiteral {
		converted = &java.TypeCast{
			Clazz: &java.ControlParentheses{
				Tree: java.RightPadded[java.Expression]{Element: &java.Identifier{Name: to, Type: lstutil.NamedType(to)}},
			},
			Expr: lstutil.SetExprPrefix(inner, java.EmptySpace),
		}
	}
	renamed := *mi
	renamed.Name = &java.Identifier{Prefix: mi.Name.Prefix, Name: awsHelperFor(to), Type: mi.Name.Type}
	renamed.MethodType = lstutil.FuncType(v2Aws, awsHelperFor(to), nil)
	renamed.Arguments = mi.Arguments
	renamed.Arguments.Elements = []java.RightPadded[java.Expression]{{Element: converted}}
	return &renamed, true
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
		return nil, false
	}

	// The argument was relocated on the way in, so a constant already names the
	// types alias.
	if fa, ok := inner.(*java.FieldAccess); ok && fa.Name.Element != nil {
		if target, ok := fa.Target.(*java.Identifier); ok && v.typesAliases[target.Name] {
			if awsmanifest.EnumOf(service, fa.Name.Element.Name) == enum {
				return lstutil.SetExprPrefix(fa, expr.GetPrefix()), true
			}
			// A constant of some other enum fitted v1's *string field and does
			// not fit v2's typed one; the guard has already blocked the file.
			return nil, false
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
		Target: &java.Identifier{Name: alias, Type: lstutil.NamedType(v2ServicePkg + service + "/types")},
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: enum}},
		Type:   lstutil.NamedType(v2ServicePkg + service + "/types." + enum),
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
