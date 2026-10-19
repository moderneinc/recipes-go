/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// v1's enum constants were untyped strings and v2's are typed. aidansteele/jwtex
// keeps them in a map of strings, getsocial-rnd/ecs-drain-lambda compares them
// against fields v2 still holds as strings: outside a field v2 types as the
// enum, the constant converts back to the string it was used as.
func TestMigrateEnumConstantsUsedAsStrings(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"crypto"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/ecs"
				"github.com/aws/aws-sdk-go/service/kms"
			)

			var algorithms = map[crypto.Hash]string{
				crypto.SHA256: kms.SigningAlgorithmSpecRsassaPkcs1V15Sha256,
			}

			func draining(instance *ecs.ContainerInstance) bool {
				return *instance.Status == ecs.ContainerInstanceStatusDraining
			}

			func status() *string {
				return aws.String(ecs.ContainerInstanceStatusDraining)
			}
		`, `
			package main

			import (
				"crypto"

				"github.com/aws/aws-sdk-go-v2/aws"
				ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
				kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
			)

			var algorithms = map[crypto.Hash]string{
				crypto.SHA256: string(kmstypes.SigningAlgorithmSpecRsassaPkcs1V15Sha256),
			}

			func draining(instance *ecstypes.ContainerInstance) bool {
				return *instance.Status == string(ecstypes.ContainerInstanceStatusDraining)
			}

			func status() *string {
				return aws.String(string(ecstypes.ContainerInstanceStatusDraining))
			}
		`),
	)
}

// A shape's enum field takes the typed constant, the aws.String around it
// gone, while a read of one compares as the string v1 dereferenced it to. The
// read reaches the field through list elements, which v2 holds by value.
func TestMigrateEnumConstantsInEnumSlotsAndReads(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/ec2"
			)

			func filter() *ec2.InstanceStatus {
				return &ec2.InstanceStatus{
					InstanceState: &ec2.InstanceState{Name: aws.String(ec2.InstanceStateNameRunning)},
				}
			}

			func running(svc *ec2.EC2) (bool, error) {
				resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{})
				if err != nil {
					return false, err
				}
				switch *resp.Reservations[0].Instances[0].State.Name {
				case ec2.InstanceStateNameRunning:
					return true, nil
				}
				return false, nil
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/ec2"
				ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
			)

			func filter() *ec2types.InstanceStatus {
				return &ec2types.InstanceStatus{
					InstanceState: &ec2types.InstanceState{Name: ec2types.InstanceStateNameRunning},
				}
			}

			func running(svc *ec2.Client) (bool, error) {
				resp, err := svc.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{})
				if err != nil {
					return false, err
				}
				switch string(awsCompatPtrs(awsCompatPtrs(resp.Reservations)[0].Instances)[0].State.Name) {
				case string(ec2types.InstanceStateNameRunning):
					return true, nil
				}
				return false, nil
			}
		`),
		test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("main")),
	)
}

// aws/aws-node-termination-handler's shape: a constant of one enum in a list of
// another, which v1's *string list took by its string. A conversion hands v2's
// typed list the same string.
func TestMigrateEnumConstantOfAnotherEnum(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/sqs"
			)

			var input = &sqs.ReceiveMessageInput{
				AttributeNames: []*string{aws.String(sqs.MessageSystemAttributeNameSentTimestamp)},
			}
		`, `
			package main

			import (
				"github.com/aws/aws-sdk-go-v2/service/sqs"
				sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
			)

			var input = &sqs.ReceiveMessageInput{
				AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeName(sqstypes.MessageSystemAttributeNameSentTimestamp)},
			}
		`),
	)
}

// v2 renamed some shapes: autoscaling's InstanceDetails is AutoScalingInstanceDetails.
func TestMigrateRenamedShape(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/autoscaling"
			)

			var details = &autoscaling.InstanceDetails{InstanceId: aws.String("i-1")}
		`, `
			package main

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				autoscalingtypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
			)

			var details = &autoscalingtypes.AutoScalingInstanceDetails{InstanceId: aws.String("i-1")}
		`),
	)
}

// hypnoglow/helm-s3's shape: an enum field set from a function returning the
// *string v1 typed it with, nil when unset. v2 reads it through aws.ToString,
// nil becoming the empty enum v2 leaves unset.
func TestMigrateEnumFieldFromStringPointer(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package awss3

			import (
				"os"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func getSSE() *string {
				sse := os.Getenv("AWS_S3_SSE")
				if sse == "" {
					return nil
				}
				return &sse
			}

			func input(bucket, key string) *s3.PutObjectInput {
				return &s3.PutObjectInput{
					Bucket:               aws.String(bucket),
					Key:                  aws.String(key),
					ServerSideEncryption: getSSE(),
				}
			}
		`, `
			package awss3

			import (
				"os"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
				s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
			)

			func getSSE() *string {
				sse := os.Getenv("AWS_S3_SSE")
				if sse == "" {
					return nil
				}
				return &sse
			}

			func input(bucket, key string) *s3.PutObjectInput {
				return &s3.PutObjectInput{
					Bucket:               aws.String(bucket),
					Key:                  aws.String(key),
					ServerSideEncryption: s3types.ServerSideEncryption(aws.ToString(getSSE())),
				}
			}
		`),
	)
}

// in4it/ecs-deploy's shape: a union shape v2 made an interface, set to one
// field's literal. It becomes that field's member type, holding the shape by
// value.
func TestMigrateUnionLiteralOfOneMember(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package ecs

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/appmesh"
			)

			func spec(host string) *appmesh.VirtualNodeSpec {
				return &appmesh.VirtualNodeSpec{
					ServiceDiscovery: &appmesh.ServiceDiscovery{
						Dns: &appmesh.DnsServiceDiscovery{
							Hostname: aws.String(host),
						},
					},
				}
			}
		`, `
			package ecs

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				appmeshtypes "github.com/aws/aws-sdk-go-v2/service/appmesh/types"
			)

			func spec(host string) *appmeshtypes.VirtualNodeSpec {
				return &appmeshtypes.VirtualNodeSpec{
					ServiceDiscovery: &appmeshtypes.ServiceDiscoveryMemberDns{
						Value: appmeshtypes.DnsServiceDiscovery{
							Hostname: aws.String(host),
						},
					},
				}
			}
		`),
	)
}

// A union set to two fields has no single member to become.
func TestMigrateUnionLiteralOfTwoMembersHeldBack(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package ecs

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/appmesh"
			)

			func discovery(host, ns string) *appmesh.ServiceDiscovery {
				return &appmesh.ServiceDiscovery{
					Dns:         &appmesh.DnsServiceDiscovery{Hostname: aws.String(host)},
					AwsCloudMap: &appmesh.AwsCloudMapServiceDiscovery{NamespaceName: aws.String(ns)},
				}
			}
		`),
	)
}

// in4it/ecs-deploy's shape: an enum list built at runtime from the program's
// own strings. v2 types the list with the enum, which a generated conversion
// builds, nil staying nil.
func TestMigrateEnumListFromStrings(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package ecs

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/appmesh"
			)

			func policy(events []string) *appmesh.HttpRetryPolicy {
				return &appmesh.HttpRetryPolicy{
					MaxRetries:     aws.Int64(3),
					TcpRetryEvents: aws.StringSlice(events),
				}
			}
		`, `
			package ecs

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				appmeshtypes "github.com/aws/aws-sdk-go-v2/service/appmesh/types"
			)

			func policy(events []string) *appmeshtypes.HttpRetryPolicy {
				return &appmeshtypes.HttpRetryPolicy{
					MaxRetries:     aws.Int64(3),
					TcpRetryEvents: awsCompatEnumsOf[appmeshtypes.TcpRetryPolicyEvent](events),
				}
			}
		`),
		test.Generated("zz_awssdkv2_enums.go", `
			// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

			package ecs

			// awsCompatEnumsOf converts a list of strings to the enum list aws-sdk-go-v2
			// types a field with, where v1 took the strings. nil stays nil.
			func awsCompatEnumsOf[E ~string](values []string) []E {
				if values == nil {
					return nil
				}
				out := make([]E, len(values))
				for i, v := range values {
					out[i] = E(v)
				}
				return out
			}
		`),
	)
}

// in4it/ecs-deploy's shape: a union with a single member, held in a list of
// pointers. v2's union is an interface, which the list holds without one.
func TestMigrateSingleMemberUnionList(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package ecs

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/appmesh"
			)

			func backends(names []string) []*appmesh.Backend {
				var out []*appmesh.Backend
				for _, name := range names {
					out = append(out, &appmesh.Backend{
						VirtualService: &appmesh.VirtualServiceBackend{
							VirtualServiceName: aws.String(name),
						},
					})
				}
				return out
			}
		`, `
			package ecs

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				appmeshtypes "github.com/aws/aws-sdk-go-v2/service/appmesh/types"
			)

			func backends(names []string) []appmeshtypes.Backend {
				var out []appmeshtypes.Backend
				for _, name := range names {
					out = append(out, &appmeshtypes.BackendMemberVirtualService{
						Value: appmeshtypes.VirtualServiceBackend{
							VirtualServiceName: aws.String(name),
						},
					})
				}
				return out
			}
		`),
	)
}

// keikoproj/governor's shape: an enum field read through aws.StringValue. v2
// holds the enum by value, which is the string the helper answered.
func TestMigrateEnumReadThroughStringValue(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package reaper

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/ec2"
			)

			func state(instance *ec2.Instance) string {
				return aws.StringValue(instance.State.Name)
			}
		`, `
			package reaper

			import ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

			func state(instance *ec2types.Instance) string {
				return string(instance.State.Name)
			}
		`),
	)
}
