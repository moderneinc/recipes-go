/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package newrelicotel

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// The two telemetry metrics with a faithful OpenTelemetry instrument. A
// telemetry.Summary carries a pre-aggregated count, sum, min and max, which no
// OpenTelemetry instrument accepts, so it is reported rather than rewritten.
var instrumentFor = map[string]struct{ constructor, record string }{
	"Count": {"Float64Counter", "Add"},
	"Gauge": {"Float64Gauge", "Record"},
}

var recordingTemplates = map[string]*template.GoTemplate{
	"Count": recordingTemplate(instrumentFor["Count"].constructor, instrumentFor["Count"].record),
	"Gauge": recordingTemplate(instrumentFor["Gauge"].constructor, instrumentFor["Gauge"].record),
}

var attributeTemplates = func() map[string]*template.GoTemplate {
	m := map[string]*template.GoTemplate{}
	for _, kind := range attributeConstructors {
		m[kind] = attributeTemplate(kind)
	}
	return m
}()

// contextType attributes the context argument the emitted recording passes.
var contextType = lstutil.NamedType("context.Context")

// Rewrites harvester metric recordings as OpenTelemetry instrument recordings.
type MigrateNewRelicMetricRecording struct {
	recipe.Base
}

func (r *MigrateNewRelicMetricRecording) Name() string {
	return "org.openrewrite.golang.migration.MigrateNewRelicMetricRecording"
}
func (r *MigrateNewRelicMetricRecording) DisplayName() string {
	return "Record New Relic metrics through OpenTelemetry"
}
func (r *MigrateNewRelicMetricRecording) Description() string {
	return "Rewrite `harvester.RecordMetric(telemetry.Count{…})` and `telemetry.Gauge{…}` as an OpenTelemetry `Float64Counter.Add` / `Float64Gauge.Record` on a meter from the global provider. The instrument is created at the call site inside a scoping block, which keeps one statement replacing one; hoist it to a package-level instrument when reviewing. `Timestamp` and `Interval` are dropped, since OpenTelemetry stamps a measurement when it is collected. A call site is left alone when its attribute values are not all `string`, `bool`, `int`, `int64` or `float64`, when the enclosing function has no `context.Context` in scope, or when the harvester is a local variable the rewrite would leave unused."
}
func (r *MigrateNewRelicMetricRecording) Tags() []string {
	return []string{"migration", "newrelic", "opentelemetry", "observability"}
}

func (r *MigrateNewRelicMetricRecording) Editor() recipe.TreeVisitor {
	return visitor.Init(&metricRecordingVisitor{})
}

type metricRecordingVisitor struct {
	visitor.GoVisitor
	telemetryPkg string
	// meterName is the package clause, which names the meter the emitted
	// instruments are created on.
	meterName string
	// ctxName is the context.Context parameter of the function being visited,
	// empty when it has none.
	ctxName string
	// localNames are the names the enclosing function binds itself; a harvester
	// among them would be left unused by the rewrite.
	localNames map[string]bool
	rewrote    bool
}

func (v *metricRecordingVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.telemetryPkg = telemetryQualifier(cu)
	if v.telemetryPkg == "" || bindsAnyEmittedName(cu) {
		return cu
	}
	v.meterName = packageName(cu)
	if v.meterName == "" {
		return cu
	}

	v.rewrote = false
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)
	if !v.rewrote {
		return cu
	}

	// Added in import-path order, which is how a gofmt'd group is sorted.
	for _, path := range []string{otelAPI, otelAttribute, otelMetric} {
		recipegolang.MaybeAddImport(v, path, nil, false)
	}
	if drained, ok := visitor.DrainAfterVisits(v, cu, p).(*golang.CompilationUnit); ok {
		return drained
	}
	return cu
}

// The context to record against and the names that would be stranded are both
// properties of the enclosing function, so they are gathered on the way in. A
// function literal is a method declaration too, and it closes over its enclosing
// function, so both carry inwards: a closure with no context of its own still
// has the outer one, and a harvester the outer function declared is still
// stranded when the closure was its only reader.
func (v *metricRecordingVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	outerCtx, outerLocals := v.ctxName, v.localNames
	if name := contextParamName(md); name != "" {
		v.ctxName = name
	}
	v.localNames = union(outerLocals, locallyBoundNames(md))
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)
	v.ctxName, v.localNames = outerCtx, outerLocals
	return md
}

// union returns the names in either set.
func union(a, b map[string]bool) map[string]bool {
	out := make(map[string]bool, len(a)+len(b))
	for k := range a {
		out[k] = true
	}
	for k := range b {
		out[k] = true
	}
	return out
}

// The replacement is a block, so it is swapped in at the statement slot rather
// than returned from VisitMethodInvocation, which would also reach a call in
// expression position — where a block is not a legal replacement.
func (v *metricRecordingVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	if v.ctxName == "" {
		return block
	}

	statements := make([]java.RightPadded[java.Statement], len(block.Statements))
	copy(statements, block.Statements)
	changed := false
	for i, rp := range statements {
		mi, ok := rp.Element.(*java.MethodInvocation)
		if !ok {
			continue
		}
		replacement := v.rewriteRecordMetric(mi, visitor.NewCursor(v.Cursor(), mi))
		if replacement == nil {
			continue
		}
		statements[i].Element = replacement
		changed = true
	}
	if !changed {
		return block
	}
	v.rewrote = true
	return block.WithStatements(statements)
}

// rewriteRecordMetric returns the OpenTelemetry block replacing a
// `<harvester>.RecordMetric(telemetry.Count{…})` call, or nil when the call is
// not one this recipe migrates.
func (v *metricRecordingVisitor) rewriteRecordMetric(mi *java.MethodInvocation, cursor *visitor.Cursor) java.Statement {
	if mi.Name == nil || mi.Name.Name != "RecordMetric" || mi.Select == nil {
		return nil
	}
	// A harvester held in a local variable would be left unused, which does not
	// compile; one reached through a field or a parameter is unaffected.
	if recv, ok := mi.Select.Element.(*java.Identifier); ok && v.localNames[recv.Name] {
		return nil
	}
	args := realArgs(mi)
	if len(args) != 1 {
		return nil
	}

	kind := compositeKind(args[0], v.telemetryPkg)
	tmpl, ok := recordingTemplates[kind]
	if !ok {
		return nil
	}
	comp := args[0].(*golang.Composite)

	name, ok := compositeField(comp, "Name")
	if !ok {
		return nil
	}
	value, ok := compositeField(comp, "Value")
	if !ok {
		return nil
	}
	attrs, ok := v.attributeArgs(comp)
	if !ok {
		return nil
	}

	values := template.NewMatchResult().
		Bind(nrMeterName, &java.Literal{Source: `"` + v.meterName + `"`, Value: v.meterName}).
		Bind(nrMetricKey, detached(name)).
		Bind(nrCtx, &java.Identifier{Name: v.ctxName, Type: contextType}).
		Bind(nrValue, detached(value)).
		BindList(nrAttrs, attrs)

	applied := tmpl.Apply(cursor, values)
	if applied == nil {
		return nil
	}
	stmt, ok := applied.(java.Statement)
	if !ok {
		return nil
	}
	return stmt
}

// attributeArgs converts the Attributes map literal into attribute.KeyValue
// constructor calls. It reports false when any entry cannot be converted
// faithfully — a non-literal key, or a value whose type has no attribute
// constructor — so the whole call site is left alone rather than half moved.
func (v *metricRecordingVisitor) attributeArgs(comp *golang.Composite) ([]java.J, bool) {
	attrExpr, present := compositeField(comp, "Attributes")
	if !present {
		return nil, true
	}
	attrMap, ok := attrExpr.(*golang.Composite)
	if !ok {
		return nil, false
	}

	var out []java.J
	for _, rp := range attrMap.Elements.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		kv, ok := rp.Element.(*golang.KeyValue)
		if !ok {
			return nil, false
		}
		key, ok := kv.Key.(*java.Literal)
		if !ok {
			return nil, false
		}
		value := kv.Value.Element
		kind, ok := attributeKind(value)
		if !ok {
			return nil, false
		}
		call := attributeTemplates[kind].Instantiate(template.NewMatchResult().
			Bind(nrAttrKey, detached(key)).
			Bind(nrAttrValue, detached(value)))
		if call == nil {
			return nil, false
		}
		expr, ok := call.(java.Expression)
		if !ok {
			return nil, false
		}
		if len(out) > 0 {
			expr = lstSpace(expr)
		}
		out = append(out, expr)
	}
	return out, true
}

// packageName returns the file's package clause, which names the meter.
func packageName(cu *golang.CompilationUnit) string {
	if cu.PackageDecl == nil || cu.PackageDecl.Element == nil {
		return ""
	}
	return cu.PackageDecl.Element.Name
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
	// The parser leaves a context.Context reference in a parameter list without a
	// resolved type in some files, so the spelled name is read as well.
	fa, ok := expr.(*java.FieldAccess)
	if !ok || fa.Name.Element == nil || fa.Name.Element.Name != "Context" {
		return false
	}
	target, ok := fa.Target.(*java.Identifier)
	return ok && target.Name == contextPkg
}

// locallyBoundNames returns the identifiers md binds in its own body, which are
// the ones a rewrite could strand.
func locallyBoundNames(md *java.MethodDeclaration) map[string]bool {
	scan := visitor.Init(&localBindingScan{names: map[string]bool{}})
	if md.Body != nil {
		scan.Visit(md.Body, nil)
	}
	return scan.names
}

type localBindingScan struct {
	visitor.GoVisitor
	names map[string]bool
}

func (s *localBindingScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if id, ok := a.Variable.(*java.Identifier); ok {
		s.names[id.Name] = true
	}
	return s.GoVisitor.VisitAssignment(a, p)
}

func (s *localBindingScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	for _, rp := range ma.Variables {
		if id, ok := rp.Element.(*java.Identifier); ok {
			s.names[id.Name] = true
		}
	}
	return s.GoVisitor.VisitMultiAssignment(ma, p)
}

func (s *localBindingScan) VisitVariableDeclarator(vd *java.VariableDeclarator, p any) java.J {
	if vd.Name != nil {
		s.names[vd.Name.Name] = true
	}
	return s.GoVisitor.VisitVariableDeclarator(vd, p)
}

// detached returns expr with no leading whitespace, since a bound subtree is
// spliced into a position the template already spaces.
func detached(expr java.Expression) java.Expression {
	return lstutil.SetExprPrefix(expr, java.EmptySpace)
}

// lstSpace returns expr with a single leading space, for a position following a
// comma the template did not write.
func lstSpace(expr java.Expression) java.Expression {
	return lstutil.SetExprPrefix(expr, java.SingleSpace)
}
