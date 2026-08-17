/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package tests

import (
	"fmt"
	"go/ast"
	"go/build"
	goparser "go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/moderneinc/recipes-go/recipes"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// realWorldEnv gates the sweep, which reaches the network and takes longer than
// the rest of the suite.
const realWorldEnv = "REALWORLD_REPOS"

// realWorldRepo is an upstream repository the sweep runs over. The revision is
// pinned so a run's outcome depends on the pin rather than on the day it ran.
type realWorldRepo struct {
	Repo string // owner/name on github.com
	SHA  string
}

var realWorldRepos = []realWorldRepo{
	{"gorilla/mux", "db9d1d0073d27a0a2d9a8c1bc52aa0af4374d265"},
	{"spf13/cobra", "adbc8813901bba65827259daa8e22ff94ec1f30e"},
	{"sirupsen/logrus", "457e372460c7a80ca7c800b51ebeee5362aaa180"},
	{"go-chi/chi", "8b258c7bb28f97a5f2a856ff7ef962578fec9215"},
	{"labstack/echo", "05489dc1730161df26b72d1ae2a3ba6fb8178fc7"},
}

// registeredRecipes instantiates every recipe in the catalog. The registry is
// the set the library ships, so the sweep covers exactly that set.
func registeredRecipes() []recipe.Recipe {
	registry := recipe.NewRegistry()
	recipes.Activate(registry)

	var all []recipe.Recipe
	for _, registration := range registry.AllRegistrations() {
		if registration.Constructor == nil {
			continue
		}
		// Descriptor-only registrations construct to nil.
		if r := registration.Constructor(nil); r != nil {
			all = append(all, r)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name() < all[j].Name() })
	return all
}

// TestParseRealRepos runs every registered recipe over real Go source, checking
// that recipes neither panic nor emit output that fails to reparse or reprint.
func TestParseRealRepos(t *testing.T) {
	if os.Getenv(realWorldEnv) == "" {
		t.Skipf("set %s=1 to sweep every recipe over the pinned upstream repositories", realWorldEnv)
	}
	t.Logf("Sweeping %d registered recipes", len(registeredRecipes()))
	for _, repo := range realWorldRepos {
		t.Run(repo.Repo, func(t *testing.T) { sweepRepo(t, repo) })
	}
}

// fetchRepo checks the pinned revision out under build/, which is gitignored,
// and returns its path. The path carries the SHA, so each pin has its own
// checkout.
func fetchRepo(t *testing.T, repo realWorldRepo) string {
	t.Helper()

	dir := filepath.Join("..", "build", "realworld", filepath.FromSlash(repo.Repo)+"@"+repo.SHA)
	if _, err := os.Stat(dir); err == nil {
		return dir
	}

	// Populate a scratch path and rename, so an interrupted fetch cannot leave
	// a partial checkout that later runs would treat as cached.
	scratch := dir + ".partial"
	if err := os.RemoveAll(scratch); err != nil {
		t.Fatalf("clearing %s: %v", scratch, err)
	}
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatalf("creating %s: %v", scratch, err)
	}

	url := "https://github.com/" + repo.Repo + ".git"
	for _, args := range [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", url},
		{"fetch", "--depth", "1", "-q", "origin", repo.SHA},
		{"checkout", "-q", "FETCH_HEAD"},
	} {
		cmd := exec.Command("git", append([]string{"-C", scratch}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), scratch, err, out)
		}
	}

	if err := os.Rename(scratch, dir); err != nil {
		t.Fatalf("renaming %s to %s: %v", scratch, dir, err)
	}
	return dir
}

func sweepRepo(t *testing.T, repo realWorldRepo) {
	repoDir := fetchRepo(t, repo)

	var goFiles []string
	err := filepath.WalkDir(repoDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() && (d.Name() == "vendor" || d.Name() == "testdata" || d.Name() == ".git") {
			return filepath.SkipDir
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			goFiles = append(goFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", repoDir, err)
	}
	if len(goFiles) == 0 {
		t.Fatalf("no .go files under %s; the checkout of %s is not usable", repoDir, repo.Repo)
	}
	sort.Strings(goFiles)
	t.Logf("Found %d .go files in %s", len(goFiles), repo.Repo)

	results := sweepFiles(t, repoDir, goFiles)
	reportSweep(t, repo, results)
}

// fileResult is one file's outcome. Results are collected rather than logged in
// place so that the report reads the same however the work was scheduled.
type fileResult struct {
	path        string
	constrained bool     // build constraints exclude the file on this platform
	parseFail   string   // why the file did not parse and reprint; empty when it did
	spaceIssues []string // whitespace validation failures on the parsed tree
	problems    []string // recipes that panicked or produced unusable output
	findings    map[string]int
}

func sweepFiles(t *testing.T, repoDir string, goFiles []string) []fileResult {
	results := make([]fileResult, len(goFiles))

	indices := make(chan int)
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Per-worker parser and recipe instances: a recipe may carry state
			// across the files it visits, and the sweep visits them concurrently.
			p := parser.NewGoParser()
			recipes := registeredRecipes()
			for i := range indices {
				results[i] = sweepFile(p, recipes, repoDir, goFiles[i])
			}
		}()
	}
	for i := range goFiles {
		indices <- i
	}
	close(indices)
	wg.Wait()

	return results
}

func sweepFile(p *parser.GoParser, recipes []recipe.Recipe, repoDir, goFile string) fileResult {
	relPath, _ := filepath.Rel(repoDir, goFile)
	res := fileResult{path: relPath, findings: map[string]int{}}

	// The parser yields nothing for a file the current GOOS/GOARCH excludes, so
	// classifying those here keeps the sweep's report identical across platforms.
	if match, err := build.Default.MatchFile(filepath.Dir(goFile), filepath.Base(goFile)); err == nil && !match {
		res.constrained = true
		return res
	}

	srcBytes, err := os.ReadFile(goFile)
	if err != nil {
		res.parseFail = fmt.Sprintf("read: %v", err)
		return res
	}
	src := string(srcBytes)

	cu, err := p.Parse(relPath, src)
	if err != nil {
		res.parseFail = fmt.Sprintf("parse: %v", err)
		return res
	}
	if printer.Print(cu) != src {
		res.parseFail = "parse/print is not idempotent"
		return res
	}
	res.spaceIssues = test.ValidateSpaces(cu)

	for _, r := range recipes {
		editor := r.Editor()
		if editor == nil {
			continue
		}
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					res.problems = append(res.problems, fmt.Sprintf("PANIC        %s: %v", r.Name(), rec))
				}
			}()

			result := editor.Visit(cu, recipe.NewExecutionContext())
			if result == nil {
				return
			}

			after := printer.Print(result)
			if after != src {
				res.findings[r.DisplayName()]++
				cu2, err := p.Parse(relPath, after)
				if err != nil {
					res.problems = append(res.problems, fmt.Sprintf("UNPARSEABLE  %s: %v", r.Name(), err))
				} else if printer.Print(cu2) != after {
					res.problems = append(res.problems, fmt.Sprintf("ROUND-TRIP   %s: output does not reprint", r.Name()))
				}
			} else if printer.PrintWithMarkers(result, printer.DefaultMarkerPrinter) != src {
				res.findings[r.DisplayName()]++ // search-only recipe, reporting through markers
			}
		}()
	}
	return res
}

func reportSweep(t *testing.T, repo realWorldRepo, results []fileResult) {
	var parseOK, parseFail, constrained, spaceIssues int
	findings := map[string]int{}

	for _, res := range results {
		if res.constrained {
			constrained++
			continue
		}
		if res.parseFail != "" {
			parseFail++
			t.Logf("  PARSE FAIL: %s: %s", res.path, res.parseFail)
			continue
		}
		parseOK++
		spaceIssues += len(res.spaceIssues)
		for _, e := range res.spaceIssues {
			t.Logf("  SPACE: %s: %s", res.path, e)
		}
		for _, p := range res.problems {
			t.Errorf("  %s\n    on %s", p, res.path)
		}
		for name, count := range res.findings {
			findings[name] += count
		}
	}

	t.Logf("  Parse: %d OK, %d fail/idempotence issues, %d excluded by build constraints",
		parseOK, parseFail, constrained)
	t.Logf("  Space validation issues: %d", spaceIssues)
	t.Logf("  Recipe findings:")
	names := make([]string, 0, len(findings))
	for name := range findings {
		names = append(names, name)
	}
	sort.Strings(names)
	totalFindings := 0
	for _, name := range names {
		t.Logf("    %s: %d", name, findings[name])
		totalFindings += findings[name]
	}
	if totalFindings == 0 {
		t.Logf("    (none)")
	}
	fmt.Printf("\n[%s] Parse: %d OK, %d fail, %d constrained | Findings: %d\n",
		repo.Repo, parseOK, parseFail, constrained, totalFindings)
}

// unregisteredByDesign maps a recipe name to the reason it is kept out of
// recipes.Activate. Any other recipe missing from the registry is invisible to
// the catalog and the CLI, which the test below reports as a failure.
var unregisteredByDesign = map[string]string{
	// MigrateToJSONV2 composes these five, so registering them as well would
	// list them twice in the catalog.
	"org.openrewrite.golang.migration.MigrateImportOnlyToJSONV2":    "composed by MigrateToJSONV2",
	"org.openrewrite.golang.migration.MigrateStreamingEncodeDecode": "composed by MigrateToJSONV2",
	"org.openrewrite.golang.migration.RelocateEncoderDecoderTypes":  "composed by MigrateToJSONV2",
	"org.openrewrite.golang.migration.RelocateRawMessage":           "composed by MigrateToJSONV2",
	"org.openrewrite.golang.migration.ReplaceMarshalIndent":         "composed by MigrateToJSONV2",
}

func TestEveryDeclaredRecipeIsRegistered(t *testing.T) {
	registry := recipe.NewRegistry()
	registry.Activate(recipes.Activate)
	registered := make(map[string]bool)
	for _, desc := range registry.AllRecipes() {
		registered[desc.Name] = true
	}

	declared := declaredRecipeNames(t)

	var missing []string
	for name, file := range declared {
		if registered[name] || unregisteredByDesign[name] != "" {
			continue
		}
		missing = append(missing, fmt.Sprintf("%s (%s)", name, file))
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%d recipe(s) declared but not registered in recipes/activate.go:\n\t%s",
			len(missing), strings.Join(missing, "\n\t"))
	}

	// A stale allowlist entry hides a recipe as effectively as a missing
	// registration does.
	for name := range unregisteredByDesign {
		if _, ok := declared[name]; !ok {
			t.Errorf("allowlisted recipe %s no longer exists; remove it from unregisteredByDesign", name)
		}
		if registered[name] {
			t.Errorf("%s is registered; remove it from unregisteredByDesign", name)
		}
	}
}

// declaredRecipeNames maps every recipe name declared under recipes/ to the file
// declaring it. Reflection cannot enumerate a package's types, so the set is
// read from the source.
func declaredRecipeNames(t *testing.T) map[string]string {
	t.Helper()

	names := make(map[string]string)
	fset := token.NewFileSet()
	err := filepath.WalkDir(filepath.Join("..", "recipes"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := goparser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !isRecipeNameMethod(fn) {
				continue
			}
			name, ok := singleStringReturn(fn)
			if !ok {
				t.Errorf("%s: %s.Name() does not return a string literal; teach declaredRecipeNames how to read it",
					fset.Position(fn.Pos()), receiverTypeName(fn))
				continue
			}
			names[name] = path
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scanning recipes/: %v", err)
	}
	return names
}

// Every recipe declares its name through a pointer-receiver `Name() string`.
func isRecipeNameMethod(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) != 1 || fn.Name.Name != "Name" || fn.Body == nil {
		return false
	}
	if _, ok := fn.Recv.List[0].Type.(*ast.StarExpr); !ok {
		return false
	}
	if len(fn.Type.Params.List) != 0 || fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	ident, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	return ok && ident.Name == "string"
}

func singleStringReturn(fn *ast.FuncDecl) (string, bool) {
	if len(fn.Body.List) != 1 {
		return "", false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return "", false
	}
	lit, ok := ret.Results[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	return value, err == nil
}

func receiverTypeName(fn *ast.FuncDecl) string {
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return "?"
	}
	ident, ok := star.X.(*ast.Ident)
	if !ok {
		return "?"
	}
	return ident.Name
}
