/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// A package whose exported API takes a v1 type stays on v1, so a package
// importing it stays too: migrating it alone would hand v2's types to a
// function still taking v1's.
func TestMigrateHoldsBackAcrossPackageBoundary(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.24

				require github.com/aws/aws-sdk-go v1.55.8
			`),
			test.Golang(`
				package store

				import (
					"github.com/aws/aws-sdk-go/aws/session"
					"github.com/aws/aws-sdk-go/service/s3"
				)

				func Size(sess *session.Session, out *s3.HeadObjectOutput) int64 {
					if sess == nil {
						return 0
					}
					return *out.ContentLength
				}
			`).WithPath("store/store.go"),
			test.Golang(`
				package main

				import (
					"example.com/app/store"
					"github.com/aws/aws-sdk-go/aws"
					"github.com/aws/aws-sdk-go/aws/session"
					"github.com/aws/aws-sdk-go/service/s3"
				)

				func main() {
					sess := session.Must(session.NewSession())
					out, _ := s3.New(sess).HeadObject(&s3.HeadObjectInput{Bucket: aws.String("b"), Key: aws.String("k")})
					store.Size(sess, out)
				}
			`).WithPath("main.go"),
		),
	)
}
