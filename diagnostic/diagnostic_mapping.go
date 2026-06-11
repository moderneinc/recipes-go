/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package diagnostic

// AnalyzerTool identifies the static analysis tool a diagnostic comes from.
type AnalyzerTool int

const (
	Staticcheck  AnalyzerTool = iota // staticcheck (SA*, S*, ST*, QF*)
	GoVet                            // go vet
	GolangciLint                     // golangci-lint (meta-linter)
)

// Mapping maps a recipe to its equivalent static analysis diagnostic.
type Mapping struct {
	DiagnosticID string       // e.g., "S1012", "SA4000"
	Tool         AnalyzerTool // which tool produces this diagnostic
	HasFix       bool         // whether the tool can auto-fix this diagnostic
}

// HasMappings is implemented by recipes that correspond to
// known static analysis diagnostics. Used by the comparison harness.
type HasMappings interface {
	DiagnosticMappings() []Mapping
}
