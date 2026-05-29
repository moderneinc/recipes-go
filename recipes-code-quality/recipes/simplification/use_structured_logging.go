/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseStructuredLogging finds calls to the standard `log` package such as `log.Println`,
// `log.Printf`, `log.Fatal`, and `log.Fatalf`. In Go 1.21+ consider migrating
// to `log/slog` for structured logging.
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

var useStructuredLoggingPrintln = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseStructuredLogging$Println"),
	template.WithDisplayName("log.Println → slog.Info"),
	template.WithBefore(fmt.Sprintf(`log.Println(%s)`, slogMsg), template.Imports("log")),
	template.WithAfter(fmt.Sprintf(`slog.Info(%s)`, slogMsg), template.Imports("log/slog")),
	template.WithCaptures(slogMsg),
)

var useStructuredLoggingPrint = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseStructuredLogging$Print"),
	template.WithDisplayName("log.Print → slog.Info"),
	template.WithBefore(fmt.Sprintf(`log.Print(%s)`, slogMsg), template.Imports("log")),
	template.WithAfter(fmt.Sprintf(`slog.Info(%s)`, slogMsg), template.Imports("log/slog")),
	template.WithCaptures(slogMsg),
)

func (r *UseStructuredLogging) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{useStructuredLoggingPrintln, useStructuredLoggingPrint}
}

func (r *UseStructuredLogging) Editor() recipe.TreeVisitor {
	return visitor.Init(&findStdLogVisitor{})
}

type findStdLogVisitor struct {
	visitor.GoVisitor
}

// stdLogPrefixes lists the method-name prefixes on the standard `log` package
// that should be flagged.
var stdLogPrefixes = []string{"Print", "Fatal"}

// stdLogAutoFixed lists method names that are auto-converted by the template
// sub-recipes (single-argument calls only).
var stdLogAutoFixed = map[string]bool{"Print": true, "Println": true}

func (v *findStdLogVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "log" {
		return mi
	}

	for _, prefix := range stdLogPrefixes {
		if strings.HasPrefix(mi.Name.Name, prefix) {
			// Skip single-argument Print/Println — handled by template sub-recipes.
			if stdLogAutoFixed[mi.Name.Name] && len(mi.Arguments.Elements) == 1 {
				return mi
			}
			mi = mi.WithMarkers(
				java.MarkupInfo(mi.Markers, "consider migrating to log/slog for structured logging (Go 1.21+)"),
			)
			return mi
		}
	}

	return mi
}
