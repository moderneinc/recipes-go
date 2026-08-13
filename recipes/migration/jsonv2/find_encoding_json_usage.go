/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// EncodingJsonUsageRow is one row of the FindEncodingJsonUsage data table.
type EncodingJsonUsageRow struct {
	SourcePath string
	Category   string
	API        string
	Detail     string
	Suggestion string
}

var encodingJsonUsageTable = recipe.NewDataTable[EncodingJsonUsageRow](
	"org.openrewrite.golang.migration.table.EncodingJsonUsage",
	"encoding/json usage for the v2 migration",
	"Every encoding/json touchpoint that an encoding/json/v2 migration must address.",
	[]recipe.ColumnDescriptor{
		{Name: "sourcePath", DisplayName: "Source path", Description: "File containing the usage.", Type: "String"},
		{Name: "category", DisplayName: "Category", Description: "One of import, rewrite, review, or modernize.", Type: "String"},
		{Name: "api", DisplayName: "API", Description: "The v1 symbol or pattern detected.", Type: "String"},
		{Name: "detail", DisplayName: "Detail", Description: "Struct field, receiver variable, import alias, or other context.", Type: "String"},
		{Name: "suggestion", DisplayName: "Suggestion", Description: "How the v2 migration should handle it.", Type: "String"},
	},
)

// A single pass over a file cataloguing every `encoding/json` touchpoint an
// `encoding/json/v2` migration must address (the import, package functions,
// Encoder/Decoder and other type methods resolved through the type system,
// exported types, custom marshalers, and struct field and tag concerns),
// reported to a data table without modifying any code, so a migration can be
// scoped and risk-ranked from one report.
type FindEncodingJsonUsage struct {
	recipe.Base
}

func (r *FindEncodingJsonUsage) Name() string {
	return "org.openrewrite.golang.migration.FindEncodingJsonUsage"
}
func (r *FindEncodingJsonUsage) DisplayName() string {
	return "Find `encoding/json` usage for the v2 migration"
}
func (r *FindEncodingJsonUsage) Description() string {
	return "Inventory every `encoding/json` (v1) touchpoint that an `encoding/json/v2` migration must address: the import, package functions, `Encoder`/`Decoder` and other type methods (resolved through the type system, so receivers reached via variables, parameters, or fields are all found), exported types, `[N]byte`/`time.Duration` struct fields, `omitempty` tags classified by field type, and custom `MarshalJSON`/`UnmarshalJSON` implementations. Findings populate a data table categorized as import, rewrite, review, or modernize. This recipe reports only and does not modify code."
}
func (r *FindEncodingJsonUsage) Tags() []string { return []string{"migration", "json"} }

func (r *FindEncodingJsonUsage) DataTables() []recipe.DataTableDescriptor {
	return []recipe.DataTableDescriptor{encodingJsonUsageTable.Descriptor()}
}

func (r *FindEncodingJsonUsage) Editor() recipe.TreeVisitor {
	return visitor.Init(&findEncodingJsonUsageVisitor{})
}

type findEncodingJsonUsageVisitor struct {
	visitor.GoVisitor
	sourcePath   string
	jsonPkg      string
	currentType  string
	insideStruct int
}

func (v *findEncodingJsonUsageVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.sourcePath = cu.SourcePath
	v.jsonPkg = localJsonPackage(cu)
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *findEncodingJsonUsageVisitor) VisitImport(imp *java.Import, p any) java.J {
	imp = v.GoVisitor.VisitImport(imp, p).(*java.Import)
	if path, ok := importPath(imp); ok && path == "encoding/json" {
		v.insertRow(p, "import", "encoding/json", importAlias(imp), "rewrite the import to encoding/json/v2")
	}
	return imp
}

func (v *findEncodingJsonUsageVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	sel := selectIdentName(mi)
	declaring := declaringFQN(mi)
	switch {
	case v.jsonPkg != "" && sel == v.jsonPkg:
		// Qualified package function; works from syntax without type info.
		v.reportJsonFunc(p, mi.Name.Name)
	case declaring == "encoding/json":
		// Package function reached without the `json.` qualifier (dot import).
		v.reportJsonFunc(p, mi.Name.Name)
	case declaring == "encoding/json.Encoder":
		category, suggestion := classifyEncoderMethod(mi.Name.Name)
		v.insertRow(p, category, "json.Encoder."+mi.Name.Name, sel, suggestion)
	case declaring == "encoding/json.Decoder":
		category, suggestion := classifyDecoderMethod(mi.Name.Name)
		v.insertRow(p, category, "json.Decoder."+mi.Name.Name, sel, suggestion)
	case strings.HasPrefix(declaring, "encoding/json."):
		v.insertRow(p, "review", strings.TrimPrefix(declaring, "encoding/")+"."+mi.Name.Name, sel,
			"method on an encoding/json type; verify the v2 equivalent")
	}
	return mi
}

func (v *findEncodingJsonUsageVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	ident, ok := fa.Target.(*java.Identifier)
	if !ok || v.jsonPkg == "" || ident.Name != v.jsonPkg {
		return fa
	}
	category, suggestion := classifyJsonType(fa.Name.Element.Name)
	v.insertRow(p, category, "json."+fa.Name.Element.Name, "", suggestion)
	return fa
}

func (v *findEncodingJsonUsageVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	if md.Name != nil {
		switch md.Name.Name {
		case "MarshalJSON":
			v.insertRow(p, "modernize", "MarshalJSON", "", "optionally implement streaming MarshalJSONTo(*jsontext.Encoder)")
		case "UnmarshalJSON":
			v.insertRow(p, "modernize", "UnmarshalJSON", "", "optionally implement streaming UnmarshalJSONFrom(*jsontext.Decoder)")
		}
	}
	return v.GoVisitor.VisitMethodDeclaration(md, p)
}

func (v *findEncodingJsonUsageVisitor) VisitTypeDecl(td *golang.TypeDecl, p any) java.J {
	prev := v.currentType
	if td.Name != nil {
		v.currentType = td.Name.Name
	}
	td = v.GoVisitor.VisitTypeDecl(td, p).(*golang.TypeDecl)
	v.currentType = prev
	return td
}

func (v *findEncodingJsonUsageVisitor) VisitStructType(st *golang.StructType, p any) java.J {
	v.insideStruct++
	st = v.GoVisitor.VisitStructType(st, p).(*golang.StructType)
	v.insideStruct--
	return st
}

func (v *findEncodingJsonUsageVisitor) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	vd = v.GoVisitor.VisitVariableDeclarations(vd, p).(*java.VariableDeclarations)

	// A bare (unqualified) type identifier resolving to encoding/json is a
	// dot-imported type reference at a var, field, or parameter position.
	// Qualified references are FieldAccess nodes handled in VisitFieldAccess.
	if name, ok := bareJsonTypeName(vd.TypeExpr); ok {
		category, suggestion := classifyJsonType(name)
		v.insertRow(p, category, "json."+name, "", suggestion)
	}

	if v.insideStruct == 0 || vd.TypeExpr == nil {
		return vd
	}

	if at, ok := vd.TypeExpr.(*golang.ArrayType); ok {
		if elem, ok := at.ElementType.(*java.Identifier); ok && elem.Name == "byte" {
			fieldType := "[" + arrayLengthText(at) + "]byte"
			for _, field := range fieldNames(vd) {
				v.insertRow(p, "review", fieldType, qualifyField(v.currentType, field),
					"encodes as a base64 string in v2; add json:\",format:array\" to keep v1 output")
			}
		}
	} else if fa, ok := vd.TypeExpr.(*java.FieldAccess); ok {
		if pkg, ok := fa.Target.(*java.Identifier); ok && pkg.Name == "time" && fa.Name.Element.Name == "Duration" {
			for _, field := range fieldNames(vd) {
				v.insertRow(p, "review", "time.Duration", qualifyField(v.currentType, field),
					"representation changes in v2; add json:\",format:nano\" to keep v1 output")
			}
		}
	}

	if hasOmitemptyTag(vd) {
		for _, decl := range vd.Variables {
			d := decl.Element
			if d == nil || d.Name == nil {
				continue
			}
			v.insertRow(p, "review", "omitempty tag", qualifyField(v.currentType, d.Name.Name), classifyOmitempty(d.Name.Type))
		}
	}
	return vd
}

func (v *findEncodingJsonUsageVisitor) reportJsonFunc(p any, name string) {
	if category, suggestion, ok := classifyJsonFunc(name); ok {
		v.insertRow(p, category, "json."+name, "", suggestion)
	} else {
		v.insertRow(p, "review", "json."+name, "", "encoding/json call; verify against the v2 API")
	}
}

func (v *findEncodingJsonUsageVisitor) insertRow(p any, category, api, detail, suggestion string) {
	ctx, ok := p.(*recipe.ExecutionContext)
	if !ok {
		return
	}
	encodingJsonUsageTable.InsertRow(ctx, EncodingJsonUsageRow{
		SourcePath: v.sourcePath,
		Category:   category,
		API:        api,
		Detail:     detail,
		Suggestion: suggestion,
	})
}

// classifyJsonFunc categorizes an encoding/json package function by name.
func classifyJsonFunc(name string) (category, suggestion string, ok bool) {
	switch name {
	case "Marshal", "Unmarshal":
		return "review", "defaults differ in v2; pass json.DefaultOptionsV1() for byte-identical output", true
	case "MarshalIndent":
		return "rewrite", "removed in v2; use json.Marshal with jsontext.WithIndent and jsontext.WithIndentPrefix", true
	case "NewEncoder":
		return "rewrite", "moves to jsontext.NewEncoder; encode via json.MarshalEncode", true
	case "NewDecoder":
		return "rewrite", "moves to jsontext.NewDecoder; decode via json.UnmarshalDecode", true
	case "Indent":
		return "rewrite", "removed in v2; use jsontext.Value.Indent", true
	case "Compact":
		return "rewrite", "removed in v2; use jsontext.Value.Compact", true
	case "HTMLEscape":
		return "rewrite", "removed in v2; use the jsontext.EscapeForHTML option", true
	case "Valid":
		return "rewrite", "removed in v2; validate via jsontext", true
	}
	return "", "", false
}

// classifyEncoderMethod categorizes a method called on a json.Encoder.
func classifyEncoderMethod(name string) (category, suggestion string) {
	switch name {
	case "Encode":
		return "rewrite", "encode via json.MarshalEncode(enc, v) in v2"
	case "SetIndent":
		return "rewrite", "configure indentation via the jsontext.WithIndent option in v2"
	case "SetEscapeHTML":
		return "review", "configure HTML escaping via the jsontext.EscapeForHTML option in v2"
	}
	return "review", "method on a json.Encoder; verify the v2 equivalent"
}

// classifyDecoderMethod categorizes a method called on a json.Decoder.
func classifyDecoderMethod(name string) (category, suggestion string) {
	switch name {
	case "Decode":
		return "rewrite", "decode via json.UnmarshalDecode(dec, &v) in v2"
	case "DisallowUnknownFields":
		return "review", "use the json.RejectUnknownMembers option in v2"
	case "UseNumber":
		return "review", "numeric handling differs in v2"
	case "Token", "More", "Buffered", "InputOffset":
		return "review", "streaming token API; verify jsontext equivalents in v2"
	}
	return "review", "method on a json.Decoder; verify the v2 equivalent"
}

// classifyJsonType categorizes an encoding/json exported type reference,
// defaulting to a generic review row so no reference is missed.
func classifyJsonType(name string) (category, suggestion string) {
	switch name {
	case "RawMessage":
		return "rewrite", "the raw JSON type moves to jsontext.Value in v2"
	case "Number":
		return "review", "review numeric handling; v2 tightens parsing and adds StringifyNumbers"
	case "Encoder", "Decoder":
		return "rewrite", "the streaming type moves to jsontext in v2"
	case "Marshaler":
		return "modernize", "consider the streaming MarshalerTo interface (MarshalJSONTo) in v2"
	case "Unmarshaler":
		return "modernize", "consider the streaming UnmarshalerFrom interface (UnmarshalJSONFrom) in v2"
	case "Token", "Delim":
		return "review", "streaming token API; verify jsontext token equivalents in v2"
	case "SyntaxError", "UnmarshalTypeError", "UnsupportedTypeError", "UnsupportedValueError",
		"InvalidUnmarshalError", "MarshalerError", "InvalidUTF8Error", "UnmarshalFieldError":
		return "review", "error type; verify error types and handling in v2"
	}
	return "review", "encoding/json reference; verify against the v2 API"
}

// classifyOmitempty tailors the suggestion for an omitempty field by its type;
// v2 redefines omitempty to omit empty JSON values rather than Go zero values.
func classifyOmitempty(fieldType java.JavaType) string {
	base := "v2 omits empty JSON values, not Go zero values"
	if fq, ok := fieldType.(java.FullyQualified); ok {
		switch fq.GetFullyQualifiedName() {
		case "time.Time", "time.Duration":
			return base + "; " + fq.GetFullyQualifiedName() + " is a known divergence, switch to omitzero"
		}
		return base + "; the named type " + fq.GetFullyQualifiedName() + " may diverge (structs and custom marshalers), verify and consider omitzero"
	}
	if _, ok := fieldType.(*java.JavaTypePrimitive); ok {
		return base + ", but a primitive field is unaffected"
	}
	return base + "; verify this field and consider omitzero"
}

// bareJsonTypeName returns the encoding/json type named by a bare (unqualified)
// type identifier, unwrapping a single pointer. Bare json identifiers only
// occur under a dot import; qualified references are FieldAccess nodes.
func bareJsonTypeName(expr java.Expression) (string, bool) {
	if pt, ok := expr.(*golang.PointerType); ok {
		expr = pt.Elem
	}
	id, ok := expr.(*java.Identifier)
	if !ok || id.Type == nil {
		return "", false
	}
	fq, ok := id.Type.(java.FullyQualified)
	if !ok {
		return "", false
	}
	const prefix = "encoding/json."
	if name := fq.GetFullyQualifiedName(); strings.HasPrefix(name, prefix) {
		return strings.TrimPrefix(name, prefix), true
	}
	return "", false
}

// declaringFQN returns the fully qualified name of the type declaring the
// invoked method, or an empty string when type information is unavailable.
func declaringFQN(mi *java.MethodInvocation) string {
	if mi.MethodType != nil && mi.MethodType.DeclaringType != nil {
		return mi.MethodType.DeclaringType.GetFullyQualifiedName()
	}
	return ""
}

// selectIdentName returns the receiver identifier of a call, or "".
func selectIdentName(mi *java.MethodInvocation) string {
	if mi.Select == nil {
		return ""
	}
	if id, ok := mi.Select.Element.(*java.Identifier); ok {
		return id.Name
	}
	return ""
}

// hasOmitemptyTag reports whether a struct field carries `json:"...,omitempty"`.
func hasOmitemptyTag(vd *java.VariableDeclarations) bool {
	for _, ann := range vd.LeadingAnnotations {
		id, ok := ann.AnnotationType.(*java.Identifier)
		if !ok || id.Name != "json" || ann.Arguments == nil {
			continue
		}
		for _, arg := range ann.Arguments.Elements {
			lit, ok := arg.Element.(*java.Literal)
			if !ok {
				continue
			}
			value, _ := lit.Value.(string)
			for _, opt := range strings.Split(value, ",") {
				if opt == "omitempty" {
					return true
				}
			}
		}
	}
	return false
}

// localJsonPackage returns the local name the file uses for encoding/json (its
// alias, or "json"), or an empty string when the package is not imported.
func localJsonPackage(cu *golang.CompilationUnit) string {
	if cu.Imports == nil {
		return ""
	}
	for _, imp := range cu.Imports.Elements {
		if path, ok := importPath(imp.Element); ok && path == "encoding/json" {
			if alias := importAlias(imp.Element); alias != "" {
				return alias
			}
			return "json"
		}
	}
	return ""
}

// importPath returns the unquoted path of an import, e.g. "encoding/json".
func importPath(imp *java.Import) (string, bool) {
	lit, ok := imp.Qualid.(*java.Literal)
	if !ok {
		return "", false
	}
	if s, ok := lit.Value.(string); ok {
		return s, true
	}
	return strings.Trim(lit.Source, "`\""), true
}

// importAlias returns the import alias, or an empty string when unaliased.
func importAlias(imp *java.Import) string {
	if imp.Alias != nil && imp.Alias.Element != nil {
		return imp.Alias.Element.Name
	}
	return ""
}

// qualifyField renders a `Struct.Field` label, or just the field for an
// anonymous struct.
func qualifyField(structName, field string) string {
	if structName == "" {
		return field
	}
	return structName + "." + field
}

// arrayLengthText renders the fixed-array length expression for reporting.
func arrayLengthText(at *golang.ArrayType) string {
	switch e := at.Length.Element.(type) {
	case *java.Literal:
		return e.Source
	case *java.Identifier:
		return e.Name
	default:
		return "N"
	}
}

// fieldNames returns the declared names of a struct field declaration, or a
// single empty name for an embedded field with no explicit name.
func fieldNames(vd *java.VariableDeclarations) []string {
	var names []string
	for _, decl := range vd.Variables {
		if decl.Element != nil && decl.Element.Name != nil {
			names = append(names, decl.Element.Name.Name)
		}
	}
	if len(names) == 0 {
		names = append(names, "")
	}
	return names
}
