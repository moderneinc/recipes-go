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
	return visitor.Init(&migrateImportOnlyVisitor{})
}

type migrateImportOnlyVisitor struct {
	visitor.GoVisitor
}

func (v *migrateImportOnlyVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	jsonPkg := regularJsonPackage(cu)
	if jsonPkg == "" {
		return cu
	}
	if importsEncodingJsonV2(cu) {
		return cu
	}
	// A time.Duration field marshals to a runtime error under v2 without an
	// explicit format, so the file is left for review.
	if fileHasDurationField(cu) {
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
