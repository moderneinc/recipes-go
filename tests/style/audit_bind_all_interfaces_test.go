/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditBindAllInterfacesEmptyHost(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditBindAllInterfaces{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func main() {
				http.ListenAndServe(":8080", nil)
			}
		`, `
			package main

			import "net/http"

			func main() {
				/*~~(listener bound to all interfaces; prefer an explicit host such as 127.0.0.1)~~>*/http.ListenAndServe(":8080", nil)
			}
		`),
	)
}

func TestAuditBindAllInterfacesZeroHost(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditBindAllInterfaces{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net"

			func main() {
				net.Listen("tcp", "0.0.0.0:9000")
			}
		`, `
			package main

			import "net"

			func main() {
				/*~~(listener bound to all interfaces; prefer an explicit host such as 127.0.0.1)~~>*/net.Listen("tcp", "0.0.0.0:9000")
			}
		`),
	)
}

func TestAuditBindAllInterfacesExplicitHostNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditBindAllInterfaces{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net"

			func main() {
				net.Listen("tcp", "127.0.0.1:9000")
			}
		`),
	)
}
