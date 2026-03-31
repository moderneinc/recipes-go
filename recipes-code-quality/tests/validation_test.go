/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/parser"
	"github.com/openrewrite/rewrite/pkg/printer"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/test"
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
		&simplification.FindChannelLenCheck{},
		&simplification.FindRedundantRangeBlank{},
		&simplification.FindSingleCaseSelect{},
		&simplification.FindStdLog{},
		&simplification.FindSwitchTrue{},
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
		&redundancy.FindBlankImport{},
		&redundancy.FindEmptyDefault{},
		&redundancy.FindEmptySwitch{},
		&redundancy.FindNilCheckBeforeClose{},
		&redundancy.FindRedundantElse{},
		&redundancy.FindRedundantLenBeforeRange{},
		&redundancy.FindSelfAssignment{},
		&redundancy.FindUnreachableCode{},
		&redundancy.FindConstantCondition{},
		&redundancy.FindEmptyLoop{},
		&redundancy.FindEmptyFmtSprintf{},
		&redundancy.FindRedundantGoroutineClosure{},

		// Style
		&style.UseErrorsNewForSimpleErrors{},
		&style.PreferStringsContains{},
		&style.PreferBytesContains{},
		&style.ExportedFuncShouldHaveComment{},
		&style.PreferStringsEqualFold{},
		&style.PreferStringsEqualFoldSingle{},
		&style.PreferRegexpMustCompile{},
		&style.FindInitFunction{},
		&style.FindGlobalVar{},
		&style.PreferRawStringForRegex{},
		&style.FindMathRand{},
		&style.FindDotImport{},
		&style.PreferHexEncoding{},
		&style.PreferStrconvQuote{},
		&style.FindBareReturnNilError{},
		&style.FindChannelClose{},
		&style.FindContextBackground{},
		&style.FindContextTodo{},
		&style.FindContextWithValue{},
		&style.FindDeeplyNestedBlock{},
		&style.FindDeprecatedAtomic{},
		&style.FindEmptyFunction{},
		&style.FindEmptyGoroutine{},
		&style.FindEmptyInterfaceParam{},
		&style.FindExecCommand{},
		&style.FindGoroutineClosure{},
		&style.FindHardcodedCredentials{},
		&style.FindHttpDefaultClient{},
		&style.FindHTTPListenAndServe{},
		&style.FindHttpRedirect{},
		&style.FindHttpListenAndServe{},
		&style.FindJsonNumber{},
		&style.FindJsonRawMessage{},
		&style.FindLargeInterface{},
		&style.FindLongFunction{},
		&style.FindMagicNumber{},
		&style.FindManyParams{},
		&style.FindManyReturns{},
		&style.FindDeeplyNestedErrorCheck{},
		&style.FindOsFileOpen{},
		&style.FindSkipWithoutReason{},
		&style.FindSQLStringConcat{},
		&style.FindTemplateExecute{},
		&style.FindTestFatal{},
		&style.FindTestMain{},
		&style.FindTimeSleep{},
		&style.FindTypeAssertionWithoutOk{},
		&style.FindUnbufferedChannel{},
		&style.FindUnsafeUsage{},
		&style.PreferMakeForEmptyMap{},
		&style.FindHttpNewRequest{},
		&style.FindSqlOpen{},
		&style.FindYamlUnmarshal{},
		&style.FindFormatStringVariable{},
		&style.FindMapRangeClear{},
		&style.FindNestedGoroutine{},
		&style.FindDebugPrint{},
		&style.FindSelectDefaultOnly{},

		// Error handling
		&errorhandling.PreferErrorsIsOverEquality{},
		&errorhandling.CheckErrorReturn{},
		&errorhandling.WrapErrorWithContext{},
		&errorhandling.FindPanic{},
		&errorhandling.FindEmptyErrorCheck{},
		&errorhandling.FindCloseCall{},
		&errorhandling.FindDeferredClose{},
		&errorhandling.FindErrorStringComparison{},
		&errorhandling.FindErrorTypeAssertion{},
		&errorhandling.FindLogFatal{},
		&errorhandling.FindMultipleErrorWraps{},
		&errorhandling.FindOsExit{},
		&errorhandling.FindRecover{},
		&errorhandling.PreferErrorfWrapVerb{},
		&errorhandling.SimplifyRedundantErrorWrap{},
		&errorhandling.FindErrorsNew{},
		&errorhandling.PreferErrorsIsContext{},
		&errorhandling.PreferErrorsIsEOF{},
		&errorhandling.PreferErrorsIsForFieldAccess{},
		&errorhandling.FindErrorFmtSprint{},
		&errorhandling.FindContextErr{},
		&errorhandling.FindMustFunction{},
		&errorhandling.FindSwallowedError{},

		// Performance
		&performance.PreallocateSlice{},
		&performance.PreferStrconvItoa{},
		&performance.PreferStrconvFormatBool{},
		&performance.FindDeferInLoop{},
		&performance.FindJsonInLoop{},
		&performance.FindMapAllocInLoop{},
		&performance.FindNewInLoop{},
		&performance.FindReflection{},
		&performance.FindRegexInLoop{},
		&performance.FindStringConcatInLoop{},
		&performance.PreferBytesBufferString{},
		&performance.SimplifySprintfChar{},
		&performance.FindChannelCreateInLoop{},
		&performance.FindFmtInLoop{},
		&performance.FindGoroutineInLoop{},
		&performance.FindLockInLoop{},

		// Naming
		&naming.FindMisnamedErrorVar{},
		&naming.FindStutteringName{},
		&naming.FindUnderscoredExportedName{},
		&naming.FindAllCapsConstant{},
		&naming.FindContextParamNotCtx{},
		&naming.FindGetterPrefix{},
		&naming.FindLongReceiverName{},
		&naming.FindSingleLetterVar{},
	}
}

// TestParseRealRepos validates that we can parse real Go files from the
// working set without crashing, and that recipes run without panicking.
func TestParseRealRepos(t *testing.T) {
	// Resolve the working set directory relative to the module root.
	// The test runs from recipes-code-quality/tests/, and the working set
	// is at the repo root: recipes-go/working-set-code-quality/
	workingSet := filepath.Join("..", "..", "working-set-code-quality")
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
