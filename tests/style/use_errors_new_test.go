/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

func TestUseErrorsNewSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseErrorsNewForSimpleErrors{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() error {
				return fmt.Errorf("something went wrong")
			}
		`, `
			package main

			import "errors"

			func f() error {
				return errors.New("something went wrong")
			}
		`),
	)
}

func TestUseErrorsNewNoChangeFormatVerbS(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseErrorsNewForSimpleErrors{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(msg string) error {
				return fmt.Errorf("error: %s", msg)
			}
		`),
	)
}

func TestUseErrorsNewNoChangeFormatVerbD(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseErrorsNewForSimpleErrors{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(n int) error {
				return fmt.Errorf("count: %d", n)
			}
		`),
	)
}

// sentinelStandIn is UsePackageLevelErrorSentinel reduced to the part that matters
// here: it takes the `errors` reference back out of the function body. It scans, so
// the runner builds one editor per recipe and reuses it across every file.
type sentinelStandIn struct{ recipe.ScanningBase }

func (s *sentinelStandIn) Name() string        { return "test.SentinelStandIn" }
func (s *sentinelStandIn) DisplayName() string { return "Sentinel stand-in" }
func (s *sentinelStandIn) Description() string { return "Replace `errors.New(...)` with a sentinel." }
func (s *sentinelStandIn) InitialValue(ctx *recipe.ExecutionContext) any {
	return nil
}
func (s *sentinelStandIn) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&sentinelStandInVisitor{})
}

type sentinelStandInVisitor struct{ visitor.GoVisitor }

func (v *sentinelStandInVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Select == nil {
		return mi
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	if !ok || recv.Name != "errors" || mi.Name.Name != "New" {
		return mi
	}
	golang.MaybeRemoveImport(v, "errors")
	prefix := mi.Prefix
	if !recv.Prefix.IsEmpty() {
		prefix = recv.Prefix
	}
	return &java.Identifier{ID: uuid.New(), Prefix: prefix, Name: "ErrDup"}
}

type useErrorsNewThenSentinel struct{ recipe.Base }

func (r *useErrorsNewThenSentinel) Name() string        { return "test.UseErrorsNewThenSentinel" }
func (r *useErrorsNewThenSentinel) DisplayName() string { return "Use errors.New, then hoist" }
func (r *useErrorsNewThenSentinel) Description() string { return "Pair the two recipes." }
func (r *useErrorsNewThenSentinel) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{&style.UseErrorsNewForSimpleErrors{}, &sentinelStandIn{}}
}

func TestUseErrorsNewDropsFmtInEveryFile(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&useErrorsNewThenSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			package sentinel

			import "fmt"

			func A() error {
				return fmt.Errorf("dup")
			}
		`, `
			package sentinel

			func A() error {
				return ErrDup
			}
		`),
		test.Golang(`
			package sentinel

			import "fmt"

			func B() error {
				return fmt.Errorf("dup")
			}
		`, `
			package sentinel

			func B() error {
				return ErrDup
			}
		`),
	)
}
