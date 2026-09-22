/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

// Package mapstructurev2 migrates github.com/mitchellh/mapstructure, unmaintained
// since 2023, to the community fork github.com/go-viper/mapstructure/v2.
//
// v2 is a superset of v1 with one removal: the exported `Error` struct, whose
// `Errors []string` field callers ranged over, is gone in favour of
// `errors.Join`, and the v2 `Error` is an unimplementable interface. Everything
// else — `Decode`, `WeakDecode`, `NewDecoder`, every hook constructor, every
// `DecoderConfig` field — survives unchanged, so the rest is a path swap.
package mapstructurev2

import (
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

const (
	oldModule = "github.com/mitchellh/mapstructure"
	newModule = "github.com/go-viper/mapstructure/v2"

	// The fork's current release.
	newVersion = "v2.5.0"

	// removedError is the one v1 export v2 does not carry.
	removedError = "Error"
)

var swapRules = []pathswap.Rule{{Old: oldModule, New: newModule}}

// referencesRemovedError reports whether cu names mapstructure.Error, the single
// v1 export the fork dropped. Such a file cannot compile after the swap, so the
// migration leaves it for the hand rework to errors.Is / errors.As.
func referencesRemovedError(cu *golang.CompilationUnit) bool {
	pkg := pathswap.Qualifier(cu, oldModule)
	if pkg == "" {
		return false
	}
	scan := visitor.Init(&removedErrorScan{pkg: pkg})
	scan.Visit(cu, nil)
	return scan.found
}

type removedErrorScan struct {
	visitor.GoVisitor
	pkg   string
	found bool
}

func (s *removedErrorScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if isRemovedErrorRef(fa, s.pkg) {
		s.found = true
	}
	return s.GoVisitor.VisitFieldAccess(fa, p)
}

// isRemovedErrorRef reports whether fa is a `<pkg>.Error` reference, the shape
// every use takes: `&mapstructure.Error{}`, `*mapstructure.Error` and
// `err.(*mapstructure.Error)` all reach the name through a field access.
func isRemovedErrorRef(fa *java.FieldAccess, pkg string) bool {
	target, ok := fa.Target.(*java.Identifier)
	if !ok || target.Name != pkg {
		return false
	}
	return fa.Name.Element != nil && fa.Name.Element.Name == removedError
}
