/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"go/version"
	"path"
	"sort"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/mod/semver"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

//go:generate go run ./internal/genmodules
//go:generate go run ./internal/genmanifest

// moduleVersion is one `require` line.
type moduleVersion struct{ path, version string }

// Brings go.mod and go.sum into line with what the migrated source imports.
type UpdateAwsSdkDependency struct {
	recipe.ScanningBase
}

func (r *UpdateAwsSdkDependency) Name() string {
	return "org.openrewrite.golang.migration.UpdateAwsSdkDependency"
}
func (r *UpdateAwsSdkDependency) DisplayName() string {
	return "Require `aws-sdk-go-v2` instead of `aws-sdk-go`"
}
func (r *UpdateAwsSdkDependency) Description() string {
	return "Require the `github.com/aws/aws-sdk-go-v2` modules the migrated source imports — the core, `config`, `credentials`, the relocated `feature` packages and each service — along with the indirect requirements they bring, and add their go.sum lines, so the module builds without a `go mod tidy`. Drop `github.com/aws/aws-sdk-go` once no file imports it. A module the build list already selects at a higher version keeps it. Then run `go mod tidy`: the build does not need it, but it drops what the migration leaves behind — the requirements only v1 needed, the go.sum lines nothing reads any more, and an `// indirect` marker on a module the source now imports directly."
}
func (r *UpdateAwsSdkDependency) Tags() []string {
	return []string{"migration", "aws", "gomod"}
}

func (r *UpdateAwsSdkDependency) InitialValue(*recipe.ExecutionContext) any {
	acc := newAwsUsageAcc()
	return &acc
}
func (r *UpdateAwsSdkDependency) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&awsUsageScanner{acc: acc.(*awsUsageAcc)})
}
func (r *UpdateAwsSdkDependency) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&awsRequireEditor{acc: acc.(*awsUsageAcc)})
}

type awsUsageAcc struct {
	// files are what the scan saw of each file, resolved against held once the
	// scan is done, since a package migrates whole or not at all.
	files []usageFile
	// held are the directories a file keeps on v1.
	held     map[string]bool
	resolved bool
	// imported holds the v2 modules the source will import once migrated.
	imported map[string]bool
	// v1Remains is set by a file that stays on v1.
	v1Remains bool
	// migrated is set by a file that moves to v2.
	migrated bool
	// modules maps a go.mod's directory to what its build list already holds.
	modules map[string]*buildList
	plans   map[string]*requirePlan
}

// buildList is what a module requires before the migration: the versions its
// go.mod names, the versions the toolchain selected, and the go.sum hashes it
// recorded.
type buildList struct {
	// goVersion is the module's go directive before the migration raises it.
	goVersion string
	required  map[string]string
	selected  map[string]string
	hashes    map[string]string
}

// usageFile is one file's part in the requirements: whether it imports v1, and
// the v2 modules it imports once migrated.
type usageFile struct {
	dir     string
	v1      bool
	modules []string
}

func newAwsUsageAcc() awsUsageAcc {
	return awsUsageAcc{held: map[string]bool{}, imported: map[string]bool{}, modules: map[string]*buildList{}, plans: map[string]*requirePlan{}}
}

// resolve settles what the module migrates once every hold-back is known: a v1
// file in a held package stays, and its imports with it.
func (acc *awsUsageAcc) resolve() {
	if acc.resolved {
		return
	}
	acc.resolved = true
	for _, f := range acc.files {
		if f.v1 && acc.held[f.dir] {
			acc.v1Remains = true
			continue
		}
		if f.v1 {
			acc.migrated = true
		}
		for _, module := range f.modules {
			acc.imported[module] = true
		}
	}
}

type awsUsageScanner struct {
	visitor.GoVisitor
	acc *awsUsageAcc
}

// The scan asks where each file ends up rather than where it is, so the answer
// does not depend on whether the source rewrite has run yet — the Moderne CLI
// scans before a recipe list's edits are applied.
func (v *awsUsageScanner) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	recordAwsUsage(v.acc, cu)
	return cu
}

func (v *awsUsageScanner) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	recordBuildList(v.acc, gm)
	return gm
}

// recordAwsUsage notes which v2 modules a file will import once migrated.
func recordAwsUsage(acc *awsUsageAcc, cu *golang.CompilationUnit) {
	if cu.Imports == nil || vendored(cu) {
		return
	}
	f := usageFile{dir: path.Dir(cu.SourcePath), v1: importsV1(cu)}
	// A mock regenerated against v2 imports only the service.
	service := ""
	if m, isMock := ifaceMockOf(cu); isMock {
		service = m.service
	} else if m, isMock := testifyMockOf(cu); isMock {
		service = m.service
	}
	if service != "" {
		if module, ok := v2ModuleFor(v2ServicePath(service)); ok {
			f.modules = append(f.modules, module)
		}
		acc.files = append(acc.files, f)
		return
	}
	// A test the rewrite cannot carry over is commented out where the rest of
	// its package migrates, so it holds its package only through that decision.
	if f.v1 && !strings.HasSuffix(cu.SourcePath, "_test.go") && !scanFile(cu).migratable {
		acc.held[f.dir] = true
	}
	for _, rp := range cu.Imports.Elements {
		path := rp.Element.Path()
		switch service, _, isIface := ifacePackageName(path); {
		case isIface && service == managerIfaceService:
			// The regenerated manager interfaces import the manager and S3.
			if module, ok := v2ModuleFor(v2ServicePkg + s3Service); ok {
				f.modules = append(f.modules, module)
			}
			path = v2S3Manager
		case isIface:
			// The regenerated package lives in this module and imports the service.
			path = v2ServicePath(service)
		case path == v1Awserr:
			path = smithyGo
		default:
			if mapped, ok := MapPath(path); ok {
				path = mapped
			}
		}
		if module, ok := v2ModuleFor(path); ok {
			f.modules = append(f.modules, module)
		}
	}
	acc.files = append(acc.files, f)
}

// recordBuildList keeps what a go.mod requires and what its build list
// selected, which the migration must not lower.
func recordBuildList(acc *awsUsageAcc, gm *golang.GoMod) {
	bl := &buildList{required: map[string]string{}, selected: map[string]string{}, hashes: map[string]string{}}
	if mrr := java.FindMarker[golang.GoResolutionResult](gm.Markers); mrr != nil {
		bl.goVersion = mrr.GoVersion
		for _, req := range mrr.Requires {
			bl.required[req.ModulePath] = req.Version
		}
		for _, d := range mrr.ResolvedDependencies {
			// The build list comes first and holds one version per module; the
			// go.sum rows after it may name versions nothing selects.
			if _, seen := bl.selected[d.ModulePath]; !seen && mrr.ResolutionStatus == golang.GoResolutionResolved {
				bl.selected[d.ModulePath] = d.Version
			}
			if d.ModuleHash != "" {
				bl.hashes[d.ModulePath+" "+d.Version] = d.ModuleHash
			}
			if d.GoModHash != "" {
				bl.hashes[d.ModulePath+" "+d.Version+"/go.mod"] = d.GoModHash
			}
		}
	}
	acc.modules[path.Dir(gm.SourcePath)] = bl
}

// v2ModuleFor returns the module providing a v2 import path, when it is one
// the requirements table covers.
func v2ModuleFor(importPath string) (string, bool) {
	under := func(module string) bool {
		return importPath == module || strings.HasPrefix(importPath, module+"/")
	}
	if under(smithyGo) {
		return smithyGo, true
	}
	if under(v2Aws) {
		return v2Module, true
	}
	best := ""
	for module := range v2Requirements {
		if module != v2Module && under(module) && len(module) > len(best) {
			best = module
		}
	}
	return best, best != ""
}

// requirePlan is the requirements a module ends up with, and the go.sum lines
// they need.
type requirePlan struct {
	versions map[string]string
	sums     map[string]string
	// goVersion is the highest go directive an imported module requires.
	goVersion string
}

// plan resolves what the go.mod in dir requires once migrated: each imported
// module's requirements from the table, highest version winning, unless the
// build list already holds a higher one.
func (acc *awsUsageAcc) plan(dir string) *requirePlan {
	acc.resolve()
	if p, ok := acc.plans[dir]; ok {
		return p
	}
	bl := acc.modules[dir]
	if bl == nil {
		bl = &buildList{}
	}
	p := &requirePlan{versions: map[string]string{}, sums: map[string]string{}, goVersion: v2MinGo}
	raise := func(module, version string) {
		if version != "" && semver.Compare(version, p.versions[module]) > 0 {
			p.versions[module] = version
		}
	}
	for module := range acc.imported {
		for _, mv := range v2Requirements[module] {
			raise(mv.path, mv.version)
		}
		if goVersion := v2GoVersions[module]; goVersion != "" && version.Compare("go"+goVersion, "go"+p.goVersion) > 0 {
			p.goVersion = goVersion
		}
	}
	for module := range p.versions {
		raise(module, bl.selected[module])
		raise(module, bl.required[module])
	}
	for module := range acc.imported {
		for _, key := range v2SumKeys[module] {
			mod, version, _ := strings.Cut(key, " ")
			// A module's own hash is only needed at the version selected; the
			// go.mod hashes of the others are read building the graph.
			if strings.HasSuffix(version, "/go.mod") || p.versions[mod] == version {
				p.sums[key] = v2Sums[key]
			}
		}
	}
	for module, version := range p.versions {
		for _, key := range []string{module + " " + version, module + " " + version + "/go.mod"} {
			if _, ok := p.sums[key]; !ok && bl.hashes[key] != "" {
				p.sums[key] = bl.hashes[key]
			}
		}
	}
	acc.plans[dir] = p
	return p
}

type awsRequireEditor struct {
	visitor.GoVisitor
	acc *awsUsageAcc
}

func (v *awsRequireEditor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	return applyAwsRequires(gm, v.acc)
}

func (v *awsRequireEditor) VisitGoSum(gs *golang.GoSum, p any) java.Tree {
	return applyAwsSums(gs, v.acc)
}

// applyAwsRequires brings the go.mod requires into line with what the migrated
// source imports. Everything the source imports directly is required directly;
// the rest of what those modules need is marked indirect.
func applyAwsRequires(gm *golang.GoMod, acc *awsUsageAcc) java.Tree {
	acc.resolve()
	if !acc.migrated {
		return gm
	}
	plan := acc.plan(path.Dir(gm.SourcePath))
	modules := make([]string, 0, len(plan.versions))
	for module := range plan.versions {
		modules = append(modules, module)
	}
	// Added in module-path order so the block does not depend on the order the
	// scan happened to see files in.
	sort.Strings(modules)
	// Direct requirements first, so a go.mod with no require block yet gets the
	// direct block ahead of the indirect one, as tidy lays them out.
	for _, indirect := range []bool{false, true} {
		for _, module := range modules {
			if acc.imported[module] != indirect {
				gm = migration.RequireAtLeast(gm, module, plan.versions[module], indirect)
			}
		}
	}
	if !acc.v1Remains {
		gm = migration.RemoveRequire(gm, v1Module)
	}
	mrr := java.FindMarker[golang.GoResolutionResult](gm.Markers)
	below117 := mrr != nil && mrr.GoVersion != "" && version.Compare("go"+mrr.GoVersion, "go1.17") < 0
	gm = raiseGoDirective(gm, plan.goVersion)
	if below117 {
		// Past 1.17 go.mod must name every module the build reads, which one
		// written for an earlier release left to the graph. Their go.sum lines
		// are already there, since the build read them before.
		gm = migration.AddMissingRequires(gm, mrr, func(modulePath string) bool {
			return modulePath == v1Module && !acc.v1Remains
		})
	}
	return gm
}

// applyAwsSums adds the go.sum lines the new requirements need, in the order
// the toolchain keeps them.
func applyAwsSums(gs *golang.GoSum, acc *awsUsageAcc) java.Tree {
	acc.resolve()
	if !acc.migrated {
		return gs
	}
	plan := acc.plan(path.Dir(gs.SourcePath))
	present := map[string]bool{}
	for _, rp := range gs.Lines {
		present[sumKey(rp.Element)] = true
	}
	lines := append([]java.RightPadded[*golang.GoSumLine]{}, gs.Lines...)
	added := false
	for key, hash := range plan.sums {
		if present[key] {
			continue
		}
		mod, version, _ := strings.Cut(key, " ")
		version, goMod := strings.CutSuffix(version, "/go.mod")
		lines = append(lines, java.RightPadded[*golang.GoSumLine]{
			Element: &golang.GoSumLine{
				Ident:      uuid.New(),
				Markers:    java.EmptyMarkers,
				ModulePath: mod,
				Version:    version,
				GoMod:      goMod,
				Hash:       hash,
			},
			After:   java.MakeSpace(nil, "\n"),
			Markers: java.EmptyMarkers,
		})
		added = true
	}
	if !added {
		return gs
	}
	if n := len(gs.Lines); n > 0 && !strings.HasSuffix(gs.Lines[n-1].After.Whitespace(), "\n") {
		lines[n-1].After = java.MakeSpace(gs.Lines[n-1].After.Comments(), gs.Lines[n-1].After.Whitespace()+"\n")
	}
	sort.SliceStable(lines, func(i, j int) bool { return sumLess(lines[i].Element, lines[j].Element) })
	return gs.WithLines(lines)
}

func sumKey(l *golang.GoSumLine) string {
	if l.GoMod {
		return l.ModulePath + " " + l.Version + "/go.mod"
	}
	return l.ModulePath + " " + l.Version
}

// sumLess orders go.sum lines as the toolchain writes them: by module path, then
// version, a module's hash ahead of its go.mod's.
func sumLess(a, b *golang.GoSumLine) bool {
	if a.ModulePath != b.ModulePath {
		return a.ModulePath < b.ModulePath
	}
	if c := semver.Compare(a.Version, b.Version); c != 0 {
		return c < 0
	}
	return !a.GoMod && b.GoMod
}

// raiseGoDirective lifts the `go` directive to the release aws-sdk-go-v2
// requires, since the module cannot build against v2 below it.
func raiseGoDirective(gm *golang.GoMod, minVersion string) *golang.GoMod {
	statements := make([]java.RightPadded[golang.GoModStatement], len(gm.Statements))
	copy(statements, gm.Statements)
	changed := false
	for i, rp := range statements {
		d, ok := rp.Element.(*golang.GoModDirective)
		if !ok || d.Keyword != "go" || len(d.Values) != 1 {
			continue
		}
		if version.Compare("go"+d.Values[0].Text, "go"+minVersion) >= 0 {
			continue
		}
		values := []*golang.GoModValue{d.Values[0].WithText(minVersion)}
		statements[i].Element = d.WithValues(values)
		changed = true
	}
	if !changed {
		return gm
	}
	return gm.WithStatements(statements)
}
