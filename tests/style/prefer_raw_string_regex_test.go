/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferRawStringRegexCompile(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRawStringForRegex{})
	spec.RewriteRun(t,
		test.Golang(
			"package main\n\nimport \"regexp\"\n\nfunc f() *regexp.Regexp {\n\tr, _ := regexp.Compile(\"\\\\d+\")\n\treturn r\n}\n",
			"package main\n\nimport \"regexp\"\n\nfunc f() *regexp.Regexp {\n\tr, _ := regexp.Compile(`\\d+`)\n\treturn r\n}\n",
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

// Skips regexp.Compile in a single-value context, where the two-value call does not compile.
func TestPreferRawStringRegexNoChangeSingleValueContext(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRawStringForRegex{})
	spec.RewriteRun(t,
		test.Golang("package main\n\nimport \"regexp\"\n\nvar r = regexp.Compile(\"\\\\d+\")\n"),
	)
}

// Skips a real newline escape, which a raw string would embed as a literal line break.
func TestPreferRawStringRegexNoChangeControlChar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRawStringForRegex{})
	spec.RewriteRun(t,
		test.Golang("package main\n\nimport \"regexp\"\n\nvar re = regexp.MustCompile(\"a\\nb\")\n"),
	)
}

// Rewrites a `\\t` metacharacter escape, which is a backslash rather than a control character.
func TestPreferRawStringRegexMetacharTab(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRawStringForRegex{})
	spec.RewriteRun(t,
		test.Golang(
			"package main\n\nimport \"regexp\"\n\nvar re = regexp.MustCompile(\"\\\\t+\")\n",
			"package main\n\nimport \"regexp\"\n\nvar re = regexp.MustCompile(`\\t+`)\n",
		),
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
