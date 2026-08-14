/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"strings"
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// The inventory recipe reports only and must not modify code.
func TestFindEncodingJsonUsageReportsOnly(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&jsonv2.FindEncodingJsonUsage{})
	spec.RewriteRun(t,
		test.Golang(`
			package demo

			import "encoding/json"

			type Payload struct {
				Raw json.RawMessage
			}

			func run(p Payload) {
				_, _ = json.Marshal(p)
			}
		`),
	)
}

func TestFindEncodingJsonUsageDataTable(t *testing.T) {
	// `@` stands in for a backtick so struct tags read cleanly in a raw string.
	source := strings.ReplaceAll(`package demo

import (
	"encoding/json"
	"time"
)

type Config struct {
	ID      [8]byte
	Timeout time.Duration @json:",omitempty"@
	Name    string        @json:",omitempty"@
	Raw     json.RawMessage
}

func (Config) MarshalJSON() ([]byte, error) { return nil, nil }

func run(c Config) {
	_, _ = json.Marshal(c)
	_ = json.MarshalIndent(c, "", "  ")

	enc := json.NewEncoder(nil)
	enc.SetIndent("", "  ")
	_ = enc.Encode(c)

	dec := json.NewDecoder(nil)
	dec.DisallowUnknownFields()
	_ = dec.Decode(&c)

	var e *json.Encoder
	_ = e
	var s *json.SyntaxError
	_ = s
}
`, "@", "`")

	cu, err := parser.NewGoParser().Parse("demo/config.go", source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	store := recipe.NewInMemoryDataTableStore()
	ctx := recipe.NewExecutionContext()
	ctx.PutMessage(recipe.DataTableStoreKey, store)
	(&jsonv2.FindEncodingJsonUsage{}).Editor().Visit(cu, ctx)

	raw := store.GetRows("org.openrewrite.golang.migration.table.EncodingJsonUsage", "")
	got := map[string]int{}
	for _, r := range raw {
		row := r.(jsonv2.EncodingJsonUsageRow)
		got[row.Category+"|"+row.API]++
	}

	want := map[string]int{
		"import|encoding/json":                      1,
		"review|[8]byte":                            1,
		"review|time.Duration":                      1,
		"review|omitempty tag":                      2,
		"rewrite|json.RawMessage":                   1,
		"modernize|MarshalJSON":                     1,
		"review|json.Marshal":                       1,
		"rewrite|json.MarshalIndent":                1,
		"rewrite|json.NewEncoder":                   1,
		"rewrite|json.Encoder.SetIndent":            1,
		"rewrite|json.Encoder.Encode":               1,
		"rewrite|json.NewDecoder":                   1,
		"review|json.Decoder.DisallowUnknownFields": 1,
		"rewrite|json.Decoder.Decode":               1,
		"rewrite|json.Encoder":                      1,
		"review|json.SyntaxError":                   1,
	}

	for k, n := range want {
		if got[k] != n {
			t.Errorf("category|api %q: got %d, want %d", k, got[k], n)
		}
	}
	total := 0
	for _, n := range want {
		total += n
	}
	if len(raw) != total {
		t.Errorf("row count = %d, want %d; all rows: %v", len(raw), total, got)
	}
}

// An aliased import binds detection to the local package name, and findings are
// still reported under the canonical json.* labels.
func TestFindEncodingJsonUsageAliasedImport(t *testing.T) {
	source := `package demo

import j "encoding/json"

func run(v any) {
	_, _ = j.Marshal(v)
	enc := j.NewEncoder(nil)
	_ = enc.Encode(v)
}
`
	cu, err := parser.NewGoParser().Parse("demo/aliased.go", source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	store := recipe.NewInMemoryDataTableStore()
	ctx := recipe.NewExecutionContext()
	ctx.PutMessage(recipe.DataTableStoreKey, store)
	(&jsonv2.FindEncodingJsonUsage{}).Editor().Visit(cu, ctx)

	raw := store.GetRows("org.openrewrite.golang.migration.table.EncodingJsonUsage", "")
	got := map[string]int{}
	for _, r := range raw {
		row := r.(jsonv2.EncodingJsonUsageRow)
		got[row.Category+"|"+row.API]++
	}

	want := map[string]int{
		"import|encoding/json":        1,
		"review|json.Marshal":         1,
		"rewrite|json.NewEncoder":     1,
		"rewrite|json.Encoder.Encode": 1,
	}
	for k, n := range want {
		if got[k] != n {
			t.Errorf("category|api %q: got %d, want %d (all: %v)", k, got[k], n, got)
		}
	}
	if len(raw) != 4 {
		t.Errorf("row count = %d, want 4; all rows: %v", len(raw), got)
	}
}

func runInventory(t *testing.T, path, source string) []jsonv2.EncodingJsonUsageRow {
	t.Helper()
	cu, err := parser.NewGoParser().Parse(path, source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	store := recipe.NewInMemoryDataTableStore()
	ctx := recipe.NewExecutionContext()
	ctx.PutMessage(recipe.DataTableStoreKey, store)
	(&jsonv2.FindEncodingJsonUsage{}).Editor().Visit(cu, ctx)
	raw := store.GetRows("org.openrewrite.golang.migration.table.EncodingJsonUsage", "")
	rows := make([]jsonv2.EncodingJsonUsageRow, len(raw))
	for i, r := range raw {
		rows[i] = r.(jsonv2.EncodingJsonUsageRow)
	}
	return rows
}

// The package functions and streaming token types that v2 removed without a
// shipped rewrite are reported for review, not rewrite.
func TestFindEncodingJsonUsageReviewLabels(t *testing.T) {
	rows := runInventory(t, "demo/review.go", `package demo

import (
	"bytes"
	"encoding/json"
)

func run(dst *bytes.Buffer, src []byte, tok json.Token, d json.Delim) {
	_ = json.Compact(dst, src)
	_ = json.Indent(dst, src, "", "  ")
	json.HTMLEscape(dst, src)
	_ = json.Valid(src)
	_, _ = tok, d
}
`)
	got := map[string]int{}
	for _, row := range rows {
		got[row.Category+"|"+row.API]++
	}
	for _, k := range []string{
		"review|json.Compact", "review|json.Indent", "review|json.HTMLEscape",
		"review|json.Valid", "review|json.Token", "review|json.Delim",
	} {
		if got[k] != 1 {
			t.Errorf("expected %q once, got %d (all: %v)", k, got[k], got)
		}
	}
}

// Codecs reached through a parameter (not a local NewEncoder/NewDecoder) are
// found via the method's declaring type.
func TestFindEncodingJsonUsageIndirectCodec(t *testing.T) {
	rows := runInventory(t, "demo/indirect.go", `package demo

import "encoding/json"

func write(enc *json.Encoder, v any) { _ = enc.Encode(v) }
func read(dec *json.Decoder, v any)  { _ = dec.Decode(v) }
`)
	got := map[string]int{}
	for _, row := range rows {
		got[row.Category+"|"+row.API]++
	}
	for _, k := range []string{"rewrite|json.Encoder.Encode", "rewrite|json.Decoder.Decode"} {
		if got[k] != 1 {
			t.Errorf("expected %q once, got %d (all: %v)", k, got[k], got)
		}
	}
}

// v2 tightens decoding in two silent ways (case-sensitive member matching drops
// mixed-case keys; strict base64 rejects embedded newlines v1 ignored), so the
// finder flags each decode entry point with both risks rather than a generic note.
func TestFindEncodingJsonUsageDecodeStrictness(t *testing.T) {
	rows := runInventory(t, "demo/dec.go", `package demo

import "encoding/json"

func run(data []byte, dec *json.Decoder, v any) {
	_, _ = json.Marshal(v)
	_ = json.Unmarshal(data, v)
	_ = dec.Decode(v)
}
`)
	suggestion := map[string]string{}
	for _, row := range rows {
		suggestion[row.Category+"|"+row.API] = row.Suggestion
	}
	for _, k := range []string{"review|json.Unmarshal", "rewrite|json.Decoder.Decode"} {
		if !strings.Contains(suggestion[k], "case-sensitiv") {
			t.Errorf("%q should warn about case-sensitive matching, got %q", k, suggestion[k])
		}
		if !strings.Contains(suggestion[k], "base64") {
			t.Errorf("%q should warn about strict base64 decoding, got %q", k, suggestion[k])
		}
	}
	// Marshal keeps its own output-oriented message, not the decode warnings.
	if strings.Contains(suggestion["review|json.Marshal"], "base64") {
		t.Errorf("json.Marshal should not carry the decode warnings, got %q", suggestion["review|json.Marshal"])
	}
}

// Dot-imported package functions carry no `json.` qualifier and are found via
// the declaring type.
func TestFindEncodingJsonUsageDotImport(t *testing.T) {
	rows := runInventory(t, "demo/dot.go", `package demo

import . "encoding/json"

func run(v any) {
	_, _ = Marshal(v)
	_ = Valid(nil)
}
`)
	got := map[string]int{}
	for _, row := range rows {
		got[row.Category+"|"+row.API]++
	}
	for _, k := range []string{"review|json.Marshal", "review|json.Valid"} {
		if got[k] != 1 {
			t.Errorf("expected %q once, got %d (all: %v)", k, got[k], got)
		}
	}
}

// Dot-imported type references are bare identifiers resolved through the type
// system (var, field, and parameter positions, including a pointer).
func TestFindEncodingJsonUsageDotImportTypes(t *testing.T) {
	rows := runInventory(t, "demo/dottypes.go", `package demo

import . "encoding/json"

type Holder struct {
	Raw RawMessage
}

func use(m Marshaler, enc *Encoder) {
	var n Number
	_, _, _ = m, enc, n
}
`)
	got := map[string]int{}
	for _, row := range rows {
		got[row.Category+"|"+row.API]++
	}
	want := []string{"rewrite|json.RawMessage", "modernize|json.Marshaler", "rewrite|json.Encoder", "review|json.Number"}
	for _, k := range want {
		if got[k] != 1 {
			t.Errorf("expected %q once, got %d (all: %v)", k, got[k], got)
		}
	}
}

// omitempty rows are classified by the field type resolved through the type system.
func TestFindEncodingJsonUsageOmitemptyClassification(t *testing.T) {
	rows := runInventory(t, "demo/omit.go", strings.ReplaceAll(`package demo

import "time"

type Config struct {
	When  time.Time @json:",omitempty"@
	Count int       @json:",omitempty"@
}
`, "@", "`"))

	byField := map[string]string{}
	for _, row := range rows {
		if row.API == "omitempty tag" {
			byField[row.Detail] = row.Suggestion
		}
	}
	if s := byField["Config.When"]; !strings.Contains(s, "known divergence") {
		t.Errorf("time.Time omitempty suggestion = %q, want it to flag a known divergence", s)
	}
	if s := byField["Config.Count"]; !strings.Contains(s, "primitive") {
		t.Errorf("int omitempty suggestion = %q, want it to note a primitive is unaffected", s)
	}
}
