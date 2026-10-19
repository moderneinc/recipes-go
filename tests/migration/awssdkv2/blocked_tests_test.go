/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// A test exercising v1's request machinery has nothing in v2 to port to, so
// rather than hold back the package it tests, it is commented out for a person
// to port.
func TestMigrateCommentsOutUnportableTest(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
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
	)
}

// Where another file uses a helper the test declares, commenting it out would
// break that file, so the package stays on v1.
func TestMigrateHoldsBackTestOthersUse(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package queue

			import (
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/sqs"
			)

			func Client(sess *session.Session) *sqs.SQS {
				return sqs.New(sess)
			}
		`).WithPath("queue/client.go"),
		test.Golang(`
			package queue

			import "github.com/aws/aws-sdk-go/aws/request"

			func newRequest() *request.Request {
				return &request.Request{}
			}
		`).WithPath("queue/helpers_test.go"),
		test.Golang(`
			package queue

			import "testing"

			func TestRetries(t *testing.T) {
				if newRequest().Retryable != nil {
					t.Fatal("retryable")
				}
			}
		`).WithPath("queue/client_test.go"),
	)
}
