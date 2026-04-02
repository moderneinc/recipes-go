/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferRawStringRegexCompile(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRawStringForRegex{})
	spec.RewriteRun(t,
		test.Golang(
			"package main\n\nimport \"regexp\"\n\nvar r = regexp.Compile(\"\\\\d+\")\n",
			"package main\n\nimport \"regexp\"\n\nvar r = regexp.Compile(`\\d+`)\n",
		),
	)
}

func TestPreferRawStringRegexMustCompile(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRawStringForRegex{})
	spec.RewriteRun(t,
		test.Golang(
			"package main\n\nimport \"regexp\"\n\nvar r = regexp.MustCompile(\"\\\\d+\")\n",
			"package main\n\nimport \"regexp\"\n\nvar r = regexp.MustCompile(`\\d+`)\n",
		),
	)
}

func TestPreferRawStringRegexNoChangeRawString(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRawStringForRegex{})
	spec.RewriteRun(t,
		test.Golang("package main\n\nimport \"regexp\"\n\nvar r = regexp.MustCompile(`\\d+`)\n"),
	)
}

func TestPreferRawStringRegexNoChangeNoBackslash(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRawStringForRegex{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "regexp"

			var r = regexp.MustCompile("[a-z]+")
		`),
	)
}
