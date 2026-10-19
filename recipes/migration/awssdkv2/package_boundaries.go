/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"path"
	"strings"
	"unicode"

	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// A package migrates whole or not at all, but packages hand SDK values to one
// another too: a mock package implementing an interface over v1's client, a
// controller taking v1's output from a package that fetched it. Where one side
// of such a boundary moves and the other stays, the types disagree and the
// module stops building. So a package importing one whose exported API names
// aws-sdk-go types moves with it: either both migrate or both stay.

// recordBoundaries notes the packages cu imports and whether its exported API
// names a v1 type.
func (a *migrateAcc) recordBoundaries(cu *golang.CompilationUnit) {
	dir := path.Dir(cu.SourcePath)
	if cu.Imports != nil {
		for _, rp := range cu.Imports.Elements {
			if a.dirImports[dir] == nil {
				a.dirImports[dir] = map[string]bool{}
			}
			a.dirImports[dir][rp.Element.Path()] = true
		}
	}
	if strings.HasSuffix(cu.SourcePath, "_test.go") || !importsV1(cu) {
		return
	}
	locals := map[string]bool{}
	for _, rp := range cu.Imports.Elements {
		if p := rp.Element.Path(); underV1(p) || p == gureguV1 {
			if local := pathswap.LocalName(rp.Element); local != "" {
				locals[local] = true
			}
		}
	}
	if exportsV1Types(cu, locals) {
		a.sdkAPI[dir] = true
	}
}

// exportsV1Types reports whether an exported declaration of cu names a type
// from one of the v1 packages bound to locals.
func exportsV1Types(cu *golang.CompilationUnit, locals map[string]bool) bool {
	scan := visitor.Init(&v1TypeScan{locals: locals})
	exported := func(id *java.Identifier) bool {
		return id != nil && id.Name != "" && unicode.IsUpper(rune(id.Name[0]))
	}
	for _, rp := range cu.Statements {
		switch d := rp.Element.(type) {
		case *golang.TypeDecl:
			if exported(d.Name) {
				scan.Visit(d.Definition, nil)
			}
		case *java.MethodDeclaration:
			if exported(d.Name) {
				for _, param := range d.Parameters.Elements {
					scan.Visit(param.Element, nil)
				}
				if d.ReturnType != nil {
					scan.Visit(d.ReturnType, nil)
				}
			}
		case *golang.MethodDeclaration:
			if md := d.Declaration; md != nil && exported(md.Name) {
				for _, param := range md.Parameters.Elements {
					scan.Visit(param.Element, nil)
				}
				if md.ReturnType != nil {
					scan.Visit(md.ReturnType, nil)
				}
			}
		case *java.VariableDeclarations:
			for _, v := range d.Variables {
				if v.Element != nil && exported(v.Element.Name) && d.TypeExpr != nil {
					scan.Visit(d.TypeExpr, nil)
				}
			}
		}
		if scan.found {
			return true
		}
	}
	return false
}

type v1TypeScan struct {
	visitor.GoVisitor
	locals map[string]bool
	found  bool
}

func (v *v1TypeScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if target, isIdent := fa.Target.(*java.Identifier); isIdent && v.locals[target.Name] {
		v.found = true
	}
	return v.GoVisitor.VisitFieldAccess(fa, p)
}

// holdBackAcrossBoundaries holds back each package on the far side of an SDK
// boundary from one that stays, repeating until nothing changes.
func (a *migrateAcc) holdBackAcrossBoundaries() {
	if a.modulePath == "" {
		return
	}
	for changed := true; changed; {
		changed = false
		for _, dir := range sortedKeys(a.dirImports) {
			for _, imported := range sortedKeys(a.dirImports[dir]) {
				target, inModule := strings.CutPrefix(imported, a.modulePath+"/")
				if !inModule || target == dir || !a.sdkAPI[target] {
					continue
				}
				_, dirHeld := a.heldBack[dir]
				_, targetHeld := a.heldBack[target]
				switch {
				case targetHeld && !dirHeld && a.usesSDK(dir):
					a.holdBackDir(dir, "imports "+target+", whose API names aws-sdk-go types and stays on v1")
					changed = true
				case dirHeld && !targetHeld:
					a.holdBackDir(target, "its API names aws-sdk-go types, and "+dir+", which imports it, stays on v1")
					changed = true
				}
			}
		}
	}
}

// usesSDK reports whether a package's files would change at all.
func (a *migrateAcc) usesSDK(dir string) bool {
	for imported := range a.dirImports[dir] {
		if underV1(imported) || imported == gureguV1 {
			return true
		}
	}
	return false
}

// holdBackDir holds back every file of a package the scan saw.
func (a *migrateAcc) holdBackDir(dir, reason string) {
	for file := range a.fileDecls {
		if path.Dir(file) == dir {
			a.holdBack(file, reason)
		}
	}
}
