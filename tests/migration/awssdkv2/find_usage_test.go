/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func findSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&awssdkv2.FindAwsSdkGoV1Usage{})
}

// The constructs with no mechanical v2 form are each reported with what
// replaces them, so a hand migration has the list. A package v2 dropped outright
// is called out at its import rather than at every reference to it.
func TestFindReportsUnmigratableConstructs(t *testing.T) {
	findSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws/awserr"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func list(ctx context.Context) error {
				sess, err := session.NewSession()
				if err != nil {
					return err
				}
				svc := s3.New(sess)
				if err := svc.ListObjectsPages(&s3.ListObjectsInput{}, nil); err != nil {
					var aerr awserr.Error
					return aerr
				}
				return svc.WaitUntilBucketExists(&s3.HeadBucketInput{})
			}
		`, `
			package main

			import (
				"context"

				/*~~(awserr was replaced by smithy's typed errors: an assertion to awserr.Error becomes errors.As against smithy.APIError)~~>*/"github.com/aws/aws-sdk-go/aws/awserr"
				/*~~(the v1 session is gone: load a config with github.com/aws/aws-sdk-go-v2/config.LoadDefaultConfig(ctx) and build clients from it)~~>*/"github.com/aws/aws-sdk-go/aws/session"
				/*~~(migrates to github.com/aws/aws-sdk-go-v2/service/s3)~~>*/"github.com/aws/aws-sdk-go/service/s3"
			)

			func list(ctx context.Context) error {
				sess, err := /*~~(build a config instead: config.LoadDefaultConfig(ctx, config.WithRegion(region)), then pass it to each client's NewFromConfig)~~>*/session.NewSession()
				if err != nil {
					return err
				}
				svc := /*~~(a v2 client is built from a config: New(sess) became NewFromConfig(cfg))~~>*/s3.New(sess)
				if err := /*~~(ListObjectsPages became a paginator: build one with NewListObjectsPaginator(client, params) and loop while HasMorePages())~~>*/svc.ListObjectsPages(&s3.ListObjectsInput{}, nil); err != nil {
					var aerr awserr.Error
					return aerr
				}
				return /*~~(WaitUntilBucketExists became a waiter type: build one with BucketExistsWaiter and call Wait(ctx, params, maxWait))~~>*/svc.WaitUntilBucketExists(&s3.HeadBucketInput{})
			}
		`),
	)
}

// No AWS SDK import: nothing reported.
func TestFindQuietOnUnrelatedFile(t *testing.T) {
	findSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func run(ctx context.Context) {}
		`),
	)
}
