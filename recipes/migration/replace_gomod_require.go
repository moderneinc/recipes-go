/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// ReplaceRequire returns gm with the `require oldPath` directive repointed at
// newPath and newVersion, in place, so the entry keeps its comments and its
// `// indirect` marker. When newPath is already required the old entry is
// dropped instead, and when oldPath is not required gm comes back unchanged.
//
// Repointing in place leaves the block out of module-path order; the migrations
// that call this run FormatGoMod afterwards to restore it. It does not touch
// go.sum, so a `go mod tidy` is still needed to complete resolution.
func ReplaceRequire(gm *golang.GoMod, oldPath, newPath, newVersion string) *golang.GoMod {
	if oldPath == "" || newPath == "" || !requiredModuleSet(gm)[oldPath] {
		return gm
	}
	drop := requiredModuleSet(gm)[newPath]

	var out []java.RightPadded[golang.GoModStatement]
	changed := false
	for _, rp := range gm.Statements {
		switch el := rp.Element.(type) {
		case *golang.GoModDirective:
			if el.Keyword == "require" && directiveModulePath(el) == oldPath {
				changed = true
				if drop {
					continue
				}
				rp.Element = repointDirective(el, newPath, newVersion)
			}
		case *golang.GoModBlock:
			if el.Keyword == "require" {
				if block, blockChanged := repointRequireBlock(el, oldPath, newPath, newVersion, drop); blockChanged {
					changed = true
					if len(block.Entries) == 0 {
						continue
					}
					rp.Element = block
				}
			}
		}
		out = append(out, rp)
	}
	if !changed {
		return gm
	}
	return gm.WithStatements(out)
}

// repointRequireBlock repoints or drops the oldPath entry of a factored block.
func repointRequireBlock(b *golang.GoModBlock, oldPath, newPath, newVersion string, drop bool) (*golang.GoModBlock, bool) {
	var kept []java.RightPadded[golang.GoModStatement]
	changed, firstDropped := false, false
	for i, e := range b.Entries {
		d, ok := e.Element.(*golang.GoModDirective)
		if ok && directiveModulePath(d) == oldPath {
			changed = true
			if drop {
				if i == 0 {
					firstDropped = true
				}
				continue
			}
			e.Element = repointDirective(d, newPath, newVersion)
		}
		kept = append(kept, e)
	}
	if !changed {
		return b, false
	}
	if firstDropped {
		kept = restoreBlockOpeningNewline(kept)
	}
	return b.WithEntries(kept), true
}

// restoreBlockOpeningNewline puts back the leading newline the dropped first
// entry carried, so the block still opens on its own line.
func restoreBlockOpeningNewline(entries []java.RightPadded[golang.GoModStatement]) []java.RightPadded[golang.GoModStatement] {
	if len(entries) == 0 {
		return entries
	}
	d, ok := entries[0].Element.(*golang.GoModDirective)
	if !ok || strings.HasPrefix(d.Prefix.Whitespace, "\n") {
		return entries
	}
	sp := java.Space{Whitespace: "\n" + d.Prefix.Whitespace, Comments: d.Prefix.Comments}
	entries[0].Element = d.WithPrefix(sp)
	return entries
}

// repointDirective rewrites the module path and version values of a require
// line. The path is Values[0] for both a standalone `require path version`
// directive and a block entry, since a directive's values follow its keyword.
func repointDirective(d *golang.GoModDirective, newPath, newVersion string) *golang.GoModDirective {
	if len(d.Values) == 0 {
		return d
	}
	values := make([]*golang.GoModValue, len(d.Values))
	copy(values, d.Values)
	values[0] = values[0].WithText(newPath)
	if newVersion != "" && len(values) > 1 {
		values[1] = values[1].WithText(newVersion)
	}
	return d.WithValues(values)
}

// directiveModulePath returns the module path a require line names.
func directiveModulePath(d *golang.GoModDirective) string {
	if d == nil {
		return ""
	}
	return firstValueText(d)
}

// RemoveRequire returns gm with the direct `require modulePath` directive
// dropped. An entry marked `// indirect` is kept: it pins a version the module
// graph asked for rather than one the source imports, and dropping it changes
// resolution.
func RemoveRequire(gm *golang.GoMod, modulePath string) *golang.GoMod {
	if modulePath == "" {
		return gm
	}

	var out []java.RightPadded[golang.GoModStatement]
	changed := false
	for _, rp := range gm.Statements {
		switch el := rp.Element.(type) {
		case *golang.GoModDirective:
			if el.Keyword == "require" && directiveModulePath(el) == modulePath && !isIndirect(rp) {
				changed = true
				continue
			}
		case *golang.GoModBlock:
			if el.Keyword == "require" {
				if block, blockChanged := dropRequireEntry(el, modulePath); blockChanged {
					changed = true
					if len(block.Entries) == 0 {
						continue
					}
					rp.Element = block
				}
			}
		}
		out = append(out, rp)
	}
	if !changed {
		return gm
	}
	return gm.WithStatements(out)
}

func dropRequireEntry(b *golang.GoModBlock, modulePath string) (*golang.GoModBlock, bool) {
	var kept []java.RightPadded[golang.GoModStatement]
	changed, firstDropped := false, false
	for i, e := range b.Entries {
		d, ok := e.Element.(*golang.GoModDirective)
		if ok && directiveModulePath(d) == modulePath && !isIndirect(e) {
			changed = true
			if i == 0 {
				firstDropped = true
			}
			continue
		}
		kept = append(kept, e)
	}
	if !changed {
		return b, false
	}
	if firstDropped {
		kept = restoreBlockOpeningNewline(kept)
	}
	return b.WithEntries(kept), true
}

// isIndirect reports whether a require line carries the `// indirect` marker,
// which the parser attaches as a comment on the whitespace following the entry.
func isIndirect(rp java.RightPadded[golang.GoModStatement]) bool {
	for _, c := range rp.After.Comments {
		if strings.Contains(c.Text, "indirect") {
			return true
		}
	}
	return false
}
