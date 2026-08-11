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

// Removes the dot alias from `import . "pkg"` and re-qualifies the references it
// brought into scope, e.g. `Println` to `fmt.Println`.
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
	// dotPkgs maps a dot-imported package path to its re-qualification info,
	// rebuilt per compilation unit.
	dotPkgs map[string]dotImport
}

// Pre-scans imports to record which packages are dot-imported (imports are
// visited before the body), so references can be matched and re-qualified.
func (v *avoidDotImportVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.dotPkgs = map[string]dotImport{}
	if cu.Imports != nil {
		// Record packages already imported normally, so stripping a dot alias for
		// one of them does not create a duplicate import.
		normal := map[string]bool{}
		for _, rp := range cu.Imports.Elements {
			if imp := rp.Element; !isDotImport(imp) {
				if path := importPath(imp); path != "" {
					normal[path] = true
				}
			}
		}
		for _, rp := range cu.Imports.Elements {
			imp := rp.Element
			if !isDotImport(imp) {
				continue
			}
			fq, ok := imp.Alias.Element.Type.(java.FullyQualified)
			if !ok || fq == nil {
				continue
			}
			path := fq.GetFullyQualifiedName()
			if path == "" || normal[path] {
				continue
			}
			// Only register packages whose name we can confidently derive.
			name, confident := packageName(path)
			if !confident {
				continue
			}
			v.dotPkgs[path] = dotImport{qualifier: name, pkgType: imp.Alias.Element.Type}
		}
	}
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

// isDotImport reports whether imp uses the "." alias.
func isDotImport(imp *java.Import) bool {
	return imp.Alias != nil && imp.Alias.Element != nil && imp.Alias.Element.Name == "."
}

// importPath returns the package path of a non-dot import, unquoted.
func importPath(imp *java.Import) string {
	lit, ok := imp.Qualid.(*java.Literal)
	if !ok {
		return ""
	}
	if s, ok := lit.Value.(string); ok && s != "" {
		return strings.Trim(s, "\"`")
	}
	return strings.Trim(lit.Source, "\"`")
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

	// Only remove the alias for a package we know how to re-qualify, else leave
	// the dot import intact.
	fq, ok := aliasIdent.Type.(java.FullyQualified)
	if !ok || fq == nil {
		return imp
	}
	if _, ok := v.dotPkgs[fq.GetFullyQualifiedName()]; !ok {
		return imp
	}

	// Remove the dot alias and clear the qualid's leading whitespace, now covered
	// by the import prefix.
	c := *imp
	c.Alias = nil
	if lit, ok := c.Qualid.(*java.Literal); ok {
		c.Qualid = lit.WithPrefix(java.EmptySpace)
	} else if ident, ok := c.Qualid.(*java.Identifier); ok {
		c.Qualid = ident.WithPrefix(java.EmptySpace)
	}
	return &c
}

// Re-qualifies a free function call from a dot-imported package, e.g.
// `Println("x")` to `fmt.Println("x")`.
func (v *avoidDotImportVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select != nil {
		return mi
	}
	dp, ok := v.callPackage(mi)
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

// callPackage returns the dot-imported package a receiver-less call resolves to,
// via the function's declaring type or, for a var holding a func, the name's owner.
func (v *avoidDotImportVisitor) callPackage(mi *java.MethodInvocation) (dotImport, bool) {
	if mt := mi.MethodType; mt != nil && mt.DeclaringType != nil {
		if dp, ok := v.dotPkgs[mt.DeclaringType.GetFullyQualifiedName()]; ok {
			return dp, true
		}
	}
	if mi.Name != nil && mi.Name.FieldType != nil {
		if owner, ok := mi.Name.FieldType.Owner.(java.FullyQualified); ok && owner != nil {
			if dp, ok := v.dotPkgs[owner.GetFullyQualifiedName()]; ok {
				return dp, true
			}
		}
	}
	return dotImport{}, false
}

// Re-qualifies a bare reference from a dot-imported package, e.g. `Writer` to
// `io.Writer` or a `math.Pi` constant used bare.
func (v *avoidDotImportVisitor) VisitIdentifier(ident *java.Identifier, p any) java.J {
	ident = v.GoVisitor.VisitIdentifier(ident, p).(*java.Identifier)

	if parent := v.Cursor().Parent(); parent != nil {
		// Skip an identifier already used as a FieldAccess selector (idempotency).
		if fa, ok := parent.Value().(*java.FieldAccess); ok && fa.Name.Element == v.Cursor().Value() {
			return ident
		}
		// A call's name is re-qualified by VisitMethodInvocation, not here.
		if mi, ok := parent.Value().(*java.MethodInvocation); ok && mi.Name == v.Cursor().Value() {
			return ident
		}
	}

	// A const/var reference carries a FieldType whose Owner is its package.
	if ft := ident.FieldType; ft != nil {
		if owner, ok := ft.Owner.(java.FullyQualified); ok && owner != nil && ft.Name == ident.Name {
			if dp, found := v.dotPkgs[owner.GetFullyQualifiedName()]; found {
				return v.qualify(ident, dp)
			}
		}
		return ident
	}

	// A function used as a value (a bare reference, not a call).
	if mt, ok := ident.Type.(*java.JavaTypeMethod); ok && mt.DeclaringType != nil {
		if dp, found := v.dotPkgs[mt.DeclaringType.GetFullyQualifiedName()]; found {
			return v.qualify(ident, dp)
		}
		return ident
	}

	// A type reference's fully-qualified name is `<dot-package>.<Name>`.
	fq, ok := ident.Type.(java.FullyQualified)
	if !ok || fq == nil {
		return ident
	}
	fqn := fq.GetFullyQualifiedName()
	for path, dp := range v.dotPkgs {
		if fqn == path+"."+ident.Name {
			return v.qualify(ident, dp)
		}
	}
	return ident
}

// qualify wraps ident in a `<qualifier>.<ident>` FieldAccess for a dot-imported package.
func (v *avoidDotImportVisitor) qualify(ident *java.Identifier, dp dotImport) java.J {
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

// Derives the package-name qualifier from the import path's last segment (or the
// part before a gopkg.in ".vN" suffix), reporting confident=false for ambiguous
// "/vN" and non-identifier segments.
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
		// Bare "/vN" is ambiguous across module conventions, so decline.
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
