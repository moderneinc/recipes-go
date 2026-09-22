/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package newrelicotel

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Reports every telemetry SDK construct with its OpenTelemetry counterpart.
type FindNewRelicTelemetrySdkUsage struct {
	recipe.Base
}

func (r *FindNewRelicTelemetrySdkUsage) Name() string {
	return "org.openrewrite.golang.migration.FindNewRelicTelemetrySdkUsage"
}
func (r *FindNewRelicTelemetrySdkUsage) DisplayName() string {
	return "Find `newrelic-telemetry-sdk-go` usage"
}
func (r *FindNewRelicTelemetrySdkUsage) Description() string {
	return "Mark every `github.com/newrelic/newrelic-telemetry-sdk-go/telemetry` construct with the OpenTelemetry Go SDK shape that replaces it, so the parts no recipe can move mechanically — the harvester wiring, spans, events and logs — are enumerated for a hand migration."
}
func (r *FindNewRelicTelemetrySdkUsage) Tags() []string {
	return []string{"search", "migration", "newrelic", "opentelemetry", "observability"}
}

func (r *FindNewRelicTelemetrySdkUsage) Editor() recipe.TreeVisitor {
	return visitor.Init(&findUsageVisitor{})
}

type findUsageVisitor struct {
	visitor.GoVisitor
	telemetryPkg string
}

func (v *findUsageVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.telemetryPkg = telemetryQualifier(cu)
	if v.telemetryPkg == "" {
		return cu
	}
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *findUsageVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Name == nil {
		return mi
	}

	// A call on the telemetry package itself: the harvester constructor and its
	// Config options.
	if mi.Select == nil {
		return mi
	}
	if recv, ok := mi.Select.Element.(*java.Identifier); ok && recv.Name == v.telemetryPkg {
		if guidance := packageCallGuidance(mi.Name.Name); guidance != "" {
			return mi.WithMarkers(java.MarkupWarn(mi.Markers, guidance))
		}
		return mi
	}

	// A call on a harvester value, identified by the telemetry argument it takes
	// or by the name the SDK gives it.
	if guidance := harvesterCallGuidance(mi, v.telemetryPkg); guidance != "" {
		return mi.WithMarkers(java.MarkupWarn(mi.Markers, guidance))
	}
	return mi
}

func (v *findUsageVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	target, ok := fa.Target.(*java.Identifier)
	if !ok || target.Name != v.telemetryPkg || fa.Name.Element == nil {
		return fa
	}
	guidance := typeGuidance(fa.Name.Element.Name)
	if guidance == "" {
		return fa
	}
	return fa.WithMarkers(java.MarkupWarn(fa.Markers, guidance))
}

// packageCallGuidance maps a telemetry package function to its OpenTelemetry
// equivalent.
func packageCallGuidance(name string) string {
	switch name {
	case "NewHarvester":
		return "telemetry.NewHarvester has no single OpenTelemetry counterpart: build an OTLP exporter (go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc) behind a sdkmetric.NewMeterProvider with a PeriodicReader, and register it with otel.SetMeterProvider"
	case "ConfigAPIKey":
		return "telemetry.ConfigAPIKey becomes the `api-key` entry of otlpmetricgrpc.WithHeaders on the exporter"
	case "ConfigMetricsURLOverride", "ConfigSpansURLOverride", "ConfigEventsURLOverride", "ConfigLogsURLOverride":
		return "a telemetry URL override becomes otlpmetricgrpc.WithEndpoint (otlp.nr-data.net:4317 for New Relic's OTLP ingest)"
	case "ConfigCommonAttributes":
		return "telemetry.ConfigCommonAttributes becomes a resource.WithAttributes(...) passed to sdkmetric.WithResource"
	case "ConfigHarvestPeriod":
		return "telemetry.ConfigHarvestPeriod becomes sdkmetric.WithInterval on the PeriodicReader"
	case "ConfigBasicErrorLogger", "ConfigBasicDebugLogger", "ConfigBasicAuditLogger":
		return "a telemetry logger option becomes otel.SetErrorHandler, which OpenTelemetry uses for the same diagnostics"
	}
	return ""
}

// harvesterCallGuidance maps a Harvester method to its OpenTelemetry
// equivalent. The receiver's type is unresolvable when the SDK is not on the
// parse classpath, so a call is recognised by the telemetry composite it is
// handed, falling back to the method name — which only reaches here in a file
// that imports the SDK.
func harvesterCallGuidance(mi *java.MethodInvocation, telemetryPkg string) string {
	args := realArgs(mi)
	kind := ""
	if len(args) == 1 {
		kind = compositeKind(args[0], telemetryPkg)
	}

	switch mi.Name.Name {
	case "RecordMetric":
		switch kind {
		case "Count":
			return "telemetry.Count becomes a metric.Float64Counter; MigrateNewRelicMetricRecording rewrites this call where its attributes and context allow"
		case "Gauge":
			return "telemetry.Gauge becomes a metric.Float64Gauge; MigrateNewRelicMetricRecording rewrites this call where its attributes and context allow"
		case "Summary":
			return "telemetry.Summary carries a pre-aggregated count, sum, min and max, which no OpenTelemetry instrument accepts; record the observations into a metric.Float64Histogram instead of the aggregate"
		}
		return "RecordMetric becomes a recording on an OpenTelemetry instrument taken from a metric.Meter"
	case "RecordSpan":
		return "telemetry.Span is recorded after the fact with an explicit ID, ParentID and Duration, where an OpenTelemetry span brackets the work: replace with tracer.Start(ctx, name) and span.End(), letting the SDK derive the ids and duration from the context"
	case "RecordEvent":
		return "telemetry.Event has no OpenTelemetry metric or trace counterpart; emit it as a log record with go.opentelemetry.io/otel/log, or as a span event via span.AddEvent"
	case "RecordLog":
		return "telemetry.Log becomes a log.Record emitted through go.opentelemetry.io/otel/log"
	case "HarvestNow":
		return "HarvestNow becomes ForceFlush on the sdkmetric.MeterProvider"
	case "MetricAggregator":
		return "the telemetry MetricAggregator has no counterpart: OpenTelemetry instruments aggregate on their own, so replace ag.Count(name, attrs).Increment() with a Float64Counter.Add and ag.Gauge(name, attrs).Value(v) with a Float64Gauge.Record"
	}
	return ""
}

// typeGuidance maps a telemetry type reference to its OpenTelemetry equivalent.
func typeGuidance(name string) string {
	switch name {
	case "Harvester":
		return "a *telemetry.Harvester is a provider, an exporter and a batcher at once; hold a metric.Meter (and a trace.Tracer where spans are recorded) instead, with the sdkmetric.MeterProvider owned by the process"
	case "Config":
		return "telemetry.Config becomes the sdkmetric.MeterProvider and exporter options"
	case "Metric":
		return "the telemetry.Metric interface has no counterpart: an OpenTelemetry measurement is made through a typed instrument rather than a value passed to a recorder"
	case "MetricAggregator", "AggregatedCount", "AggregatedGauge", "AggregatedSummary":
		return "the telemetry aggregator types have no counterpart: OpenTelemetry instruments aggregate on their own"
	case "Count":
		return "telemetry.Count becomes a metric.Float64Counter"
	case "Gauge":
		return "telemetry.Gauge becomes a metric.Float64Gauge"
	case "Summary":
		return "telemetry.Summary becomes a metric.Float64Histogram, recording each observation rather than the aggregate"
	case "Span":
		return "telemetry.Span becomes a span started with tracer.Start and finished with span.End"
	case "Event":
		return "telemetry.Event has no counterpart; emit it as a log record or a span event"
	case "Log":
		return "telemetry.Log becomes a log.Record emitted through go.opentelemetry.io/otel/log"
	case "RequestFactory", "ClientOption":
		return "the telemetry request factories build New Relic ingest payloads directly; the OTLP exporter replaces them entirely"
	}
	return ""
}
