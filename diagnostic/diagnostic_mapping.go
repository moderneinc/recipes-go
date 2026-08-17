/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package diagnostic

// AnalyzerTool identifies the static analysis tool a diagnostic comes from.
type AnalyzerTool int

const (
	Staticcheck  AnalyzerTool = iota // staticcheck (S*, SA*, ST*, U*)
	GoVet                            // go vet
	GolangciLint                     // golangci-lint (meta-linter)
)

func (t AnalyzerTool) String() string {
	switch t {
	case Staticcheck:
		return "Staticcheck"
	case GoVet:
		return "GoVet"
	case GolangciLint:
		return "GolangciLint"
	}
	return "unknown"
}

// Mapping maps a recipe to its equivalent static analysis diagnostic.
type Mapping struct {
	DiagnosticID string       // e.g., "S1012", "SA4000"
	Tool         AnalyzerTool // which tool produces this diagnostic
	HasFix       bool         // whether the tool can auto-fix this diagnostic
}

// HasMappings is implemented by recipes that correspond to
// known static analysis diagnostics.
type HasMappings interface {
	DiagnosticMappings() []Mapping
}
