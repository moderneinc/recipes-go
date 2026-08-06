/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
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

var useStrongHashMd5New = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseStrongHash$Md5New"),
	template.WithDisplayName("md5.New() -> sha256.New()"),
	template.WithBefore(`md5.New()`, template.Imports("crypto/md5")),
	template.WithAfter(`sha256.New()`, template.Imports("crypto/sha256"), template.SourceImports("crypto/sha256")),
)

var useStrongHashSha1New = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseStrongHash$Sha1New"),
	template.WithDisplayName("sha1.New() -> sha256.New()"),
	template.WithBefore(`sha1.New()`, template.Imports("crypto/sha1")),
	template.WithAfter(`sha256.New()`, template.Imports("crypto/sha256"), template.SourceImports("crypto/sha256")),
)

func (r *UseStrongHash) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		useStrongHashMd5New,
		useStrongHashSha1New,
	}
}
