/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import "testing"

// The qualifier-inference heuristic and its confidence signal, one case per
// path shape packageName distinguishes.
func TestPackageName(t *testing.T) {
	tests := []struct {
		path      string
		wantName  string // only asserted when confident
		confident bool
	}{
		{"fmt", "fmt", true},                     // single segment
		{"math/rand", "rand", true},              // last segment of a multi-segment path
		{"gopkg.in/yaml.v2", "yaml", true},       // gopkg.in ".vN": strip the version suffix
		{"github.com/onsi/ginkgo/v2", "", false}, // bare "/vN" is ambiguous (package "ginkgo" here, but "v1"/"v2" for k8s): declined
		{"example.com/foo.bar", "", false},       // dotted non-version segment: unknown
		{"github.com/foo/go-bar", "", false},     // last segment "go-bar" can't be a package name (invalid identifier)
	}
	for _, tt := range tests {
		name, confident := packageName(tt.path)
		if confident != tt.confident {
			t.Errorf("packageName(%q) confident = %v, want %v (name=%q)", tt.path, confident, tt.confident, name)
			continue
		}
		if confident && name != tt.wantName {
			t.Errorf("packageName(%q) name = %q, want %q", tt.path, name, tt.wantName)
		}
	}
}
