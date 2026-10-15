/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"fmt"
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
