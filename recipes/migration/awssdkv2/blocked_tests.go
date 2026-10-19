/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"path"
	"sort"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// A test can exercise v1 itself — its retryer's internals, a request built by
// hand — where there is no v2 counterpart to test the same way. Holding its
// package back for it would leave the code under test on v1, so where the rest
// of the package migrates the test is commented out instead, with what held it
// back, for a person to port. `go build` never reads it; `go test` runs the
// package without it until then.

// decideBlockedTests settles each test the rewrite could not carry over: held
// back with its package where the package stays on v1 anyway, or where another
// file uses a name it declares; otherwise commented out.
func (a *migrateAcc) decideBlockedTests() {
	paths := make([]string, 0, len(a.blockedTests))
	for p := range a.blockedTests {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, test := range paths {
		reason := a.blockedTests[test]
		dir := path.Dir(test)
		if _, held := a.heldBack[dir]; held || a.helpsAnotherFile(test) {
			a.holdBack(test, reason)
			continue
		}
		a.disabledTests[test] = reason
	}
	// A test held back holds the package back, which every other blocked test
	// in it then follows.
	for test := range a.disabledTests {
		if _, held := a.heldBack[path.Dir(test)]; held {
			delete(a.disabledTests, test)
			a.holdBack(test, a.blockedTests[test])
		}
	}
	for test, reason := range a.disabledTests {
		a.blockerReason(test, "test commented out, to port by hand: "+reason)
	}
}

// helpsAnotherFile reports whether another file of test's directory uses a name
// test declares beyond its test functions.
func (a *migrateAcc) helpsAnotherFile(test string) bool {
	for name := range a.fileDecls[test] {
		if isTestFunc(name) {
			continue
		}
		for file, idents := range a.fileIdents {
			if file != test && path.Dir(file) == path.Dir(test) && idents[name] {
				return true
			}
		}
	}
	return false
}

func isTestFunc(name string) bool {
	for _, prefix := range []string{"Test", "Benchmark", "Example", "Fuzz"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// usedNames returns every name cu uses.
func usedNames(cu *golang.CompilationUnit) map[string]bool {
	scan := visitor.Init(&usedNameScan{names: map[string]bool{}})
	scan.Visit(cu, nil)
	return scan.names
}

type usedNameScan struct {
	visitor.GoVisitor
	names map[string]bool
}

func (v *usedNameScan) VisitIdentifier(id *java.Identifier, p any) java.J {
	v.names[id.Name] = true
	return id
}

// commentedOutTest keeps the package clause and comments out the rest, under
// a note saying what held it back.
func commentedOutTest(cu *golang.CompilationUnit, reason string) *golang.CompilationUnit {
	source := printer.Print(cu)
	clause := "package " + cu.PackageDecl.Element.Name
	body := source
	if i := strings.Index(source, clause); i >= 0 {
		body = source[i+len(clause):]
	}
	var b strings.Builder
	b.WriteString(clause + "\n\n")
	b.WriteString("// MigrateAwsSdkGoToV2 commented this test out: " + strings.TrimSuffix(reason, ".") + ".\n")
	b.WriteString("// It tests aws-sdk-go v1 in a way v2 has no counterpart for; port it to the\n")
	b.WriteString("// migrated code and restore it.\n//\n")
	for _, line := range strings.Split(strings.Trim(body, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			b.WriteString("//\n")
			continue
		}
		b.WriteString("// " + line + "\n")
	}
	parsed, err := parser.NewGoParser().Parse(cu.SourcePath, b.String())
	if err != nil {
		return cu
	}
	c := *cu
	c.PackageDecl = parsed.PackageDecl
	c.Imports = parsed.Imports
	c.Statements = parsed.Statements
	c.EOF = parsed.EOF
	return &c
}

// recordAdjunct notes the packages a changed adjunct file imports, among
// which are the module's own that hand it SDK values. The module path may not
// be known yet, so they are resolved once every file has been seen.
func (a *migrateAcc) recordAdjunct(cu *golang.CompilationUnit) {
	var imports []string
	if cu.Imports != nil {
		for _, rp := range cu.Imports.Elements {
			imports = append(imports, rp.Element.Path())
		}
	}
	a.adjuncts[cu.SourcePath] = imports
}

// holdBackStrandedAdjuncts holds back an adjunct file reading values from a
// module package that stays on v1, which hands it v1's shapes still. Holding
// one back can strand another, so it repeats until nothing changes.
func (a *migrateAcc) holdBackStrandedAdjuncts() {
	if a.modulePath == "" {
		return
	}
	for changed := true; changed; {
		changed = false
		for _, file := range sortedKeys(a.adjuncts) {
			if _, held := a.heldBack[path.Dir(file)]; held {
				continue
			}
			for _, imported := range a.adjuncts[file] {
				dir, inModule := strings.CutPrefix(imported, a.modulePath+"/")
				if _, held := a.heldBack[dir]; inModule && held {
					a.holdBack(file, "reads SDK values from "+dir+", which stays on v1")
					changed = true
					break
				}
			}
		}
	}
}
