/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"net"
	"strconv"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// BindAllInterfacesRow is one row of the AuditBindAllInterfaces data table.
type BindAllInterfacesRow struct {
	SourcePath string
	Call       string
	Address    string
}

var bindAllInterfacesTable = recipe.NewDataTable[BindAllInterfacesRow](
	"org.openrewrite.golang.codequality.AuditBindAllInterfaces$Findings",
	"Listeners bound to all interfaces",
	"Network listeners bound to all interfaces.",
	[]recipe.ColumnDescriptor{
		{Name: "sourcePath", DisplayName: "Source path", Description: "File containing the listener.", Type: "String"},
		{Name: "call", DisplayName: "Call", Description: "The listen function invoked.", Type: "String"},
		{Name: "address", DisplayName: "Address", Description: "The bound address.", Type: "String"},
	},
)

// AuditBindAllInterfaces finds network listeners bound to all interfaces
// (an empty host such as `:8080`, or `0.0.0.0`/`[::]`). Binding to all
// interfaces exposes the service beyond the loopback address and may be
// unintended; prefer an explicit host like `127.0.0.1`. This mirrors gosec
// G102.
type AuditBindAllInterfaces struct {
	recipe.Base
}

func (r *AuditBindAllInterfaces) Name() string {
	return "org.openrewrite.golang.codequality.AuditBindAllInterfaces"
}
func (r *AuditBindAllInterfaces) DisplayName() string { return "Audit binding to all interfaces" }
func (r *AuditBindAllInterfaces) Description() string {
	return "Find network listeners bound to all interfaces (e.g. `:8080`, `0.0.0.0`, `[::]`). This exposes the service beyond loopback and may be unintended; prefer an explicit host such as `127.0.0.1` (gosec G102)."
}
func (r *AuditBindAllInterfaces) Tags() []string { return []string{"style", "security"} }

func (r *AuditBindAllInterfaces) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "G102", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *AuditBindAllInterfaces) DataTables() []recipe.DataTableDescriptor {
	return []recipe.DataTableDescriptor{bindAllInterfacesTable.Descriptor()}
}

func (r *AuditBindAllInterfaces) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditBindAllInterfacesVisitor{})
}

type auditBindAllInterfacesVisitor struct {
	visitor.GoVisitor
	sourcePath string
}

func (v *auditBindAllInterfacesVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.sourcePath = cu.SourcePath
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *auditBindAllInterfacesVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || !isListenCall(ident.Name, mi.Name.Name) {
		return mi
	}

	for _, arg := range mi.Arguments.Elements {
		lit, ok := arg.Element.(*java.Literal)
		if !ok {
			continue
		}
		addr, err := strconv.Unquote(lit.Source)
		if err != nil {
			continue
		}
		if bindsAllInterfaces(addr) {
			if ctx, ok := p.(*recipe.ExecutionContext); ok {
				bindAllInterfacesTable.InsertRow(ctx, BindAllInterfacesRow{
					SourcePath: v.sourcePath,
					Call:       ident.Name + "." + mi.Name.Name,
					Address:    addr,
				})
			}
			return mi.WithMarkers(java.MarkupWarn(mi.Markers, "listener bound to all interfaces; prefer an explicit host such as 127.0.0.1"))
		}
	}
	return mi
}

func isListenCall(pkg, fn string) bool {
	switch pkg {
	case "net":
		return fn == "Listen" || fn == "ListenPacket"
	case "tls":
		return fn == "Listen"
	case "http":
		return fn == "ListenAndServe" || fn == "ListenAndServeTLS"
	}
	return false
}

func bindsAllInterfaces(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	switch host {
	case "", "0.0.0.0", "::":
		return true
	}
	return false
}
