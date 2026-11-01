/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

func TestHandleDeferredCloseError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleDeferredCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				f, _ := os.Open("file.txt")
				defer f.Close()
			}
		`, `
			package main

			import "os"

			func main() {
				f, _ := os.Open("file.txt")
				defer func() { _ = f.Close() }()
			}
		`),
	)
}

func TestHandleDeferredCloseErrorNoChangeDone(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleDeferredCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sync"

			func main() {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Done()
			}
		`),
	)
}

// Skips a void Close().
func TestHandleDeferredCloseErrorNoChangeVoidClose(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleDeferredCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type T struct{}

			func (T) Close() {}

			func f(t T) {
				defer t.Close()
			}
		`),
	)
}

// The closure the recipe writes is made of nodes of its own, each with an id,
// as the parser gives every node it makes.
func TestHandleDeferredCloseErrorGivesEachNodeAnID(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleDeferredCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				f, _ := os.Open("file.txt")
				defer f.Close()
			}
		`, `
			package main

			import "os"

			func main() {
				f, _ := os.Open("file.txt")
				defer func() { _ = f.Close() }()
			}
		`).WithAfterRecipe(func(t *testing.T, cu *golang.CompilationUnit) {
			ids := visitor.Init(&idCollector{})
			ids.Visit(cu, nil)
			var missing, shared []string
			seen := map[uuid.UUID]bool{}
			for i, id := range ids.ids {
				switch {
				case id == uuid.Nil:
					missing = append(missing, fmt.Sprintf("%T", ids.nodes[i]))
				case seen[id]:
					shared = append(shared, fmt.Sprintf("%T", ids.nodes[i]))
				}
				seen[id] = true
			}
			assert.Empty(t, missing, "nodes without an id")
			assert.Empty(t, shared, "nodes sharing an id")
		}),
	)
}

// idCollector records the id of every node it visits.
type idCollector struct {
	visitor.GoVisitor
	ids   []uuid.UUID
	nodes []java.Tree
}

func (c *idCollector) Visit(t java.Tree, p any) java.Tree {
	if j, ok := t.(java.J); ok {
		c.ids = append(c.ids, j.GetID())
		c.nodes = append(c.nodes, t)
	}
	return c.GoVisitor.Visit(t, p)
}
