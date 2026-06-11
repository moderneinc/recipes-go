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
	httpGetUrl = template.Expr("url")

	httpPostUrl         = template.Expr("postUrl")
	httpPostContentType = template.Expr("contentType")
	httpPostBody        = template.Expr("body")

	httpHeadUrl = template.Expr("headUrl")

	httpPostFormUrl    = template.Expr("pfUrl")
	httpPostFormValues = template.Expr("pfValues")
)

var (
	useCustomHttpClientGet = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.UseCustomHttpClient$Get"),
		template.WithDisplayName("http.Get → http.DefaultClient.Get"),
		template.WithBefore(fmt.Sprintf(`http.Get(%s)`, httpGetUrl), template.Imports("net/http")),
		template.WithAfter(fmt.Sprintf(`http.DefaultClient.Get(%s)`, httpGetUrl), template.Imports("net/http")),
		template.WithCaptures(httpGetUrl),
	)

	useCustomHttpClientPost = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.UseCustomHttpClient$Post"),
		template.WithDisplayName("http.Post → http.DefaultClient.Post"),
		template.WithBefore(fmt.Sprintf(`http.Post(%s, %s, %s)`, httpPostUrl, httpPostContentType, httpPostBody), template.Imports("net/http")),
		template.WithAfter(fmt.Sprintf(`http.DefaultClient.Post(%s, %s, %s)`, httpPostUrl, httpPostContentType, httpPostBody), template.Imports("net/http")),
		template.WithCaptures(httpPostUrl, httpPostContentType, httpPostBody),
	)

	useCustomHttpClientHead = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.UseCustomHttpClient$Head"),
		template.WithDisplayName("http.Head → http.DefaultClient.Head"),
		template.WithBefore(fmt.Sprintf(`http.Head(%s)`, httpHeadUrl), template.Imports("net/http")),
		template.WithAfter(fmt.Sprintf(`http.DefaultClient.Head(%s)`, httpHeadUrl), template.Imports("net/http")),
		template.WithCaptures(httpHeadUrl),
	)

	useCustomHttpClientPostForm = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.UseCustomHttpClient$PostForm"),
		template.WithDisplayName("http.PostForm → http.DefaultClient.PostForm"),
		template.WithBefore(fmt.Sprintf(`http.PostForm(%s, %s)`, httpPostFormUrl, httpPostFormValues), template.Imports("net/http")),
		template.WithAfter(fmt.Sprintf(`http.DefaultClient.PostForm(%s, %s)`, httpPostFormUrl, httpPostFormValues), template.Imports("net/http")),
		template.WithCaptures(httpPostFormUrl, httpPostFormValues),
	)
)

// UseCustomHttpClient replaces calls to `http.Get`, `http.Post`, `http.Head`,
// and `http.PostForm` with their `http.DefaultClient` equivalents. These
// convenience functions use the default HTTP client which has no timeout
// configured, potentially leading to resource leaks. Making the default client
// explicit is the first step toward replacing it with a custom client that has
// appropriate timeouts.
type UseCustomHttpClient struct {
	recipe.Base
}

func (r *UseCustomHttpClient) Name() string {
	return "org.openrewrite.golang.codequality.UseCustomHttpClient"
}
func (r *UseCustomHttpClient) DisplayName() string { return "Use custom HTTP client" }
func (r *UseCustomHttpClient) Description() string {
	return "Replace `http.Get`, `http.Post`, `http.Head`, and `http.PostForm` with `http.DefaultClient` equivalents to make the default client explicit."
}
func (r *UseCustomHttpClient) Tags() []string { return []string{"style", "net/http"} }

func (r *UseCustomHttpClient) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		useCustomHttpClientGet,
		useCustomHttpClientPost,
		useCustomHttpClientHead,
		useCustomHttpClientPostForm,
	}
}
