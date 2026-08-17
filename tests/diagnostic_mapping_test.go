/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package tests

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// mappedRecipes returns every registered recipe that claims a diagnostic,
// keyed by recipe name. The registry is the set the library ships, so these
// checks cover exactly that set.
func mappedRecipes(t *testing.T) map[string][]diagnostic.Mapping {
	t.Helper()
	registry := recipe.NewRegistry()
	recipes.Activate(registry)

	mapped := map[string][]diagnostic.Mapping{}
	for _, registration := range registry.AllRegistrations() {
		if registration.Constructor == nil {
			continue
		}
		r := registration.Constructor(nil)
		if m, ok := r.(diagnostic.HasMappings); ok {
			mapped[r.Name()] = m.DiagnosticMappings()
		}
	}
	return mapped
}

func TestDiagnosticIDsAreReal(t *testing.T) {
	for name, mappings := range mappedRecipes(t) {
		for _, m := range mappings {
			// An ID no analyzer defines claims coverage of a diagnostic that does not exist.
			if !diagnostic.Known(m.Tool, m.DiagnosticID) {
				t.Errorf("%s maps to %s %q, which %s does not define", name, m.Tool, m.DiagnosticID, m.Tool)
			}
		}
	}
}

func TestDiagnosticMappingsAreDistinct(t *testing.T) {
	for name, mappings := range mappedRecipes(t) {
		seen := map[string]bool{}
		for _, m := range mappings {
			key := fmt.Sprintf("%s %s", m.Tool, m.DiagnosticID)
			if seen[key] {
				t.Errorf("%s maps to %s twice", name, key)
			}
			seen[key] = true
		}
	}
}

// Reports rather than asserts: the covered set is a fact about the library,
// not a threshold. Run with -v to read it.
func TestStaticcheckCoverage(t *testing.T) {
	claimed := map[string]bool{}
	for _, mappings := range mappedRecipes(t) {
		for _, m := range mappings {
			if m.Tool == diagnostic.Staticcheck {
				claimed[m.DiagnosticID] = true
			}
		}
	}

	all := diagnostic.IDs(diagnostic.Staticcheck)
	var covered, uncovered []string
	for _, id := range all {
		if claimed[id] {
			covered = append(covered, id)
		} else {
			uncovered = append(uncovered, id)
		}
	}
	sort.Strings(covered)

	t.Logf("staticcheck: %d/%d checks covered", len(covered), len(all))
	t.Logf("covered:   %s", strings.Join(covered, " "))
	t.Logf("uncovered: %s", strings.Join(uncovered, " "))
}
