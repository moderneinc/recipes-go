/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// envato/ejsonkms's shape: a session's region set by calling its config's
// setter for the effect, and a client given a config built as a NewConfig
// chain. The setter is v2's field assignment, and the chain the literal merged
// over the session's config.
func TestMigrateSessionConfigSetterAndChainClientConfig(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"os"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/kms"
			)

			func newKmsClient(awsRegion string) *kms.KMS {
				awsSession := session.Must(session.NewSession())
				awsSession.Config.WithRegion(awsRegion)
				fakeKmsEndpoint := os.Getenv("FAKE_AWSKMS_URL")
				if len(fakeKmsEndpoint) != 0 {
					return kms.New(awsSession, aws.NewConfig().WithEndpoint(fakeKmsEndpoint))
				}
				return kms.New(awsSession)
			}
		`, `
			package main

			import (
				"context"
				"os"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/kms"
			)

			func newKmsClient(awsRegion string) *kms.Client {
				awsSession, err := config.LoadDefaultConfig(context.TODO())
				if err != nil {
					panic(err)
				}
				awsSession.Region = awsRegion
				fakeKmsEndpoint := os.Getenv("FAKE_AWSKMS_URL")
				if len(fakeKmsEndpoint) != 0 {
					return kms.NewFromConfig(awsCompatMergeConfig(awsSession, &aws.Config{
						BaseEndpoint: aws.String(awsCompatEndpoint(fakeKmsEndpoint)),
					}))
				}
				return kms.NewFromConfig(awsSession)
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("main")),
	)
}
