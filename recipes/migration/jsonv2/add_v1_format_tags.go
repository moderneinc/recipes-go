/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/google/uuid"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Pins the wire format of a time.Duration or fixed [N]byte struct field to its
// v1 encoding with a json format option, so the field migrates to
// encoding/json/v2 without a call-site compatibility option.
type AddV1FormatTags struct {
	recipe.Base
}

func (r *AddV1FormatTags) Name() string {
	return "org.openrewrite.golang.migration.AddV1FormatTags"
}
func (r *AddV1FormatTags) DisplayName() string {
	return "Add v1-preserving `format` tags to `time.Duration` and `[N]byte` fields"
}
func (r *AddV1FormatTags) Description() string {
	return "Add `json:\",format:nano\"` to `time.Duration` struct fields and `json:\",format:array\"` to fixed `[N]byte` array struct fields, whose default `encoding/json/v2` encoding otherwise diverges from v1 (a duration string rather than a nanosecond number, and a base64 string rather than a number array). With the tag the field encodes identically under `encoding/json` and `encoding/json/v2`, so `MigrateToJSONV2` migrates the file on its default path rather than leaving it for review. An existing `json` tag gains the option while its name and other options are kept, and a field that already pins a `format` is left unchanged."
}
func (r *AddV1FormatTags) Tags() []string { return []string{"migration", "json"} }

func (r *AddV1FormatTags) Editor() recipe.TreeVisitor {
	return visitor.Init(&addV1FormatTagsVisitor{})
}

type addV1FormatTagsVisitor struct {
	visitor.GoVisitor
	insideStruct int
}

func (v *addV1FormatTagsVisitor) VisitStructType(st *golang.StructType, p any) java.J {
	v.insideStruct++
	st = v.GoVisitor.VisitStructType(st, p).(*golang.StructType)
	v.insideStruct--
	return st
}

func (v *addV1FormatTagsVisitor) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	vd = v.GoVisitor.VisitVariableDeclarations(vd, p).(*java.VariableDeclarations)
	if v.insideStruct == 0 {
		return vd
	}
	format := v1FormatFor(vd)
	if format == "" || hasJsonFormatOption(vd) {
		return vd
	}
	return withJsonFormatTag(vd, format)
}

// Returns the json format option that keeps a struct field byte-identical to v1
// (nano for a time.Duration, array for a fixed [N]byte), or an empty string when
// the field's v2 encoding already matches v1.
func v1FormatFor(vd *java.VariableDeclarations) string {
	switch {
	case isDurationField(vd):
		return "nano"
	case isFixedByteArrayField(vd):
		return "array"
	}
	return ""
}

// Returns a copy of the struct field carrying `json:",format:<format>"`,
// appending the option to an existing json tag or adding a json tag when the
// field has none.
func withJsonFormatTag(vd *java.VariableDeclarations, format string) java.J {
	if idx := jsonAnnotationIndex(vd.LeadingAnnotations); idx >= 0 {
		updated, ok := appendFormatOption(vd.LeadingAnnotations[idx], format)
		if !ok {
			return vd
		}
		anns := append([]*java.Annotation{}, vd.LeadingAnnotations...)
		anns[idx] = updated
		c := *vd
		c.LeadingAnnotations = anns
		return &c
	}
	c := *vd
	c.LeadingAnnotations = append(append([]*java.Annotation{}, vd.LeadingAnnotations...), newJsonFormatAnnotation(format))
	return &c
}

// Returns the index of the field's json struct tag within its leading
// annotations, or -1 when the field has no json tag.
func jsonAnnotationIndex(anns []*java.Annotation) int {
	for i, ann := range anns {
		if id, ok := ann.AnnotationType.(*java.Identifier); ok && id.Name == "json" {
			return i
		}
	}
	return -1
}

// Returns a copy of a json tag annotation with `,format:<format>` appended to its
// value, or false when the annotation carries no string literal to extend.
func appendFormatOption(ann *java.Annotation, format string) (*java.Annotation, bool) {
	if ann.Arguments == nil || len(ann.Arguments.Elements) == 0 {
		return nil, false
	}
	lit, ok := ann.Arguments.Elements[0].Element.(*java.Literal)
	if !ok || len(lit.Source) < 2 || lit.Source[len(lit.Source)-1] != '"' {
		return nil, false
	}
	value, _ := lit.Value.(string)
	option := ",format:" + format
	newLit := *lit
	newLit.Value = value + option
	newLit.Source = lit.Source[:len(lit.Source)-1] + option + `"`

	elems := append([]java.RightPadded[java.Expression]{}, ann.Arguments.Elements...)
	elems[0] = java.RightPadded[java.Expression]{Element: &newLit, After: ann.Arguments.Elements[0].After}
	args := *ann.Arguments
	args.Elements = elems
	c := *ann
	c.Arguments = &args
	return &c, true
}

// Builds a `json:",format:<format>"` struct tag annotation spaced with the single
// leading space gofmt places before a field tag.
func newJsonFormatAnnotation(format string) *java.Annotation {
	value := ",format:" + format
	return &java.Annotation{
		ID:             uuid.New(),
		Prefix:         java.SingleSpace,
		AnnotationType: &java.Identifier{ID: uuid.New(), Name: "json"},
		Arguments: &java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{Element: &java.Literal{ID: uuid.New(), Source: `"` + value + `"`, Value: value}},
			},
		},
	}
}
