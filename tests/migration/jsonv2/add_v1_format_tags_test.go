/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func addV1FormatTagsSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&jsonv2.AddV1FormatTags{})
}

// `@` stands in for a backtick so struct tags read cleanly in a raw string.
func backticks(s string) string {
	return strings.ReplaceAll(s, "@", "`")
}

// A time.Duration field gains json:",format:nano" whether it starts untagged,
// with a json name, with other options, or under a non-json tag.
func TestAddV1FormatTagsDuration(t *testing.T) {
	source := func(tag string) string {
		return backticks(fmt.Sprintf(`
			package main

			import "time"

			type Config struct {
				Timeout time.Duration%s
			}
		`, tag))
	}
	cases := []struct {
		name          string
		before, after string
	}{
		{"untagged", "", ` @json:",format:nano"@`},
		{"json name", ` @json:"timeout"@`, ` @json:"timeout,format:nano"@`},
		{"json name and omitempty", ` @json:"timeout,omitempty"@`, ` @json:"timeout,omitempty,format:nano"@`},
		{"non-json tag", ` @xml:"timeout"@`, ` @xml:"timeout" json:",format:nano"@`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addV1FormatTagsSpec().RewriteRun(t, test.Golang(source(tc.before), source(tc.after)))
		})
	}
}

// A fixed [N]byte field gains json:",format:array" whether untagged or already
// carrying a json name.
func TestAddV1FormatTagsFixedByteArray(t *testing.T) {
	source := func(tag string) string {
		return backticks(fmt.Sprintf(`
			package main

			type Header struct {
				Magic [16]byte%s
			}
		`, tag))
	}
	cases := []struct {
		name          string
		before, after string
	}{
		{"untagged", "", ` @json:",format:array"@`},
		{"json name", ` @json:"magic"@`, ` @json:"magic,format:array"@`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addV1FormatTagsSpec().RewriteRun(t, test.Golang(source(tc.before), source(tc.after)))
		})
	}
}

// Fields whose v2 encoding already matches v1, or that carry no wire format, are
// left unchanged.
func TestAddV1FormatTagsNoChange(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{"already pins format:nano", backticks(`
			package main

			import "time"

			type Config struct {
				Timeout time.Duration @json:",format:nano"@
			}
		`)},
		{"explicit different format", backticks(`
			package main

			import "time"

			type Config struct {
				Timeout time.Duration @json:",format:units"@
			}
		`)},
		{"byte slice", backticks(`
			package main

			type Blob struct {
				Data []byte
			}
		`)},
		{"duration outside a struct", backticks(`
			package main

			import "time"

			func wait(d time.Duration) time.Duration {
				var next time.Duration
				return next
			}
		`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addV1FormatTagsSpec().RewriteRun(t, test.Golang(tc.source))
		})
	}
}
