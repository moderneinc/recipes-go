/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
	"github.com/stretchr/testify/require"
)

// compileEnv gates the tests that migrate a module on disk and build the result
// with the real toolchain. They fetch modules, so they are off by default.
const compileEnv = "COMPILE_TESTS"

func requireCompile(t *testing.T) {
	t.Helper()
	if os.Getenv(compileEnv) == "" {
		t.Skipf("set %s=1 to run the tests that build migrated modules with the go toolchain", compileEnv)
	}
}

// writeModule lays files out under a fresh directory and tidies it, so go.mod
// and go.sum are the toolchain's own rather than hand-written.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(strings.TrimLeft(content, "\n")), 0o644))
	}
	goCmd(t, dir, "mod", "tidy")
	return dir
}

func goCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := runGo(dir, args...)
	require.NoErrorf(t, err, "go %s:\n%s", strings.Join(args, " "), out)
	return out
}

func runGo(dir string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=", "GOWORK=off")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// migrateOnDisk runs the migration over a module the way `mod run` would see
// it — the build list resolved by the toolchain, every package parsed together —
// and writes the results back.
func migrateOnDisk(t *testing.T, dir string, r recipe.Recipe) []awssdkv2.AwsSdkGoV1BlockerRow {
	t.Helper()
	modContent, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	mrr, err := parser.ParseGoMod("go.mod", string(modContent))
	require.NoError(t, err)
	sumContent, _ := os.ReadFile(filepath.Join(dir, "go.sum"))
	// As the RPC server's ParseProject resolves it for `mod build`.
	listed, pkgs, err := parser.ResolveModuleGraph(dir)
	require.NoError(t, err)
	mrr.ResolvedDependencies = parser.MergeResolvedDependencies(parser.ParseGoSum(string(sumContent)), listed)
	if pkgs.Incomplete {
		mrr.ResolutionStatus = golang.GoResolutionIncomplete
	} else {
		mrr.ResolutionStatus = golang.GoResolutionResolved
		mrr.PackageModules = pkgs.Packages
	}
	project := golang.NewGoProject(filepath.Base(dir), mrr.ModulePath)

	pi := parser.NewProjectImporter(mrr.ModulePath, nil)
	pi.SetProjectRoot(dir)
	for _, dep := range mrr.ResolvedDependencies {
		if !dep.Main {
			pi.AddRequire(dep.ModulePath)
			pi.AddModule(dep.ModulePath, "", dep.Version)
		}
	}
	byDir := map[string][]parser.FileInput{}
	require.NoError(t, filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		content, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		pi.AddSource(rel, string(content))
		byDir[filepath.Dir(rel)] = append(byDir[filepath.Dir(rel)], parser.FileInput{Path: rel, Content: string(content)})
		return nil
	}))

	p := parser.NewGoParser()
	p.Importer = pi
	var trees []java.Tree
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		cus, err := p.ParsePackage(byDir[d])
		require.NoError(t, err)
		for _, cu := range cus {
			cu.Markers = java.AddMarker(cu.Markers, project)
			trees = append(trees, cu)
		}
	}
	gm, err := parser.ParseGoModFile("go.mod", string(modContent))
	require.NoError(t, err)
	gm = gm.WithMarkers(java.AddMarker(java.AddMarker(gm.Markers, *mrr), project))
	trees = append(trees, gm)
	if len(sumContent) > 0 {
		gs, err := parser.ParseGoSumFile("go.sum", string(sumContent))
		require.NoError(t, err)
		trees = append(trees, gs.WithMarkers(java.AddMarker(gs.Markers, project)))
	}

	ctx := recipe.NewExecutionContext()
	for _, tr := range runLifecycle(ctx, r, trees) {
		path, content := printSource(tr)
		out := filepath.Join(dir, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(out), 0o755))
		require.NoError(t, os.WriteFile(out, []byte(content), 0o644))
	}
	var blockers []awssdkv2.AwsSdkGoV1BlockerRow
	if store, ok := ctx.GetMessage(recipe.DataTableStoreKey); ok {
		for _, dt := range store.(recipe.DataTableStore).GetDataTables() {
			for _, row := range store.(recipe.DataTableStore).GetRows(dt.Descriptor().Name, dt.Group()) {
				if b, isBlocker := row.(awssdkv2.AwsSdkGoV1BlockerRow); isBlocker {
					blockers = append(blockers, b)
				}
			}
		}
	}
	return blockers
}

// runLifecycle mirrors pkg/test's scanning lifecycle, returning every source
// after the run, generated ones included.
func runLifecycle(ctx *recipe.ExecutionContext, r recipe.Recipe, trees []java.Tree) []java.Tree {
	apply := func(editor recipe.TreeVisitor) {
		for i, tr := range trees {
			if res := editor.Visit(tr, ctx); res != nil {
				tr = res
			}
			trees[i] = visitor.DrainAfterVisits(editor, tr, ctx)
		}
	}
	var walk func(r recipe.Recipe)
	walk = func(r recipe.Recipe) {
		if sr, ok := r.(recipe.ScanningRecipe); ok {
			acc := sr.InitialValue(ctx)
			if scanner := sr.Scanner(acc); scanner != nil {
				for _, tr := range trees {
					scanner.Visit(tr, ctx)
				}
			}
			trees = append(trees, sr.Generate(acc, ctx)...)
			if editor := sr.EditorWithData(acc); editor != nil {
				apply(editor)
			}
		} else if editor := r.Editor(); editor != nil {
			apply(editor)
		}
		for _, sub := range r.RecipeList() {
			walk(sub)
		}
	}
	walk(r)
	return trees
}

func printSource(tr java.Tree) (string, string) {
	switch s := tr.(type) {
	case *golang.GoMod:
		return "go.mod", printer.PrintGoMod(s)
	case *golang.GoSum:
		return "go.sum", printer.PrintGoSum(s)
	case *golang.CompilationUnit:
		return s.SourcePath, printer.Print(s)
	}
	panic("unexpected source type")
}

// The shape behind `missing go.sum entry` and `undefined: dynamodb.SearchVectorsInput`:
// a dependency already requires a v2 dynamodb older than the manifest, so the
// module graph holds that version with only its go.mod hash summed.
func TestMigratedDynamoDBModuleBuilds(t *testing.T) {
	requireCompile(t)
	lib := writeModule(t, map[string]string{
		"go.mod": `
module example.com/lib

go 1.22

require github.com/aws/aws-sdk-go-v2/service/dynamodb v1.40.0
`,
		"table/table.go": `
package table

import "github.com/aws/aws-sdk-go-v2/service/dynamodb"

var _ *dynamodb.Client
`,
		"name/name.go": `
package name

func Table() string { return "items" }
`,
	})
	dir := writeModule(t, map[string]string{
		"go.mod": `
module main

go 1.22

require example.com/lib v0.0.0

replace example.com/lib => ` + lib + `
`,
		"store.go": `
package main

import (
	"fmt"

	"example.com/lib/name"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

func describe(svc dynamodbiface.DynamoDBAPI) (*dynamodb.DescribeTableOutput, error) {
	return svc.DescribeTable(&dynamodb.DescribeTableInput{TableName: aws.String(name.Table())})
}

func main() {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))
	out, err := describe(dynamodb.New(sess))
	fmt.Println(out, err)
}
`,
	})
	migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{})
	assertBuilds(t, dir)
}

// ec2AndS3Manager uses a service module and the S3 transfer manager, neither of
// which the migration pins a version for.
const ec2AndS3Manager = `
package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func main() {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))
	out, err := ec2.New(sess).DescribeInstances(&ec2.DescribeInstancesInput{})
	fmt.Println(out, err)

	f, _ := os.Open("x")
	_, err = s3manager.NewUploader(sess).Upload(&s3manager.UploadInput{
		Bucket: aws.String("b"),
		Key:    aws.String("k"),
		Body:   f,
	})
	fmt.Println(err)
}
`

// Nothing of v2 in the graph beforehand, so the pinned requires arrive without
// the indirect requires a pruned module graph needs alongside them.
func TestMigratedEc2AndS3ManagerModuleBuilds(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": `
module main

go 1.22
`,
		"main.go": ec2AndS3Manager,
	})
	migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{})
	assertBuilds(t, dir)
}

// The shape behind `no required module provides package`: the module is already
// partly on v2, so go.mod is consistent for the core modules and lacks only the
// ones the migrated imports newly need.
func TestMigratedPartlyV2ModuleBuilds(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": `
module main

go 1.24
`,
		"main.go": ec2AndS3Manager,
		"identity.go": `
package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func identity(ctx context.Context) (*sts.GetCallerIdentityOutput, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
}
`,
	})
	migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{})
	assertBuilds(t, dir)
}

// The expression builder moved to a feature module of its own, which brings
// attributevalue and dynamodbstreams into the build with it.
func TestMigratedDynamoDBExpressionModuleBuilds(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule main\n\ngo 1.22\n",
		"main.go": `
package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

func main() {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))
	filter := expression.Name("updated").GreaterThan(expression.Value(int64(10)))
	expr, err := expression.NewBuilder().WithFilter(filter).Build()
	if err != nil {
		panic(err)
	}
	out, err := dynamodb.New(sess).Scan(&dynamodb.ScanInput{
		TableName:                 aws.String("t"),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	fmt.Println(out, err)
}
`,
	})
	migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{})
	migrated, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	require.Contains(t, string(migrated), `"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"`)
	assertBuilds(t, dir)
}

// The token signer moved to feature/rds/auth, a module the migration requires
// alongside the config the credentials now come from.
func TestMigratedRdsAuthModuleBuilds(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule main\n\ngo 1.22\n",
		"main.go": `
package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
)

func main() {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))
	token, err := rdsutils.BuildAuthToken("db.example.com:5432", "us-east-1", "app", sess.Config.Credentials)
	fmt.Println(token, err)
}
`,
	})
	migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{})
	migrated, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	require.Contains(t, string(migrated), `"github.com/aws/aws-sdk-go-v2/feature/rds/auth"`)
	assertBuilds(t, dir)
}

// Every regenerated iface declares the manifest's operations, which the version
// the migration pins has to have: a manifest read off a newer release than the
// pin names operations the build cannot find.
func TestMigratedIfaceForEveryServiceBuilds(t *testing.T) {
	requireCompile(t)
	var src strings.Builder
	src.WriteString("package main\n\nimport (\n")
	for _, service := range awsmanifest.Services() {
		fmt.Fprintf(&src, "\t%q\n", "github.com/aws/aws-sdk-go/service/"+service+"/"+service+"iface")
	}
	src.WriteString(")\n\n")
	for _, service := range awsmanifest.Services() {
		client, ok := awsmanifest.ClientType(service)
		require.True(t, ok, service)
		fmt.Fprintf(&src, "var _ %siface.%sAPI\n", service, client)
	}
	src.WriteString("\nfunc main() {}\n")
	dir := writeModule(t, map[string]string{
		"go.mod":  "\nmodule main\n\ngo 1.22\n",
		"main.go": src.String(),
	})
	migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{})
	assertBuilds(t, dir)
}

// assertBuilds builds the module as it stands after the migration, then with
// go.sum filled in for the requires go.mod already lists — what a pipeline
// running RegenerateGoSum's `go mod download all` sees — and finally after
// `go mod tidy`.
func assertBuilds(t *testing.T, dir string) {
	t.Helper()
	gomod, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	t.Logf("migrated go.mod:\n%s", gomod)
	if out, err := runGo(dir, "build", "./..."); err != nil {
		t.Errorf("go build ./... as migrated:\n%s", out)
	}
	if out, err := runGo(dir, "mod", "download", "all"); err != nil {
		t.Errorf("go mod download all:\n%s", out)
	} else if out, err := runGo(dir, "build", "./..."); err != nil {
		t.Errorf("go build ./... after go mod download all:\n%s", out)
	}
	if out, err := runGo(dir, "mod", "tidy"); err != nil {
		t.Errorf("go mod tidy:\n%s", out)
		return
	}
	if out, err := runGo(dir, "build", "./..."); err != nil {
		t.Errorf("go build ./... after tidying:\n%s", out)
	}
}

// realWorldAwsRepos are the corpus repositories on aws-sdk-go v1, as fetched by
// TestParseRealRepos into build/realworld.
var realWorldAwsRepos = []string{
	"dmfutcher/git-s3-push",
	"ZipRecruiter/cloudwatching",
	"99designs/iamy",
	"awslabs/aws-cloudmap-prometheus-sd",
	"PaddleHQ/go-aws-ssm",
}

// TestMigrateRealWorldAwsRepos migrates a copy of each corpus repository on
// aws-sdk-go v1 and builds it. It reports rather than fails: a module with a
// file held back is not expected to build until that file is migrated by hand.
func TestMigrateRealWorldAwsRepos(t *testing.T) {
	requireCompile(t)
	for _, repo := range realWorldAwsRepos {
		t.Run(repo, func(t *testing.T) {
			matches, _ := filepath.Glob(filepath.Join("..", "..", "..", "build", "realworld", filepath.FromSlash(repo)+"@*"))
			if len(matches) == 0 {
				t.Skipf("%s is not fetched; run REALWORLD_REPOS=1 go test ./tests/ -run TestParseRealRepos first", repo)
			}
			dir := t.TempDir()
			out, err := exec.Command("cp", "-a", matches[0]+"/.", dir).CombinedOutput()
			require.NoError(t, err, string(out))
			if out, err := runGo(dir, "build", "./..."); err != nil {
				t.Logf("does not build before migrating; skipping:\n%s", out)
				return
			}
			before := goSources(t, dir)
			blockers := migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{})
			after := goSources(t, dir)

			var changed, added []string
			for path, content := range after {
				if old, existed := before[path]; !existed {
					added = append(added, path)
				} else if old != content {
					changed = append(changed, path)
				}
			}
			sort.Strings(changed)
			sort.Strings(added)
			t.Logf("migrated %d file(s): %v", len(changed), changed)
			t.Logf("generated %d file(s): %v", len(added), added)
			for _, b := range blockers {
				t.Logf("held back %s: %s", b.SourcePath, b.Reason)
			}
			for _, stage := range []struct {
				name string
				prep []string
			}{{"as migrated", nil}, {"after go mod download all", []string{"mod", "download", "all"}}, {"after go mod tidy", []string{"mod", "tidy"}}} {
				if stage.prep != nil {
					if out, err := runGo(dir, stage.prep...); err != nil {
						t.Logf("go %s failed:\n%s", strings.Join(stage.prep, " "), out)
						continue
					}
				}
				if out, err := runGo(dir, "build", "./..."); err != nil {
					t.Logf("BUILD FAILS %s:\n%s", stage.name, out)
				} else {
					t.Logf("builds %s", stage.name)
				}
			}
		})
	}
}

// goSources reads every .go file and go.mod/go.sum under dir, keyed by path.
func goSources(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	require.NoError(t, filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		base := filepath.Base(p)
		if !strings.HasSuffix(base, ".go") && base != "go.mod" && base != "go.sum" {
			return nil
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		files[filepath.ToSlash(rel)] = string(content)
		return nil
	}))
	return files
}

// A session's request handlers become middleware through a generated adapter.
// Building is not enough to trust it: the v1 module and its migration run
// against the same scripted server, and the handlers must record the same
// attempts, the same span and the same labels.
func TestMigratedRequestHandlersRecordAlike(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule main\n\ngo 1.22\n",
		"deps.go": `
package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func client(endpoint string) *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Endpoint:    aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials("id", "secret", ""),
	}))
	sess.Handlers.Send.PushFront(func(r *request.Request) {
		record("before", r.ClientInfo.ServiceName, r.Operation.Name)
	})
	sess.Handlers.Send.SetBackNamed(request.NamedHandler{
		Name: "Counters",
		Fn: func(r *request.Request) {
			took := time.Now().Sub(r.AttemptTime).Round(100 * time.Millisecond)
			record(fmt.Sprintf("after %v", took), r.ClientInfo.ServiceName, r.Operation.Name)
		},
	})
	sess.Handlers.UnmarshalError.SetBackNamed(request.NamedHandler{
		Name: "Counters",
		Fn: func(r *request.Request) {
			record("error", r.ClientInfo.ServiceName, r.Operation.Name)
		},
	})
	// Order and replacement by name: v1 runs a list front to back, and Set
	// swaps a handler in where the one it replaces stood.
	sess.Handlers.Send.PushBackNamed(request.NamedHandler{
		Name: "Tag",
		Fn:   func(r *request.Request) { record("tag replaced", "", "") },
	})
	sess.Handlers.Send.PushBack(func(r *request.Request) { record("last", "", "") })
	sess.Handlers.Send.SetBackNamed(request.NamedHandler{
		Name: "Tag",
		Fn:   func(r *request.Request) { record("tag", "", "") },
	})
	sess.Handlers.UnmarshalError.PushFront(func(r *request.Request) { record("first error", "", "") })
	return cloudwatch.New(sess)
}

func listMetrics(svc *cloudwatch.CloudWatch) error {
	_, err := svc.ListMetrics(&cloudwatch.ListMetricsInput{})
	return err
}
`,
		"main.go": `
package main

import (
	"fmt"
	"os"
	"strings"
)

var events []string

func record(what, service, call string) {
	events = append(events, strings.Join([]string{what, service, call}, " "))
}

func main() {
	err := listMetrics(client(os.Args[1]))
	for _, e := range events {
		fmt.Println(e)
	}
	fmt.Println("failed:", err != nil)
}
`,
	})
	v1Bin := filepath.Join(t.TempDir(), "v1")
	goCmd(t, dir, "build", "-o", v1Bin, ".")

	blockers := migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{})
	require.Empty(t, blockers)
	deps, err := os.ReadFile(filepath.Join(dir, "deps.go"))
	require.NoError(t, err)
	require.NotContains(t, string(deps), "aws-sdk-go/aws/request")
	assertBuilds(t, dir)
	v2Bin := filepath.Join(t.TempDir(), "v2")
	goCmd(t, dir, "build", "-o", v2Bin, ".")

	// Each step is a status, held back for the delay before the headers go out.
	// No script fails more often than both SDKs' default retries absorb.
	type step struct {
		status int
		delay  time.Duration
	}
	for name, script := range map[string][]step{
		"ok":            {{200, 0}},
		"slow":          {{200, 300 * time.Millisecond}},
		"retried":       {{500, 0}, {200, 0}},
		"retried twice": {{503, 200 * time.Millisecond}, {500, 0}, {200, 0}},
		"client error":  {{400, 0}},
	} {
		t.Run(name, func(t *testing.T) {
			run := func(bin string) string {
				var mu sync.Mutex
				i := 0
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					s := step{200, 0}
					if i < len(script) {
						s = script[i]
					}
					i++
					mu.Unlock()
					time.Sleep(s.delay)
					// v2's CloudWatch speaks CBOR, v1's the query protocol.
					cbor := r.Header.Get("Smithy-Protocol") != ""
					switch {
					case cbor:
						w.Header().Set("Smithy-Protocol", "rpc-v2-cbor")
						w.Header().Set("Content-Type", "application/cbor")
						w.WriteHeader(s.status)
						w.Write([]byte{0xa0})
					case s.status == 200:
						w.Header().Set("Content-Type", "text/xml")
						w.Write([]byte(`<ListMetricsResponse><ListMetricsResult><Metrics></Metrics></ListMetricsResult></ListMetricsResponse>`))
					default:
						w.WriteHeader(s.status)
						w.Write([]byte(`<ErrorResponse><Error><Code>Failure</Code><Message>m</Message></Error></ErrorResponse>`))
					}
				}))
				defer srv.Close()
				out, err := exec.Command(bin, srv.URL).CombinedOutput()
				require.NoError(t, err, string(out))
				return string(out)
			}
			v1Out, v2Out := run(v1Bin), run(v2Bin)
			t.Logf("v1:\n%sv2:\n%s", v1Out, v2Out)
			require.Equal(t, v1Out, v2Out)
		})
	}
}

// v1's MaxRetries counted retries and v2's RetryMaxAttempts counts attempts.
// Against a server that always fails, the v1 module and its migration have to
// make the same number of requests for the same setting.
func TestMigratedMaxRetriesMakeTheSameAttempts(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule main\n\ngo 1.22\n",
		"client.go": `
package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func listMetrics(endpoint string, retries int) error {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Endpoint:    aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials("id", "secret", ""),
		MaxRetries:  aws.Int(retries),
	}))
	_, err := cloudwatch.New(sess).ListMetrics(&cloudwatch.ListMetricsInput{})
	return err
}
`,
		"main.go": `
package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	retries, _ := strconv.Atoi(os.Args[2])
	fmt.Println("failed:", listMetrics(os.Args[1], retries) != nil)
}
`,
	})
	v1Bin := filepath.Join(t.TempDir(), "v1")
	goCmd(t, dir, "build", "-o", v1Bin, ".")
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	assertBuilds(t, dir)
	v2Bin := filepath.Join(t.TempDir(), "v2")
	goCmd(t, dir, "build", "-o", v2Bin, ".")

	attempts := func(bin string, retries int) int {
		var mu sync.Mutex
		n := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			n++
			mu.Unlock()
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()
		out, err := exec.Command(bin, srv.URL, fmt.Sprint(retries)).CombinedOutput()
		require.NoError(t, err, string(out))
		require.Contains(t, string(out), "failed: true")
		mu.Lock()
		defer mu.Unlock()
		return n
	}
	// -1 asks each SDK for its own default, which v1 and v2 set differently.
	for _, retries := range []int{0, 1, 2} {
		v1, v2 := attempts(v1Bin, retries), attempts(v2Bin, retries)
		t.Logf("MaxRetries %d: v1 made %d attempts, v2 %d", retries, v1, v2)
		require.Equal(t, retries+1, v1)
		require.Equal(t, v1, v2)
	}
}

// session.Must outside an `x :=` statement becomes the generated loader, or the
// config it was handed; either way the result has to build.
func TestMigratedSessionMustInExpressionBuilds(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule main\n\ngo 1.22\n",
		"main.go": `
package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var shared = session.Must(session.NewSession())

func returned(region string) *session.Session {
	return session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
}

func passed() *sqs.SQS {
	return sqs.New(session.Must(session.NewSession()))
}

func carried(cfg *aws.Config) *sqs.SQS {
	return sqs.New(session.Must(session.NewSessionWithOptions(session.Options{Config: *cfg})))
}

func main() {
	_, _, _, _ = shared, returned("us-east-1"), passed(), carried(&aws.Config{})
}
`,
	})
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	migrated, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	require.NotContains(t, string(migrated), "session.")
	assertBuilds(t, dir)
}

// S3's path-style addressing reaches the v2 client through the config's service
// options: the v1 module and its migration have to address the same URL.
func TestMigratedPathStyleAddressesAlike(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule main\n\ngo 1.22\n",
		"client.go": `
package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func head(endpoint string) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String("us-east-1"),
		Endpoint:         aws.String(endpoint),
		Credentials:      credentials.NewStaticCredentials("id", "secret", ""),
		S3ForcePathStyle: aws.Bool(true),
	}))
	s3.New(sess).HeadObject(&s3.HeadObjectInput{Bucket: aws.String("bucket"), Key: aws.String("key")})
}
`,
		"main.go": `
package main

import "os"

func main() { head(os.Args[1]) }
`,
	})
	v1Bin := filepath.Join(t.TempDir(), "v1")
	goCmd(t, dir, "build", "-o", v1Bin, ".")
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	assertBuilds(t, dir)
	v2Bin := filepath.Join(t.TempDir(), "v2")
	goCmd(t, dir, "build", "-o", v2Bin, ".")

	requested := func(bin string) string {
		var mu sync.Mutex
		var paths []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			paths = append(paths, r.Host+" "+r.URL.Path)
			mu.Unlock()
		}))
		defer srv.Close()
		out, err := exec.Command(bin, srv.URL).CombinedOutput()
		require.NoError(t, err, string(out))
		mu.Lock()
		defer mu.Unlock()
		require.NotEmpty(t, paths)
		return strings.Replace(paths[0], strings.TrimPrefix(srv.URL, "http://"), "server", 1)
	}
	v1, v2 := requested(v1Bin), requested(v2Bin)
	t.Logf("v1 requested %q, v2 %q", v1, v2)
	require.Equal(t, "server /bucket/key", v1)
	require.Equal(t, v1, v2)
}

// DisableSSL sent v1's requests over http, to a custom endpoint given without a
// scheme and to S3's own endpoint alike. The migrated binary must too.
func TestMigratedDisableSSLRequestsAlike(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule main\n\ngo 1.22\n",
		"client.go": `
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type recorder struct{}

func (recorder) RoundTrip(r *http.Request) (*http.Response, error) {
	fmt.Println(r.URL.Scheme)
	os.Exit(0)
	return nil, errors.New("unreachable")
}

var recordingClient = &http.Client{Transport: recorder{}}

func head(endpoint string, insecure bool) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Endpoint:    aws.String(endpoint),
		DisableSSL:  aws.Bool(insecure),
		Credentials: credentials.NewStaticCredentials("id", "secret", ""),
		HTTPClient:  recordingClient,
	}))
	s3.New(sess).HeadObject(&s3.HeadObjectInput{Bucket: aws.String("bucket"), Key: aws.String("key")})
}
`,
		"main.go": `
package main

import "os"

func main() { head(os.Args[1], os.Args[2] == "true") }
`,
	})
	v1Bin := filepath.Join(t.TempDir(), "v1")
	goCmd(t, dir, "build", "-o", v1Bin, ".")
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	assertBuilds(t, dir)
	v2Bin := filepath.Join(t.TempDir(), "v2")
	goCmd(t, dir, "build", "-o", v2Bin, ".")

	scheme := func(bin, endpoint, insecure string) string {
		out, err := exec.Command(bin, endpoint, insecure).CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	for _, c := range [][3]string{
		{"minio:9000", "true", "http"},
		{"https://minio:9000", "true", "https"},
		{"", "true", "http"},
		{"", "false", "https"},
		{"minio:9000", "false", "https"},
	} {
		v1, v2 := scheme(v1Bin, c[0], c[1]), scheme(v2Bin, c[0], c[1])
		require.Equal(t, c[2], v1, "v1 with endpoint %q, DisableSSL %s", c[0], c[1])
		require.Equal(t, v1, v2, "endpoint %q, DisableSSL %s", c[0], c[1])
	}
}

// profefe/profefe's shape: the uploader and downloader held as s3manageriface
// interfaces, with mocks embedding them. v2 dropped the package; it is
// regenerated in the module, the mocks take v2's methods, and the package's
// tests still pass against them.
func TestMigratedManagerIfaceBuildsAndTests(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule example.com/prof\n\ngo 1.22\n",
		"storage/storage.go": `
package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
)

type Storage struct {
	svc        s3iface.S3API
	uploader   s3manageriface.UploaderAPI
	downloader s3manageriface.DownloaderAPI
	bucket     string
}

func NewStorage(svc s3iface.S3API, bucket string) *Storage {
	return &Storage{
		svc:        svc,
		uploader:   s3manager.NewUploaderWithClient(svc),
		downloader: s3manager.NewDownloaderWithClient(svc),
		bucket:     bucket,
	}
}

func (st *Storage) Put(ctx context.Context, key string, r io.Reader) (string, error) {
	resp, err := st.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(st.bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	if err != nil {
		return "", err
	}
	return resp.Location, nil
}

func (st *Storage) Get(ctx context.Context, key string, w io.WriterAt) (int64, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(st.bucket),
		Key:    aws.String(key),
	}
	return st.downloader.DownloadWithContext(ctx, w, input)
}
`,
		"storage/storage_test.go": `
package s3

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
)

type mockUploaderAPI struct {
	s3manageriface.UploaderAPI
	inputs []*s3manager.UploadInput
}

func (m *mockUploaderAPI) UploadWithContext(ctx aws.Context, input *s3manager.UploadInput, _ ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	m.inputs = append(m.inputs, input)
	return &s3manager.UploadOutput{Location: *input.Key}, nil
}

type mockDownloaderAPI struct {
	s3manageriface.DownloaderAPI
}

func (m *mockDownloaderAPI) DownloadWithContext(ctx aws.Context, wa io.WriterAt, input *s3.GetObjectInput, _ ...func(*s3manager.Downloader)) (int64, error) {
	return 0, nil
}

func TestStorage(t *testing.T) {
	up := &mockUploaderAPI{}
	st := &Storage{uploader: up, downloader: &mockDownloaderAPI{}, bucket: "b"}
	loc, err := st.Put(context.Background(), "k", strings.NewReader("x"))
	if err != nil || loc != "k" || len(up.inputs) != 1 {
		t.Fatal(loc, err)
	}
}
`,
	})
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	mocks, err := os.ReadFile(filepath.Join(dir, "storage", "storage_test.go"))
	require.NoError(t, err)
	require.Contains(t, string(mocks), "Upload(ctx context.Context, input *s3.PutObjectInput")
	assertBuilds(t, dir)
	if out, err := runGo(dir, "test", "./..."); err != nil {
		t.Errorf("go test ./... after migration:\n%s", out)
	}
}

// A MockGen mock of a v1 iface, as keikoproj/iam-manager keeps, is rendered
// afresh against the regenerated interface; the client field still takes it
// and the expectations recorded on it still match the calls made.
func TestMigratedMockGenMockBuildsAndTests(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule example.com/meter\n\ngo 1.22\n",
		"meter.go": `
package meter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/marketplacemetering"
	"github.com/aws/aws-sdk-go/service/marketplacemetering/marketplacemeteringiface"
)

type Meter struct {
	Client marketplacemeteringiface.MarketplaceMeteringAPI
}

func (m *Meter) Customer(token string) (string, error) {
	out, err := m.Client.ResolveCustomer(&marketplacemetering.ResolveCustomerInput{RegistrationToken: aws.String(token)})
	if err != nil {
		return "", err
	}
	return aws.StringValue(out.CustomerIdentifier), nil
}

func (m *Meter) Register(product string) error {
	_, err := m.Client.RegisterUsage(&marketplacemetering.RegisterUsageInput{ProductCode: aws.String(product)})
	return err
}
`,
		"mocks/mock.go": `
// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/aws/aws-sdk-go/service/marketplacemetering/marketplacemeteringiface (interfaces: MarketplaceMeteringAPI)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	request "github.com/aws/aws-sdk-go/aws/request"
	marketplacemetering "github.com/aws/aws-sdk-go/service/marketplacemetering"
	gomock "go.uber.org/mock/gomock"
)

type MockMarketplaceMeteringAPI struct {
	ctrl     *gomock.Controller
	recorder *MockMarketplaceMeteringAPIMockRecorder
}

type MockMarketplaceMeteringAPIMockRecorder struct {
	mock *MockMarketplaceMeteringAPI
}

func NewMockMarketplaceMeteringAPI(ctrl *gomock.Controller) *MockMarketplaceMeteringAPI {
	mock := &MockMarketplaceMeteringAPI{ctrl: ctrl}
	mock.recorder = &MockMarketplaceMeteringAPIMockRecorder{mock}
	return mock
}

func (m *MockMarketplaceMeteringAPI) EXPECT() *MockMarketplaceMeteringAPIMockRecorder {
	return m.recorder
}

func (m *MockMarketplaceMeteringAPI) ResolveCustomerRequest(arg0 *marketplacemetering.ResolveCustomerInput) (*request.Request, *marketplacemetering.ResolveCustomerOutput) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveCustomerRequest", arg0)
	ret0, _ := ret[0].(*request.Request)
	ret1, _ := ret[1].(*marketplacemetering.ResolveCustomerOutput)
	return ret0, ret1
}

func (mr *MockMarketplaceMeteringAPIMockRecorder) ResolveCustomerRequest(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveCustomerRequest", reflect.TypeOf((*MockMarketplaceMeteringAPI)(nil).ResolveCustomerRequest), arg0)
}
`,
		"meter_test.go": `
package meter

import (
	"testing"

	"example.com/meter/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/marketplacemetering"
	"go.uber.org/mock/gomock"
)

func TestMeter(t *testing.T) {
	client := mocks.NewMockMarketplaceMeteringAPI(gomock.NewController(t))
	client.EXPECT().ResolveCustomer(&marketplacemetering.ResolveCustomerInput{RegistrationToken: aws.String("t")}).Return(&marketplacemetering.ResolveCustomerOutput{CustomerIdentifier: aws.String("c")}, nil)
	var seen string
	client.EXPECT().RegisterUsage(gomock.Any()).Do(func(in *marketplacemetering.RegisterUsageInput) {
		seen = aws.StringValue(in.ProductCode)
	}).Return(&marketplacemetering.RegisterUsageOutput{}, nil)
	m := &Meter{Client: client}
	if c, err := m.Customer("t"); err != nil || c != "c" {
		t.Fatal(c, err)
	}
	if err := m.Register("p"); err != nil || seen != "p" {
		t.Fatal(seen, err)
	}
}
`,
	})
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	assertBuilds(t, dir)
	if out, err := runGo(dir, "test", "./..."); err != nil {
		t.Errorf("go test ./... after migration:\n%s", out)
	}
}

// A mockery mock is regenerated the same way, and the expectations recorded on
// it still match the calls the migrated code makes.
func TestMigratedMockeryMockBuildsAndTests(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule example.com/meter\n\ngo 1.22\n\nrequire github.com/stretchr/testify v1.10.0\n",
		"meter.go": `
package meter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/marketplacemetering"
	"github.com/aws/aws-sdk-go/service/marketplacemetering/marketplacemeteringiface"
)

type Meter struct {
	Client marketplacemeteringiface.MarketplaceMeteringAPI
}

func (m *Meter) Customer(token string) (string, error) {
	out, err := m.Client.ResolveCustomer(&marketplacemetering.ResolveCustomerInput{RegistrationToken: aws.String(token)})
	if err != nil {
		return "", err
	}
	return aws.StringValue(out.CustomerIdentifier), nil
}

func (m *Meter) Register(product string) error {
	_, err := m.Client.RegisterUsage(&marketplacemetering.RegisterUsageInput{ProductCode: aws.String(product)})
	return err
}
`,
		"mocks/MarketplaceMeteringAPI.go": `
// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	request "github.com/aws/aws-sdk-go/aws/request"
	mock "github.com/stretchr/testify/mock"

	marketplacemetering "github.com/aws/aws-sdk-go/service/marketplacemetering"
)

// MarketplaceMeteringAPI is an autogenerated mock type for the MarketplaceMeteringAPI type
type MarketplaceMeteringAPI struct {
	mock.Mock
}

// ResolveCustomer provides a mock function with given fields: _a0
func (_m *MarketplaceMeteringAPI) ResolveCustomer(_a0 *marketplacemetering.ResolveCustomerInput) (*marketplacemetering.ResolveCustomerOutput, error) {
	ret := _m.Called(_a0)

	var r0 *marketplacemetering.ResolveCustomerOutput
	if rf, ok := ret.Get(0).(func(*marketplacemetering.ResolveCustomerInput) *marketplacemetering.ResolveCustomerOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*marketplacemetering.ResolveCustomerOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*marketplacemetering.ResolveCustomerInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ResolveCustomerRequest provides a mock function with given fields: _a0
func (_m *MarketplaceMeteringAPI) ResolveCustomerRequest(_a0 *marketplacemetering.ResolveCustomerInput) (*request.Request, *marketplacemetering.ResolveCustomerOutput) {
	ret := _m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*marketplacemetering.ResolveCustomerInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *marketplacemetering.ResolveCustomerOutput
	if rf, ok := ret.Get(1).(func(*marketplacemetering.ResolveCustomerInput) *marketplacemetering.ResolveCustomerOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*marketplacemetering.ResolveCustomerOutput)
		}
	}

	return r0, r1
}

type mockConstructorTestingTNewMarketplaceMeteringAPI interface {
	mock.TestingT
	Cleanup(func())
}

// NewMarketplaceMeteringAPI creates a new instance of MarketplaceMeteringAPI.
func NewMarketplaceMeteringAPI(t mockConstructorTestingTNewMarketplaceMeteringAPI) *MarketplaceMeteringAPI {
	mock := &MarketplaceMeteringAPI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
`,
		"meter_test.go": `
package meter

import (
	"testing"

	"example.com/meter/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/marketplacemetering"
	"github.com/stretchr/testify/mock"
)

func TestMeter(t *testing.T) {
	client := mocks.NewMarketplaceMeteringAPI(t)
	client.On("ResolveCustomer", &marketplacemetering.ResolveCustomerInput{RegistrationToken: aws.String("t")}).Return(&marketplacemetering.ResolveCustomerOutput{CustomerIdentifier: aws.String("c")}, nil).Once()
	client.On("RegisterUsage", mock.MatchedBy(func(in *marketplacemetering.RegisterUsageInput) bool {
		return aws.StringValue(in.ProductCode) == "p"
	})).Return(&marketplacemetering.RegisterUsageOutput{}, nil)
	m := &Meter{Client: client}
	if c, err := m.Customer("t"); err != nil || c != "c" {
		t.Fatal(c, err)
	}
	if err := m.Register("p"); err != nil {
		t.Fatal(err)
	}
	client.AssertCalled(t, "RegisterUsage", mock.Anything)
	client.AssertNumberOfCalls(t, "ResolveCustomer", 1)
	client.AssertNotCalled(t, "MeterUsage", mock.Anything)
}
`,
	})
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	assertBuilds(t, dir)
	if out, err := runGo(dir, "test", "./..."); err != nil {
		t.Errorf("go test ./... after migration:\n%s", out)
	}
}

// A config built up through setters under conditions, as willnorris/imageproxy
// builds one, sends its requests over the scheme v1's did, whichever settings
// were applied and in whatever order.
func TestMigratedConfigBuilderRequestsAlike(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule main\n\ngo 1.22\n",
		"client.go": `
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type recorder struct{}

func (recorder) RoundTrip(r *http.Request) (*http.Response, error) {
	fmt.Println(r.URL.Scheme)
	os.Exit(0)
	return nil, errors.New("unreachable")
}

func head(endpoint, insecure string) {
	config := aws.NewConfig().WithRegion("us-east-1").WithCredentials(credentials.NewStaticCredentials("id", "secret", "")).WithHTTPClient(&http.Client{Transport: recorder{}})
	if endpoint != "" {
		config = config.WithEndpoint(endpoint)
	}
	if insecure == "1" {
		config = config.WithDisableSSL(true)
	}
	sess, err := session.NewSession(config)
	if err != nil {
		panic(err)
	}
	s3.New(sess).HeadObject(&s3.HeadObjectInput{Bucket: aws.String("bucket"), Key: aws.String("key")})
}
`,
		"main.go": `
package main

import "os"

func main() { head(os.Args[1], os.Args[2]) }
`,
	})
	v1Bin := filepath.Join(t.TempDir(), "v1")
	goCmd(t, dir, "build", "-o", v1Bin, ".")
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	client, err := os.ReadFile(filepath.Join(dir, "client.go"))
	require.NoError(t, err)
	require.Contains(t, string(client), "awsCompatNewConfig()")
	assertBuilds(t, dir)
	v2Bin := filepath.Join(t.TempDir(), "v2")
	goCmd(t, dir, "build", "-o", v2Bin, ".")

	scheme := func(bin, endpoint, insecure string) string {
		out, err := exec.Command(bin, endpoint, insecure).CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	for _, c := range [][3]string{
		{"minio:9000", "1", "http"},
		{"https://minio:9000", "1", "https"},
		{"", "1", "http"},
		{"", "0", "https"},
		{"minio:9000", "0", "https"},
	} {
		v1, v2 := scheme(v1Bin, c[0], c[1]), scheme(v2Bin, c[0], c[1])
		require.Equal(t, c[2], v1, "v1 with endpoint %q, disableSSL %s", c[0], c[1])
		require.Equal(t, v1, v2, "endpoint %q, disableSSL %s", c[0], c[1])
	}
}

// hypnoglow/helm-s3's awsutil package: functional options over a
// session.Options value, one of which reads a bucket's region off the raw
// response of a HEAD request it sends. The migrated package builds, and asks
// the same server for the region as v1 did.
func TestMigratedSessionOptionsValueBuilds(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule example.com/hs3\n\ngo 1.22\n",
		"awsutil/session.go": `
package awsutil

import (
	"net/url"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	// awsEndpoint can be set to a custom endpoint to use alternative AWS S3
	// server like minio (https://minio.io).
	awsEndpoint = "AWS_ENDPOINT"

	// awsDisableSSL can be set to true to disable SSL for AWS S3 server.
	awsDisableSSL = "AWS_DISABLE_SSL"

	// awsBucketLocation can be set to an AWS region to force the session region
	// if AWS_DEFAULT_REGION and AWS_REGION cannot be trusted.
	awsBucketLocation = "HELM_S3_REGION"

	// awsDynamicRegionEnabled can be set to a falsy value to disable dynamic
	// bucket region discovery. By default, dynamic region discovery is enabled.
	awsDynamicRegionEnabled = "HELM_S3_DYNAMIC_REGION_ENABLED"
)

// SessionOption is an option for session.
type SessionOption func(*session.Options)

// AssumeRoleTokenProvider is an option for setting custom assume role token provider.
func AssumeRoleTokenProvider(provider func() (string, error)) SessionOption {
	return func(options *session.Options) {
		options.AssumeRoleTokenProvider = provider
	}
}

// DynamicBucketRegion is an option for determining the Helm S3 bucket's AWS
// region dynamically thus allowing the mixed use of buckets residing in
// different regions without requiring manual updates on the HELM_S3_REGION,
// AWS_REGION, or AWS_DEFAULT_REGION environment variables.
//
// This HEAD bucket solution works with all kinds of S3 URIs containing
// the bucket name in the host part.
//
// The basic idea behind the HEAD bucket solution and the "official
// confirmation" this behavior is expected and supported came from a comment on
// the AWS SDK Go repository:
// https://github.com/aws/aws-sdk-go/issues/720#issuecomment-243891223
func DynamicBucketRegion(s3URL string) SessionOption {
	return func(options *session.Options) {
		if v, err := strconv.ParseBool(os.Getenv(awsDynamicRegionEnabled)); err == nil && !v {
			return
		}

		parsedS3URL, err := url.Parse(s3URL)
		if err != nil {
			return
		}

		// Note: The dummy credentials are required in case no other credential
		// provider is found, but even if the HEAD bucket request fails and
		// returns a non-200 status code indicating no access to the bucket, the
		// actual bucket region is returned in a response header.
		//
		// Note: A signing region **MUST** be configured, otherwise the signed
		// request fails. The configured region itself is irrelevant, the
		// endpoint officially works and returns the bucket region in a response
		// header regardless of whether the signing region matches the bucket's
		// region.
		//
		// Note: The default S3 endpoint **MUST** be configured to avoid making
		// the request region specific thus avoiding regional redirect responses
		// (301 Permanently moved) on HEAD bucket. This setting is only required
		// because any other region than "us-east-1" would configure a
		// region-specific endpoint as well, so it's more safe to explicitly
		// configure the default endpoint.
		//
		// Source:
		// https://github.com/aws/aws-sdk-go/issues/720#issuecomment-243891223
		configuration := aws.NewConfig().
			WithCredentials(credentials.NewStaticCredentials("dummy", "dummy", "")).
			WithRegion("us-east-1").
			WithEndpoint("s3.amazonaws.com")
		sess := session.Must(session.NewSession())
		s3Client := s3.New(sess, configuration)

		bucketRegionHeader := "X-Amz-Bucket-Region"
		input := &s3.HeadBucketInput{
			Bucket: aws.String(parsedS3URL.Host),
		}
		request, _ := s3Client.HeadBucketRequest(input)
		_ = request.Send()
		if request.HTTPResponse == nil ||
			len(request.HTTPResponse.Header[bucketRegionHeader]) == 0 {
			return
		}

		options.Config.Region = aws.String(request.HTTPResponse.Header[bucketRegionHeader][0])
	}
}

// Session returns an AWS session as described http://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html
func Session(opts ...SessionOption) (*session.Session, error) {
	disableSSL := os.Getenv(awsDisableSSL) == "true"

	so := session.Options{
		Config: aws.Config{
			DisableSSL:       aws.Bool(disableSSL),
			S3ForcePathStyle: aws.Bool(true),
			Endpoint:         aws.String(os.Getenv(awsEndpoint)),
		},
		SharedConfigState:       session.SharedConfigEnable,
		AssumeRoleTokenProvider: StderrTokenProvider,
	}

	bucketRegion := os.Getenv(awsBucketLocation)
	// if not set, we don't update the config,
	// so that the AWS SDK can still rely on either AWS_REGION or AWS_DEFAULT_REGION
	if bucketRegion != "" {
		so.Config.Region = aws.String(bucketRegion)
	}

	for _, opt := range opts {
		opt(&so)
	}

	return session.NewSessionWithOptions(so)
}
`,
		"awsutil/token.go": `
package awsutil

func StderrTokenProvider() (string, error) { return "", nil }
`,
		"main.go": `
package main

import (
	"fmt"

	"example.com/hs3/awsutil"
)

func main() {
	sess, err := awsutil.Session(awsutil.AssumeRoleTokenProvider(awsutil.StderrTokenProvider))
	fmt.Println(sess, err)
}
`,
	})
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	session, err := os.ReadFile(filepath.Join(dir, "awsutil", "session.go"))
	require.NoError(t, err)
	t.Logf("migrated session.go:\n%s", session)
	assertBuilds(t, dir)
}

// The raw response a v1 request held after Send, even for a failed call, is
// what hypnoglow/helm-s3 reads a bucket's region from. The stand-in hands the
// migrated binary the same response.
func TestMigratedRequestReadsRawResponseAlike(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule main\n\ngo 1.22\n",
		"client.go": `
package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func region(endpoint string) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String("us-east-1"),
		Endpoint:         aws.String(endpoint),
		Credentials:      credentials.NewStaticCredentials("id", "secret", ""),
		S3ForcePathStyle: aws.Bool(true),
		MaxRetries:       aws.Int(0),
	}))
	client := s3.New(sess)
	request, _ := client.HeadBucketRequest(&s3.HeadBucketInput{Bucket: aws.String("bucket")})
	err := request.Send()
	if request.HTTPResponse == nil {
		fmt.Println("no response")
		return
	}
	fmt.Println(request.HTTPResponse.StatusCode, request.HTTPResponse.Header.Get("X-Amz-Bucket-Region"), err != nil, request.Error != nil)
}
`,
		"main.go": `
package main

import "os"

func main() { region(os.Args[1]) }
`,
	})
	v1Bin := filepath.Join(t.TempDir(), "v1")
	goCmd(t, dir, "build", "-o", v1Bin, ".")
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	client, err := os.ReadFile(filepath.Join(dir, "client.go"))
	require.NoError(t, err)
	require.Contains(t, string(client), "awsCompatNewRequest(")
	assertBuilds(t, dir)
	v2Bin := filepath.Join(t.TempDir(), "v2")
	goCmd(t, dir, "build", "-o", v2Bin, ".")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Amz-Bucket-Region", "eu-west-2")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	run := func(bin string) string {
		out, err := exec.Command(bin, srv.URL).CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	v1, v2 := run(v1Bin), run(v2Bin)
	t.Logf("v1 %q, v2 %q", v1, v2)
	require.Equal(t, "403 eu-west-2 true true", v1)
	require.Equal(t, v1, v2)
}

// in4it/ecs-deploy's shape: github.com/guregu/dynamo, built on aws-sdk-go v1,
// over a session. Its v2 takes the config, a context at every call reaching
// DynamoDB and an int limit; the migrated module builds against it.
func TestMigratedGureguDynamoBuilds(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule example.com/svc\n\ngo 1.22\n\nrequire github.com/guregu/dynamo v1.23.0\n",
		"service/service.go": `
package service

import (
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
)

type Deployment struct {
	ServiceName string ` + "`" + `dynamo:"ServiceName,hash"` + "`" + `
	Time        time.Time ` + "`" + `dynamo:"Time,range"` + "`" + `
	Version     int64
}

type Service struct {
	db    *dynamo.DB
	table dynamo.Table
}

func New(table string) *Service {
	s := &Service{}
	s.db = dynamo.New(session.New(), &aws.Config{})
	s.table = s.db.Table(table)
	return s
}

func (s *Service) Put(d *Deployment) error {
	return s.table.Put(d).If("$ = ?", "Version", d.Version-1).Run()
}

func (s *Service) Latest(name string, n int) ([]Deployment, error) {
	var out []Deployment
	err := s.table.Get("ServiceName", name).Range("Time", dynamo.LessOrEqual, time.Now()).Order(dynamo.Descending).Limit(int64(math.Max(float64(10), float64(n)))).All(&out)
	return out, err
}

func (s *Service) One(name string, t time.Time) (Deployment, error) {
	var d Deployment
	err := s.table.Get("ServiceName", name).Range("Time", dynamo.Equal, t).Limit(1).One(&d)
	return d, err
}
`,
	})
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	migrated, err := os.ReadFile(filepath.Join(dir, "service", "service.go"))
	require.NoError(t, err)
	t.Logf("migrated service.go:\n%s", migrated)
	require.Contains(t, string(migrated), `"github.com/guregu/dynamo/v2"`)
	assertBuilds(t, dir)
}

// guregu/dynamo's v2 marshals items itself, so the item a migrated binary
// stores is compared with the one v1 stored: each sends its PutItem to a fake
// endpoint, which records the item.
func TestMigratedGureguDynamoStoresAlike(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule example.com/svc\n\ngo 1.22\n\nrequire github.com/guregu/dynamo v1.23.0\n",
		"main.go": `
package main

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
)

type Element struct {
	C, S        string
	MemoryLimit int64    ` + "`" + `dynamo:"ML"` + "`" + `
	Tags        []string ` + "`" + `dynamo:",set,omitempty"` + "`" + `
}

type Deployment struct {
	ServiceName string ` + "`" + `dynamo:"ServiceName,hash"` + "`" + `
	Time        time.Time ` + "`" + `dynamo:"Time,range"` + "`" + `
	Version     int64
	Empty       string
	Services    []*Element
	Labels      map[string]string
}

func main() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Endpoint:    aws.String(os.Args[1]),
		Credentials: credentials.NewStaticCredentials("id", "secret", ""),
		MaxRetries:  aws.Int(0),
	}))
	db := dynamo.New(sess)
	d := Deployment{
		ServiceName: "svc",
		Time:        time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Version:     3,
		Services:    []*Element{{C: "c", S: "s", MemoryLimit: 512, Tags: []string{"a", "b"}}, {C: "d", S: "t"}},
		Labels:      map[string]string{"team": "x", "empty": ""},
	}
	db.Table("Services").Put(d).Run()
}
`,
	})
	v1Bin := filepath.Join(t.TempDir(), "v1")
	goCmd(t, dir, "build", "-o", v1Bin, ".")
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	main, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	require.Contains(t, string(main), `"github.com/guregu/dynamo/v2"`)
	assertBuilds(t, dir)
	v2Bin := filepath.Join(t.TempDir(), "v2")
	goCmd(t, dir, "build", "-o", v2Bin, ".")

	stored := func(bin string) string {
		var mu sync.Mutex
		var item string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body struct{ Item json.RawMessage }
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			var canonical any
			_ = json.Unmarshal(body.Item, &canonical)
			out, _ := json.Marshal(canonical)
			mu.Lock()
			item = string(out)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/x-amz-json-1.0")
			_, _ = w.Write([]byte("{}"))
		}))
		defer srv.Close()
		out, err := exec.Command(bin, srv.URL).CombinedOutput()
		require.NoError(t, err, string(out))
		mu.Lock()
		defer mu.Unlock()
		return item
	}
	v1, v2 := stored(v1Bin), stored(v2Bin)
	t.Logf("v1 stored %s\nv2 stored %s", v1, v2)
	require.NotEmpty(t, v1)
	require.Equal(t, v1, v2)
}

// Items written through dynamodbattribute keep v1's encoding once the import
// moves to the generated port: json tags still name attributes, and an empty
// string or list is still stored as NULL, so items written before read back.
func TestMigratedDynamodbAttributeStoresAlike(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule example.com/leases\n\ngo 1.22\n",
		"leases.go": `
package leases

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type Lease struct {
	ID    string   ` + "`json:\"id\"`" + `
	Note  string   ` + "`json:\"note\"`" + `
	Tags  []string ` + "`json:\"tags\"`" + `
	Count int      ` + "`json:\"count\"`" + `
}

func Encode(l Lease) (map[string]*dynamodb.AttributeValue, error) {
	return dynamodbattribute.MarshalMap(l)
}

func Decode(item map[string]*dynamodb.AttributeValue) (Lease, error) {
	var l Lease
	err := dynamodbattribute.UnmarshalMap(item, &l)
	return l, err
}
`,
		"leases_test.go": `
package leases

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
)

func TestStorage(t *testing.T) {
	item, err := Encode(Lease{ID: "a", Count: 3})
	if err != nil {
		t.Fatal(err)
	}
	if got := aws.StringValue(item["id"].S); got != "a" {
		t.Fatalf("id = %q", got)
	}
	if !aws.BoolValue(item["note"].NULL) || !aws.BoolValue(item["tags"].NULL) {
		t.Fatalf("empty values not stored as NULL: %v", item)
	}
	if got := aws.StringValue(item["count"].N); got != "3" {
		t.Fatalf("count = %q", got)
	}
	l, err := Decode(item)
	if err != nil || l.ID != "a" || l.Count != 3 {
		t.Fatal(l, err)
	}
}
`,
	})
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	assertBuilds(t, dir)
	if out, err := runGo(dir, "test", "./..."); err != nil {
		t.Errorf("go test ./... after migration:\n%s", out)
	}
}

// kube-secret-syncer's stubs override ListSecretsPages on the iface they
// embed. v2's paginator calls ListSecrets once per page, which the stub gains,
// so the code under test still sees every page, the last flagged as such, and
// the error a failing stub returns.
func TestMigratedPageIteratorStubServesPages(t *testing.T) {
	requireCompile(t)
	dir := writeModule(t, map[string]string{
		"go.mod": "\nmodule example.com/secrets\n\ngo 1.22\n",
		"secrets.go": `
package secrets

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
)

func Names(c secretsmanageriface.SecretsManagerAPI) ([]string, []bool, error) {
	var names []string
	var lasts []bool
	err := c.ListSecretsPages(&secretsmanager.ListSecretsInput{MaxResults: aws.Int64(2)}, func(page *secretsmanager.ListSecretsOutput, lastPage bool) bool {
		for _, s := range page.SecretList {
			names = append(names, aws.StringValue(s.Name))
		}
		lasts = append(lasts, lastPage)
		return !lastPage
	})
	return names, lasts, err
}
`,
		"secrets_test.go": `
package secrets

import (
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
)

type stub struct {
	secretsmanageriface.SecretsManagerAPI
	pages []*secretsmanager.ListSecretsOutput
}

func (s *stub) ListSecretsPages(input *secretsmanager.ListSecretsInput, fn func(*secretsmanager.ListSecretsOutput, bool) bool) error {
	if input.NextToken != nil || aws.Int64Value(input.MaxResults) != 2 {
		return errors.New("unexpected input")
	}
	for i, p := range s.pages {
		if !fn(p, i == len(s.pages)-1) {
			break
		}
	}
	return nil
}

type failing struct {
	secretsmanageriface.SecretsManagerAPI
}

func (failing) ListSecretsPages(*secretsmanager.ListSecretsInput, func(*secretsmanager.ListSecretsOutput, bool) bool) error {
	return errors.New("denied")
}

func page(names ...string) *secretsmanager.ListSecretsOutput {
	out := &secretsmanager.ListSecretsOutput{}
	for _, n := range names {
		out.SecretList = append(out.SecretList, &secretsmanager.SecretListEntry{Name: aws.String(n)})
	}
	return out
}

func TestStorage(t *testing.T) {
	names, lasts, err := Names(&stub{pages: []*secretsmanager.ListSecretsOutput{page("a", "b"), page("c"), page("d")}})
	if err != nil || !reflect.DeepEqual(names, []string{"a", "b", "c", "d"}) || !reflect.DeepEqual(lasts, []bool{false, false, true}) {
		t.Fatal(names, lasts, err)
	}
	if _, _, err := Names(failing{}); err == nil || err.Error() != "denied" {
		t.Fatal(err)
	}
}
`,
	})
	require.Empty(t, migrateOnDisk(t, dir, &awssdkv2.MigrateAwsSdkGoModuleToV2{}))
	assertBuilds(t, dir)
	if out, err := runGo(dir, "test", "./..."); err != nil {
		t.Errorf("go test ./... after migration:\n%s", out)
	}
}
