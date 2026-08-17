/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package tests

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/moderneinc/recipes-go/recipes"
	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/moderneinc/recipes-go/recipes/naming"
	"github.com/moderneinc/recipes-go/recipes/performance"
	"github.com/moderneinc/recipes-go/recipes/redundancy"
	"github.com/moderneinc/recipes-go/recipes/simplification"
	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// allRecipes returns all code quality recipes for validation.
func allRecipes() []recipe.Recipe {
	return []recipe.Recipe{
		// Simplification
		&simplification.SimplifyBooleanExpression{},
		&simplification.ReplaceTimeSinceWithSince{},
		&simplification.ReplaceTimeUntilWithUntil{},
		&simplification.SimplifyRedundantNilCheck{},
		&simplification.SimplifySliceRange{},
		&simplification.SimplifyFmtSprintf{},
		&simplification.PreferBytesEqual{},
		&simplification.PreferSortInts{},
		&simplification.PreferStringsHasPrefix{},
		&simplification.UseStringsReplaceAll{},
		&simplification.SimplifyRedundantTrimSpace{},
		&simplification.PreferStringsContainsOverCount{},
		&simplification.PreferEmptyStringCheck{},
		&simplification.PreferLenCheck{},
		&simplification.PreferIoDiscard{},
		&simplification.PreferIoNopCloser{},
		&simplification.PreferIoReadAll{},
		&simplification.PreferOsReadFile{},
		&simplification.PreferOsWriteFile{},
		&simplification.PreferOsMkdirTemp{},
		&simplification.PreferOsCreateTemp{},
		&simplification.AvoidChannelLenCheck{},
		&simplification.RemoveRedundantRangeBlank{},
		&simplification.SimplifySingleCaseSelect{},
		&simplification.UseStructuredLogging{},
		&simplification.RemoveSwitchTrueTag{},
		&simplification.PreferBytesHasPrefix{},
		&simplification.PreferCopyString{},
		&simplification.PreferErrorsIsForOsCheck{},
		&simplification.PreferErrorsIsForPermission{},
		&simplification.PreferFilepathClean{},
		&simplification.PreferIoWriteString{},
		&simplification.PreferOsReadDir{},
		&simplification.PreferSlicesSort{},
		&simplification.PreferStrconvAtoi{},
		&simplification.PreferStringsBuilderWriteString{},
		&simplification.PreferStringsNewReader{},
		&simplification.SimplifyTrimLeftNoop{},
		&simplification.SimplifySprintfConcat{},
		&simplification.SimplifyDoubleNegation{},
		&simplification.SimplifyBytesBufferRoundtrip{},
		&simplification.SimplifyBytesSplitN{},
		&simplification.SimplifyRedundantBytesTrimSpace{},
		&simplification.SimplifySplitN{},
		&simplification.UseBytesReplaceAll{},
		&simplification.PreferBytesContainsAny{},
		&simplification.PreferStringsContainsAny{},
		&simplification.PreferStringsContainsRune{},
		&simplification.UseHttpNewRequestWithContext{},
		&simplification.PreferOsIsTimeout{},
		&simplification.SimplifyErrorsIsNil{},
		&simplification.PreferMinMaxBuiltin{},
		&simplification.PreferBytesContainsRune{},
		&simplification.SimplifyBytesEqualNil{},
		&simplification.SimplifyIfReturnBool{},

		// Redundancy
		&redundancy.RemoveRedundantReturn{},
		&redundancy.RemoveRedundantBreak{},
		&redundancy.RemoveRedundantBreakInSelect{},
		&redundancy.RemoveRedundantSprintf{},
		&redundancy.RemoveRedundantTypeConversion{},
		&redundancy.UseDocumentedBlankImport{},
		&redundancy.RemoveEmptyDefault{},
		&redundancy.RemoveEmptySwitch{},
		&redundancy.SimplifyNilCheckBeforeClose{},
		&redundancy.RemoveRedundantElse{},
		&redundancy.SimplifyRedundantLenBeforeRange{},
		&redundancy.RemoveSelfAssignment{},
		&redundancy.RemoveUnreachableCode{},
		&redundancy.RemoveConstantCondition{},
		&redundancy.RemoveEmptyLoop{},
		&redundancy.FindEmptyFmtSprintf{},
		&redundancy.SimplifyGoroutineClosure{},
		&redundancy.RemoveRedundantInterfaceAssertion{},
		&redundancy.UseMeaningfulReturnValues{},
		&redundancy.RemoveDoubleDeref{},

		// Style
		&style.UseErrorsNewForSimpleErrors{},
		&style.PreferStringsContains{},
		&style.PreferBytesContains{},
		&style.AddExportedFuncComment{},
		&style.PreferStringsEqualFold{},
		&style.PreferStringsEqualFoldSingle{},
		&style.PreferRegexpMustCompile{},
		&style.AvoidInitFunction{},
		&style.AvoidGlobalVariable{},
		&style.PreferRawStringForRegex{},
		&style.UseCryptoRand{},
		&style.AvoidDotImport{},
		&style.PreferHexEncoding{},
		&style.PreferStrconvQuote{},
		&style.WrapErrorBeforeReturn{},
		&style.AuditChannelClose{},
		&style.AuditContextBackground{},
		&style.ResolveContextTodo{},
		&style.AvoidContextWithValue{},
		&style.ReduceNestingDepth{},
		&style.FindDeprecatedAtomicFunctions{},
		&style.RemoveEmptyFunction{},
		&style.RemoveEmptyGoroutine{},
		&style.AvoidEmptyInterfaceParam{},
		&style.AuditExecCommand{},
		&style.AuditGoroutineClosure{},
		&style.AvoidHardcodedCredentials{},
		&style.UseCustomHttpClient{},
		&style.UseHttpServerWithTimeout{},
		&style.AuditHttpRedirect{},
		&style.UseTlsForHttp{},
		&style.AuditJsonNumber{},
		&style.AuditJsonRawMessage{},
		&style.KeepInterfacesSmall{},
		&style.KeepFunctionsShort{},
		&style.UseNamedConstant{},
		&style.LimitFunctionParameters{},
		&style.LimitReturnValues{},
		&style.ReduceErrorCheckNesting{},
		&style.EnsureFileClosed{},
		&style.UseSkipWithReason{},
		&style.AvoidSqlStringConcat{},
		&style.CheckTemplateExecuteError{},
		&style.AuditTestFatal{},
		&style.AuditTestMain{},
		&style.AvoidTimeSleep{},
		&style.UseCommaOkTypeAssertion{},
		&style.UseBufferedChannel{},
		&style.AvoidUnsafePackage{},
		&style.PreferMakeForEmptyMap{},
		&style.EnsureSqlConnectionClosed{},
		&style.AuditYamlUnmarshal{},
		&style.AvoidFormatStringVariable{},
		&style.FindMapRangeClear{},
		&style.AvoidNestedGoroutine{},
		&style.RemoveDebugPrint{},
		&style.SimplifySelectDefaultOnly{},

		// Error handling
		&errorhandling.PreferErrorsIsOverEquality{},
		&errorhandling.HandleErrorReturn{},
		&errorhandling.WrapErrorWithContext{},
		&errorhandling.AvoidPanic{},
		&errorhandling.HandleCheckedError{},
		&errorhandling.CheckCloseError{},
		&errorhandling.HandleDeferredCloseError{},
		&errorhandling.UseErrorsIsOverStringComparison{},
		&errorhandling.UseErrorsAs{},
		&errorhandling.AvoidLogFatal{},
		&errorhandling.AuditMultipleErrorWraps{},
		&errorhandling.AvoidOsExit{},
		&errorhandling.AuditRecover{},
		&errorhandling.PreferErrorfWrapVerb{},
		&errorhandling.SimplifyRedundantErrorWrap{},
		&errorhandling.UsePackageLevelErrorSentinel{},
		&errorhandling.PreferErrorsIsContext{},
		&errorhandling.PreferErrorsIsEOF{},
		&errorhandling.PreferErrorsIsForFieldAccess{},
		&errorhandling.UseErrorMethod{},
		&errorhandling.CheckContextError{},
		&errorhandling.AuditMustFunction{},
		&errorhandling.HandleSwallowedError{},

		// Performance
		&performance.PreallocateSlice{},
		&performance.PreferStrconvItoa{},
		&performance.PreferStrconvFormatBool{},
		&performance.AvoidDeferInLoop{},
		&performance.ReuseJsonCodecInLoop{},
		&performance.AllocateMapOutsideLoop{},
		&performance.AllocateOutsideLoop{},
		&performance.AvoidReflection{},
		&performance.CompileRegexOutsideLoop{},
		&performance.UseStringsBuilderInLoop{},
		&performance.PreferBytesBufferString{},
		&performance.SimplifySprintfChar{},
		&performance.CreateChannelOutsideLoop{},
		&performance.AvoidFmtInLoop{},
		&performance.LimitGoroutinesInLoop{},
		&performance.AvoidLockInLoop{},

		// Naming
		&naming.UseErrPrefixForErrors{},
		&naming.RemovePackagePrefixFromName{},
		&naming.UseMixedCaps{},
		&naming.UseMixedCapsForConstants{},
		&naming.UseCtxForContextParam{},
		&naming.RemoveGetterPrefix{},
		&naming.UseShortReceiverName{},
		&naming.UseDescriptiveVarNames{},
		&naming.UseDescriptivePackageName{},
	}
}

// TestParseRealRepos validates that we can parse real Go files from the
// working set without crashing, and that recipes run without panicking.
func TestParseRealRepos(t *testing.T) {
	// Resolve the working set directory relative to the module root.
	// The test runs from tests/, and the working set
	// is at the repo root: recipes-go/working-set-code-quality/
	workingSet := filepath.Join("..", "working-set-code-quality")
	if _, err := os.Stat(workingSet); os.IsNotExist(err) {
		t.Skip("working-set-code-quality not found; run `git clone` to populate")
	}

	repos := []string{"gorilla/mux", "spf13/cobra", "sirupsen/logrus", "go-chi/chi", "labstack/echo"}
	p := parser.NewGoParser()
	recipes := allRecipes()

	for _, repo := range repos {
		repoDir := filepath.Join(workingSet, repo)
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			t.Logf("Skipping %s (not cloned)", repo)
			continue
		}

		t.Run(repo, func(t *testing.T) {
			var goFiles []string
			err := filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // skip errors
				}
				if info.IsDir() && (info.Name() == "vendor" || info.Name() == "testdata" || info.Name() == ".git") {
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

			t.Logf("Found %d .go files in %s", len(goFiles), repo)

			var parseOK, parseFail int
			var spaceIssues int
			recipeFindings := make(map[string]int) // recipe name -> findings count

			for _, goFile := range goFiles {
				src, err := os.ReadFile(goFile)
				if err != nil {
					continue
				}

				relPath, _ := filepath.Rel(repoDir, goFile)

				// Parse
				cu, err := p.Parse(relPath, string(src))
				if err != nil {
					parseFail++
					if parseFail <= 5 {
						t.Logf("  PARSE FAIL: %s: %v", relPath, err)
					}
					continue
				}
				parseOK++

				// Check parse-print idempotence
				printed := printer.Print(cu)
				if printed != string(src) {
					if parseFail+1 <= 3 {
						t.Logf("  IDEMPOTENCE FAIL: %s", relPath)
					}
					parseFail++
					continue
				}

				// Space validation
				if errs := test.ValidateSpaces(cu); len(errs) > 0 {
					spaceIssues += len(errs)
					for _, e := range errs {
						t.Logf("  SPACE: %s: %s", relPath, e)
					}
				}

				// Run each recipe
				for _, r := range recipes {
					editor := r.Editor()
					if editor == nil {
						continue
					}
					func() {
						defer func() {
							if rec := recover(); rec != nil {
								t.Errorf("  PANIC in %s on %s: %v", r.Name(), relPath, rec)
							}
						}()

						ctx := recipe.NewExecutionContext()
						result := editor.Visit(cu, ctx)
						if result == nil {
							return
						}

						after := printer.Print(result)
						if after != string(src) {
							recipeFindings[r.DisplayName()]++

							// Verify the modified output is still parseable
							cu2, err2 := p.Parse(relPath, after)
							if err2 != nil {
								t.Errorf("  CORRUPT OUTPUT: %s produced unparseable output on %s: %v",
									r.DisplayName(), relPath, err2)
							} else {
								// Verify round-trip of modified output
								reprinted := printer.Print(cu2)
								if reprinted != after {
									t.Errorf("  ROUND-TRIP FAIL: %s output on %s is not idempotent",
										r.DisplayName(), relPath)
								}
							}
						}

						// Check search results via marker printing
						markerOutput := printer.PrintWithMarkers(result, printer.DefaultMarkerPrinter)
						if markerOutput != string(src) && after == string(src) {
							// Search-only recipe found something
							recipeFindings[r.DisplayName()]++
						}
					}()
				}
			}

			t.Logf("  Parse: %d OK, %d fail/idempotence issues", parseOK, parseFail)
			t.Logf("  Space validation issues: %d", spaceIssues)
			t.Logf("  Recipe findings:")
			totalFindings := 0
			for name, count := range recipeFindings {
				t.Logf("    %s: %d", name, count)
				totalFindings += count
			}
			if totalFindings == 0 {
				t.Logf("    (none)")
			}
			fmt.Printf("\n[%s] Parse: %d OK, %d fail | Findings: %d\n", repo, parseOK, parseFail, totalFindings)
		})
	}
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

	// These produce code a customer would have to undo by hand.
	"org.openrewrite.golang.codequality.AvoidFallthrough":     "deleting fallthrough changes what the switch does, and still compiles",
	"org.openrewrite.golang.codequality.EnsureHttpBodyClosed": "matches any method named Do, so `err := retry.Do(f)` gains `defer err.Body.Close()`",
	"org.openrewrite.golang.codequality.EnsureSqlRowsClosed":  "matches any method named Query, so `q := r.URL.Query()` gains `defer q.Close()`",
	"org.openrewrite.golang.codequality.EnsureTimerStopped":   "`defer t.Stop()` on a time.AfterFunc timer cancels the callback it scheduled",
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
