/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/stretchr/testify/assert"
)

const cloudwatchV1Sum = `
	github.com/aws/aws-sdk-go v1.55.8 h1:JRmEUbU52aJQZ2AjX4q4Wu7t4uZjOu71uyNmaWlUkJQ=
	github.com/aws/aws-sdk-go v1.55.8/go.mod h1:ZkViS9AqA6otK+JBBNH2++sx1sgxrPKcSzPPvQkUtXk=
	github.com/jmespath/go-jmespath v0.4.0 h1:BEgLn5cpjn8UN1mAw4NjwDrS35OdebyEtFe+9YPoQUg=
	github.com/jmespath/go-jmespath v0.4.0/go.mod h1:T8mJZnbsbmF+m6zOOFylbeCJqk5+pHWvzYPziyZiYoo=
`

const cloudwatchV1Source = `
	package main

	import "github.com/aws/aws-sdk-go/service/cloudwatch"

	var _ *cloudwatch.CloudWatch
`

// The build runs -mod=readonly, so go.sum has to hold every module the new
// requirements bring before anything compiles. The lines go where the
// toolchain would sort them.
func TestUpdateAwsSdkDependencyAddsGoSumLines(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.UpdateAwsSdkDependency{}).RewriteRun(t,
		test.GoProject("cw",
			test.GoMod(`
				module example.com/cw

				go 1.22

				require github.com/aws/aws-sdk-go v1.55.8

				require github.com/jmespath/go-jmespath v0.4.0 // indirect
			`, `
				module example.com/cw

				go 1.24

				require github.com/jmespath/go-jmespath v0.4.0 // indirect

				require (
					github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.73.3
				)

				require (
					github.com/aws/aws-sdk-go-v2 v1.47.3 // indirect
					github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 // indirect
					github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 // indirect
					github.com/aws/smithy-go v1.28.5 // indirect
				)
			`),
			test.GoSum(cloudwatchV1Sum, `
				github.com/aws/aws-sdk-go v1.55.8 h1:JRmEUbU52aJQZ2AjX4q4Wu7t4uZjOu71uyNmaWlUkJQ=
				github.com/aws/aws-sdk-go v1.55.8/go.mod h1:ZkViS9AqA6otK+JBBNH2++sx1sgxrPKcSzPPvQkUtXk=
				github.com/aws/aws-sdk-go-v2 v1.47.3 h1:HRoT/MkNi3bjO5xF4dCzLzvVmkvRGHj3/cLINDnJejk=
				github.com/aws/aws-sdk-go-v2 v1.47.3/go.mod h1:xSl+OSI5ZUw6bSMupWc218zE5Pl+65MCCu3ySWFY99A=
				github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 h1:CqL8NN3p4I0JTwwV5MEoatIl4Y2amZ2X9dRD+0hbRL0=
				github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6/go.mod h1:exdJ32hMqXl0Wyo6NwqEC5yIhtua3+PDaKE8P29m4qE=
				github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 h1:FpBq24TG6WVSObTNhQOv7Q0088rounUUlwn3zlNItkM=
				github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6/go.mod h1:ONzdPVYC95oBOu0VWNEsKEntcoemP3xVKjT+qWml1s4=
				github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.73.3 h1:VmR1+YQW2vc3Sy2fLGSFtijUq9bAOMzzFxNC1zmHiA0=
				github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.73.3/go.mod h1:Q6H59zrEAgECYo4bjSh3IAvr+A+KklmThMoZjqnXbHY=
				github.com/aws/smithy-go v1.28.5 h1:OqbqKtzY9NUnr2yAe9+hiTdFb20tNaK/lI0PMOGXX9k=
				github.com/aws/smithy-go v1.28.5/go.mod h1:YE2RhdIuDbA5E5bTdciG9KrW3+TiEONeUWCqxX9i1Fc=
				github.com/jmespath/go-jmespath v0.4.0 h1:BEgLn5cpjn8UN1mAw4NjwDrS35OdebyEtFe+9YPoQUg=
				github.com/jmespath/go-jmespath v0.4.0/go.mod h1:T8mJZnbsbmF+m6zOOFylbeCJqk5+pHWvzYPziyZiYoo=
			`),
			test.Golang(cloudwatchV1Source),
		),
	)
}

// A dependency that already pulls in a v2 module keeps the higher of its
// version and the one the migration would pick: lowering it breaks the
// dependency, and raising a lower one is what keeps a regenerated iface
// declaring only operations the module has.
func TestUpdateAwsSdkDependencyKeepsHigherAndRaisesLowerVersions(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.UpdateAwsSdkDependency{}).RewriteRun(t,
		test.GoProject("cw",
			test.GoMod(`
				module example.com/cw

				go 1.22

				require (
					github.com/aws/aws-sdk-go v1.55.8
					github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.40.0
					github.com/aws/smithy-go v1.99.0 // indirect
				)
			`, `
				module example.com/cw

				go 1.24

				require (
					github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.73.3
					github.com/aws/smithy-go v1.99.0 // indirect
				)

				require (
					github.com/aws/aws-sdk-go-v2 v1.47.3 // indirect
					github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 // indirect
					github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 // indirect
				)
			`),
			test.Golang(cloudwatchV1Source),
		),
	)
}

// 99designs/iamy's shape: one file in the package compares its session to nil,
// so the whole package stays on v1 — the file beside it importing an iface
// package included. Nothing migrates, so neither go.mod nor a regenerated iface
// package may appear.
func TestMigrateLeavesGoModAloneWhenEveryPackageIsHeldBack(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{})
	test.ExpectDataTable(spec, awssdkv2.AwsSdkGoV1BlockersTable, func(t *testing.T, rows []awssdkv2.AwsSdkGoV1BlockerRow) {
		assert.Equal(t, []awssdkv2.AwsSdkGoV1BlockerRow{
			{SourcePath: "app/session.go", Reason: "a session compared to nil, which the config replacing it cannot be"},
			{SourcePath: "app/sts.go", Reason: "in the same package as app/session.go, which stays on v1"},
		}, rows)
	})
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.14

				require github.com/aws/aws-sdk-go v1.55.8
			`),
			test.Golang(`
				package app

				import "github.com/aws/aws-sdk-go/aws/session"

				var sess *session.Session

				func ready() bool { return sess != nil }
			`).WithPath("app/session.go"),
			test.Golang(`
				package app

				import (
					"github.com/aws/aws-sdk-go/service/sts"
					"github.com/aws/aws-sdk-go/service/sts/stsiface"
				)

				func account(svc stsiface.STSAPI) (string, error) {
					out, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
					if err != nil {
						return "", err
					}
					return *out.Account, nil
				}
			`).WithPath("app/sts.go"),
		),
	)
}

// resolvedGoMod marks a go.mod's module graph as resolved at parse time, with
// the build list and package map given, as the RPC server's ParseProject does.
func resolvedGoMod(mod test.SourceSpec, deps []golang.GoResolvedDependency, pkgs []golang.GoPackageModule) test.SourceSpec {
	mod = test.GoModGraph(mod, deps, pkgs)
	for i, m := range mod.Markers {
		if mrr, ok := m.(golang.GoResolutionResult); ok {
			mrr.ResolutionStatus = golang.GoResolutionResolved
			mod.Markers[i] = mrr
		}
	}
	return mod
}

// ZipRecruiter/cloudwatching's shape: a go.mod written for go 1.12 leaves the
// modules it only reaches indirectly to the graph. Raised past 1.17, go.mod must
// name each one, or the build stops at "updates to go.mod needed".
func TestUpdateAwsSdkDependencyNamesIndirectModulesPastGo117(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.UpdateAwsSdkDependency{}).RewriteRun(t,
		test.GoProject("cw",
			resolvedGoMod(test.GoMod(`
				module example.com/cw

				go 1.12

				require (
					github.com/aws/aws-sdk-go v1.55.8
					github.com/sirupsen/logrus v1.9.3
				)
			`, `
				module example.com/cw

				go 1.24

				require (
					github.com/sirupsen/logrus v1.9.3
					github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.73.3
				)

				require (
					github.com/aws/aws-sdk-go-v2 v1.47.3 // indirect
					github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 // indirect
					github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 // indirect
					github.com/aws/smithy-go v1.28.5 // indirect
					golang.org/x/sys v0.13.0 // indirect
				)
			`), []golang.GoResolvedDependency{
				{ModulePath: "example.com/cw", Main: true},
				{ModulePath: "github.com/aws/aws-sdk-go", Version: "v1.55.8"},
				{ModulePath: "github.com/sirupsen/logrus", Version: "v1.9.3"},
				{ModulePath: "golang.org/x/sys", Version: "v0.13.0", Indirect: true},
			}, []golang.GoPackageModule{
				{ImportPath: "github.com/aws/aws-sdk-go/service/cloudwatch", ModulePath: "github.com/aws/aws-sdk-go"},
				{ImportPath: "github.com/sirupsen/logrus", ModulePath: "github.com/sirupsen/logrus"},
				{ImportPath: "golang.org/x/sys/unix", ModulePath: "golang.org/x/sys"},
			}),
			test.Golang(cloudwatchV1Source),
		),
	)
}
