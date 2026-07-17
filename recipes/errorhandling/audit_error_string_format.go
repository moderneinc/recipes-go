/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// ErrorStringFormatRow is one row of the AuditErrorStringFormat data table.
type ErrorStringFormatRow struct {
	SourcePath  string
	Constructor string
	Issue       string
	Message     string
}

var errorStringFormatTable = recipe.NewDataTable[ErrorStringFormatRow](
	"org.openrewrite.golang.codequality.AuditErrorStringFormat$Findings",
	"Error strings that violate ST1005",
	"Error messages that are capitalized or end with punctuation.",
	[]recipe.ColumnDescriptor{
		{Name: "sourcePath", DisplayName: "Source path", Description: "File containing the error string.", Type: "String"},
		{Name: "constructor", DisplayName: "Constructor", Description: "errors.New or fmt.Errorf.", Type: "String"},
		{Name: "issue", DisplayName: "Issue", Description: "Why the message violates ST1005.", Type: "String"},
		{Name: "message", DisplayName: "Message", Description: "The offending error message.", Type: "String"},
	},
)

// AuditErrorStringFormat finds `errors.New` and `fmt.Errorf` messages that are
// capitalized or end with punctuation. Error strings are often wrapped by a
// caller (`fmt.Errorf("doing x: %w", err)`), so a leading capital or trailing
// period reads badly mid-sentence. This mirrors staticcheck ST1005.
type AuditErrorStringFormat struct {
	recipe.Base
}

func (r *AuditErrorStringFormat) Name() string {
	return "org.openrewrite.golang.codequality.AuditErrorStringFormat"
}
func (r *AuditErrorStringFormat) DisplayName() string { return "Audit error string format" }
func (r *AuditErrorStringFormat) Description() string {
	return "Find `errors.New` and `fmt.Errorf` messages that are capitalized or end with punctuation. Error strings are often embedded in a larger message, so they should not be capitalized or end with punctuation (staticcheck ST1005)."
}
func (r *AuditErrorStringFormat) Tags() []string { return []string{"error-handling", "lint"} }

func (r *AuditErrorStringFormat) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "ST1005", Tool: diagnostic.Staticcheck, HasFix: false},
	}
}

func (r *AuditErrorStringFormat) DataTables() []recipe.DataTableDescriptor {
	return []recipe.DataTableDescriptor{errorStringFormatTable.Descriptor()}
}

func (r *AuditErrorStringFormat) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditErrorStringFormatVisitor{})
}

type auditErrorStringFormatVisitor struct {
	visitor.GoVisitor
	sourcePath string
}

func (v *auditErrorStringFormatVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.sourcePath = cu.SourcePath
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *auditErrorStringFormatVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok {
		return mi
	}
	pkg, fn := ident.Name, mi.Name.Name
	if !((pkg == "errors" && fn == "New") || (pkg == "fmt" && fn == "Errorf")) {
		return mi
	}

	args := mi.Arguments.Elements
	if len(args) == 0 {
		return mi
	}
	lit, ok := args[0].Element.(*java.Literal)
	if !ok {
		return mi
	}
	msg, err := strconv.Unquote(lit.Source)
	if err != nil || msg == "" {
		return mi
	}

	reason := errorStringIssue(msg)
	if reason == "" {
		return mi
	}
	if ctx, ok := p.(*recipe.ExecutionContext); ok {
		errorStringFormatTable.InsertRow(ctx, ErrorStringFormatRow{
			SourcePath:  v.sourcePath,
			Constructor: pkg + "." + fn,
			Issue:       reason,
			Message:     msg,
		})
	}
	return mi.WithMarkers(java.MarkupInfo(mi.Markers, "error string should not "+reason+" (ST1005)"))
}

// errorStringIssue reports why an error message violates ST1005, or "" if it is
// fine. Capitalization is only flagged when the first word looks like an
// ordinary word (leading upper followed by lower), so initialisms and proper
// nouns like "HTTP" or "TLS" are left alone.
func errorStringIssue(msg string) string {
	runes := []rune(msg)
	if len(runes) >= 2 && unicode.IsUpper(runes[0]) && unicode.IsLower(runes[1]) {
		return "be capitalized"
	}
	if !strings.HasSuffix(msg, "...") {
		switch runes[len(runes)-1] {
		case '.', ':', '!':
			return "end with punctuation"
		}
	}
	return ""
}
