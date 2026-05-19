/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package codequality

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

func Activate(r *recipe.Registry) {
	recipes.Activate(r)
}
