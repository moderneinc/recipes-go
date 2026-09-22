/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package newrelicotel_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/newrelicotel"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func recordingSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&newrelicotel.MigrateNewRelicMetricRecording{})
}

// bcneng/candebot counts filtered messages off a harvester held on its bot
// context. A field-held harvester survives the rewrite, so the counter moves.
func TestRecordCountFromHarvesterField(t *testing.T) {
	recordingSpec().RewriteRun(t,
		test.Golang(`
			package handlers

			import (
				"context"

				"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
			)

			type messageEvent struct {
				Channel string
				Filter  string
			}

			type botContext struct {
				Harvester *telemetry.Harvester
			}

			func handleMessage(ctx context.Context, botCtx botContext, event messageEvent) {
				botCtx.Harvester.RecordMetric(telemetry.Count{
					Name: "candebot.inclusion.message_filtered",
					Attributes: map[string]interface{}{
						"channel": event.Channel,
						"filter":  event.Filter,
					},
					Value: 1,
				})
			}
		`, `
			package handlers

			import (
				"context"

				"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
				"go.opentelemetry.io/otel"
				"go.opentelemetry.io/otel/attribute"
				"go.opentelemetry.io/otel/metric"
			)

			type messageEvent struct {
				Channel string
				Filter  string
			}

			type botContext struct {
				Harvester *telemetry.Harvester
			}

			func handleMessage(ctx context.Context, botCtx botContext, event messageEvent) {
				{
					nrInstrument, _ := otel.Meter("handlers").Float64Counter("candebot.inclusion.message_filtered")
					nrInstrument.Add(ctx, 1, metric.WithAttributes(attribute.String("channel", event.Channel), attribute.String("filter", event.Filter)))
				}
			}
		`),
	)
}

// A gauge records rather than adds.
func TestRecordGauge(t *testing.T) {
	recordingSpec().RewriteRun(t,
		test.Golang(`
			package instrumentation

			import (
				"context"

				"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
			)

			type agent struct {
				harvester *telemetry.Harvester
			}

			func (a *agent) record(ctx context.Context, name string, value float64, host string) {
				a.harvester.RecordMetric(telemetry.Gauge{
					Name:       name,
					Value:      value,
					Attributes: map[string]interface{}{"hostname": host},
				})
			}
		`, `
			package instrumentation

			import (
				"context"

				"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
				"go.opentelemetry.io/otel"
				"go.opentelemetry.io/otel/attribute"
				"go.opentelemetry.io/otel/metric"
			)

			type agent struct {
				harvester *telemetry.Harvester
			}

			func (a *agent) record(ctx context.Context, name string, value float64, host string) {
				{
					nrInstrument, _ := otel.Meter("instrumentation").Float64Gauge(name)
					nrInstrument.Record(ctx, value, metric.WithAttributes(attribute.String("hostname", host)))
				}
			}
		`),
	)
}

// A metric with no attributes records against an empty option set.
func TestRecordCountWithoutAttributes(t *testing.T) {
	recordingSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				"context"

				"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
			)

			type svc struct {
				h *telemetry.Harvester
			}

			func (s *svc) tick(ctx context.Context) {
				s.h.RecordMetric(telemetry.Count{Name: "ticks", Value: 1})
			}
		`, `
			package app

			import (
				"context"

				"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
				"go.opentelemetry.io/otel"
				"go.opentelemetry.io/otel/attribute"
				"go.opentelemetry.io/otel/metric"
			)

			type svc struct {
				h *telemetry.Harvester
			}

			func (s *svc) tick(ctx context.Context) {
				{
					nrInstrument, _ := otel.Meter("app").Float64Counter("ticks")
					nrInstrument.Add(ctx, 1, metric.WithAttributes())
				}
			}
		`),
	)
}

// No context in scope: the recording has nothing to pass, so the call stands.
func TestRecordBlockedWithoutContext(t *testing.T) {
	recordingSpec().RewriteRun(t,
		test.Golang(`
			package app

			import "github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"

			type svc struct {
				h *telemetry.Harvester
			}

			func (s *svc) tick() {
				s.h.RecordMetric(telemetry.Count{Name: "ticks", Value: 1})
			}
		`),
	)
}

// An attribute value whose type has no attribute constructor is not guessed at.
func TestRecordBlockedByUntypedAttribute(t *testing.T) {
	recordingSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				"context"

				"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
			)

			type svc struct {
				h *telemetry.Harvester
			}

			func (s *svc) tick(ctx context.Context, payload interface{}) {
				s.h.RecordMetric(telemetry.Count{
					Name:       "ticks",
					Value:      1,
					Attributes: map[string]interface{}{"payload": payload},
				})
			}
		`),
	)
}

// A harvester in a local variable would be left unused, which does not compile.
func TestRecordBlockedByLocalHarvester(t *testing.T) {
	recordingSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				"context"

				"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
			)

			func tick(ctx context.Context, key string) error {
				h, err := telemetry.NewHarvester(telemetry.ConfigAPIKey(key))
				if err != nil {
					return err
				}
				h.RecordMetric(telemetry.Count{Name: "ticks", Value: 1})
				return nil
			}
		`),
	)
}

// A Summary is a pre-aggregated observation no instrument accepts.
func TestRecordBlockedBySummary(t *testing.T) {
	recordingSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				"context"

				"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
			)

			type svc struct {
				h *telemetry.Harvester
			}

			func (s *svc) tick(ctx context.Context) {
				s.h.RecordMetric(telemetry.Summary{Name: "latency", Count: 3, Sum: 9, Min: 1, Max: 5})
			}
		`),
	)
}

// No telemetry import at all.
func TestRecordUnrelatedFile(t *testing.T) {
	recordingSpec().RewriteRun(t,
		test.Golang(`
			package app

			import "context"

			func tick(ctx context.Context) {}
		`),
	)
}
