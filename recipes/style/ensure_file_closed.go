/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// osFileOpenMethods lists os package functions that open files.
var osFileOpenMethods = map[string]bool{
	"Open":     true,
	"Create":   true,
	"OpenFile": true,
}

// EnsureFileClosed finds calls to `os.Open()`, `os.Create()`, and
// `os.OpenFile()` and inserts `defer f.Close()` after the assignment.
type EnsureFileClosed struct {
	recipe.Base
}

func (r *EnsureFileClosed) Name() string {
	return "org.openrewrite.golang.codequality.EnsureFileClosed"
}
func (r *EnsureFileClosed) DisplayName() string { return "Ensure file closed" }
func (r *EnsureFileClosed) Description() string {
	return "Find calls to `os.Open`, `os.Create`, and `os.OpenFile`. Ensure the returned file is closed to avoid resource leaks."
}
func (r *EnsureFileClosed) Tags() []string { return []string{"style", "os"} }

func (r *EnsureFileClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureFileClosedVisitor{})
}

type ensureFileClosedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureFileClosedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	return insertDeferMethodCall(block, isOsFileOpen, "Close")
}

// isOsFileOpen returns true if the method invocation is os.Open, os.Create, or os.OpenFile.
func isOsFileOpen(a acquisition) bool {
	declaring, ok := declaringType(a.call)
	if !ok || declaring != "os" {
		return false
	}
	return osFileOpenMethods[a.call.Name.Name]
}
