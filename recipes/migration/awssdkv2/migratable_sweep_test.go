/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
)

// TestMigratableSweep reports, over a directory of checked-out repositories, how
// many v1 files migrate and which construct held each of the rest back. The
// migration is module-wide, so the count that matters is modules rather than
// files, and the blocker list is what says where to look next.
//
//	AWS_SWEEP_ROOT=/path/to/checkouts go test ./recipes/migration/awssdkv2/ \
//	    -run TestMigratableSweep -count=1 -v
//
// The files are parsed one at a time, without the cross-file attribution a real
// LST carries, so the answer is a lower bound on what the CLI migrates.
func TestMigratableSweep(t *testing.T) {
	root := os.Getenv("AWS_SWEEP_ROOT")
	if root == "" {
		t.Skip("set AWS_SWEEP_ROOT to a directory of checked-out repositories")
	}
	repos, _ := os.ReadDir(root)
	reasons := map[string]int{}
	totalV1, totalOK, modOK, modTotal := 0, 0, 0, 0
	for _, r := range repos {
		if !r.IsDir() {
			continue
		}
		dir := filepath.Join(root, r.Name())
		p := parser.NewGoParser()
		v1, ok := 0, 0
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && (d.Name() == "vendor" || d.Name() == ".git" || d.Name() == ".moderne") {
				return filepath.SkipDir
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			cu, err := p.Parse(path, string(src))
			if err != nil || cu == nil || !importsV1(cu) {
				return nil
			}
			v1++
			s := scanFile(cu)
			if s.migratable {
				ok++
			} else {
				reason := s.reason
				if reason == "" {
					reason = "(unset)"
				}
				reasons[reason]++
				rel, _ := filepath.Rel(root, path)
				t.Logf("    BLOCK %-60s %s", rel, reason)
			}
			return nil
		})
		if v1 == 0 {
			continue
		}
		modTotal++
		if v1 == ok {
			modOK++
		}
		totalV1 += v1
		totalOK += ok
		t.Logf("%-32s %3d/%3d files", r.Name(), ok, v1)
	}
	t.Logf("TOTAL %d/%d files, %d/%d modules fully migratable", totalOK, totalV1, modOK, modTotal)
	type kv struct {
		k string
		n int
	}
	var list []kv
	for k, n := range reasons {
		list = append(list, kv{k, n})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })
	for _, e := range list {
		t.Logf("  %3d  %s", e.n, e.k)
	}
}
