/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package newrelicotel

import (
	"github.com/moderneinc/recipes-go/recipes/migration/internal/depswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// telemetryModules lists both modules the emitted recordings import: the
// attribute package ships in go.opentelemetry.io/otel, the metric API is its
// own module.
var telemetryModules = depswap.Modules{
	Old:     nrModule,
	New:     []string{otelAPI, otelMetric},
	Version: otelVersion,
}

// Brings the go.mod requires into line with what the source imports.
type UpdateNewRelicTelemetryDependency struct {
	recipe.ScanningBase
}

func (r *UpdateNewRelicTelemetryDependency) Name() string {
	return "org.openrewrite.golang.migration.UpdateNewRelicTelemetryDependency"
}
func (r *UpdateNewRelicTelemetryDependency) DisplayName() string {
	return "Require the OpenTelemetry Go SDK instead of `newrelic-telemetry-sdk-go`"
}
func (r *UpdateNewRelicTelemetryDependency) Description() string {
	return "Require `go.opentelemetry.io/otel " + otelVersion + "` and `go.opentelemetry.io/otel/metric` once the source records through them, and drop `github.com/newrelic/newrelic-telemetry-sdk-go` once nothing imports it. Exporting to New Relic also needs an OTLP exporter and `go.opentelemetry.io/otel/sdk`, which the hand-written provider wiring pulls in. Does not sync go.sum, so a `go mod tidy` is still needed."
}
func (r *UpdateNewRelicTelemetryDependency) Tags() []string {
	return []string{"migration", "newrelic", "opentelemetry", "gomod"}
}

func (r *UpdateNewRelicTelemetryDependency) InitialValue(*recipe.ExecutionContext) any {
	return depswap.NewAcc()
}
func (r *UpdateNewRelicTelemetryDependency) Scanner(acc any) recipe.TreeVisitor {
	return depswap.Scanner(acc.(*depswap.Acc), telemetryModules,
		depswap.EditorProbe((&MigrateNewRelicMetricRecording{}).Editor))
}
func (r *UpdateNewRelicTelemetryDependency) EditorWithData(acc any) recipe.TreeVisitor {
	return depswap.Editor(acc.(*depswap.Acc), telemetryModules)
}
