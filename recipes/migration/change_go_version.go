/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/preconditions"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// ChangeGoVersion rewrites the `go` directive in go.mod to a given version.
// It sets the version unconditionally; for upgrade-only semantics (never
// downgrade) compose ChangeGoVersionVisitor behind a version guard, as
// UpgradeGoTo126 does.
type ChangeGoVersion struct {
	recipe.Base
	NewVersion string
}

func (r *ChangeGoVersion) Name() string { return "org.openrewrite.golang.migration.ChangeGoVersion" }

func (r *ChangeGoVersion) DisplayName() string { return "Change the `go` directive version" }

func (r *ChangeGoVersion) Description() string {
	return "Rewrites the `go` directive in go.mod to a new version."
}

func (r *ChangeGoVersion) Options() []recipe.OptionDescriptor {
	return []recipe.OptionDescriptor{
		recipe.Option("newVersion", "New version",
			"The version to set on the `go` directive, e.g. `1.26`.").
			WithExample("1.26").
			WithValue(r.NewVersion),
	}
}

func (r *ChangeGoVersion) Editor() recipe.TreeVisitor {
	return preconditions.Check(
		preconditions.HasSourcePath("**/go.mod"),
		visitor.Init(&ChangeGoVersionVisitor{NewVersion: r.NewVersion}),
	)
}

// ChangeGoVersionVisitor rewrites the single-valued `go` directive to
// NewVersion. It is exported so upgrade-style recipes can embed it and add
// their own gating (e.g. a no-downgrade check) on top of the rewrite.
type ChangeGoVersionVisitor struct {
	visitor.GoVisitor
	NewVersion string
}

func (v *ChangeGoVersionVisitor) VisitGoModDirective(d *golang.GoModDirective, p any) java.Tree {
	d = v.GoVisitor.VisitGoModDirective(d, p).(*golang.GoModDirective)
	if d.Keyword == "go" && len(d.Values) == 1 {
		if updated := d.Values[0].WithText(v.NewVersion); updated != d.Values[0] {
			d = d.WithValues([]*golang.GoModValue{updated})
		}
	}
	return d
}
