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
				switch string(awsCompatPtrs(resp.Reservations)[0].Instances[0].State.Name) {
				case string(ec2types.InstanceStateNameRunning):
					return true, nil
				}
				return false, nil
			}
		`),
		test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("main")),
	)
}
