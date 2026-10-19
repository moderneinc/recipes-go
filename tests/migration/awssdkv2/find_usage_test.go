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

// A sub-package v2 moved or dropped is reported with where it went, not as a
// service the manifest does not know.
func TestFindReportsRelocatedSubpackages(t *testing.T) {
	findSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				_ "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
				_ "github.com/aws/aws-sdk-go/service/dynamodb/expression"
				_ "github.com/aws/aws-sdk-go/service/rds/rdsutils"
				_ "github.com/aws/aws-sdk-go/service/s3/s3crypto"
			)
		`, `
			package main

			import (
				/*~~(migrates to a port of this package generated under internal/awscompat/dynamodbattribute, which stores and reads items as v1 did; v2's attributevalue ignores json struct tags and stores empty values as themselves where v1 stored NULL. A type implementing its Marshaler or Unmarshaler holds the package on v1)~~>*/_ "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
				/*~~(migrates to github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression)~~>*/_ "github.com/aws/aws-sdk-go/service/dynamodb/expression"
				/*~~(migrates to github.com/aws/aws-sdk-go-v2/feature/rds/auth)~~>*/_ "github.com/aws/aws-sdk-go/service/rds/rdsutils"
				/*~~(s3crypto has no v2 package: move to github.com/aws/amazon-s3-encryption-client-go/v3, whose default policy reads and writes the format v1's V2 client did, and set EnableLegacyUnauthenticatedModes to read objects v1's original client wrote; move on to v4's key-committing format only once every reader of the bucket is on v3)~~>*/_ "github.com/aws/aws-sdk-go/service/s3/s3crypto"
			)
		`),
	)
}

// The connection-string builder is the part of rdsutils v2 dropped, so it is
// called out where it is used, with what to write instead.
func TestFindReportsRdsConnectionStringBuilder(t *testing.T) {
	findSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/service/rds/rdsutils"
			)

			func dsn(creds *credentials.Credentials) (string, error) {
				return rdsutils.NewConnectionStringBuilder("db:3306", "us-east-1", "app", "orders", creds).Build()
			}
		`, `
			package main

			import (
				/*~~(migrates to github.com/aws/aws-sdk-go-v2/credentials)~~>*/"github.com/aws/aws-sdk-go/aws/credentials"
				/*~~(migrates to github.com/aws/aws-sdk-go-v2/feature/rds/auth)~~>*/"github.com/aws/aws-sdk-go/service/rds/rdsutils"
			)

			func dsn(creds *credentials.Credentials) (string, error) {
				return /*~~(rdsutils.NewConnectionStringBuilder has no v2 counterpart: build the DSN by hand, with the token from github.com/aws/aws-sdk-go-v2/feature/rds/auth.BuildAuthToken as the password and the tls and allowCleartextPasswords parameters v1's builder set)~~>*/rdsutils.NewConnectionStringBuilder("db:3306", "us-east-1", "app", "orders", creds).Build()
			}
		`),
	)
}
