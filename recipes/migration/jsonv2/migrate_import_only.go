/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Swaps the encoding/json import to encoding/json/v2 for a file whose json usage
// is entirely v2-compatible, so no code changes are needed.
type MigrateImportOnlyToJSONV2 struct {
	recipe.Base
	preserveV1 bool
}

func (r *MigrateImportOnlyToJSONV2) Name() string {
	return "org.openrewrite.golang.migration.MigrateImportOnlyToJSONV2"
}
func (r *MigrateImportOnlyToJSONV2) DisplayName() string {
	return "Migrate an import-only `encoding/json` file to `encoding/json/v2`"
}
func (r *MigrateImportOnlyToJSONV2) Description() string {
	return "Swap the `encoding/json` import to `encoding/json/v2` for a file whose entire json usage already exists in v2 (`Marshal`, `Unmarshal`, `Marshaler`, `Unmarshaler`), so only the import changes and the calls adopt v2 semantics."
}
func (r *MigrateImportOnlyToJSONV2) Tags() []string { return []string{"migration", "json"} }

func (r *MigrateImportOnlyToJSONV2) Editor() recipe.TreeVisitor {
	return visitor.Init(&migrateImportOnlyVisitor{preserveV1: r.preserveV1})
}

type migrateImportOnlyVisitor struct {
	visitor.GoVisitor
	preserveV1 bool
}

func (v *migrateImportOnlyVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	jsonPkg := regularJsonPackage(cu)
	if jsonPkg == "" {
		return cu
	}
	if importsEncodingJsonV2(cu) {
		return cu
	}
	// A time.Duration or fixed [N]byte field encodes incompatibly under bare v2,
	// so the default path leaves the file for review; the compat path migrates it,
	// since PreserveV1Semantics restores the v1 encoding.
	if !v.preserveV1 && fileNeedsV1Compat(cu) {
		return cu
	}

	scan := visitor.Init(&importOnlyScan{jsonPkg: jsonPkg})
	scan.Visit(cu, nil)
	// Migrate only when every json touchpoint already exists in v2 and at least
	// one such reference remains, so the swapped import is used and nothing is
	// stranded.
	if scan.blocked || !scan.jsonReferenced {
		return cu
	}

	queueImportSwapToV2(v)
	return cu
}

type importOnlyScan struct {
	visitor.GoVisitor
	jsonPkg        string
	blocked        bool
	jsonReferenced bool
}

func (s *importOnlyScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if s.blocked {
		return mi
	}
	// A method call on a v1 Marshaler/Unmarshaler (both survive in v2) is fine, so
	// only a json package function that v2 dropped blocks the file; a removed
	// Encoder/Decoder value is already caught by its constructor or type reference.
	if mi.Name != nil && selectIdentName(mi) == s.jsonPkg {
		if !isV2SurvivingJsonName(mi.Name.Name) {
			s.blocked = true
			return mi
		}
		s.jsonReferenced = true
	}
	return s.GoVisitor.VisitMethodInvocation(mi, p)
}

func (s *importOnlyScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if s.blocked {
		return fa
	}
	if ident, ok := fa.Target.(*java.Identifier); ok && ident.Name == s.jsonPkg {
		if fa.Name.Element == nil || !isV2SurvivingJsonName(fa.Name.Element.Name) {
			s.blocked = true
			return fa
		}
		s.jsonReferenced = true
	}
	return s.GoVisitor.VisitFieldAccess(fa, p)
}
