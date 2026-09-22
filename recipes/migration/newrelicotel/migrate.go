/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package newrelicotel

import (
	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// The entry point for the part of the telemetry SDK migration that is
// mechanical.
type MigrateNewRelicTelemetryToOpenTelemetry struct {
	recipe.Base
}

func (r *MigrateNewRelicTelemetryToOpenTelemetry) Name() string {
	return "org.openrewrite.golang.migration.MigrateNewRelicTelemetryToOpenTelemetry"
}
func (r *MigrateNewRelicTelemetryToOpenTelemetry) DisplayName() string {
	return "Migrate `newrelic-telemetry-sdk-go` to the OpenTelemetry Go SDK"
}
func (r *MigrateNewRelicTelemetryToOpenTelemetry) Description() string {
	return "Move the mechanical part of a `github.com/newrelic/newrelic-telemetry-sdk-go` migration: metric recordings become OpenTelemetry instrument recordings, and go.mod gains the OpenTelemetry requires. This is a partial migration by design — the harvester wiring, spans, events and logs have no faithful one-to-one rewrite, so run `FindNewRelicTelemetrySdkUsage` to enumerate what is left and replace the harvester with an `sdkmetric.MeterProvider` behind an OTLP exporter by hand. Run `go mod tidy` afterwards."
}
func (r *MigrateNewRelicTelemetryToOpenTelemetry) Tags() []string {
	return []string{"migration", "newrelic", "opentelemetry", "observability"}
}

func (r *MigrateNewRelicTelemetryToOpenTelemetry) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&MigrateNewRelicMetricRecording{},
		// Runs after the rewrite, so it sees the imports it introduced.
		&UpdateNewRelicTelemetryDependency{},
		&migration.FormatGoMod{},
	}
}
