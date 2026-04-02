/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UsePackageLevelErrorSentinel moves inline `errors.New("...")` calls from
// function bodies to package-level `var` declarations. Inline error sentinels
// cannot be compared with errors.Is; this recipe hoists them automatically.
type UsePackageLevelErrorSentinel struct {
	recipe.Base
}

func (r *UsePackageLevelErrorSentinel) Name() string {
	return "org.openrewrite.golang.codequality.UsePackageLevelErrorSentinel"
}
func (r *UsePackageLevelErrorSentinel) DisplayName() string {
	return "Use package-level error sentinel"
}
func (r *UsePackageLevelErrorSentinel) Description() string {
	return "Move inline `errors.New(\"...\")` calls to package-level sentinel variables so they can be compared with `errors.Is`."
}
func (r *UsePackageLevelErrorSentinel) Tags() []string { return []string{"error-handling", "lint"} }

func (r *UsePackageLevelErrorSentinel) Editor() recipe.TreeVisitor {
	return visitor.Init(&usePackageLevelErrorSentinelVisitor{})
}

type usePackageLevelErrorSentinelVisitor struct {
	visitor.GoVisitor
}

// errNewEntry records an inline errors.New call found inside a function body.
type errNewEntry struct {
	message string // the raw string literal value, e.g. "not found"
	varName string // generated sentinel name, e.g. "ErrNotFound"
	mi      *tree.MethodInvocation
}

func (v *usePackageLevelErrorSentinelVisitor) VisitCompilationUnit(cu *tree.CompilationUnit, p any) tree.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*tree.CompilationUnit)

	// Pass 1: Collect inline errors.New("...") calls inside function bodies.
	collector := &errNewCollector{}
	collector.Self = collector
	collector.Visit(cu, nil)
	if len(collector.found) == 0 {
		return cu
	}

	// Deduplicate by message so the same string produces one sentinel.
	seen := map[string]*errNewEntry{}
	var entries []*errNewEntry
	for _, e := range collector.found {
		if _, ok := seen[e.message]; !ok {
			seen[e.message] = e
			entries = append(entries, e)
		}
	}

	// Build a lookup from message -> varName for the replacer.
	msgToVar := make(map[string]string, len(entries))
	for _, e := range entries {
		msgToVar[e.message] = e.varName
	}

	// Pass 2: Replace inline errors.New("...") calls with identifier references.
	replacer := &errNewReplacer{msgToVar: msgToVar}
	replacer.Self = replacer
	replaced := replacer.Visit(cu, nil).(*tree.CompilationUnit)

	// Build new var declarations and prepend them to the top-level statements.
	varStmts := buildVarDecls(entries)
	newStmts := make([]tree.RightPadded[tree.Statement], 0, len(varStmts)+len(replaced.Statements))
	newStmts = append(newStmts, varStmts...)
	newStmts = append(newStmts, replaced.Statements...)
	return replaced.WithStatements(newStmts)
}

// errNewCollector walks the entire tree looking for errors.New("...") calls
// that are inside a function body (i.e. inside a Block that is a child of a
// MethodDeclaration). It skips calls that are already at package level inside
// a var declaration.
type errNewCollector struct {
	visitor.GoVisitor
	found   []*errNewEntry
	inFunc  int // depth counter: >0 means we are inside a function body
}

func (c *errNewCollector) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	c.inFunc++
	md = c.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)
	c.inFunc--
	return md
}

func (c *errNewCollector) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = c.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if c.inFunc == 0 {
		return mi
	}

	if mi.Select == nil {
		return mi
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "errors" {
		return mi
	}
	if mi.Name.Name != "New" {
		return mi
	}
	if len(mi.Arguments.Elements) != 1 {
		return mi
	}
	lit, ok := mi.Arguments.Elements[0].Element.(*tree.Literal)
	if !ok || lit.Kind != tree.StringLiteral {
		return mi
	}

	msg := unquote(lit.Source)
	varName := messageToVarName(msg)
	c.found = append(c.found, &errNewEntry{
		message: msg,
		varName: varName,
		mi:      mi,
	})
	return mi
}

// errNewReplacer replaces inline errors.New("msg") calls with the sentinel
// identifier.
type errNewReplacer struct {
	visitor.GoVisitor
	msgToVar map[string]string
	inFunc   int
}

func (r *errNewReplacer) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	r.inFunc++
	md = r.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)
	r.inFunc--
	return md
}

func (r *errNewReplacer) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = r.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if r.inFunc == 0 {
		return mi
	}

	if mi.Select == nil {
		return mi
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "errors" {
		return mi
	}
	if mi.Name.Name != "New" {
		return mi
	}
	if len(mi.Arguments.Elements) != 1 {
		return mi
	}
	lit, ok := mi.Arguments.Elements[0].Element.(*tree.Literal)
	if !ok || lit.Kind != tree.StringLiteral {
		return mi
	}

	msg := unquote(lit.Source)
	varName, ok := r.msgToVar[msg]
	if !ok {
		return mi
	}

	// Replace the entire errors.New("...") call with the sentinel identifier.
	// The leading whitespace lives on the Select element (the "errors" identifier),
	// not on the MethodInvocation itself.
	prefix := mi.Prefix
	if mi.Select != nil {
		if selIdent, ok := mi.Select.Element.(*tree.Identifier); ok && !selIdent.Prefix.IsEmpty() {
			prefix = selIdent.Prefix
		}
	}
	return &tree.Identifier{
		ID:     uuid.New(),
		Prefix: prefix,
		Name:   varName,
	}
}

// buildVarDecls creates `var ErrFoo = errors.New("msg")` statements for each entry.
func buildVarDecls(entries []*errNewEntry) []tree.RightPadded[tree.Statement] {
	result := make([]tree.RightPadded[tree.Statement], len(entries))
	for i, e := range entries {
		nameIdent := &tree.Identifier{
			ID:     uuid.New(),
			Prefix: tree.SingleSpace,
			Name:   e.varName,
		}

		// Build the initializer: errors.New("msg")
		selectIdent := &tree.Identifier{
			ID:   uuid.New(),
			Name: "errors",
		}
		methodName := &tree.Identifier{
			ID:   uuid.New(),
			Name: "New",
		}
		argLit := &tree.Literal{
			ID:     uuid.New(),
			Kind:   tree.StringLiteral,
			Source: quote(e.message),
			Value:  e.message,
		}
		initCall := &tree.MethodInvocation{
			ID:     uuid.New(),
			Prefix: tree.SingleSpace,
			Select: &tree.RightPadded[tree.Expression]{Element: selectIdent},
			Name:   methodName,
			Arguments: tree.Container[tree.Expression]{
				Before: tree.EmptySpace,
				Elements: []tree.RightPadded[tree.Expression]{
					{Element: argLit},
				},
			},
		}

		declarator := &tree.VariableDeclarator{
			ID:   uuid.New(),
			Name: nameIdent,
			Initializer: &tree.LeftPadded[tree.Expression]{
				Before:  tree.SingleSpace,
				Element: initCall,
			},
		}

		vd := &tree.VariableDeclarations{
			ID:      uuid.New(),
			Prefix:  tree.Space{Whitespace: "\n\n"},
			Markers: tree.Markers{ID: uuid.New(), Entries: []tree.Marker{tree.VarKeyword{Ident: uuid.New()}}},
			Variables: []tree.RightPadded[*tree.VariableDeclarator]{
				{Element: declarator},
			},
		}

		result[i] = tree.RightPadded[tree.Statement]{Element: vd}
	}
	return result
}

// messageToVarName converts a message string to an ErrFoo variable name.
// "not found" -> "ErrNotFound", "connection refused" -> "ErrConnectionRefused"
func messageToVarName(msg string) string {
	words := strings.Fields(msg)
	var b strings.Builder
	b.WriteString("Err")
	for _, w := range words {
		// Strip non-alphanumeric characters from each word, then title-case.
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

// unquote strips surrounding quotes from a Go string literal source.
func unquote(source string) string {
	if len(source) >= 2 && source[0] == '"' && source[len(source)-1] == '"' {
		return source[1 : len(source)-1]
	}
	return source
}

// quote wraps a string in double quotes.
func quote(s string) string {
	return "\"" + s + "\""
}
