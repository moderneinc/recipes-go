/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"go/version"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/preconditions"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// upgradeGoVersionEditor builds the editor shared by every UpgradeGoTo12X
// recipe: a go.mod precondition plus ChangeGoVersionVisitor gated by a
// no-downgrade guard, so a module already targeting the version or newer is
// left untouched.
//
// Bumping the `go` directive is the whole migration. Go's compatibility
// promise keeps source compiling across releases, version-gated language
// changes (e.g. the Go 1.22 loop-variable semantics) activate from the
// directive alone, and `go mod edit -go` itself never rewrites the
// `toolchain` line or go.sum — so there is nothing else for these recipes to
// touch.
func upgradeGoVersionEditor(target string) recipe.TreeVisitor {
	return preconditions.Check(
		preconditions.HasSourcePath("**/go.mod"),
		visitor.Init(&upgradeGoVersionVisitor{
			ChangeGoVersionVisitor: ChangeGoVersionVisitor{NewVersion: target},
		}),
	)
}

type upgradeGoVersionVisitor struct {
	ChangeGoVersionVisitor
}

func (v *upgradeGoVersionVisitor) VisitGoModDirective(d *golang.GoModDirective, p any) java.Tree {
	if d.Keyword == "go" && len(d.Values) == 1 &&
		version.Compare("go"+d.Values[0].Text, "go"+v.NewVersion) >= 0 {
		return d
	}
	return v.ChangeGoVersionVisitor.VisitGoModDirective(d, p)
}

type UpgradeGoTo118 struct{ recipe.Base }

func (r *UpgradeGoTo118) Name() string        { return "org.openrewrite.golang.migration.UpgradeGoTo118" }
func (r *UpgradeGoTo118) DisplayName() string { return "Upgrade Go to 1.18" }
func (r *UpgradeGoTo118) Description() string {
	return "Raise the `go` directive in go.mod to Go 1.18, unless it already targets 1.18 or newer."
}
func (r *UpgradeGoTo118) Editor() recipe.TreeVisitor { return upgradeGoVersionEditor("1.18") }

type UpgradeGoTo119 struct{ recipe.Base }

func (r *UpgradeGoTo119) Name() string        { return "org.openrewrite.golang.migration.UpgradeGoTo119" }
func (r *UpgradeGoTo119) DisplayName() string { return "Upgrade Go to 1.19" }
func (r *UpgradeGoTo119) Description() string {
	return "Raise the `go` directive in go.mod to Go 1.19, unless it already targets 1.19 or newer."
}
func (r *UpgradeGoTo119) Editor() recipe.TreeVisitor { return upgradeGoVersionEditor("1.19") }

type UpgradeGoTo120 struct{ recipe.Base }

func (r *UpgradeGoTo120) Name() string        { return "org.openrewrite.golang.migration.UpgradeGoTo120" }
func (r *UpgradeGoTo120) DisplayName() string { return "Upgrade Go to 1.20" }
func (r *UpgradeGoTo120) Description() string {
	return "Raise the `go` directive in go.mod to Go 1.20, unless it already targets 1.20 or newer."
}
func (r *UpgradeGoTo120) Editor() recipe.TreeVisitor { return upgradeGoVersionEditor("1.20") }

type UpgradeGoTo121 struct{ recipe.Base }

func (r *UpgradeGoTo121) Name() string        { return "org.openrewrite.golang.migration.UpgradeGoTo121" }
func (r *UpgradeGoTo121) DisplayName() string { return "Upgrade Go to 1.21" }
func (r *UpgradeGoTo121) Description() string {
	return "Raise the `go` directive in go.mod to Go 1.21, unless it already targets 1.21 or newer."
}
func (r *UpgradeGoTo121) Editor() recipe.TreeVisitor { return upgradeGoVersionEditor("1.21") }

type UpgradeGoTo122 struct{ recipe.Base }

func (r *UpgradeGoTo122) Name() string        { return "org.openrewrite.golang.migration.UpgradeGoTo122" }
func (r *UpgradeGoTo122) DisplayName() string { return "Upgrade Go to 1.22" }
func (r *UpgradeGoTo122) Description() string {
	return "Raise the `go` directive in go.mod to Go 1.22, unless it already targets 1.22 or newer."
}
func (r *UpgradeGoTo122) Editor() recipe.TreeVisitor { return upgradeGoVersionEditor("1.22") }

type UpgradeGoTo123 struct{ recipe.Base }

func (r *UpgradeGoTo123) Name() string        { return "org.openrewrite.golang.migration.UpgradeGoTo123" }
func (r *UpgradeGoTo123) DisplayName() string { return "Upgrade Go to 1.23" }
func (r *UpgradeGoTo123) Description() string {
	return "Raise the `go` directive in go.mod to Go 1.23, unless it already targets 1.23 or newer."
}
func (r *UpgradeGoTo123) Editor() recipe.TreeVisitor { return upgradeGoVersionEditor("1.23") }

type UpgradeGoTo124 struct{ recipe.Base }

func (r *UpgradeGoTo124) Name() string        { return "org.openrewrite.golang.migration.UpgradeGoTo124" }
func (r *UpgradeGoTo124) DisplayName() string { return "Upgrade Go to 1.24" }
func (r *UpgradeGoTo124) Description() string {
	return "Raise the `go` directive in go.mod to Go 1.24, unless it already targets 1.24 or newer."
}
func (r *UpgradeGoTo124) Editor() recipe.TreeVisitor { return upgradeGoVersionEditor("1.24") }

type UpgradeGoTo125 struct{ recipe.Base }

func (r *UpgradeGoTo125) Name() string        { return "org.openrewrite.golang.migration.UpgradeGoTo125" }
func (r *UpgradeGoTo125) DisplayName() string { return "Upgrade Go to 1.25" }
func (r *UpgradeGoTo125) Description() string {
	return "Raise the `go` directive in go.mod to Go 1.25, unless it already targets 1.25 or newer."
}
func (r *UpgradeGoTo125) Editor() recipe.TreeVisitor { return upgradeGoVersionEditor("1.25") }

type UpgradeGoTo126 struct{ recipe.Base }

func (r *UpgradeGoTo126) Name() string        { return "org.openrewrite.golang.migration.UpgradeGoTo126" }
func (r *UpgradeGoTo126) DisplayName() string { return "Upgrade Go to 1.26" }
func (r *UpgradeGoTo126) Description() string {
	return "Raise the `go` directive in go.mod to Go 1.26, unless it already targets 1.26 or newer."
}
func (r *UpgradeGoTo126) Editor() recipe.TreeVisitor { return upgradeGoVersionEditor("1.26") }
