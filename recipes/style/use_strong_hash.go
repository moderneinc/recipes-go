/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Replaces the weak hash constructors md5.New() and sha1.New() with
// sha256.New(), leaving md5.Sum/sha1.Sum alone since their [16]byte/[20]byte
// results differ from sha256.Sum256's [32]byte and would need a whole-usage
// migration.
type UseStrongHash struct {
	recipe.Base
}

func (r *UseStrongHash) Name() string {
	return "org.openrewrite.golang.codequality.UseStrongHash"
}
func (r *UseStrongHash) DisplayName() string { return "Use strong hash functions" }
func (r *UseStrongHash) Description() string {
	return "Replace weak hash constructors (md5.New, sha1.New) with sha256.New."
}
func (r *UseStrongHash) Tags() []string { return []string{"style", "security"} }

func (r *UseStrongHash) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "G401", Tool: diagnostic.GolangciLint, HasFix: false},
		{DiagnosticID: "G501", Tool: diagnostic.GolangciLint, HasFix: false},
		{DiagnosticID: "G505", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *UseStrongHash) Editor() recipe.TreeVisitor {
	return visitor.Init(&useStrongHashVisitor{})
}

var (
	md5NewPattern     = template.Expression(`md5.New()`).Imports("crypto/md5").Build()
	sha1NewPattern    = template.Expression(`sha1.New()`).Imports("crypto/sha1").Build()
	sha256NewTemplate = template.ExpressionTemplate(`sha256.New()`).Imports("crypto/sha256").Build()
)

type useStrongHashVisitor struct {
	visitor.GoVisitor
}

func (v *useStrongHashVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := md5NewPattern.Match(mi, nil)
	if match == nil {
		match = sha1NewPattern.Match(mi, nil)
	}
	if match == nil {
		return mi
	}

	replaced, ok := sha256NewTemplate.Apply(nil, match).(*java.MethodInvocation)
	if !ok {
		return mi
	}

	recipegolang.MaybeAddImport(v, "crypto/sha256", nil, false)
	v.DoAfterVisit(recipe.Service[*recipegolang.ImportService](nil).RemoveUnusedImportsVisitor())
	return replaced.WithPrefix(mi.GetPrefix())
}
