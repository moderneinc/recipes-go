/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var shData = template.Expr("shData")

// UseStrongHash replaces weak hash functions (md5, sha1) with sha256 equivalents.
// `md5.New()` and `sha1.New()` become `sha256.New()`.
// `md5.Sum(d)` and `sha1.Sum(d)` become `sha256.Sum256(d)`.
type UseStrongHash struct {
	recipe.Base
}

func (r *UseStrongHash) Name() string {
	return "org.openrewrite.golang.codequality.UseStrongHash"
}
func (r *UseStrongHash) DisplayName() string { return "Use strong hash functions" }
func (r *UseStrongHash) Description() string {
	return "Replace weak hash functions (md5, sha1) with SHA-256 equivalents."
}
func (r *UseStrongHash) Tags() []string { return []string{"style", "security"} }

var useStrongHashMd5New = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseStrongHash$Md5New"),
	template.WithDisplayName("md5.New() -> sha256.New()"),
	template.WithBefore(`md5.New()`, template.Imports("crypto/md5")),
	template.WithAfter(`sha256.New()`, template.Imports("crypto/sha256")),
)

var useStrongHashMd5Sum = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseStrongHash$Md5Sum"),
	template.WithDisplayName("md5.Sum(d) -> sha256.Sum256(d)"),
	template.WithBefore(fmt.Sprintf(`md5.Sum(%s)`, shData), template.Imports("crypto/md5")),
	template.WithAfter(fmt.Sprintf(`sha256.Sum256(%s)`, shData), template.Imports("crypto/sha256")),
	template.WithCaptures(shData),
)

var useStrongHashSha1New = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseStrongHash$Sha1New"),
	template.WithDisplayName("sha1.New() -> sha256.New()"),
	template.WithBefore(`sha1.New()`, template.Imports("crypto/sha1")),
	template.WithAfter(`sha256.New()`, template.Imports("crypto/sha256")),
)

var useStrongHashSha1Sum = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseStrongHash$Sha1Sum"),
	template.WithDisplayName("sha1.Sum(d) -> sha256.Sum256(d)"),
	template.WithBefore(fmt.Sprintf(`sha1.Sum(%s)`, shData), template.Imports("crypto/sha1")),
	template.WithAfter(fmt.Sprintf(`sha256.Sum256(%s)`, shData), template.Imports("crypto/sha256")),
	template.WithCaptures(shData),
)

func (r *UseStrongHash) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		useStrongHashMd5New,
		useStrongHashMd5Sum,
		useStrongHashSha1New,
		useStrongHashSha1Sum,
	}
}
