/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify

import (
	"strconv"
	"strings"

	"github.com/google/uuid"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// guardBase carries the flavour selection shared by the single-condition adoption
// visitors (bool / nil / len): pkg + importPath choose require vs assert and
// isReporter selects the fatal or non-fatal reporter set.
type guardBase struct {
	visitor.GoVisitor
	pkg        string
	importPath string
	isReporter func(string) bool
}

// rewriteBlockGuards replaces each top-level statement of block for which match
// returns a call, adding importPath when anything changed.
func rewriteBlockGuards(av visitor.AfterVisitsProvider, block *java.Block, importPath string, match func(*java.If) *java.MethodInvocation) java.J {
	changed := false
	newStmts := make([]java.RightPadded[java.Statement], 0, len(block.Statements))
	for _, rp := range block.Statements {
		var call *java.MethodInvocation
		if ifStmt, ok := rp.Element.(*java.If); ok {
			call = match(ifStmt)
		}
		if call == nil {
			newStmts = append(newStmts, rp)
			continue
		}
		changed = true
		newStmts = append(newStmts, java.RightPadded[java.Statement]{
			Element: call,
			After:   rp.After,
			Markers: rp.Markers,
		})
	}

	if !changed {
		return block
	}
	recipegolang.MaybeAddImport(av, importPath, nil, false)
	return block.WithStatements(newStmts)
}

// reporterBodyCall returns the single reporter call and its testing receiver when
// thenPart is a block whose only statement is such a call, or (nil, nil).
func reporterBodyCall(thenPart java.Statement, isReporter func(string) bool) (*java.MethodInvocation, *java.Identifier) {
	body, ok := thenPart.(*java.Block)
	if !ok || len(body.Statements) != 1 {
		return nil, nil
	}
	call, ok := body.Statements[0].Element.(*java.MethodInvocation)
	if !ok {
		return nil, nil
	}
	recv := reporterReceiver(call, isReporter)
	if recv == nil {
		return nil, nil
	}
	return call, recv
}

// isComparison reports whether expr is a top-level relational/equality comparison,
// which has a more specific assertion (Equal/Nil/Len) and so is excluded from the
// bool True/False adoption.
func isComparison(expr java.Expression) bool {
	bin, ok := expr.(*java.Binary)
	if !ok {
		return false
	}
	switch bin.Operator.Element {
	case java.Equal, java.NotEqual, java.LessThan, java.LessThanOrEqual, java.GreaterThan, java.GreaterThanOrEqual:
		return true
	}
	return false
}

// finishAssertion builds the testify replacement for a matched guard: the core
// assertion `<pkg>.<assertion>(recv, coreArgs...)` plus the reporter's message
// carried over (with its trailing value-dump stripped). surv is the set of
// identifiers the assertion already shows (their dumps are redundant); forbidden
// are identifiers that will not exist after the rewrite (the inline-init
// variable).
func finishAssertion(prefix java.Space, pkg, recvName, assertion string, call *java.MethodInvocation, coreArgs []java.Expression, surv, forbidden map[string]bool) *java.MethodInvocation {
	msgArgs, useF := messageArgs(call, forbidden, surv)
	return buildAssertionCall(prefix, pkg, recvName, assertion, useF, coreArgs, msgArgs)
}

// buildAssertionCall constructs `<pkg>.<assertion>[f](<recv>, <coreArgs...>, <msgArgs...>)`,
// indented to sit where the original statement did. useF appends the `f` suffix
// (the format-string assertion variant).
func buildAssertionCall(prefix java.Space, pkg, recvName, assertion string, useF bool, coreArgs, msgArgs []java.Expression) *java.MethodInvocation {
	name := assertion
	if useF {
		name += "f"
	}
	elements := make([]java.RightPadded[java.Expression], 0, len(coreArgs)+len(msgArgs)+1)
	elements = append(elements, java.RightPadded[java.Expression]{
		Element: &java.Identifier{ID: uuid.New(), Name: recvName},
	})
	for _, a := range coreArgs {
		elements = append(elements, java.RightPadded[java.Expression]{Element: withPrefix(a, java.SingleSpace)})
	}
	for _, a := range msgArgs {
		elements = append(elements, java.RightPadded[java.Expression]{Element: withPrefix(a, java.SingleSpace)})
	}
	return &java.MethodInvocation{
		ID:     uuid.New(),
		Prefix: prefix,
		Select: &java.RightPadded[java.Expression]{
			Element: &java.Identifier{ID: uuid.New(), Name: pkg},
		},
		Name:      &java.Identifier{ID: uuid.New(), Name: name},
		Arguments: java.Container[java.Expression]{Elements: elements},
	}
}

// identSet returns the set of value-identifier names referenced across exprs.
func identSet(exprs ...java.Expression) map[string]bool {
	s := map[string]bool{}
	for _, e := range exprs {
		collectIdentNames(e, s)
	}
	return s
}

// messageArgs computes the trailing message arguments to carry onto a testify
// assertion from the original reporter call, and whether the `f` (format) variant
// is needed. The trailing value-dump (`, got %v` and the like) whose values the
// assertion already shows is stripped; genuinely contextual args are kept. When
// the format cannot be parsed with confidence the original template is kept
// verbatim. Returns (nil, false) for no message.
func messageArgs(call *java.MethodInvocation, forbidden, surv map[string]bool) ([]java.Expression, bool) {
	args := call.Arguments.Elements
	if len(args) == 0 {
		return nil, false
	}
	name := ""
	if call.Name != nil {
		name = call.Name.Name
	}
	if strings.HasSuffix(name, "f") { // Errorf / Fatalf
		return formatMessageArgs(args, forbidden, surv)
	}
	return plainMessageArgs(args, forbidden, surv)
}

// formatMessageArgs handles the Sprintf-style reporters: args[0] is the format
// literal, args[1:] the values.
func formatMessageArgs(args []java.RightPadded[java.Expression], forbidden, surv map[string]bool) ([]java.Expression, bool) {
	lit, ok := args[0].Element.(*java.Literal)
	if !ok {
		return nil, false // non-literal format: cannot reason -> drop message
	}
	format := literalString(lit)
	values := elementsOf(args[1:])
	spans, complexFmt := scanFormatVerbs(format)
	if complexFmt || len(spans) != len(values) {
		return keepOriginal(lit, values, forbidden)
	}
	if len(values) == 0 {
		if strings.TrimSpace(format) == "" {
			return nil, false
		}
		return []java.Expression{lit}, false // the literal is the whole message
	}

	redundant := make([]bool, len(values))
	for i, v := range values {
		redundant[i] = isRedundantArg(v, surv)
	}
	msg, keep, stripped := stripValueDump(tokenizeFormat(format, spans), redundant)
	if !stripped {
		return keepOriginal(lit, values, forbidden)
	}
	if referencesAny(values[:keep], forbidden) {
		return nil, false // a kept arg would be undefined after the rewrite
	}
	if keep == 0 {
		if msg == "" {
			return nil, false
		}
		return []java.Expression{stringLiteral(msg)}, false
	}
	return prepend(stringLiteral(msg), values[:keep]), true
}

// plainMessageArgs handles the Sprint-style reporters (t.Error / t.Fatal): the
// args are values, a trailing run of which (the redundant ones) is dropped.
func plainMessageArgs(args []java.RightPadded[java.Expression], forbidden, surv map[string]bool) ([]java.Expression, bool) {
	values := elementsOf(args)
	keep := len(values)
	for keep > 0 && isRedundantArg(values[keep-1], surv) {
		keep--
	}
	if keep == 0 || referencesAny(values[:keep], forbidden) {
		return nil, false
	}
	return values[:keep], false
}

// keepOriginal preserves the reporter's message verbatim (format literal + all
// values) as the `f`-variant, unless doing so would reference a forbidden ident.
func keepOriginal(lit *java.Literal, values []java.Expression, forbidden map[string]bool) ([]java.Expression, bool) {
	if referencesAny(values, forbidden) {
		return nil, false
	}
	return prepend(lit, values), true
}

// isRedundantArg reports whether a format arg's value is already shown by the
// assertion: it references at least one identifier and every identifier is a
// survivor, a builtin length function, or a predeclared constant. A pure literal
// is not redundant (it carries text worth keeping).
func isRedundantArg(expr java.Expression, surv map[string]bool) bool {
	names := map[string]bool{}
	if !collectIdentNames(expr, names) || len(names) == 0 {
		return false
	}
	for n := range names {
		switch n {
		case "len", "cap", "nil", "true", "false":
			continue
		}
		if !surv[n] {
			return false
		}
	}
	return true
}

// referencesAny reports whether any expr references an identifier in forbidden.
func referencesAny(values []java.Expression, forbidden map[string]bool) bool {
	if len(forbidden) == 0 {
		return false
	}
	for _, v := range values {
		names := map[string]bool{}
		collectIdentNames(v, names)
		for n := range names {
			if forbidden[n] {
				return true
			}
		}
	}
	return false
}

var connectorWords = map[string]bool{
	"got": true, "want": true, "wanted": true, "expected": true,
	"expecting": true, "actual": true, "have": true,
}

type ftokKind int

const (
	tVerb ftokKind = iota
	tWord
	tOther
)

type ftok struct {
	kind ftokKind
	text string
	vi   int // verb index (tVerb only)
}

type verbSpan struct{ start, end int }

// scanFormatVerbs returns the byte spans of each simple format verb, and true if
// the format contains a construct that consumes args unpredictably (`%*`, `%[`)
// or a trailing `%`, in which case callers keep the original template.
func scanFormatVerbs(format string) ([]verbSpan, bool) {
	var spans []verbSpan
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		if i+1 < len(format) && format[i+1] == '%' {
			i++
			continue
		}
		start := i
		i++
		for i < len(format) && strings.IndexByte("+-# 0", format[i]) >= 0 {
			i++
		}
		for i < len(format) && (format[i] >= '0' && format[i] <= '9' || format[i] == '.') {
			i++
		}
		if i >= len(format) || format[i] == '*' || format[i] == '[' {
			return nil, true
		}
		spans = append(spans, verbSpan{start: start, end: i + 1})
	}
	return spans, false
}

// tokenizeFormat splits format into verb / word / other tokens.
func tokenizeFormat(format string, spans []verbSpan) []ftok {
	var toks []ftok
	pos := 0
	for i, sp := range spans {
		if sp.start > pos {
			toks = appendTextTokens(toks, format[pos:sp.start])
		}
		toks = append(toks, ftok{kind: tVerb, text: format[sp.start:sp.end], vi: i})
		pos = sp.end
	}
	if pos < len(format) {
		toks = appendTextTokens(toks, format[pos:])
	}
	return toks
}

func appendTextTokens(toks []ftok, s string) []ftok {
	i := 0
	for i < len(s) {
		if isLetterByte(s[i]) {
			j := i
			for j < len(s) && isLetterByte(s[j]) {
				j++
			}
			toks = append(toks, ftok{kind: tWord, text: s[i:j]})
			i = j
			continue
		}
		j := i
		for j < len(s) && !isLetterByte(s[j]) {
			j++
		}
		toks = append(toks, ftok{kind: tOther, text: s[i:j]})
		i = j
	}
	return toks
}

func isLetterByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// stripValueDump removes a trailing value-dump from a tokenized format: from the
// right it drops whitespace/punctuation, connector words, and format verbs whose
// args are redundant, stopping at the first content word or contextual verb. It
// returns the remaining message text, the number of leading verbs kept (their
// args must be passed), and whether any verb was actually stripped. When no verb
// is stripped it returns ok=false so the caller keeps the original template.
func stripValueDump(toks []ftok, redundant []bool) (msg string, keep int, ok bool) {
	end := len(toks)
	strippedVerb := false
loop:
	for end > 0 {
		t := toks[end-1]
		switch t.kind {
		case tOther:
			end--
		case tWord:
			if connectorWords[strings.ToLower(t.text)] {
				end--
			} else {
				break loop
			}
		case tVerb:
			if redundant[t.vi] {
				strippedVerb = true
				end--
			} else {
				break loop
			}
		}
	}
	if !strippedVerb {
		return "", 0, false
	}
	var b strings.Builder
	for _, t := range toks[:end] {
		b.WriteString(t.text)
		if t.kind == tVerb {
			keep++
		}
	}
	msg = balanceBrackets(strings.TrimRight(b.String(), " \t\n\r,:;-.=("))
	return msg, keep, true
}

// balanceBrackets appends the closers for any bracket left open in s. Stripping
// a trailing value-dump can drop a closing bracket (e.g. `(malformed): got %d` ->
// `(malformed`); since only the tail is removed, the imbalance is always missing
// closers, restored here in nesting order.
func balanceBrackets(s string) string {
	pairs := map[byte]byte{'(': ')', '[': ']', '{': '}'}
	var open []byte
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '(', '[', '{':
			open = append(open, c)
		case ')', ']', '}':
			if n := len(open); n > 0 && pairs[open[n-1]] == c {
				open = open[:n-1]
			}
		}
	}
	if len(open) == 0 {
		return s
	}
	var b strings.Builder
	b.WriteString(s)
	for i := len(open) - 1; i >= 0; i-- {
		b.WriteByte(pairs[open[i]])
	}
	return b.String()
}

func literalString(lit *java.Literal) string {
	if s, ok := lit.Value.(string); ok {
		return s
	}
	if s, err := strconv.Unquote(lit.Source); err == nil {
		return s
	}
	return strings.Trim(lit.Source, "\"`")
}

func stringLiteral(s string) *java.Literal {
	return &java.Literal{ID: uuid.New(), Value: s, Source: strconv.Quote(s)}
}

func elementsOf(args []java.RightPadded[java.Expression]) []java.Expression {
	out := make([]java.Expression, len(args))
	for i, a := range args {
		out[i] = a.Element
	}
	return out
}

func prepend(first java.Expression, rest []java.Expression) []java.Expression {
	return append([]java.Expression{first}, rest...)
}
