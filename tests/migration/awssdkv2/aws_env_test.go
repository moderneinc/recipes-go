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
// client takes a config that is nil or sets that region. v1 merged a config's
// settings over the session's and skipped a nil one, and so does the generated
// merge the client is built from.
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

				return kms.NewFromConfig(awsCompatMergeConfig(sess, config)), nil
			}
		`),
		test.Generated("zz_awssdkv2_imds.go", imdsHelpersIn("environment")),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("environment")),
	)
}

// A config local written to field by field stays v2's config, merged over the
// session's as v1 merged it.
func TestMigrateClientConfigWrittenFieldByField(t *testing.T) {
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
		`, `
			package environment

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/kms"
			)

			func New(sess aws.Config, region string) *kms.Client {
				config := &aws.Config{Region: region}
				config.RetryMaxAttempts = 3
				return kms.NewFromConfig(awsCompatMergeConfig(sess, config))
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("environment")),
	)
}

// A setting v2 keeps somewhere other than the config has no place on the local.
func TestMigrateBlockedByClientConfigFieldV2Moved(t *testing.T) {
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
				config.DisableSSL = aws.Bool(true)
				return kms.New(sess, config)
			}
		`),
	)
}
