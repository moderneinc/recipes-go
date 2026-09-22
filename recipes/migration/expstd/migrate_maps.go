/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd

import (
	"fmt"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var mapsRules = []pathswap.Rule{{Old: expMaps, New: stdMaps}}

// The standard library's Keys and Values yield an iter.Seq where x/exp's
// returned a slice, so each call is wrapped rather than repointed. The templates
// resolve `slices` and `maps` against the standard library, which attributes
// both the wrapper and the inner call.
var (
	esKeysMap   = template.Expr("esKeysMap")
	esValuesMap = template.Expr("esValuesMap")

	collectKeysTemplate = template.ExpressionTemplate(
		fmt.Sprintf(`slices.Collect(maps.Keys(%s))`, esKeysMap),
	).Imports(stdSlices, stdMaps).Build()

	collectValuesTemplate = template.ExpressionTemplate(
		fmt.Sprintf(`slices.Collect(maps.Values(%s))`, esValuesMap),
	).Imports(stdSlices, stdMaps).Build()

	esSortedMap = template.Expr("esSortedMap")

	sortedTemplates = map[string]*template.GoTemplate{
		"Keys": template.ExpressionTemplate(
			fmt.Sprintf(`slices.Sorted(maps.Keys(%s))`, esSortedMap),
		).Imports(stdSlices, stdMaps).Build(),
		"Values": template.ExpressionTemplate(
			fmt.Sprintf(`slices.Sorted(maps.Values(%s))`, esSortedMap),
		).Imports(stdSlices, stdMaps).Build(),
	}
)

// sortCalls are the sorts that leave a key or value slice in ascending order, so
// that collecting and then sorting is exactly slices.Sorted. Each requires an
// ordered element type already, which is what makes the collapse safe; a
// sort.Slice with its own comparator is not one of them.
var sortCalls = map[string]bool{
	"slices.Sort":   true,
	"sort.Strings":  true,
	"sort.Ints":     true,
	"sort.Float64s": true,
}

// Migrates golang.org/x/exp/maps to the standard library.
type MigrateXExpMapsToStdlib struct {
	recipe.ScanningBase
}

func (r *MigrateXExpMapsToStdlib) Name() string {
	return "org.openrewrite.golang.migration.MigrateXExpMapsToStdlib"
}
func (r *MigrateXExpMapsToStdlib) DisplayName() string {
	return "Migrate `golang.org/x/exp/maps` to `maps`"
}
func (r *MigrateXExpMapsToStdlib) Description() string {
	return "Migrate `golang.org/x/exp/maps` to the standard library. `Clone`, `Copy`, `Equal`, `EqualFunc` and `DeleteFunc` carry over unchanged; `Clear(m)` becomes the `clear(m)` builtin; and `Keys(m)` and `Values(m)`, which return an `iter.Seq` in the standard library rather than a slice, are wrapped as `slices.Collect(maps.Keys(m))`. A file using `Keys` or `Values` needs Go 1.23, the rest Go 1.21."
}
func (r *MigrateXExpMapsToStdlib) Tags() []string {
	return []string{"migration", "maps", "stdlib"}
}

func (r *MigrateXExpMapsToStdlib) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "exptostd", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *MigrateXExpMapsToStdlib) InitialValue(*recipe.ExecutionContext) any {
	return newModuleAcc()
}
func (r *MigrateXExpMapsToStdlib) Scanner(acc any) recipe.TreeVisitor {
	return scanGoVersion(acc.(*moduleAcc))
}
func (r *MigrateXExpMapsToStdlib) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&migrateMapsVisitor{acc: acc.(*moduleAcc)})
}

type migrateMapsVisitor struct {
	visitor.GoVisitor
	acc *moduleAcc
	pkg string
	// needsSlices records that a Keys or Values call was wrapped, so the file
	// needs the slices import the wrapper reads.
	needsSlices bool
	// ranged holds the Keys and Values calls a `for _, k := range` iterates
	// directly. The standard library's iterator forms are rangeable as they are,
	// so those keep their shape instead of being collected into a slice.
	ranged map[*java.MethodInvocation]bool
	// sorted holds the Keys and Values calls whose result the next statement
	// sorts, which slices.Sorted does in one step.
	sorted map[*java.MethodInvocation]bool
}

func (v *migrateMapsVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	pkg := pathswap.Qualifier(cu, expMaps)
	if pkg == "" || !v.acc.atLeast(go121) {
		return cu
	}
	// Both imports would bind the same package name.
	if pathswap.Imports(cu, stdMaps) {
		return cu
	}
	// The wrapper is emitted under the plain `slices` name, so a file binding it
	// to anything but the standard library package would resolve it wrongly.
	if bindsName(cu, stdSlices) && !pathswap.Imports(cu, stdSlices) {
		return cu
	}
	// Keys and Values only have a standard-library form from Go 1.23. The import
	// has to move as a unit, so a file using them on an older module is left
	// whole rather than half migrated.
	if usesIteratorForms(cu, pkg) && !v.acc.atLeast(go123) {
		return cu
	}

	v.pkg = pkg
	v.needsSlices = false
	v.ranged = directlyRangedCalls(cu, pkg)
	v.sorted = sortedRightAfterCalls(cu, pkg)
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)

	swapped := pathswap.RewriteImports(cu, mapsRules)
	if retyped, ok := pathswap.Retype(mapsRules).Visit(swapped, p).(*golang.CompilationUnit); ok {
		swapped = retyped
	}
	if v.needsSlices {
		recipegolang.MaybeAddImport(v, stdSlices, nil, false)
	}
	if drained, ok := visitor.DrainAfterVisits(v, swapped, p).(*golang.CompilationUnit); ok {
		return drained
	}
	return swapped
}

func (v *migrateMapsVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if v.ranged[mi] || v.sorted[mi] {
		return mi
	}
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	name, ok := qualifiedCall(mi, v.pkg)
	if !ok {
		return mi
	}
	args := realArgs(mi)
	if len(args) != 1 {
		return mi
	}

	switch name {
	case "Keys":
		return v.collect(mi, collectKeysTemplate, esKeysMap, args[0])
	case "Values":
		return v.collect(mi, collectValuesTemplate, esValuesMap, args[0])
	case "Clear":
		// The builtin supersedes the helper entirely, and the parser leaves a
		// builtin call untyped, so dropping the receiver is the whole rewrite.
		c := *mi
		c.Select = nil
		c.Name = &java.Identifier{Name: "clear"}
		c.MethodType = nil
		return &c
	}
	return mi
}

// collect wraps a slice-returning x/exp call in slices.Collect over the standard
// library's iterator form, keeping the map expression as written.
func (v *migrateMapsVisitor) collect(mi *java.MethodInvocation, tmpl *template.GoTemplate, capture *template.Capture, mapExpr java.Expression) java.J {
	values := template.NewMatchResult().Bind(capture, mapExpr)
	applied := tmpl.Apply(v.Cursor(), values)
	if applied == nil {
		return mi
	}
	v.needsSlices = true
	return applied
}

// usesIteratorForms reports whether cu calls the x/exp Keys or Values whose
// standard-library counterparts return an iterator.
func usesIteratorForms(cu *golang.CompilationUnit, pkg string) bool {
	scan := visitor.Init(&iteratorFormScan{pkg: pkg})
	scan.Visit(cu, nil)
	return scan.found
}

type iteratorFormScan struct {
	visitor.GoVisitor
	pkg   string
	found bool
}

func (s *iteratorFormScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if name, ok := qualifiedCall(mi, s.pkg); ok && (name == "Keys" || name == "Values") {
		s.found = true
	}
	return s.GoVisitor.VisitMethodInvocation(mi, p)
}

// A `for _, k := range maps.Keys(m)` iterates the standard library's iterator
// directly, so the blank index the slice form needed goes away with it.
func (v *migrateMapsVisitor) VisitForEachLoop(loop *java.ForEachLoop, p any) java.J {
	loop = v.GoVisitor.VisitForEachLoop(loop, p).(*java.ForEachLoop)
	mi, ok := loop.Control.Iterable.Element.(*java.MethodInvocation)
	if !ok || !v.ranged[mi] {
		return loop
	}
	assignment, ok := loop.Control.Variable.Element.(*golang.MultiAssignment)
	if !ok {
		return loop
	}

	// The value variable moves into the key slot, taking that slot's leading
	// whitespace so the header still reads `for k := range …`.
	value := assignment.Variables[1]
	value.Element = lstutil.SetExprPrefix(value.Element, assignment.Variables[0].Element.GetPrefix())
	value.After = assignment.Variables[1].After

	rewritten := *assignment
	rewritten.Variables = []java.RightPadded[java.Expression]{value}
	control := loop.Control
	control.Variable = java.RightPadded[java.Statement]{
		Element: &rewritten,
		After:   loop.Control.Variable.After,
		Markers: loop.Control.Variable.Markers,
	}
	c := *loop
	c.Control = control
	return &c
}

// directlyRangedCalls returns the Keys and Values calls a `for _, k := range`
// iterates, which the standard library's iterator forms cover without being
// collected. A single-variable range is excluded: over the x/exp slice it yields
// indices, and over the iterator it would yield keys instead.
func directlyRangedCalls(cu *golang.CompilationUnit, pkg string) map[*java.MethodInvocation]bool {
	scan := visitor.Init(&rangedCallScan{pkg: pkg, found: map[*java.MethodInvocation]bool{}})
	scan.Visit(cu, nil)
	return scan.found
}

type rangedCallScan struct {
	visitor.GoVisitor
	pkg   string
	found map[*java.MethodInvocation]bool
}

func (s *rangedCallScan) VisitForEachLoop(loop *java.ForEachLoop, p any) java.J {
	if mi, ok := loop.Control.Iterable.Element.(*java.MethodInvocation); ok && blankKeyedRange(loop) {
		if name, ok := qualifiedCall(mi, s.pkg); ok && (name == "Keys" || name == "Values") && len(realArgs(mi)) == 1 {
			s.found[mi] = true
		}
	}
	return s.GoVisitor.VisitForEachLoop(loop, p)
}

// blankKeyedRange reports whether a loop header discards the index, as
// `for _, k := range` does.
func blankKeyedRange(loop *java.ForEachLoop) bool {
	assignment, ok := loop.Control.Variable.Element.(*golang.MultiAssignment)
	if !ok || len(assignment.Variables) != 2 {
		return false
	}
	blank, ok := assignment.Variables[0].Element.(*java.Identifier)
	if !ok || blank.Name != "_" {
		return false
	}
	_, ok = assignment.Variables[1].Element.(*java.Identifier)
	return ok
}

// A key or value slice that the next statement sorts is exactly what
// slices.Sorted produces, so the two statements collapse into one. coroot/coroot
// writes both spellings of the sort.
func (v *migrateMapsVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	if len(v.sorted) == 0 {
		return block
	}

	kept := make([]java.RightPadded[java.Statement], 0, len(block.Statements))
	changed := false
	for i := 0; i < len(block.Statements); i++ {
		rp := block.Statements[i]
		collapsed, ok := v.collapseSortedAt(block.Statements, i)
		if !ok {
			kept = append(kept, rp)
			continue
		}
		rp.Element = collapsed
		kept = append(kept, rp)
		// The sort the assignment absorbed goes with it.
		i++
		changed = true
	}
	if !changed {
		return block
	}
	return block.WithStatements(kept)
}

// collapseSortedAt returns the assignment at i rewritten to slices.Sorted, when
// it assigns a Keys or Values result that the statement at i+1 sorts.
func (v *migrateMapsVisitor) collapseSortedAt(statements []java.RightPadded[java.Statement], i int) (java.Statement, bool) {
	if i+1 >= len(statements) {
		return nil, false
	}
	assignment, ok := statements[i].Element.(*java.Assignment)
	if !ok {
		return nil, false
	}
	target, ok := assignment.Variable.(*java.Identifier)
	if !ok {
		return nil, false
	}
	mi, ok := assignment.Value.Element.(*java.MethodInvocation)
	if !ok || !v.sorted[mi] {
		return nil, false
	}
	if !sortsVariable(statements[i+1].Element, target.Name) {
		return nil, false
	}

	name, _ := qualifiedCall(mi, v.pkg)
	tmpl := sortedTemplates[name]
	if tmpl == nil {
		return nil, false
	}
	values := template.NewMatchResult().Bind(esSortedMap, detachedArg(realArgs(mi)[0]))
	replacement := tmpl.Instantiate(values)
	if replacement == nil {
		return nil, false
	}
	expr, ok := replacement.(java.Expression)
	if !ok {
		return nil, false
	}

	v.needsSlices = true
	c := *assignment
	c.Value = java.LeftPadded[java.Expression]{
		Before:  assignment.Value.Before,
		Element: lstutil.SetExprPrefix(expr, assignment.Value.Element.GetPrefix()),
		Markers: assignment.Value.Markers,
	}
	return &c, true
}

// sortsVariable reports whether stmt is one of the ascending sorts applied to
// the named variable.
func sortsVariable(stmt java.Statement, name string) bool {
	mi, ok := stmt.(*java.MethodInvocation)
	if !ok || mi.Name == nil || mi.Select == nil {
		return false
	}
	pkg, ok := mi.Select.Element.(*java.Identifier)
	if !ok {
		return false
	}
	if !sortCalls[pkg.Name+"."+mi.Name.Name] {
		return false
	}
	args := realArgs(mi)
	if len(args) != 1 {
		return false
	}
	arg, ok := args[0].(*java.Identifier)
	return ok && arg.Name == name
}

// sortedRightAfterCalls returns the Keys and Values calls whose assigned result
// the very next statement sorts.
func sortedRightAfterCalls(cu *golang.CompilationUnit, pkg string) map[*java.MethodInvocation]bool {
	scan := visitor.Init(&sortedCallScan{pkg: pkg, found: map[*java.MethodInvocation]bool{}})
	scan.Visit(cu, nil)
	return scan.found
}

type sortedCallScan struct {
	visitor.GoVisitor
	pkg   string
	found map[*java.MethodInvocation]bool
}

func (s *sortedCallScan) VisitBlock(block *java.Block, p any) java.J {
	for i := 0; i+1 < len(block.Statements); i++ {
		assignment, ok := block.Statements[i].Element.(*java.Assignment)
		if !ok {
			continue
		}
		target, ok := assignment.Variable.(*java.Identifier)
		if !ok {
			continue
		}
		mi, ok := assignment.Value.Element.(*java.MethodInvocation)
		if !ok {
			continue
		}
		name, ok := qualifiedCall(mi, s.pkg)
		if !ok || (name != "Keys" && name != "Values") || len(realArgs(mi)) != 1 {
			continue
		}
		if sortsVariable(block.Statements[i+1].Element, target.Name) {
			s.found[mi] = true
		}
	}
	return s.GoVisitor.VisitBlock(block, p)
}

// detachedArg returns expr with no leading whitespace, for splicing into a
// template position that already spaces it.
func detachedArg(expr java.Expression) java.Expression {
	return lstutil.SetExprPrefix(expr, java.EmptySpace)
}
