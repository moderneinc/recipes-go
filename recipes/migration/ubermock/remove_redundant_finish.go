/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package ubermock

import (
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Drops `defer ctrl.Finish()` where the controller finishes itself.
type RemoveRedundantGomockFinish struct {
	recipe.Base
}

func (r *RemoveRedundantGomockFinish) Name() string {
	return "org.openrewrite.golang.migration.RemoveRedundantGomockFinish"
}
func (r *RemoveRedundantGomockFinish) DisplayName() string {
	return "Remove redundant `defer ctrl.Finish()`"
}
func (r *RemoveRedundantGomockFinish) Description() string {
	return "Remove `defer ctrl.Finish()` where the controller was built by `gomock.NewController` from a `*testing.T`, `*testing.B`, `*testing.F` or `testing.TB`. `NewController` registers the finish through `Cleanup` for any such reporter, so the deferred call only repeats it."
}
func (r *RemoveRedundantGomockFinish) Tags() []string {
	return []string{"migration", "mock", "testing", "cleanup"}
}

func (r *RemoveRedundantGomockFinish) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeFinishVisitor{})
}

type removeFinishVisitor struct {
	visitor.GoVisitor
	// gomockPkg is the local name the file binds to a gomock import, empty when
	// the file imports neither fork.
	gomockPkg string
	// controllers are the variables the enclosing function assigned from
	// gomock.NewController with a self-finishing reporter.
	controllers map[string]bool
}

func (v *removeFinishVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.gomockPkg = gomockQualifier(cu)
	if v.gomockPkg == "" {
		return cu
	}
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

// gomockQualifier returns the local name a file binds to either fork's gomock
// package. Both forks are accepted: golang/mock has registered the cleanup since
// v1.5.0, so the deferred call is redundant before the path swap as well.
func gomockQualifier(cu *golang.CompilationUnit) string {
	for _, path := range []string{newGomock, oldGomock} {
		if name := pathswap.Qualifier(cu, path); name != "" {
			return name
		}
	}
	return ""
}

// The controller set is rebuilt per function, so a name that means a controller
// in one test does not license dropping a Finish call on an unrelated value of
// the same name in another. A function literal is a method declaration too and
// closes over its enclosing function, so the outer set carries inwards.
func (v *removeFinishVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	outer := v.controllers
	v.controllers = union(outer, v.collectControllers(md))
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)
	v.controllers = outer
	return md
}

func (v *removeFinishVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	if len(v.controllers) == 0 {
		return block
	}

	kept := make([]java.RightPadded[java.Statement], 0, len(block.Statements))
	dropped := false
	for _, rp := range block.Statements {
		if d, ok := rp.Element.(*golang.Defer); ok && v.isRedundantFinish(d) {
			dropped = true
			continue
		}
		kept = append(kept, rp)
	}
	if !dropped {
		return block
	}
	return block.WithStatements(kept)
}

// isRedundantFinish reports whether d is `defer <controller>.Finish()`.
func (v *removeFinishVisitor) isRedundantFinish(d *golang.Defer) bool {
	mi, ok := d.Expr.(*java.MethodInvocation)
	if !ok || mi.Name == nil || mi.Name.Name != "Finish" || mi.Select == nil {
		return false
	}
	if realArgCount(mi) != 0 {
		return false
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	return ok && v.controllers[recv.Name]
}

// collectControllers returns the names md assigns from gomock.NewController with
// a reporter that finishes itself.
func (v *removeFinishVisitor) collectControllers(md *java.MethodDeclaration) map[string]bool {
	scan := visitor.Init(&controllerScan{gomockPkg: v.gomockPkg, found: map[string]bool{}})
	scan.Visit(md, nil)
	return scan.found
}

type controllerScan struct {
	visitor.GoVisitor
	gomockPkg string
	found     map[string]bool
}

func (s *controllerScan) VisitAssignment(a *java.Assignment, p any) java.J {
	a = s.GoVisitor.VisitAssignment(a, p).(*java.Assignment)
	name, ok := a.Variable.(*java.Identifier)
	if !ok {
		return a
	}
	if s.isSelfFinishingController(a.Value.Element) {
		s.found[name.Name] = true
	}
	return a
}

// isSelfFinishingController reports whether expr is a gomock.NewController call
// whose sole argument is a testing reporter carrying a Cleanup method. Any other
// reporter — a custom TestReporter, or a testify suite's s.T() whose type the
// parser could not resolve — leaves the deferred Finish doing real work.
func (s *controllerScan) isSelfFinishingController(expr java.Expression) bool {
	mi, ok := expr.(*java.MethodInvocation)
	if !ok || mi.Name == nil || mi.Name.Name != "NewController" || mi.Select == nil {
		return false
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	if !ok || recv.Name != s.gomockPkg {
		return false
	}
	args := realArgs(mi)
	if len(args) != 1 {
		return false
	}
	return isTestingReporter(args[0])
}

// isTestingReporter reports whether expr is a testing type whose Cleanup method
// NewController hooks: *testing.T, *testing.B, *testing.F or a testing.TB.
func isTestingReporter(expr java.Expression) bool {
	switch matcher.GetFullyQualifiedName(matcher.TypeOfExpression(expr)) {
	case "testing.T", "testing.B", "testing.F", "testing.TB":
		return true
	}
	return isSuiteT(expr)
}

// isSuiteT reports whether expr is a bare `x.T()`, which in a file building a
// gomock controller means a testify suite handing over the running *testing.T —
// by far the most common way a controller is built outside a plain test
// function. The receiver's type is unresolvable wherever testify is not on the
// parse classpath, so the call is recognised by shape; a `T()` returning some
// other TestReporter would at worst leave a missing expectation reported at
// cleanup rather than aborting the test.
func isSuiteT(expr java.Expression) bool {
	mi, ok := expr.(*java.MethodInvocation)
	if !ok || mi.Name == nil || mi.Name.Name != "T" || mi.Select == nil {
		return false
	}
	return realArgCount(mi) == 0
}

// union returns the names in either set.
func union(a, b map[string]bool) map[string]bool {
	out := make(map[string]bool, len(a)+len(b))
	for k := range a {
		out[k] = true
	}
	for k := range b {
		out[k] = true
	}
	return out
}

// realArgs returns the arguments of mi, skipping the Empty sentinel an empty
// argument list carries.
func realArgs(mi *java.MethodInvocation) []java.Expression {
	var out []java.Expression
	for _, rp := range mi.Arguments.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		out = append(out, rp.Element)
	}
	return out
}

func realArgCount(mi *java.MethodInvocation) int { return len(realArgs(mi)) }
