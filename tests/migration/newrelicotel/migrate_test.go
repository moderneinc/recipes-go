/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package newrelicotel_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/newrelicotel"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// The recordings move to OpenTelemetry and go.mod gains its requires. The New
// Relic requirement stays, because the harvester the recordings hung off is
// still declared — that part has no mechanical rewrite and is left for the hand
// migration FindNewRelicTelemetrySdkUsage enumerates.
func TestMigrateWholeModule(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&newrelicotel.MigrateNewRelicTelemetryToOpenTelemetry{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.23

				require (
					github.com/newrelic/newrelic-telemetry-sdk-go v0.8.1
				)
			`, `
				module example.com/app

				go 1.23

				require (
					github.com/newrelic/newrelic-telemetry-sdk-go v0.8.1
					go.opentelemetry.io/otel v1.46.0
					go.opentelemetry.io/otel/metric v1.46.0
				)
			`),
			test.Golang(`
				package app

				import (
					"context"

					"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
				)

				type svc struct {
					h *telemetry.Harvester
				}

				func (s *svc) tick(ctx context.Context, room string) {
					s.h.RecordMetric(telemetry.Count{
						Name:       "ticks",
						Value:      1,
						Attributes: map[string]interface{}{"room": room},
					})
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

				func (s *svc) tick(ctx context.Context, room string) {
					{
						nrInstrument, _ := otel.Meter("app").Float64Counter("ticks")
						nrInstrument.Add(ctx, 1, metric.WithAttributes(attribute.String("room", room)))
					}
				}
			`),
		),
	)
}
