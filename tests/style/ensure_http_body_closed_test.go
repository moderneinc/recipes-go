/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureHttpBodyClosedGet(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				resp, err := http.Get("http://example.com")
				_ = err
				_ = resp
			}
		`, `
			package main

			import "net/http"

			func f() {
				resp, err := http.Get("http://example.com")
				defer resp.Body.Close()
				_ = err
				_ = resp
			}
		`),
	)
}

func TestEnsureHttpBodyClosedClientDo(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f(client *http.Client, req *http.Request) {
				resp, err := client.Do(req)
				_ = err
				_ = resp
			}
		`, `
			package main

			import "net/http"

			func f(client *http.Client, req *http.Request) {
				resp, err := client.Do(req)
				defer resp.Body.Close()
				_ = err
				_ = resp
			}
		`),
	)
}

func TestEnsureHttpBodyClosedAfterErrorCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() error {
				resp, err := http.Get("http://example.com")
				if err != nil {
					return err
				}
				_ = resp
				return nil
			}
		`, `
			package main

			import "net/http"

			func f() error {
				resp, err := http.Get("http://example.com")
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				_ = resp
				return nil
			}
		`),
	)
}

// Only a *http.Response has a Body to close; any other `Do` must be left alone.
func TestEnsureHttpBodyClosedNoChangeForeignDo(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "github.com/example/retry"

			func f() {
				err := retry.Do(func() error { return nil })
				_ = err
			}
		`),
	)
}

func TestEnsureHttpBodyClosedNoChangeError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f(w http.ResponseWriter) {
				http.Error(w, "err", 500)
			}
		`),
	)
}

func TestEnsureHttpBodyClosedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				resp, err := http.Get("http://example.com")
				defer resp.Body.Close()
				_ = err
			}
		`),
	)
}

// The guard, not the call, is what makes the response non-nil.
func TestEnsureHttpBodyClosedAfterErrorCheckPastDefer(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f(cancel func()) error {
				resp, err := http.Get("http://example.com")
				defer cancel()
				if err != nil {
					return err
				}
				_ = resp
				return nil
			}
		`, `
			package main

			import "net/http"

			func f(cancel func()) error {
				resp, err := http.Get("http://example.com")
				defer cancel()
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				_ = resp
				return nil
			}
		`),
	)
}

func TestEnsureHttpBodyClosedAlreadyDeferredInClosure(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() error {
				resp, err := http.Get("http://example.com")
				if err != nil {
					return err
				}
				defer func() { _ = resp.Body.Close() }()
				return nil
			}
		`),
	)
}

func TestEnsureHttpBodyClosedAlreadyDeferredViaHelper(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func closeResponse(resp *http.Response) {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}

			func f() error {
				resp, err := http.Get("http://example.com")
				defer closeResponse(resp)
				if err != nil {
					return err
				}
				_ = resp
				return nil
			}
		`),
	)
}

// A helper deferred from a branch releases the response all the same.
func TestEnsureHttpBodyClosedAlreadyDeferredInBranch(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func closeResponse(resp *http.Response) {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}

			func f() error {
				resp, err := http.Get("http://example.com")
				if resp != nil {
					defer closeResponse(resp)
				}
				if err != nil {
					return err
				}
				return nil
			}
		`),
	)
}
