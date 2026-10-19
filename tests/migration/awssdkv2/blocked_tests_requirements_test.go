/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// A test commented out because the rewrite cannot carry it over leaves its
// package migrating, so go.mod requires what the package now imports.
func TestMigrateCommentedOutTestKeepsRequirements(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoModuleToV2{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.21

				require github.com/aws/aws-sdk-go v1.55.8
			`, `
				module example.com/app

				go 1.24

				require (
					github.com/aws/aws-sdk-go-v2/config v1.33.9
					github.com/aws/aws-sdk-go-v2/service/sqs v1.52.4
				)

				require (
					github.com/aws/aws-sdk-go-v2 v1.47.3 // indirect
					github.com/aws/aws-sdk-go-v2/credentials v1.20.9 // indirect
					github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.20.3 // indirect
					github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 // indirect
					github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 // indirect
					github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.6 // indirect
					github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.21 // indirect
					github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.6 // indirect
					github.com/aws/aws-sdk-go-v2/service/signin v1.10.4 // indirect
					github.com/aws/aws-sdk-go-v2/service/sso v1.38.4 // indirect
					github.com/aws/aws-sdk-go-v2/service/ssooidc v1.43.4 // indirect
					github.com/aws/aws-sdk-go-v2/service/sts v1.51.4 // indirect
					github.com/aws/smithy-go v1.28.5 // indirect
				)
			`),
			test.Golang(`
				package queue

				import (
					"github.com/aws/aws-sdk-go/aws/session"
					"github.com/aws/aws-sdk-go/service/sqs"
				)

				func Client(sess *session.Session) *sqs.SQS {
					return sqs.New(sess)
				}
			`, `
				package queue

				import (
					"github.com/aws/aws-sdk-go-v2/aws"
					"github.com/aws/aws-sdk-go-v2/service/sqs"
				)

				func Client(sess aws.Config) *sqs.Client {
					return sqs.NewFromConfig(sess)
				}
			`).WithPath("queue/client.go"),
			test.Golang(`
				package queue

				import (
					"testing"

					"github.com/aws/aws-sdk-go/aws/request"
				)

				func TestRetries(t *testing.T) {
					req := &request.Request{}
					if req.Retryable != nil {
						t.Fatal("retryable")
					}
				}
			`, `
				package queue

				// MigrateAwsSdkGoToV2 commented this test out: request.Request: the v1 request plumbing has no v2 counterpart beyond a session's Send and UnmarshalError handlers.
				// It tests aws-sdk-go v1 in a way v2 has no counterpart for; port it to the
				// migrated code and restore it.
				//
				// import (
				// 	"testing"
				//
				// 	"github.com/aws/aws-sdk-go/aws/request"
				// )
				//
				// func TestRetries(t *testing.T) {
				// 	req := &request.Request{}
				// 	if req.Retryable != nil {
				// 		t.Fatal("retryable")
				// 	}
				// }
			`).WithPath("queue/client_test.go"),
		),
	)
}
