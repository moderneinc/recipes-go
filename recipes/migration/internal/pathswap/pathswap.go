/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

// Package pathswap moves a Go package from one import path to another, for the
// library migrations whose replacement keeps the old API under a new path.
//
// Rewriting the import literal alone is not enough. When the new path's last
// segment matches the old one the call sites are textually identical, so nothing
// else in the file changes — and every reference keeps the parse-time
// attribution naming the dead path. RemoveUnusedImports answers "is this package
// still referenced?" from those types, so it would drop the import the swap just
// introduced. Retype fixes the attribution to match the rewritten import.
package pathswap

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Rule maps an import path prefix to its replacement. Old matches the path
// exactly or as a parent of it, so one rule covers a module and every package
// under it: `github.com/golang/mock` → `go.uber.org/mock` also moves
// `github.com/golang/mock/gomock`.
type Rule struct {
	Old string
	New string
	// KeepName binds the replacement to the name the old path bound, through an
	// explicit alias, for a move that renames the package but leaves its API in
	// place. Without it every reference in the file would have to be rewritten.
	KeepName bool
	// Alias binds the replacement to a name of the caller's choosing, for a
	// destination whose own name the file already uses for something else.
	Alias string
}

// MapPath returns the replacement for path under the first matching rule, and
// whether any rule matched.
func MapPath(path string, rules []Rule) (string, bool) {
	for _, r := range rules {
		if path == r.Old {
			return r.New, true
		}
		if strings.HasPrefix(path, r.Old+"/") {
			return r.New + path[len(r.Old):], true
		}
	}
	return "", false
}

// matchRule is MapPath, also returning the rule that matched.
func matchRule(path string, rules []Rule) (string, Rule, bool) {
	for _, r := range rules {
		if path == r.Old {
			return r.New, r, true
		}
		if strings.HasPrefix(path, r.Old+"/") {
			return r.New + path[len(r.Old):], r, true
		}
	}
	return "", Rule{}, false
}

// mapFQN returns the replacement for a fully qualified type name. A Go type FQN
// is `<import path>.<Name>` and a package's own declaring type is the bare
// import path, so a rule matches through either separator.
func mapFQN(fqn string, rules []Rule) (string, bool) {
	for _, r := range rules {
		if fqn == r.Old {
			return r.New, true
		}
		if strings.HasPrefix(fqn, r.Old+"/") || strings.HasPrefix(fqn, r.Old+".") {
			return r.New + fqn[len(r.Old):], true
		}
	}
	return "", false
}

// Path returns the unquoted import path of imp, or "" when the spec is not a
// plain string literal.
func Path(imp *java.Import) string {
	if imp == nil {
		return ""
	}
	lit, ok := imp.Qualid.(*java.Literal)
	if !ok || lit == nil {
		return ""
	}
	if s, ok := lit.Value.(string); ok {
		return s
	}
	return strings.Trim(lit.Source, "`\"")
}

// Alias returns the explicit local name an import spec binds, or "" when the
// import has none and takes the package's own name.
func Alias(imp *java.Import) string {
	if imp == nil || imp.Alias == nil || imp.Alias.Element == nil {
		return ""
	}
	return imp.Alias.Element.Name
}

// LocalName returns the name an import binds in its file: the alias when there
// is one, otherwise the last path segment, skipping a semantic-import-versioning
// suffix — `github.com/go-viper/mapstructure/v2` binds `mapstructure`. It is a
// best effort for the unaliased case, since a package may declare a name
// differing from its directory, but every path these migrations touch agrees.
func LocalName(imp *java.Import) string {
	if alias := Alias(imp); alias != "" {
		return alias
	}
	return packageNameOf(Path(imp))
}

// packageNameOf returns the name an import path binds by default.
func packageNameOf(path string) string {
	for path != "" {
		i := strings.LastIndex(path, "/")
		if i < 0 {
			return path
		}
		segment := path[i+1:]
		if !isMajorVersionSegment(segment) {
			return segment
		}
		path = path[:i]
	}
	return ""
}

// isMajorVersionSegment reports whether a path segment is a `vN` major-version
// suffix, which names no directory in the package's own source.
func isMajorVersionSegment(segment string) bool {
	if len(segment) < 2 || segment[0] != 'v' {
		return false
	}
	for _, r := range segment[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Find returns the import of the exact path, or nil.
func Find(cu *golang.CompilationUnit, path string) *java.Import {
	if cu == nil || cu.Imports == nil {
		return nil
	}
	for _, rp := range cu.Imports.Elements {
		if Path(rp.Element) == path {
			return rp.Element
		}
	}
	return nil
}

// LocalNameFor returns the name cu binds to path, or "" when it does not import
// it. A blank or dot import returns "_" or "." — neither is migratable through a
// qualifier, so callers check for them.
func LocalNameFor(cu *golang.CompilationUnit, path string) string {
	return LocalName(Find(cu, path))
}

// Qualifier returns the usable local name cu binds to path, or "" when the
// package is absent or imported blank or dot.
func Qualifier(cu *golang.CompilationUnit, path string) string {
	name := LocalNameFor(cu, path)
	if name == "_" || name == "." {
		return ""
	}
	return name
}

// Imports reports whether cu imports the exact path.
func Imports(cu *golang.CompilationUnit, path string) bool {
	return Find(cu, path) != nil
}

// WithPath returns imp with its path literal replaced, keeping the original
// quoting style and the surrounding whitespace.
func WithPath(imp *java.Import, newPath string) *java.Import {
	lit, ok := imp.Qualid.(*java.Literal)
	if !ok || lit == nil {
		return imp
	}
	quote := `"`
	if strings.HasPrefix(strings.TrimSpace(lit.Source), "`") {
		quote = "`"
	}
	newLit := *lit
	newLit.Value = newPath
	newLit.Source = quote + newPath + quote
	c := *imp
	c.Qualid = &newLit
	return &c
}

// WithAlias returns imp bound to an explicit local name. The space separating
// the two is the path literal's prefix, which an unaliased import has none of.
func WithAlias(imp *java.Import, name string) *java.Import {
	lit, ok := imp.Qualid.(*java.Literal)
	if !ok || lit == nil {
		return imp
	}
	spaced := *lit
	spaced.Prefix = java.SingleSpace
	c := *imp
	c.Qualid = &spaced
	c.Alias = &java.LeftPadded[*java.Identifier]{
		Element: &java.Identifier{Prefix: java.EmptySpace, Name: name},
	}
	return &c
}

// RewriteImports returns cu with every import matching a rule pointed at its
// replacement, and the original when none matched.
func RewriteImports(cu *golang.CompilationUnit, rules []Rule) *golang.CompilationUnit {
	if cu == nil || cu.Imports == nil {
		return cu
	}
	elements := make([]java.RightPadded[*java.Import], len(cu.Imports.Elements))
	copy(elements, cu.Imports.Elements)
	changed := false
	for i, rp := range elements {
		path := Path(rp.Element)
		if path == "" {
			continue
		}
		newPath, rule, ok := matchRule(path, rules)
		if !ok || newPath == path {
			continue
		}
		rewritten := WithPath(rp.Element, newPath)
		if rewritten == rp.Element {
			continue
		}
		// A replacement whose last segment differs binds a different name, which
		// every reference in the file still spells the old way. An explicit
		// alias keeps them compiling.
		switch {
		case rule.Alias != "":
			rewritten = WithAlias(rewritten, rule.Alias)
		case rule.KeepName && Alias(rp.Element) == "" && packageNameOf(newPath) != packageNameOf(path):
			rewritten = WithAlias(rewritten, packageNameOf(path))
		}
		elements[i].Element = rewritten
		changed = true
	}
	if !changed {
		return cu
	}
	imports := *cu.Imports
	imports.Elements = elements
	c := *cu
	c.Imports = &imports
	return &c
}

// Retype returns a visitor that rewrites the parse-time attribution of every
// reference under a migrated path to name the new one, leaving the source
// unchanged. Run it alongside RewriteImports.
func Retype(rules []Rule) *retypeVisitor {
	return visitor.Init(&retypeVisitor{rules: rules})
}

type retypeVisitor struct {
	visitor.GoVisitor
	rules []Rule
}

func (v *retypeVisitor) VisitIdentifier(id *java.Identifier, p any) java.J {
	id = v.GoVisitor.VisitIdentifier(id, p).(*java.Identifier)
	retyped := v.retype(id.Type)
	if retyped == id.Type {
		return id
	}
	c := *id
	c.Type = retyped
	return &c
}

func (v *retypeVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	retyped := v.retype(fa.Type)
	if retyped == fa.Type {
		return fa
	}
	c := *fa
	c.Type = retyped
	return &c
}

func (v *retypeVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.MethodType == nil {
		return mi
	}
	method := v.retypeMethod(mi.MethodType)
	if method == mi.MethodType {
		return mi
	}
	c := *mi
	c.MethodType = method
	return &c
}

// retype returns t with any fully qualified name under a migrated path renamed,
// and t itself when nothing moved. Only the shapes a Go package reference takes
// are walked: the class naming a package or one of its types, a parameterized
// instantiation of one, and an array of either.
func (v *retypeVisitor) retype(t java.JavaType) java.JavaType {
	switch typ := t.(type) {
	case *java.JavaTypeClass:
		fqn, ok := mapFQN(typ.FullyQualifiedName, v.rules)
		if !ok {
			return t
		}
		c := *typ
		c.FullyQualifiedName = fqn
		return &c
	case *java.JavaTypeShallowClass:
		fqn, ok := mapFQN(typ.FullyQualifiedName, v.rules)
		if !ok {
			return t
		}
		c := *typ
		c.FullyQualifiedName = fqn
		return &c
	case *java.JavaTypeParameterized:
		inner := v.retype(typ.Type)
		if inner == java.JavaType(typ.Type) {
			return t
		}
		fq, ok := inner.(java.FullyQualified)
		if !ok {
			return t
		}
		c := *typ
		c.Type = fq
		return &c
	case *java.JavaTypeArray:
		elem := v.retype(typ.ElemType)
		if elem == typ.ElemType {
			return t
		}
		c := *typ
		c.ElemType = elem
		return &c
	case *java.JavaTypeMethod:
		return v.retypeMethod(typ)
	}
	return t
}

func (v *retypeVisitor) retypeMethod(m *java.JavaTypeMethod) *java.JavaTypeMethod {
	if m == nil || m.DeclaringType == nil {
		return m
	}
	declaring := v.retype(m.DeclaringType)
	if declaring == java.JavaType(m.DeclaringType) {
		return m
	}
	fq, ok := declaring.(java.FullyQualified)
	if !ok {
		return m
	}
	c := *m
	c.DeclaringType = fq
	return &c
}
