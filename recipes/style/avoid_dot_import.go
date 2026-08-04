/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Removes the dot alias from `import . "pkg"`, converting it to a normal
// `import "pkg"` and re-qualifying the references it brought into scope
// (`Println(...)` -> `fmt.Println(...)`, `Writer` -> `io.Writer`).
//
// Re-qualification is type-attribution driven, so it is scope-aware: locals,
// fields, and selectors that merely share a name are left alone. Only functions
// and type references are handled; dot-imported constants and package variables
// carry no owning-package attribution and are left as-is. When the qualifier
// cannot be confidently inferred from the path (see packageName), the dot import
// is left untouched rather than re-qualified with a wrong name.
//
// golangci-lint: revive (dot-imports)
type AvoidDotImport struct {
	recipe.Base
}

func (r *AvoidDotImport) Name() string {
	return "org.openrewrite.golang.codequality.AvoidDotImport"
}
func (r *AvoidDotImport) DisplayName() string { return "Avoid dot imports" }
func (r *AvoidDotImport) Description() string {
	return "Remove the dot alias from `import . \"pkg\"`, converting to a normal import and re-qualifying references (e.g. `Println` to `fmt.Println`)."
}
func (r *AvoidDotImport) Tags() []string { return []string{"style", "lint"} }

func (r *AvoidDotImport) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "dot-imports", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *AvoidDotImport) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidDotImportVisitor{})
}

// dotImport holds what is needed to re-qualify references to a dot-imported
// package: the qualifier to prepend and the package's own type.
type dotImport struct {
	qualifier string
	pkgType   java.JavaType
}

type avoidDotImportVisitor struct {
	visitor.GoVisitor
	// dotPkgs maps a dot-imported package path (e.g. "fmt") to its
	// re-qualification info. Rebuilt per compilation unit.
	dotPkgs map[string]dotImport
}

// Pre-scans imports to record which packages are dot-imported (imports are
// visited before the body), so references can be matched and re-qualified.
func (v *avoidDotImportVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.dotPkgs = map[string]dotImport{}
	if cu.Imports != nil {
		for _, rp := range cu.Imports.Elements {
			imp := rp.Element
			if imp.Alias == nil || imp.Alias.Element == nil || imp.Alias.Element.Name != "." {
				continue
			}
			fq, ok := imp.Alias.Element.Type.(java.FullyQualified)
			if !ok || fq == nil {
				continue
			}
			path := fq.GetFullyQualifiedName()
			if path == "" {
				continue
			}
			// Only register packages whose name we can confidently derive; a
			// low-confidence path is left untouched by VisitImport rather than
			// re-qualified with a guess.
			name, confident := packageName(path)
			if !confident {
				continue
			}
			v.dotPkgs[path] = dotImport{qualifier: name, pkgType: imp.Alias.Element.Type}
		}
	}
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *avoidDotImportVisitor) VisitImport(imp *java.Import, p any) java.J {
	imp = v.GoVisitor.VisitImport(imp, p).(*java.Import)

	if imp.Alias == nil {
		return imp
	}

	aliasIdent := imp.Alias.Element
	if aliasIdent.Name != "." {
		return imp
	}

	// Only remove the alias for a package we know how to re-qualify (registered
	// above). Otherwise leave the dot import rather than strand its references.
	fq, ok := aliasIdent.Type.(java.FullyQualified)
	if !ok || fq == nil {
		return imp
	}
	if _, ok := v.dotPkgs[fq.GetFullyQualifiedName()]; !ok {
		return imp
	}

	// Remove the dot alias, converting `import . "pkg"` to `import "pkg"`.
	// The qualid's prefix was the space between "." and the path string.
	// With the alias gone, the import prefix covers the space between
	// "import" and the path, so we clear the qualid's leading whitespace.
	c := *imp
	c.Alias = nil
	if lit, ok := c.Qualid.(*java.Literal); ok {
		c.Qualid = lit.WithPrefix(java.EmptySpace)
	} else if ident, ok := c.Qualid.(*java.Identifier); ok {
		c.Qualid = ident.WithPrefix(java.EmptySpace)
	}
	return &c
}

// Re-qualifies a free function call from a dot-imported package:
// `Println("x")` -> `fmt.Println("x")`. A call with a receiver is left alone,
// which also makes the pass idempotent.
func (v *avoidDotImportVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select != nil || mi.MethodType == nil || mi.MethodType.DeclaringType == nil {
		return mi
	}
	dp, ok := v.dotPkgs[mi.MethodType.DeclaringType.GetFullyQualifiedName()]
	if !ok {
		return mi
	}

	c := *mi
	c.Select = &java.RightPadded[java.Expression]{
		Element: &java.Identifier{
			ID:     uuid.New(),
			Prefix: java.EmptySpace,
			Name:   dp.qualifier,
			Type:   dp.pkgType,
		},
		After: java.EmptySpace,
	}
	return &c
}

// Re-qualifies a bare type reference from a dot-imported package:
// `Writer` -> `io.Writer`. Fires only for type references (nil FieldType) whose
// FQN is `<dot-package>.<Name>`, so variables, constants, fields, and selectors
// sharing the name are left alone.
func (v *avoidDotImportVisitor) VisitIdentifier(ident *java.Identifier, p any) java.J {
	ident = v.GoVisitor.VisitIdentifier(ident, p).(*java.Identifier)

	// Variable/const uses carry a FieldType; only type references have none.
	if ident.FieldType != nil {
		return ident
	}
	fq, ok := ident.Type.(java.FullyQualified)
	if !ok || fq == nil {
		return ident
	}
	// Skip an identifier already used as a FieldAccess selector, so re-running is
	// a no-op.
	if parent := v.Cursor().Parent(); parent != nil {
		if fa, ok := parent.Value().(*java.FieldAccess); ok && fa.Name.Element == v.Cursor().Value() {
			return ident
		}
	}
	fqn := fq.GetFullyQualifiedName()
	for path, dp := range v.dotPkgs {
		if fqn == path+"."+ident.Name {
			return &java.FieldAccess{
				ID:     uuid.New(),
				Prefix: ident.Prefix,
				Target: &java.Identifier{
					ID:     uuid.New(),
					Prefix: java.EmptySpace,
					Name:   dp.qualifier,
					Type:   dp.pkgType,
				},
				Name: java.LeftPadded[*java.Identifier]{
					Before:  java.EmptySpace,
					Element: ident.WithPrefix(java.EmptySpace),
				},
				Type: ident.Type,
			}
		}
	}
	return ident
}

// Derives the qualifier (the package clause name) from an import path and
// reports whether the derivation is trustworthy. The declared name is not in the
// LST, so it is inferred from the last segment ("math/rand" -> "rand"), or the
// part before a gopkg.in ".vN" suffix ("gopkg.in/yaml.v2" -> "yaml").
//
// A bare "/vN" segment is left unresolved: it may be a semantic-import-
// versioning module suffix ("github.com/onsi/ginkgo/v2" is package "ginkgo") or
// a version directory ("k8s.io/api/core/v1" is package "v1",
// "k8s.io/api/autoscaling/v2" is package "v2", ".../servicemanagement/v1" is
// package "servicemanagement"). Which one holds depends on the module's major
// version, not the path, so it is not confident. A segment that is not a valid
// Go identifier ("go-bar") is likewise not confident.
func packageName(importPath string) (name string, confident bool) {
	last := importPath
	if i := strings.LastIndex(importPath, "/"); i >= 0 {
		last = importPath[i+1:]
	}
	switch {
	case strings.ContainsRune(last, '.'):
		// A package name can't contain a dot, so a dotted last segment differs
		// from the name; gopkg.in encodes it as "<name>.vN".
		if dot := strings.LastIndexByte(last, '.'); isVersionTag(last[dot+1:]) {
			name = last[:dot]
			return name, isIdentifier(name)
		}
		return last, false
	case isVersionTag(last):
		// Bare "/vN": ambiguous across module conventions (see above), so decline.
		return last, false
	default:
		// The last segment is the name only if it is a usable identifier; a
		// segment like "go-bar" would yield the invalid qualifier "go-bar.".
		return last, isIdentifier(last)
	}
}

// Reports whether s is a valid Go identifier: a non-empty run of letters,
// digits, and underscores that does not start with a digit.
func isIdentifier(s string) bool {
	for i, r := range s {
		switch {
		case r == '_' || unicode.IsLetter(r):
			// Always allowed.
		case i > 0 && unicode.IsDigit(r):
			// Allowed after the first rune.
		default:
			return false
		}
	}
	return s != ""
}

// Reports whether s is a version tag: "v" followed by one or more digits
// (v0, v1, v2, ...).
func isVersionTag(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
