/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// allCapsWithUnderscore matches names like MAX_RETRIES or HTTP_TIMEOUT.
var allCapsWithUnderscore = regexp.MustCompile(`^[A-Z][A-Z0-9]*(_[A-Z0-9]+)+$`)

// UseMixedCapsForConstants finds constant or variable names using ALL_CAPS with
// underscores. Go convention is MixedCaps, not ALL_CAPS_WITH_UNDERSCORES.
// golangci-lint: revive (var-naming)
type UseMixedCapsForConstants struct {
	recipe.Base
}

func (r *UseMixedCapsForConstants) Name() string {
	return "org.openrewrite.golang.codequality.UseMixedCapsForConstants"
}
func (r *UseMixedCapsForConstants) DisplayName() string {
	return "Use MixedCaps for constants"
}
func (r *UseMixedCapsForConstants) Description() string {
	return "Find constant or variable names using ALL_CAPS_WITH_UNDERSCORES. Go convention is to use MixedCaps, not ALL_CAPS."
}
func (r *UseMixedCapsForConstants) Tags() []string { return []string{"naming"} }

func (r *UseMixedCapsForConstants) Editor() recipe.TreeVisitor {
	return visitor.Init(&useMixedCapsForConstantsVisitor{})
}

type useMixedCapsForConstantsVisitor struct {
	visitor.GoVisitor
}

func (v *useMixedCapsForConstantsVisitor) VisitVariableDeclarator(vd *tree.VariableDeclarator, p any) tree.J {
	vd = v.GoVisitor.VisitVariableDeclarator(vd, p).(*tree.VariableDeclarator)

	if vd.Name == nil {
		return vd
	}

	name := vd.Name.Name
	if !allCapsWithUnderscore.MatchString(name) {
		return vd
	}

	// Convert ALL_CAPS to MixedCaps.
	newName := allCapsToMixedCaps(name)
	vd = vd.WithName(vd.Name.WithName(newName))
	return vd
}

// allCapsToMixedCaps converts an ALL_CAPS_NAME to MixedCaps by splitting on "_",
// lowercasing each segment, and capitalizing the first letter.
func allCapsToMixedCaps(name string) string {
	parts := strings.Split(name, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		r, size := utf8.DecodeRuneInString(lower)
		b.WriteRune(unicode.ToUpper(r))
		b.WriteString(lower[size:])
	}
	return b.String()
}
