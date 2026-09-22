/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

// Package newrelicotel migrates github.com/newrelic/newrelic-telemetry-sdk-go,
// which New Relic has superseded by its OpenTelemetry-native ingest, to the
// OpenTelemetry Go SDK.
//
// Unlike the other library migrations here this one is not a path swap: a
// telemetry.Harvester is a provider, an exporter and a batcher at once, and its
// metric constructs carry a timestamp and an interval that OpenTelemetry sets at
// collection time instead. Only the metric-recording call sites translate
// faithfully on their own; the harvester wiring, spans, events and logs are
// reported by FindNewRelicTelemetrySdkUsage for a hand migration.
package newrelicotel

import (
	"fmt"

	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/moderneinc/recipes-go/recipes/migration/newrelicotel/otelexportdata"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

const (
	nrModule    = "github.com/newrelic/newrelic-telemetry-sdk-go"
	nrTelemetry = nrModule + "/telemetry"

	otelAPI       = "go.opentelemetry.io/otel"
	otelMetric    = otelAPI + "/metric"
	otelAttribute = otelAPI + "/attribute"
	otelSDKMetric = otelAPI + "/sdk/metric"

	// The OpenTelemetry API release the emitted code is written against.
	// Float64Gauge, which telemetry.Gauge maps onto, arrived in 1.32.
	otelVersion = "v1.46.0"

	contextPkg = "context"
)

// emittedNames are the identifiers the metric rewrite introduces. A file already
// binding one of them would have it shadowed or captured, so the rewrite stands
// off.
var emittedNames = []string{"otel", "metric", "attribute", "nrInstrument"}

// attributeConstructors maps a Go type to the attribute package constructor that
// records a value of it. A value of any other type — including an interface or a
// type the parser could not resolve — blocks its call site rather than being
// guessed at.
var attributeConstructors = map[string]string{
	"string":  "String",
	"bool":    "Bool",
	"int":     "Int",
	"int64":   "Int64",
	"float64": "Float64",
}

// The captures the emitted code binds. Names are package-unique, as the template
// engine requires.
var (
	nrMeterName = template.Expr("nrMeterName")
	nrMetricKey = template.Expr("nrMetricKey")
	nrCtx       = template.Expr("nrCtx")
	nrValue     = template.Expr("nrValue")
	nrAttrs     = template.Expr("nrAttrs").Variadic(0, 64)
	nrAttrKey   = template.Expr("nrAttrKey")
	nrAttrValue = template.Expr("nrAttrValue")
)

// A bare block scopes the instrument, so one statement still replaces one
// statement however many recordings a function makes.
func recordingTemplate(constructor, record string) *template.GoTemplate {
	return template.StatementTemplate(fmt.Sprintf(`{
	nrInstrument, _ := otel.Meter(%s).%s(%s)
	nrInstrument.%s(%s, %s, metric.WithAttributes(%s))
}`, nrMeterName, constructor, nrMetricKey, record, nrCtx, nrValue, nrAttrs)).
		Captures(nrMeterName, nrMetricKey, nrCtx, nrValue, nrAttrs).
		Imports(otelAPI, otelMetric, otelAttribute).
		ExportData(otelexportdata.FS).
		Build()
}

// attributeTemplate builds `attribute.<kind>(key, value)`.
func attributeTemplate(kind string) *template.GoTemplate {
	return template.ExpressionTemplate(fmt.Sprintf(`attribute.%s(%s, %s)`, kind, nrAttrKey, nrAttrValue)).
		Captures(nrAttrKey, nrAttrValue).
		Imports(otelAttribute).
		ExportData(otelexportdata.FS).
		Build()
}

// telemetryQualifier returns the local name a file binds to the New Relic
// telemetry package, or "" when it does not import it usably.
func telemetryQualifier(cu *golang.CompilationUnit) string {
	return pathswap.Qualifier(cu, nrTelemetry)
}

// compositeKind returns the telemetry type a composite literal constructs —
// "Count", "Gauge", "Span" and so on — or "" when it is not one.
func compositeKind(expr java.Expression, telemetryPkg string) string {
	comp, ok := expr.(*golang.Composite)
	if !ok {
		return ""
	}
	fa, ok := comp.TypeExpr.(*java.FieldAccess)
	if !ok || fa.Name.Element == nil {
		return ""
	}
	target, ok := fa.Target.(*java.Identifier)
	if !ok || target.Name != telemetryPkg {
		return ""
	}
	return fa.Name.Element.Name
}

// compositeField returns the value a composite literal assigns to the named
// field, and whether the field is present.
func compositeField(comp *golang.Composite, name string) (java.Expression, bool) {
	for _, rp := range comp.Elements.Elements {
		kv, ok := rp.Element.(*golang.KeyValue)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*java.Identifier)
		if !ok || key.Name != name {
			continue
		}
		return kv.Value.Element, true
	}
	return nil, false
}

// attributeKind returns the attribute constructor for an expression's type, and
// whether one exists. A literal carries a primitive keyword where a variable
// carries a class named for the Go type, so both are read.
func attributeKind(expr java.Expression) (string, bool) {
	t := matcher.TypeOfExpression(expr)
	if prim, ok := t.(*java.JavaTypePrimitive); ok {
		kind, found := attributeConstructors[prim.Keyword]
		return kind, found
	}
	kind, found := attributeConstructors[matcher.GetFullyQualifiedName(t)]
	return kind, found
}

// realArgs returns the arguments of mi, skipping the Empty sentinel an empty
// argument list carries.
func realArgs(mi *java.MethodInvocation) []java.Expression {
	var out []java.Expression
	for _, rp := range mi.Arguments.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		out = append(out, rp.Element)
	}
	return out
}

// bindsAnyEmittedName reports whether cu already binds one of the names the
// rewrite introduces, through an import or a top-level declaration.
func bindsAnyEmittedName(cu *golang.CompilationUnit) bool {
	if cu == nil {
		return false
	}
	if cu.Imports != nil {
		for _, rp := range cu.Imports.Elements {
			for _, name := range emittedNames {
				if pathswap.LocalName(rp.Element) == name {
					return true
				}
			}
		}
	}
	return false
}
