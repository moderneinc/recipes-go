/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package diagnostic

import (
	"embed"
	"sort"
	"strings"
	"sync"
)

//go:embed known_diagnostics.txt
var knownFS embed.FS

var known = sync.OnceValue(func() map[AnalyzerTool]map[string]bool {
	data, err := knownFS.ReadFile("known_diagnostics.txt")
	if err != nil {
		panic(err) // embedded at build time, so a read failure is a broken binary
	}
	byTool := map[AnalyzerTool]map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, id, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		tool, ok := toolByName[name]
		if !ok {
			continue
		}
		if byTool[tool] == nil {
			byTool[tool] = map[string]bool{}
		}
		byTool[tool][id] = true
	}
	return byTool
})

var toolByName = map[string]AnalyzerTool{
	"Staticcheck":  Staticcheck,
	"GoVet":        GoVet,
	"GolangciLint": GolangciLint,
}

// Known reports whether tool defines id.
func Known(tool AnalyzerTool, id string) bool {
	return known()[tool][id]
}

// IDs returns every ID the catalog records for tool, sorted.
func IDs(tool AnalyzerTool) []string {
	ids := make([]string, 0, len(known()[tool]))
	for id := range known()[tool] {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
