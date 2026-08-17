/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify

import (
	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AdoptTestify migrates hand-written test assertions to stretchr/testify and adds
// the dependency to go.mod, in a single recipe.
//
// It is deliberately NOT a composition of the individual recipes: `MaybeAddImport`
// additions are dropped when recipes are chained in a RecipeList under the CLI, so
// a composite would emit `require.NoError(...)` with no import. Instead one
// scanning recipe dispatches every pattern matcher in one edit pass (where the
// import add materialises) and adds the go.mod require in the same pass.
//
// Where conditions overlap the matchers are ordered so the most specific wins:
// Len before Equal (so `len(x) != n` becomes `Len`, not `Equal`); the NoError /
// Error and Nil matchers are disjoint (error vs non-error operands) and True /
// False skips comparisons.
//
// It does not sync go.sum or add testify's transitive dependencies (those need
// module hashes that cannot be computed offline), so a `go mod tidy` /
// `go mod download` is still needed to complete resolution.
type AdoptTestify struct {
	recipe.ScanningBase
}

func (r *AdoptTestify) Name() string { return "org.openrewrite.golang.testify.AdoptTestify" }

func (r *AdoptTestify) DisplayName() string { return "Adopt stretchr/testify" }

func (r *AdoptTestify) Description() string {
	return "Migrate hand-written test assertions to the `github.com/stretchr/testify` library and add the dependency to go.mod. " +
		"Converts error guards to `require`/`assert` `NoError`/`Error`, length checks to `Len`, equality checks to `Equal`/`NotEqual`, nil checks to `Nil`/`NotNil`, and boolean checks to `True`/`False`, then adds the testify require. " +
		"Does not sync go.sum; a `go mod tidy` is still needed to complete resolution."
}

func (r *AdoptTestify) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestify) InitialValue(ctx *recipe.ExecutionContext) any {
	return &adoptTestifyAcc{}
}

func (r *AdoptTestify) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&adoptTestifyScanner{acc: acc.(*adoptTestifyAcc)})
}

func (r *AdoptTestify) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&adoptTestifyEditor{acc: acc.(*adoptTestifyAcc)})
}

// ifMatchers dispatches an `if` guard through every concept matcher, first
// non-nil wins. Both reporter flavours (fatal -> require, non-fatal -> assert)
// are tried per concept. Len precedes Equal since both match `len(x) != n`.
var ifMatchers = []func(*java.If) *java.MethodInvocation{
	lenReq.matchLen, lenAssert.matchLen,
	eqReq.matchEquality, eqAssert.matchEquality,
	nilReq.matchNil, nilAssert.matchNil,
	noErrReq.matchNilGuard, noErrAssert.matchNilGuard,
	errReq.matchNilGuard, errAssert.matchNilGuard,
	boolReq.matchBool, boolAssert.matchBool,
}

var initMatchers = []func(*golang.StatementWithInit) *java.MethodInvocation{
	noErrReq.matchInlineInitGuard, noErrAssert.matchInlineInitGuard,
	errReq.matchInlineInitGuard, errAssert.matchInlineInitGuard,
}

// Stateless per-concept matcher configs; only their guardBase / errGuardVisitor
// config fields are read by the match methods (never the embedded visitor).
var (
	lenReq      = &lenVisitor{guardBase{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter}}
	lenAssert   = &lenVisitor{guardBase{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter}}
	nilReq      = &nilVisitor{guardBase{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter}}
	nilAssert   = &nilVisitor{guardBase{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter}}
	boolReq     = &boolVisitor{guardBase{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter}}
	boolAssert  = &boolVisitor{guardBase{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter}}
	eqReq       = &equalityVisitor{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter}
	eqAssert    = &equalityVisitor{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter}
	noErrReq    = &errGuardVisitor{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter, op: java.NotEqual, assertion: "NoError"}
	noErrAssert = &errGuardVisitor{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter, op: java.NotEqual, assertion: "NoError"}
	errReq      = &errGuardVisitor{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter, op: java.Equal, assertion: "Error"}
	errAssert   = &errGuardVisitor{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter, op: java.Equal, assertion: "Error"}
)

// dispatchTestify returns the assertion call that replaces stmt, or nil.
func dispatchTestify(stmt java.Statement) *java.MethodInvocation {
	switch el := stmt.(type) {
	case *java.If:
		for _, m := range ifMatchers {
			if c := m(el); c != nil {
				return c
			}
		}
	case *golang.StatementWithInit:
		for _, m := range initMatchers {
			if c := m(el); c != nil {
				return c
			}
		}
	}
	return nil
}

// callPackage returns the qualifier ("require" / "assert") of a built assertion
// call, used to decide which import to add.
func callPackage(call *java.MethodInvocation) string {
	if call.Select == nil {
		return ""
	}
	if id, ok := call.Select.Element.(*java.Identifier); ok {
		return id.Name
	}
	return ""
}

// adoptTestifyAcc records whether the module will import testify (so the go.mod
// require is added only when a conversion actually happens).
type adoptTestifyAcc struct {
	used bool
}

type adoptTestifyScanner struct {
	visitor.GoVisitor
	acc *adoptTestifyAcc
}

func (v *adoptTestifyScanner) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	if v.acc.used {
		return block
	}
	for _, rp := range block.Statements {
		if dispatchTestify(rp.Element) != nil {
			v.acc.used = true
			break
		}
	}
	return block
}

type adoptTestifyEditor struct {
	visitor.GoVisitor
	acc *adoptTestifyAcc
}

func (v *adoptTestifyEditor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	newStmts := make([]java.RightPadded[java.Statement], 0, len(block.Statements))
	pkgs := map[string]bool{}
	for _, rp := range block.Statements {
		call := dispatchTestify(rp.Element)
		if call == nil {
			newStmts = append(newStmts, rp)
			continue
		}
		pkgs[callPackage(call)] = true
		newStmts = append(newStmts, java.RightPadded[java.Statement]{
			Element: call,
			After:   rp.After,
			Markers: rp.Markers,
		})
	}

	if len(pkgs) == 0 {
		return block
	}
	if pkgs[requirePkg] {
		recipegolang.MaybeAddImport(v, requireImport, nil, false)
	}
	if pkgs[assertPkg] {
		recipegolang.MaybeAddImport(v, assertImport, nil, false)
	}
	return block.WithStatements(newStmts)
}

func (v *adoptTestifyEditor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	if !v.acc.used {
		return gm
	}
	return migration.AddRequire(gm, testifyModulePath, testifyVersion, false)
}
