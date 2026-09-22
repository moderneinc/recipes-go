/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package newrelicotel_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/newrelicotel"
	"github.com/moderneinc/recipes-go/recipes/migration/newrelicotel/otelexportdata"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/exportdata"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// Verify names an unreadable blob directly, where the attribution sweep in
// tests/ reports it as a missing type on every emitted recording.
// See CLAUDE.md: Type Attribution for how to regenerate.
func TestOtelExportDataIsReadable(t *testing.T) {
	require.NoError(t, exportdata.Verify(otelexportdata.FS, otelexportdata.Paths...))
}

func findSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&newrelicotel.FindNewRelicTelemetrySdkUsage{})
}

// bcneng/candebot's harvester construction: none of it has a one-to-one
// OpenTelemetry form, so each option is reported with what replaces it.
func TestFindHarvesterConstruction(t *testing.T) {
	findSpec().RewriteRun(t,
		test.Golang(`
			package bot

			import "github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"

			func newHarvester(key, version string) (*telemetry.Harvester, error) {
				return telemetry.NewHarvester(
					telemetry.ConfigAPIKey(key),
					telemetry.ConfigCommonAttributes(map[string]interface{}{
						"candebot_version": version,
					}),
				)
			}
		`, `
			package bot

			import "github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"

			func newHarvester(key, version string) (*/*~~(a *telemetry.Harvester is a provider, an exporter and a batcher at once; hold a metric.Meter (and a trace.Tracer where spans are recorded) instead, with the sdkmetric.MeterProvider owned by the process)~~>*/telemetry.Harvester, error) {
				return /*~~(telemetry.NewHarvester has no single OpenTelemetry counterpart: build an OTLP exporter (go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc) behind a sdkmetric.NewMeterProvider with a PeriodicReader, and register it with otel.SetMeterProvider)~~>*/telemetry.NewHarvester(
					/*~~(telemetry.ConfigAPIKey becomes the `+"`api-key`"+` entry of otlpmetricgrpc.WithHeaders on the exporter)~~>*/telemetry.ConfigAPIKey(key),
					/*~~(telemetry.ConfigCommonAttributes becomes a resource.WithAttributes(...) passed to sdkmetric.WithResource)~~>*/telemetry.ConfigCommonAttributes(map[string]interface{}{
						"candebot_version": version,
					}),
				)
			}
		`),
	)
}

// devopsext/sre records spans and logs off the same harvester; neither has a
// mechanical rewrite, so both are reported.
func TestFindSpanAndLogRecording(t *testing.T) {
	findSpec().RewriteRun(t,
		test.Golang(`
			package provider

			import "github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"

			type nrTracer struct {
				harvester *telemetry.Harvester
			}

			func (n *nrTracer) emit(span telemetry.Span, message string) {
				_ = n.harvester.RecordSpan(span)
				_ = n.harvester.RecordLog(telemetry.Log{Message: message})
				n.harvester.HarvestNow(nil)
			}
		`, `
			package provider

			import "github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"

			type nrTracer struct {
				harvester */*~~(a *telemetry.Harvester is a provider, an exporter and a batcher at once; hold a metric.Meter (and a trace.Tracer where spans are recorded) instead, with the sdkmetric.MeterProvider owned by the process)~~>*/telemetry.Harvester
			}

			func (n *nrTracer) emit(span /*~~(telemetry.Span becomes a span started with tracer.Start and finished with span.End)~~>*/telemetry.Span, message string) {
				_ = /*~~(telemetry.Span is recorded after the fact with an explicit ID, ParentID and Duration, where an OpenTelemetry span brackets the work: replace with tracer.Start(ctx, name) and span.End(), letting the SDK derive the ids and duration from the context)~~>*/n.harvester.RecordSpan(span)
				_ = /*~~(telemetry.Log becomes a log.Record emitted through go.opentelemetry.io/otel/log)~~>*/n.harvester.RecordLog(/*~~(telemetry.Log becomes a log.Record emitted through go.opentelemetry.io/otel/log)~~>*/telemetry.Log{Message: message})
				/*~~(HarvestNow becomes ForceFlush on the sdkmetric.MeterProvider)~~>*/n.harvester.HarvestNow(nil)
			}
		`),
	)
}

// No telemetry import: nothing reported.
func TestFindQuietOnUnrelatedFile(t *testing.T) {
	findSpec().RewriteRun(t,
		test.Golang(`
			package app

			import "context"

			func tick(ctx context.Context) {}
		`),
	)
}
