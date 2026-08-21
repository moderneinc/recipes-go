/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UsePackageLevelErrorSentinel moves inline `errors.New("...")` calls from
// function bodies to package-level `var` declarations. Inline error sentinels
// cannot be compared with errors.Is; this recipe hoists them automatically.
// A sentinel is one name in one package, so the scan spans the whole package
// before any file is edited.
type UsePackageLevelErrorSentinel struct {
	recipe.ScanningBase
}

func (r *UsePackageLevelErrorSentinel) Name() string {
	return "org.openrewrite.golang.codequality.UsePackageLevelErrorSentinel"
}
func (r *UsePackageLevelErrorSentinel) DisplayName() string {
	return "Use package-level error sentinel"
}
func (r *UsePackageLevelErrorSentinel) Description() string {
	return "Move inline `errors.New(\"...\")` calls to package-level sentinel variables so they can be compared with `errors.Is`. " +
		"Each distinct message becomes one sentinel per package, declared in the first file that uses it and referenced from every other."
}
func (r *UsePackageLevelErrorSentinel) Tags() []string { return []string{"error-handling", "lint"} }

func (r *UsePackageLevelErrorSentinel) InitialValue(*recipe.ExecutionContext) any {
	return &sentinelAcc{packages: map[string]*packageInfo{}}
}

func (r *UsePackageLevelErrorSentinel) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&sentinelScanner{acc: acc.(*sentinelAcc)})
}

func (r *UsePackageLevelErrorSentinel) EditorWithData(acc any) recipe.TreeVisitor {
	a := acc.(*sentinelAcc)
	// Resolving up front keeps the editor a pure reader of the plan, so files of
	// one package can be edited in any order or at once.
	for _, pkg := range a.packages {
		pkg.resolve()
	}
	return visitor.Init(&sentinelEditor{acc: a})
}

// sentinelAcc holds what the scan learned, one entry per Go package in the run.
type sentinelAcc struct {
	packages map[string]*packageInfo
}

type packageInfo struct {
	files map[string]*fileInfo
	plan  map[string]*sentinel
}

type fileInfo struct {
	inline   []string // literal sources of inline errors.New calls, in source order
	literals map[string]*java.Literal
	taken    []string         // names the sentinels must not collide with
	declared []messageBinding // package-level `var X = errors.New("...")`, in source order
	buildTag string
}

type messageBinding struct{ source, varName string }

// sentinel is what a message resolves to: the name every call site in the
// package refers to, and the file that declares it. declareIn is empty and decl
// nil when the declaration already exists.
type sentinel struct {
	varName   string
	declareIn string
	decl      java.Statement
}

func (a *sentinelAcc) packageOf(cu *golang.CompilationUnit) *packageInfo {
	if cu.PackageDecl == nil || cu.PackageDecl.Element == nil {
		return nil
	}
	key := path.Dir(cu.SourcePath) + "\x00" + cu.PackageDecl.Element.Name
	pkg := a.packages[key]
	if pkg == nil {
		pkg = &packageInfo{files: map[string]*fileInfo{}}
		a.packages[key] = pkg
	}
	return pkg
}

func (p *packageInfo) fileOf(sourcePath string) *fileInfo {
	f := p.files[sourcePath]
	if f == nil {
		f = &fileInfo{literals: map[string]*java.Literal{}}
		p.files[sourcePath] = f
	}
	return f
}

// sentinelScanner records, per file, the inline messages to hoist and everything
// the plan has to respect: the names already spoken for, the sentinels already
// declared, and when the toolchain builds the file.
type sentinelScanner struct {
	visitor.GoVisitor
	acc    *sentinelAcc
	file   *fileInfo
	inFunc int
}

func (s *sentinelScanner) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	pkg := s.acc.packageOf(cu)
	if pkg == nil {
		return cu
	}
	s.file = pkg.fileOf(cu.SourcePath)
	s.file.buildTag = buildTag(cu)
	s.inFunc = 0
	return s.GoVisitor.VisitCompilationUnit(cu, p)
}

func (s *sentinelScanner) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	if s.file != nil && s.inFunc == 0 && md.Name != nil {
		s.file.taken = append(s.file.taken, md.Name.Name)
	}
	s.inFunc++
	defer func() { s.inFunc-- }()
	return s.GoVisitor.VisitMethodDeclaration(md, p)
}

func (s *sentinelScanner) VisitTypeDecl(td *golang.TypeDecl, p any) java.J {
	if s.file != nil && s.inFunc == 0 && td.Name != nil {
		s.file.taken = append(s.file.taken, td.Name.Name)
	}
	return s.GoVisitor.VisitTypeDecl(td, p)
}

func (s *sentinelScanner) VisitVariableDeclarator(vd *java.VariableDeclarator, p any) java.J {
	if s.file != nil && s.inFunc == 0 && vd.Name != nil {
		s.file.taken = append(s.file.taken, vd.Name.Name)
		if vd.Initializer != nil {
			if lit, ok := errorsNewLiteral(vd.Initializer.Element); ok {
				s.file.declared = append(s.file.declared, messageBinding{lit.Source, vd.Name.Name})
			}
		}
	}
	return s.GoVisitor.VisitVariableDeclarator(vd, p)
}

func (s *sentinelScanner) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if s.file != nil && s.inFunc > 0 {
		if lit, ok := errorsNewLiteral(mi); ok {
			if _, seen := s.file.literals[lit.Source]; !seen {
				s.file.literals[lit.Source] = lit
				s.file.inline = append(s.file.inline, lit.Source)
			}
		}
	}
	return s.GoVisitor.VisitMethodInvocation(mi, p)
}

// resolve works out which message becomes which sentinel and which file declares
// it. Resolving off sorted paths keeps the answer independent of the order the
// scan happened to reach files in.
func (p *packageInfo) resolve() {
	paths := make([]string, 0, len(p.files))
	for sourcePath := range p.files {
		paths = append(paths, sourcePath)
	}
	sort.Strings(paths)

	taken := map[string]bool{}
	declared := map[string]messageBinding{} // literal source -> the sentinel already declared for it
	declaredIn := map[string]string{}
	var sources []string
	users := map[string][]string{} // literal source -> the files holding an inline call

	for _, sourcePath := range paths {
		f := p.files[sourcePath]
		for _, name := range f.taken {
			taken[name] = true
		}
		for _, b := range f.declared {
			if _, ok := declared[b.source]; !ok {
				declared[b.source] = b
				declaredIn[b.source] = sourcePath
			}
		}
		for _, source := range f.inline {
			if len(users[source]) == 0 {
				sources = append(sources, source)
			}
			users[source] = append(users[source], sourcePath)
		}
	}

	// Two messages that derive the same name leave both alone: picking a winner
	// would fold one message's call sites into the other's sentinel.
	derived := map[string]int{}
	for _, source := range sources {
		if _, reused := declared[source]; !reused {
			derived[sentinelName(source)]++
		}
	}

	p.plan = map[string]*sentinel{}
	for _, source := range sources {
		if b, reused := declared[source]; reused {
			if reaches(p, declaredIn[source], users[source]) {
				p.plan[source] = &sentinel{varName: b.varName}
			}
			continue
		}
		name := sentinelName(source)
		if name == errPrefix || taken[name] || derived[name] > 1 {
			continue
		}
		home := declaringFile(p, users[source])
		if home == "" {
			continue
		}
		decl := sentinelDecl(name, p.files[home].literals[source])
		if decl == nil {
			continue
		}
		taken[name] = true
		p.plan[source] = &sentinel{varName: name, declareIn: home, decl: decl}
	}
}

// declaringFile picks the file to hold the sentinel: the first, in path order,
// that reaches every call site.
func declaringFile(p *packageInfo, users []string) string {
	for _, sourcePath := range users {
		if reaches(p, sourcePath, users) {
			return sourcePath
		}
	}
	return ""
}

// A declaration reaches a call site in another file only when the declaring file
// is part of every build the call site is part of.
func reaches(p *packageInfo, declaringPath string, users []string) bool {
	tag := p.files[declaringPath].buildTag
	if tag == "" {
		return true
	}
	for _, sourcePath := range users {
		if p.files[sourcePath].buildTag != tag {
			return false
		}
	}
	return true
}

// sentinelEditor rewrites the call sites of every planned message and appends
// the declarations the plan assigned to the file being visited.
type sentinelEditor struct {
	visitor.GoVisitor
	acc      *sentinelAcc
	pkg      *packageInfo
	path     string
	inFunc   int
	replaced bool
}

func (e *sentinelEditor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	e.pkg = e.acc.packageOf(cu)
	if e.pkg == nil {
		return cu
	}
	e.path = cu.SourcePath
	e.inFunc = 0
	e.replaced = false
	if len(e.pkg.plan) == 0 {
		return cu
	}

	cu = e.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)

	if decls := e.declarations(); len(decls) > 0 {
		cu = cu.WithStatements(append(decls, cu.Statements...))
		recipegolang.MaybeAddImport(e, "errors", nil, false)
	} else if e.replaced {
		recipegolang.MaybeRemoveImport(e, "errors")
	}
	return cu
}

func (e *sentinelEditor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	e.inFunc++
	defer func() { e.inFunc-- }()
	return e.GoVisitor.VisitMethodDeclaration(md, p)
}

func (e *sentinelEditor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = e.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if e.inFunc == 0 {
		return mi
	}
	lit, ok := errorsNewLiteral(mi)
	if !ok {
		return mi
	}
	s := e.pkg.plan[lit.Source]
	if s == nil {
		return mi
	}
	e.replaced = true

	// The leading whitespace lives on the Select element (the "errors"
	// identifier), not on the MethodInvocation itself.
	prefix := mi.Prefix
	if selIdent, ok := mi.Select.Element.(*java.Identifier); ok && !selIdent.Prefix.IsEmpty() {
		prefix = selIdent.Prefix
	}
	return &java.Identifier{
		ID:     uuid.New(),
		Prefix: prefix,
		Name:   s.varName,
		Type:   lstutil.ErrorType,
	}
}

// declarations builds the `var ErrFoo = errors.New("msg")` statements the plan
// assigned to the file being visited, in the order that file uses them.
func (e *sentinelEditor) declarations() []java.RightPadded[java.Statement] {
	f := e.pkg.files[e.path]
	if f == nil {
		return nil
	}
	var out []java.RightPadded[java.Statement]
	for _, source := range f.inline {
		if s := e.pkg.plan[source]; s != nil && s.declareIn == e.path {
			out = append(out, java.RightPadded[java.Statement]{Element: s.decl})
		}
	}
	return out
}

// sentinelDecl builds `var ErrFoo = errors.New("msg")`. Parsing it as a template
// is what types the call; the message goes in as its own literal source, so a
// raw string literal stays raw.
func sentinelDecl(varName string, message *java.Literal) java.Statement {
	tmpl := template.TopLevelTemplate(fmt.Sprintf("var %s = errors.New(%s)", varName, message.Source)).
		Imports("errors").Build()
	decl, ok := tmpl.Apply(nil, nil).(*java.VariableDeclarations)
	if !ok {
		return nil
	}
	return decl.WithPrefix(java.Space{Whitespace: "\n\n"})
}

// errorsNewLiteral returns the message literal of an `errors.New("...")` call.
func errorsNewLiteral(expr java.Expression) (*java.Literal, bool) {
	mi, ok := expr.(*java.MethodInvocation)
	if !ok || mi.Select == nil || mi.Name == nil || mi.Name.Name != "New" {
		return nil, false
	}
	if ident, ok := mi.Select.Element.(*java.Identifier); !ok || ident.Name != "errors" {
		return nil, false
	}
	if len(mi.Arguments.Elements) != 1 {
		return nil, false
	}
	lit, ok := mi.Arguments.Elements[0].Element.(*java.Literal)
	if !ok || !isStringLiteral(lit) {
		return nil, false
	}
	return lit, true
}

const errPrefix = "Err"

// sentinelName converts a message literal to an ErrFoo variable name, dropping
// the quotes along with every other non-alphanumeric character:
// `"not found"` -> "ErrNotFound", `"connection refused"` -> "ErrConnectionRefused".
func sentinelName(source string) string {
	var b strings.Builder
	b.WriteString(errPrefix)
	for _, w := range strings.Fields(source) {
		cleaned := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, w)
		if cleaned == "" {
			continue
		}
		runes := []rune(cleaned)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

// The GOOS and GOARCH values the Go toolchain reads off a `_suffix` in a file
// name, per `go help buildconstraint`.
var goosOrGoarch = map[string]bool{
	"aix": true, "android": true, "darwin": true, "dragonfly": true, "freebsd": true,
	"hurd": true, "illumos": true, "ios": true, "js": true, "linux": true, "nacl": true,
	"netbsd": true, "openbsd": true, "plan9": true, "solaris": true, "wasip1": true,
	"windows": true, "zos": true,
	"386": true, "amd64": true, "amd64p32": true, "arm": true, "arm64": true,
	"arm64be": true, "armbe": true, "loong64": true, "mips": true, "mips64": true,
	"mips64le": true, "mips64p32": true, "mips64p32le": true, "mipsle": true,
	"ppc": true, "ppc64": true, "ppc64le": true, "riscv": true, "riscv64": true,
	"s390": true, "s390x": true, "sparc": true, "sparc64": true, "wasm": true,
}

// buildTag summarizes when the toolchain builds a file: empty for every build,
// and otherwise what narrows it — a //go:build (or legacy // +build) line, the
// _GOOS/_GOARCH parts of the file name, and test for a _test.go file, which the
// ordinary build leaves out. Files sharing a summary are in the same builds.
func buildTag(cu *golang.CompilationUnit) string {
	var parts []string
	for _, c := range cu.Prefix.Comments {
		text := strings.TrimSpace(strings.TrimLeft(c.Text, "/"))
		if strings.HasPrefix(text, "go:build ") || strings.HasPrefix(text, "+build ") {
			parts = append(parts, text)
		}
	}
	name := strings.TrimSuffix(path.Base(cu.SourcePath), ".go")
	segments := strings.Split(name, "_")
	for _, segment := range segments[1:] {
		if goosOrGoarch[segment] {
			parts = append(parts, segment)
		}
	}
	if strings.HasSuffix(name, "_test") {
		parts = append(parts, "test")
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}
