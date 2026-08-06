/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Finds calls to the standard `log` package such as `log.Println`, `log.Printf`,
// `log.Fatal`, and `log.Fatalf` and suggests migrating to `log/slog` (Go 1.21+).
type UseStructuredLogging struct {
	recipe.Base
}

func (r *UseStructuredLogging) Name() string {
	return "org.openrewrite.golang.codequality.UseStructuredLogging"
}
func (r *UseStructuredLogging) DisplayName() string { return "Use structured logging" }
func (r *UseStructuredLogging) Description() string {
	return "Find calls to the standard `log` package (`log.Print*`, `log.Fatal*`). Consider migrating to `log/slog` for structured logging (Go 1.21+)."
}
func (r *UseStructuredLogging) Tags() []string { return []string{"simplification", "logging"} }

var slogMsg = template.Expr("slogMsg")

var (
	logPrintlnPattern = template.Expression(fmt.Sprintf(`log.Println(%s)`, slogMsg)).
				Captures(slogMsg).Imports("log").Build()
	logPrintPattern = template.Expression(fmt.Sprintf(`log.Print(%s)`, slogMsg)).
			Captures(slogMsg).Imports("log").Build()
	slogInfoTemplate = template.ExpressionTemplate(fmt.Sprintf(`slog.Info(%s)`, slogMsg)).
				Captures(slogMsg).Imports("log/slog").Build()
)

func (r *UseStructuredLogging) Editor() recipe.TreeVisitor {
	return visitor.Init(&findStdLogVisitor{})
}

type findStdLogVisitor struct {
	visitor.GoVisitor
}

// stdLogPrefixes lists the method-name prefixes on the standard `log` package
// that should be flagged.
var stdLogPrefixes = []string{"Print", "Fatal"}

func (v *findStdLogVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "log" {
		return mi
	}

	// Auto-convert a single-argument log.Print/Println to slog.Info only when the
	// argument is a string, since slog.Info takes a string message; a non-string
	// argument falls through to the markup hint below.
	if mi.Name.Name == "Print" || mi.Name.Name == "Println" {
		if replaced := v.toSlogInfo(mi); replaced != nil {
			return replaced
		}
	}

	for _, prefix := range stdLogPrefixes {
		if strings.HasPrefix(mi.Name.Name, prefix) {
			return mi.WithMarkers(
				java.MarkupInfo(mi.Markers, "consider migrating to log/slog for structured logging (Go 1.21+)"),
			)
		}
	}
	return mi
}

// toSlogInfo returns the slog.Info replacement for a single-argument log.Print /
// log.Println whose argument is a string, or nil when it does not apply.
func (v *findStdLogVisitor) toSlogInfo(mi *java.MethodInvocation) java.J {
	var pat *template.GoPattern
	switch mi.Name.Name {
	case "Println":
		pat = logPrintlnPattern
	case "Print":
		pat = logPrintPattern
	default:
		return nil
	}
	match := pat.Match(mi, nil)
	if match == nil {
		return nil
	}
	arg, ok := match.GetCapture(slogMsg).(java.Expression)
	if !ok || !matcher.IsString(matcher.TypeOfExpression(arg)) {
		return nil
	}
	replaced, ok := slogInfoTemplate.Apply(nil, match).(*java.MethodInvocation)
	if !ok {
		return nil
	}
	recipegolang.MaybeAddImport(v, "log/slog", nil, false)
	v.DoAfterVisit(recipe.Service[*recipegolang.ImportService](nil).RemoveUnusedImportsVisitor())
	return replaced.WithPrefix(mi.GetPrefix())
}
