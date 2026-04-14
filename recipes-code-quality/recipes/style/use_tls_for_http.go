/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	tlsAddr    = template.Expr("addr")
	tlsHandler = template.Expr("handler")
)

var useTlsForHttpImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseTlsForHttp$Impl"),
	template.WithDisplayName("http.ListenAndServe → http.ListenAndServeTLS"),
	template.WithBefore(fmt.Sprintf(`http.ListenAndServe(%s, %s)`, tlsAddr, tlsHandler), template.Imports("net/http")),
	template.WithAfter(fmt.Sprintf(`http.ListenAndServeTLS(%s, "cert.pem", "key.pem", %s)`, tlsAddr, tlsHandler), template.Imports("net/http")),
	template.WithCaptures(tlsAddr, tlsHandler),
)

// UseTlsForHttp replaces calls to `http.ListenAndServe(addr, handler)` with
// `http.ListenAndServeTLS(addr, "cert.pem", "key.pem", handler)`. The
// placeholder cert/key paths must be replaced with actual file paths.
type UseTlsForHttp struct {
	recipe.Base
}

func (r *UseTlsForHttp) Name() string {
	return "org.openrewrite.golang.codequality.UseTlsForHttp"
}
func (r *UseTlsForHttp) DisplayName() string {
	return "Use TLS for HTTP"
}
func (r *UseTlsForHttp) Description() string {
	return "Replace `http.ListenAndServe(addr, handler)` with `http.ListenAndServeTLS(addr, \"cert.pem\", \"key.pem\", handler)` to encrypt traffic in transit."
}
func (r *UseTlsForHttp) Tags() []string { return []string{"security"} }

func (r *UseTlsForHttp) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{useTlsForHttpImpl}
}
