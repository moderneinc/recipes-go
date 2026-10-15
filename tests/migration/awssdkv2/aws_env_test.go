/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// telia-oss/aws-env's shape: the region comes from instance metadata when the
// environment has none, after checking the metadata service is there, and each
// client takes a config that is nil or sets only that region. v1 skipped a nil
// config, and so does the option replacing it.
func TestMigrateMetadataRegionAndOptionalClientConfig(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package environment

			import (
				"errors"
				"os"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/ec2metadata"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/kms"
			)

			func New(sess *session.Session) (*kms.KMS, error) {
				var config *aws.Config

				if os.Getenv("AWS_REGION") == "" {
					metadata := ec2metadata.New(sess)
					if !metadata.Available() {
						return nil, errors.New("no region")
					}
					region, err := metadata.Region()
					if err != nil {
						return nil, err
					}
					config = &aws.Config{Region: aws.String(region)}
				}

				return kms.New(sess, config), nil
			}
		`, `
			package environment

			import (
				"context"
				"errors"
				"os"

				"github.com/aws/aws-sdk-go-v2/aws"
				ec2metadata "github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
				"github.com/aws/aws-sdk-go-v2/service/kms"
			)

			func New(sess aws.Config) (*kms.Client, error) {
				var config *aws.Config

				if os.Getenv("AWS_REGION") == "" {
					metadata := ec2metadata.NewFromConfig(sess)
					if !awsCompatIMDSAvailable(context.TODO(), metadata) {
						return nil, errors.New("no region")
					}
					region, err := awsCompatIMDSRegion(context.TODO(), metadata)
					if err != nil {
						return nil, err
					}
					config = &aws.Config{Region: region}
				}

				return kms.NewFromConfig(sess, func(o *kms.Options) {
					if config != nil {
						o.Region = config.Region
					}
				}), nil
			}
		`),
		test.Generated("zz_awssdkv2_imds.go", imdsHelpersIn("environment")),
	)
}

// A config local used any other way than as a client's trailing config could
// carry settings the option does not, so its client is left alone.
func TestMigrateBlockedByClientConfigUsedElsewhere(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package environment

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/kms"
			)

			func New(sess *session.Session, region string) *kms.KMS {
				config := &aws.Config{Region: aws.String(region)}
				config.MaxRetries = aws.Int(2)
				return kms.New(sess, config)
			}
		`),
	)
}
