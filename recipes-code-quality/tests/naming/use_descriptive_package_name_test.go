/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseDescriptivePackageNameUtils(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseDescriptivePackageName{})
	spec.RewriteRun(t,
		test.Golang(`
			package utils

			func Helper() {}
		`),
	)
}

func TestUseDescriptivePackageNameCommon(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseDescriptivePackageName{})
	spec.RewriteRun(t,
		test.Golang(`
			package common

			func Do() {}
		`),
	)
}

func TestUseDescriptivePackageNameNoChangeHttp(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseDescriptivePackageName{})
	spec.RewriteRun(t,
		test.Golang(`
			package http

			func Get() {}
		`),
	)
}

func TestUseDescriptivePackageNameNoChangeMain(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseDescriptivePackageName{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {}
		`),
	)
}
