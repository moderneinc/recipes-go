/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// A stub embedding an iface interface overrides the page iterator to hand out
// its pages, as kube-secret-syncer's ListSecretsPages does. v2's paginator
// calls the operation instead, once per page, so the override would go unused
// and the call fall through to the nil interface it embeds. The stub gains the
// operation, serving each page its override hands out in turn.
const (
	pageReplayFile   = "zz_awssdkv2_pages.go"
	pageReplayHelper = "awsCompatReplayPages"
	pageReplaySource = `func (%[1]s %[2]s) %[3]s(_ context.Context, input *%[4]s.%[5]s, _ ...func(*%[4]s.Options)) (*%[4]s.%[6]s, error) {
	return awsCompatReplayPages(input.%[7]s, func(fn func(*%[4]s.%[6]s, bool) bool) error {
		first := *input
		first.%[7]s = nil
		return %[1]s.%[3]sPages(&first, fn)
	}, func(page *%[4]s.%[6]s, token *string) { page.%[8]s = token })
}`
	overriddenFormReason = "an override of v1's %s, which v2's client never calls"
)

// pageOverride is a declaration of an operation's page iterator on a type.
type pageOverride struct {
	service, operation, input, output string
}

// pageOverrideOf reports whether md declares `<Op>Pages(*Input, func(*Output,
// bool) bool) error` for an operation of a service the file imports.
func (s *fileScan) pageOverrideOf(md *java.MethodDeclaration) (pageOverride, bool) {
	if md.Name == nil {
		return pageOverride{}, false
	}
	operation, cut := strings.CutSuffix(md.Name.Name, "Pages")
	if !cut || operation == "" {
		return pageOverride{}, false
	}
	types := parameterTypes(md)
	if len(types) != 2 || md.ReturnType == nil || compactPrint(md.ReturnType) != "error" {
		return pageOverride{}, false
	}
	for local, service := range s.services {
		input, output, isOperation := awsmanifest.OperationShapes(service, operation)
		if !isOperation {
			continue
		}
		if compactPrint(types[0]) == "*"+local+"."+input && compactPrint(types[1]) == "func(*"+local+"."+output+",bool)bool" {
			return pageOverride{service: service, operation: operation, input: input, output: output}, true
		}
	}
	return pageOverride{}, false
}

// pageReplayable reports whether the paginator carries its token in one
// string field each way, which is what the replay sets.
func pageReplayable(o pageOverride) (inToken, outToken string, ok bool) {
	inToken, outToken, ok = awsmanifest.PageTokens(o.service, o.operation)
	if !ok || awsmanifest.PointerScalar(o.service, o.input, inToken) != "string" || awsmanifest.PointerScalar(o.service, o.output, outToken) != "string" {
		return "", "", false
	}
	return inToken, outToken, true
}

// overriddenV1Form names a method declaration of a form v2's client dropped —
// the WithContext, Request and PagesWithContext forms of an operation, or a
// waiter — which a stub overriding it would no longer have called.
func (s *fileScan) overriddenV1Form(md *java.MethodDeclaration) string {
	if md.Name == nil {
		return ""
	}
	name := md.Name.Name
	types := parameterTypes(md)
	takesInput := func(operation string) bool {
		for local, service := range s.services {
			if input, _, ok := awsmanifest.OperationShapes(service, operation); ok {
				for _, t := range types {
					if compactPrint(t) == "*"+local+"."+input {
						return true
					}
				}
			}
		}
		return false
	}
	for _, suffix := range []string{"PagesWithContext", "WithContext", "Request"} {
		if operation, cut := strings.CutSuffix(name, suffix); cut && operation != "" && takesInput(operation) {
			return name
		}
	}
	if strings.HasPrefix(name, waiterPrefix) && name != waiterPrefix {
		for local := range s.services {
			for _, t := range types {
				if p := compactPrint(t); strings.HasPrefix(p, "*"+local+".") && strings.HasSuffix(p, "Input") {
					return name
				}
			}
		}
	}
	return ""
}

// pageOverrideReason names a page iterator override, or another dropped form
// declared on a type, that the migration cannot carry, or "".
func (s *fileScan) pageOverrideReason(cu *golang.CompilationUnit) string {
	declared := map[string]bool{}
	var overrides []struct {
		receiver string
		o        pageOverride
	}
	for _, rp := range cu.Statements {
		md, isMethod := rp.Element.(*golang.MethodDeclaration)
		if !isMethod || md.Declaration == nil || md.Declaration.Name == nil {
			continue
		}
		receiver := receiverTypeName(md)
		declared[receiver+"."+md.Declaration.Name.Name] = true
		if o, ok := s.pageOverrideOf(md.Declaration); ok {
			overrides = append(overrides, struct {
				receiver string
				o        pageOverride
			}{receiver, o})
			continue
		}
		if form := s.overriddenV1Form(md.Declaration); form != "" {
			return fmt.Sprintf(overriddenFormReason, form)
		}
	}
	for _, r := range overrides {
		if _, _, ok := pageReplayable(r.o); !ok {
			return fmt.Sprintf(overriddenFormReason, r.o.operation+"Pages") + ", over a paginator whose token the replay cannot carry"
		}
		if declared[r.receiver+"."+r.o.operation] {
			return fmt.Sprintf(overriddenFormReason, r.o.operation+"Pages") + ", on a type declaring " + r.o.operation + " as well"
		}
	}
	return ""
}

// addPageReplays declares the operation beside each page iterator override.
func (v *migrateVisitor) addPageReplays(cu *golang.CompilationUnit) *golang.CompilationUnit {
	var statements []java.RightPadded[java.Statement]
	added := false
	for _, rp := range cu.Statements {
		statements = append(statements, rp)
		md, isMethod := rp.Element.(*golang.MethodDeclaration)
		if !isMethod || md.Declaration == nil {
			continue
		}
		o, ok := v.scan.pageOverrideOf(md.Declaration)
		if !ok {
			continue
		}
		replay, ok := v.pageReplay(md, o)
		if !ok {
			continue
		}
		statements = append(statements, java.RightPadded[java.Statement]{Element: replay})
		added = true
	}
	if !added {
		return cu
	}
	v.needsContext = true
	v.needsPageReplay = true
	c := *cu
	c.Statements = statements
	return &c
}

// pageReplay builds the operation replaying override md's pages.
func (v *migrateVisitor) pageReplay(md *golang.MethodDeclaration, o pageOverride) (java.Statement, bool) {
	inToken, outToken, ok := pageReplayable(o)
	if !ok || len(md.Receiver.Elements) != 1 {
		return nil, false
	}
	recv, isDecl := md.Receiver.Elements[0].Element.(*java.VariableDeclarations)
	if !isDecl || recv.TypeExpr == nil {
		return nil, false
	}
	name := "awsStub"
	if len(recv.Variables) == 1 && recv.Variables[0].Element.Name != nil && recv.Variables[0].Element.Name.Name != "" && recv.Variables[0].Element.Name.Name != "_" {
		name = recv.Variables[0].Element.Name.Name
	}
	local := v.scan.qualifierFor(o.service)
	source := fmt.Sprintf(pageReplaySource, name, compactPrint(recv.TypeExpr), o.operation, local, o.input, o.output, inToken, outToken)
	cu, err := parser.NewGoParser().Parse("replay.go", "package p\n\n"+source+"\n")
	if err != nil || len(cu.Statements) != 1 {
		return nil, false
	}
	typing := visitor.Init(&pageReplayTyping{pages: md.Declaration.MethodType, operation: o.operation})
	typed, isStmt := typing.Visit(cu.Statements[0].Element, nil).(java.Statement)
	if !isStmt {
		return nil, false
	}
	return lstutil.SetStmtPrefix(typed, java.ParseSpace("\n\n")), true
}

// pageReplayTyping attributes the replay's calls: the helper, and the override
// it calls back into.
type pageReplayTyping struct {
	visitor.GoVisitor
	pages     *java.JavaTypeMethod
	operation string
}

func (v *pageReplayTyping) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Name == nil {
		return mi
	}
	c := *mi
	switch mi.Name.Name {
	case pageReplayHelper:
		c.MethodType = lstutil.FuncType("", pageReplayHelper, nil)
	case v.operation + "Pages":
		c.MethodType = v.pages
		if c.MethodType == nil {
			c.MethodType = lstutil.FuncType("", mi.Name.Name, nil)
		}
	default:
		return mi
	}
	return &c
}

// parameterTypes lists md's parameter types, one per parameter, so `a, b T`
// counts twice.
func parameterTypes(md *java.MethodDeclaration) []java.Expression {
	var out []java.Expression
	for _, rp := range md.Parameters.Elements {
		vd, isDecl := rp.Element.(*java.VariableDeclarations)
		if !isDecl || vd.TypeExpr == nil {
			continue
		}
		for range max(len(vd.Variables), 1) {
			out = append(out, vd.TypeExpr)
		}
	}
	return out
}

// compactPrint prints e without whitespace, for comparing type expressions.
func compactPrint(e java.J) string {
	return strings.Join(strings.Fields(printer.Print(e)), "")
}

const pageReplayHelperSource = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import (
	"fmt"
	"strconv"
)

// v1's page iterators called a stub's own override, which handed out every
// page; v2's paginators call the operation once per page, passing back the
// token the last page carried. This serves the page the token numbers,
// numbering the next one until the override reports its last.
func awsCompatReplayPages[O any](token *string, pages func(func(*O, bool) bool) error, next func(*O, *string)) (*O, error) {
	want := 0
	if token != nil {
		n, err := strconv.Atoi(*token)
		if err != nil {
			return nil, fmt.Errorf("page token %%q was not handed out by this stub", *token)
		}
		want = n
	}
	var page *O
	seen := 0
	err := pages(func(p *O, last bool) bool {
		if seen < want {
			seen++
			return true
		}
		c := *p
		var token *string
		if !last {
			t := strconv.Itoa(seen + 1)
			token = &t
		}
		next(&c, token)
		page = &c
		return false
	})
	if page == nil && err == nil {
		page = new(O)
	}
	return page, err
}
`
