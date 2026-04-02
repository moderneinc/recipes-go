/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseMixedCaps finds exported functions using underscores instead
// of camelCase. Go convention: use MixedCaps or mixedCaps, not underscores.
// golangci-lint: revive (var-naming)
type UseMixedCaps struct {
	recipe.Base
}

func (r *UseMixedCaps) Name() string {
	return "org.openrewrite.golang.codequality.UseMixedCaps"
}
func (r *UseMixedCaps) DisplayName() string {
	return "Use MixedCaps"
}
func (r *UseMixedCaps) Description() string {
	return "Find exported functions using underscores instead of camelCase. Go convention is to use MixedCaps or mixedCaps."
}
func (r *UseMixedCaps) Tags() []string { return []string{"naming"} }

func (r *UseMixedCaps) Editor() recipe.TreeVisitor {
	return visitor.Init(&useMixedCapsVisitor{})
}

type useMixedCapsVisitor struct {
	visitor.GoVisitor
}

func (v *useMixedCapsVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil {
		return md
	}

	funcName := md.Name.Name

	// Only check exported names (starts with uppercase).
	firstRune, _ := utf8.DecodeRuneInString(funcName)
	if !unicode.IsUpper(firstRune) {
		return md
	}

	// Check if the name contains an underscore.
	if !strings.Contains(funcName, "_") {
		return md
	}

	// Convert underscored name to MixedCaps.
	newName := toMixedCaps(funcName)
	md = md.WithName(md.Name.WithName(newName))
	return md
}

// toMixedCaps converts an underscored name to MixedCaps by splitting on "_",
// capitalizing the first letter of each segment, and joining.
func toMixedCaps(name string) string {
	parts := strings.Split(name, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		r, size := utf8.DecodeRuneInString(part)
		b.WriteRune(unicode.ToUpper(r))
		b.WriteString(part[size:])
	}
	return b.String()
}
