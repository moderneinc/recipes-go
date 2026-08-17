/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AdoptTestifyRequireError replaces `if err == nil { t.Fatal("...") }` guards in
// tests with `require.Error(t, err)` — the assertion that an error is present.
type AdoptTestifyRequireError struct {
	recipe.Base
}

func (r *AdoptTestifyRequireError) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyRequireError"
}
func (r *AdoptTestifyRequireError) DisplayName() string {
	return "Adopt testify require.Error"
}
func (r *AdoptTestifyRequireError) Description() string {
	return "Replace `if err == nil { t.Fatal(\"...\") }` guards in tests with `require.Error(t, err)` from `github.com/stretchr/testify/require`."
}
func (r *AdoptTestifyRequireError) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyRequireError) Editor() recipe.TreeVisitor {
	return visitor.Init(&errGuardVisitor{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter, op: java.Equal, assertion: "Error"})
}

// AdoptTestifyAssertError replaces `if err == nil { t.Error("...") }` guards in
// tests with `assert.Error(t, err)`.
type AdoptTestifyAssertError struct {
	recipe.Base
}

func (r *AdoptTestifyAssertError) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyAssertError"
}
func (r *AdoptTestifyAssertError) DisplayName() string {
	return "Adopt testify assert.Error"
}
func (r *AdoptTestifyAssertError) Description() string {
	return "Replace `if err == nil { t.Error(\"...\") }` guards in tests with `assert.Error(t, err)` from `github.com/stretchr/testify/assert`."
}
func (r *AdoptTestifyAssertError) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyAssertError) Editor() recipe.TreeVisitor {
	return visitor.Init(&errGuardVisitor{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter, op: java.Equal, assertion: "Error"})
}
