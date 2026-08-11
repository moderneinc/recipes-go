/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidDotImportReQualifiesReferences(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import . "fmt"

			func main() {
				Println("hello")
			}
		`, `
			package main

			import "fmt"

			func main() {
				fmt.Println("hello")
			}
		`),
	)
}

// A dot-imported type reference is re-qualified: Writer -> io.Writer.
func TestAvoidDotImportReQualifiesType(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import . "io"

			var _ Writer
		`, `
			package main

			import "io"

			var _ io.Writer
		`),
	)
}

// References are re-qualified when the dot import is in a parenthesized group.
func TestAvoidDotImportReQualifiesInGroupedImport(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"os"
				. "fmt"
			)

			func main() {
				Fprintln(os.Stdout, "hi")
			}
		`, `
			package main

			import (
				"os"
				"fmt"
			)

			func main() {
				fmt.Fprintln(os.Stdout, "hi")
			}
		`),
	)
}

// The qualifier is the package name (last path segment): for `math/rand`,
// Intn(10) -> rand.Intn(10).
func TestAvoidDotImportQualifierFromMultiSegmentPath(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import . "math/rand"

			func main() {
				_ = Intn(10)
			}
		`, `
			package main

			import "math/rand"

			func main() {
				_ = rand.Intn(10)
			}
		`),
	)
}

// With two dot imports, each reference is re-qualified to its own package.
func TestAvoidDotImportMultipleDotImports(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				. "fmt"
				. "strings"
			)

			func main() {
				Println(ToUpper("hi"))
			}
		`, `
			package main

			import (
				"fmt"
				"strings"
			)

			func main() {
				fmt.Println(strings.ToUpper("hi"))
			}
		`),
	)
}

// A local (closure parameter) that shadows the imported name is left alone;
// only the genuine reference is re-qualified.
func TestAvoidDotImportDoesNotReQualifyShadowedLocal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import . "fmt"

			func main() {
				Println("real")
				shadow := func(Println string) {
					_ = Println
				}
				shadow("inner")
			}
		`, `
			package main

			import "fmt"

			func main() {
				fmt.Println("real")
				shadow := func(Println string) {
					_ = Println
				}
				shadow("inner")
			}
		`),
	)
}

// Struct field declarations, composite-literal keys, and selectors that share
// the imported name are left untouched; only the genuine call is re-qualified.
func TestAvoidDotImportDoesNotReQualifySelector(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import . "fmt"

			type record struct {
				Println string
			}

			func main() {
				r := record{Println: "field"}
				Println(r.Println)
			}
		`, `
			package main

			import "fmt"

			type record struct {
				Println string
			}

			func main() {
				r := record{Println: "field"}
				fmt.Println(r.Println)
			}
		`),
	)
}

// Builtins (len, make) and same-package functions are not re-qualified.
func TestAvoidDotImportDoesNotReQualifyBuiltinsOrLocalFuncs(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import . "fmt"

			func helper() int { return 42 }

			func main() {
				s := make([]int, 0)
				Println(len(s), helper())
			}
		`, `
			package main

			import "fmt"

			func helper() int { return 42 }

			func main() {
				s := make([]int, 0)
				fmt.Println(len(s), helper())
			}
		`),
	)
}

// A dot-imported function used as a value (passed, not called) is re-qualified.
func TestAvoidDotImportReQualifiesFunctionValue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import . "sort"

			var less = Strings
		`, `
			package main

			import "sort"

			var less = sort.Strings
		`),
	)
}

// A package imported both normally and via a dot alias is left untouched, so
// stripping the dot does not produce a duplicate import.
func TestAvoidDotImportNoChangeWhenAlsoImportedNormally(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"fmt"
				. "fmt"
			)

			func main() {
				fmt.Print(Sprintf("%d", 1))
			}
		`),
	)
}

func TestAvoidDotImportNoChangeNormalImport(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func main() {
				fmt.Println("hello")
			}
		`),
	)
}

func TestAvoidDotImportNoChangeAliasedImport(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import f "fmt"

			func main() {
				f.Println("hello")
			}
		`),
	)
}

// A blank (`_`) import is left untouched; only the `.` alias is targeted.
func TestAvoidDotImportNoChangeBlankImport(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import _ "image/png"

			func main() {}
		`),
	)
}

// A dot import whose package name cannot be confidently derived from the path
// (here a bare "/vN" segment, which is ambiguous across module conventions) is
// left untouched.
func TestAvoidDotImportNoChangeUninferablePackageName(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import . "example.com/thing/v2"

			func main() {
				DoSomething()
			}
		`),
	)
}
