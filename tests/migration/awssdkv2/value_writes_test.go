/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// kayac/sqsjkr's test: an attribute written into a message and read back. v2
// holds the attributes by value, and the pointers a read is handed back are
// copies, so the write goes into v2's own map as the value it pointed at.
func TestMigrateWriteIntoValueMap(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package sqsjkr

			import "github.com/aws/aws-sdk-go/service/sqs"

			func stamp(msg *sqs.Message, ts string) string {
				msg.Attributes["SentTimestamp"] = &ts
				return *msg.Attributes["SentTimestamp"]
			}
		`, `
			package sqsjkr

			import sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

			func stamp(msg *sqstypes.Message, ts string) string {
				msg.Attributes["SentTimestamp"] = ts
				return *awsCompatPtrsMap(msg.Attributes)["SentTimestamp"]
			}
		`),
		test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("sqsjkr")),
	)
}

// A pointer that may be nil has no value to store, and a write through a
// pointer handed back would land in a copy.
func TestMigrateBlockedByLossyValueMapWrites(t *testing.T) {
	for name, write := range map[string]string{
		"maybe nil": `msg.Attributes["SentTimestamp"] = p`,
		"through":   `*msg.Attributes["SentTimestamp"] = "1"`,
	} {
		t.Run(name, func(t *testing.T) {
			test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
				test.Golang(`
					package sqsjkr

					import "github.com/aws/aws-sdk-go/service/sqs"

					func stamp(msg *sqs.Message, p *string) {
						`+write+`
					}
				`),
			)
		})
	}
}

// nil is a list in v2 as it was in v1; a conversion of it could not infer the
// element type.
func TestMigrateNilValueSlice(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "github.com/aws/aws-sdk-go/service/ec2"

			var empty = ec2.DescribeInstancesOutput{
				Reservations: nil,
			}
		`, `
			package main

			import "github.com/aws/aws-sdk-go-v2/service/ec2"

			var empty = ec2.DescribeInstancesOutput{
				Reservations: nil,
			}
		`),
	)
}

// in4it/ecs-deploy's shape: a field named like a DynamoDB union member on the
// program's own struct, in a file using DynamoDB. Its type says it is no union.
func TestMigrateOwnFieldNamedLikeUnionMember(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package service

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/dynamodb"
			)

			type element struct{ C, S string }

			func find(elements []*element, name string) *element {
				for _, a := range elements {
					if a.S == name {
						return a
					}
				}
				return nil
			}

			func input(table string) *dynamodb.ScanInput {
				return &dynamodb.ScanInput{TableName: aws.String(table)}
			}
		`, `
			package service

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/dynamodb"
			)

			type element struct{ C, S string }

			func find(elements []*element, name string) *element {
				for _, a := range elements {
					if a.S == name {
						return a
					}
				}
				return nil
			}

			func input(table string) *dynamodb.ScanInput {
				return &dynamodb.ScanInput{TableName: aws.String(table)}
			}
		`),
	)
}

// A pointer appended to a list v2 holds by value and written through after:
// v1's list saw the write, v2's copy would not.
func TestMigrateAppendedPointerUsedAfterHeldBack(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package ecs

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/ecs"
			)

			func add(input *ecs.RegisterTaskDefinitionInput, name string) {
				def := &ecs.ContainerDefinition{Name: aws.String(name)}
				input.ContainerDefinitions = append(input.ContainerDefinitions, def)
				def.Essential = aws.Bool(true)
			}
		`),
	)
}

// One appended as the last thing done with it carries over as its value.
func TestMigrateAppendedPointerLastUse(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package ecs

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/ecs"
			)

			func add(input *ecs.RegisterTaskDefinitionInput, name string) {
				def := &ecs.ContainerDefinition{Name: aws.String(name)}
				def.Essential = aws.Bool(true)
				input.ContainerDefinitions = append(input.ContainerDefinitions, def)
			}
		`, `
			package ecs

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/ecs"
				ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
			)

			func add(input *ecs.RegisterTaskDefinitionInput, name string) {
				def := &ecstypes.ContainerDefinition{Name: aws.String(name)}
				def.Essential = aws.Bool(true)
				input.ContainerDefinitions = awsCompatVals(append(awsCompatPtrs(input.ContainerDefinitions), def))
			}
		`),
		test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("ecs")),
	)
}

// contentful-labs/kube-secret-syncer's tests: a map of lists filled in a
// literal, and read back. Both levels are held by value in v2.
func TestMigrateMapOfValueSlicesLiteral(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package syncer

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/secretsmanager"
			)

			func current(stages map[string][]*string) string {
				for version, names := range stages {
					for _, name := range names {
						if *name == "AWSCURRENT" {
							return version
						}
					}
				}
				return ""
			}

			func entry() string {
				e := &secretsmanager.SecretListEntry{
					SecretVersionsToStages: map[string][]*string{
						"v1": {aws.String("AWSCURRENT")},
					},
				}
				return current(e.SecretVersionsToStages)
			}
		`, `
			package syncer

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				secretsmanagertypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
			)

			func current(stages map[string][]*string) string {
				for version, names := range stages {
					for _, name := range names {
						if *name == "AWSCURRENT" {
							return version
						}
					}
				}
				return ""
			}

			func entry() string {
				e := &secretsmanagertypes.SecretListEntry{
					SecretVersionsToStages: awsCompatValsSliceMap(map[string][]*string{
						"v1": {aws.String("AWSCURRENT")},
					}),
				}
				return current(awsCompatPtrsSliceMap(e.SecretVersionsToStages))
			}
		`),
		test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("syncer")),
	)
}
