/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"go/version"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

const targetGoVersion = "1.26"

// UpgradeGoTo126 raises the `go` directive in go.mod to Go 1.26. It builds on
// ChangeGoVersion: the rewrite itself is ChangeGoVersionVisitor's, and this
// recipe only adds a no-downgrade guard so a module already declaring 1.26 or
// newer is left untouched.
type UpgradeGoTo126 struct {
	recipe.Base
}

func (r *UpgradeGoTo126) Name() string { return "org.openrewrite.golang.codequality.UpgradeGoTo126" }

func (r *UpgradeGoTo126) DisplayName() string { return "Upgrade Go to 1.26" }

func (r *UpgradeGoTo126) Description() string {
	return "Raise the `go` directive in go.mod to Go 1.26, unless it already targets 1.26 or newer."
}

func (r *UpgradeGoTo126) Editor() recipe.TreeVisitor {
	return visitor.Init(&upgradeGoTo126Visitor{
		ChangeGoVersionVisitor: ChangeGoVersionVisitor{NewVersion: targetGoVersion},
	})
}

type upgradeGoTo126Visitor struct {
	ChangeGoVersionVisitor
}

func (v *upgradeGoTo126Visitor) VisitGoModDirective(d *golang.GoModDirective, p any) java.Tree {
	if d.Keyword == "go" && len(d.Values) == 1 &&
		version.Compare("go"+d.Values[0].Text, "go"+targetGoVersion) >= 0 {
		return d
	}
	return v.ChangeGoVersionVisitor.VisitGoModDirective(d, p)
}
