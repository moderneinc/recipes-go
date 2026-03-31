/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindBadPackageNameUtils(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindBadPackageName{})
	spec.RewriteRun(t,
		test.Golang(`
			package utils

			func Helper() {}
		`),
	)
}

func TestFindBadPackageNameCommon(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindBadPackageName{})
	spec.RewriteRun(t,
		test.Golang(`
			package common

			func Do() {}
		`),
	)
}

func TestFindBadPackageNameNoChangeHttp(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindBadPackageName{})
	spec.RewriteRun(t,
		test.Golang(`
			package http

			func Get() {}
		`),
	)
}

func TestFindBadPackageNameNoChangeMain(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindBadPackageName{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {}
		`),
	)
}
