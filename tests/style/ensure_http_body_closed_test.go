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

func TestEnsureHttpBodyClosedAfterNilResponseCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"errors"
				"net/http"
			)

			func f(client *http.Client, req *http.Request) (int, error) {
				resp, err := client.Do(req)
				if err != nil {
					return 0, err
				}
				if resp == nil {
					return 0, errors.New("empty response")
				}
				return resp.StatusCode, nil
			}
		`, `
			package main

			import (
				"errors"
				"net/http"
			)

			func f(client *http.Client, req *http.Request) (int, error) {
				resp, err := client.Do(req)
				if err != nil {
					return 0, err
				}
				if resp == nil {
					return 0, errors.New("empty response")
				}
				defer resp.Body.Close()
				return resp.StatusCode, nil
			}
		`),
	)
}

func TestEnsureHttpBodyClosedAfterNilResponseCheckWithoutErrorCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f(client *http.Client, req *http.Request) {
				resp, _ := client.Do(req)
				if resp == nil {
					return
				}
				_ = resp
			}
		`, `
			package main

			import "net/http"

			func f(client *http.Client, req *http.Request) {
				resp, _ := client.Do(req)
				if resp == nil {
					return
				}
				defer resp.Body.Close()
				_ = resp
			}
		`),
	)
}

// `resp.Body.Close()` panics when the response is nil.
func TestEnsureHttpBodyClosedNoChangeWhenResponseMayBeNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f(client *http.Client, req *http.Request) {
				resp, _ := client.Do(req)
				if resp != nil && resp.StatusCode != http.StatusOK {
					_ = resp
				}
			}
		`),
	)
}

// A nil check that falls through leaves the response nil below it.
func TestEnsureHttpBodyClosedNoChangeWhenNilCheckDoesNotReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"log"
				"net/http"
			)

			func f(client *http.Client, req *http.Request) {
				resp, err := client.Do(req)
				if err != nil {
					return
				}
				if resp == nil {
					log.Println("no response")
				}
				_ = resp
			}
		`),
	)
}
