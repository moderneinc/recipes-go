/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// v1 counted retries and v2 counts attempts, so every value crossing over gains
// one — v1's -1, the service default, landing on v2's 0, which means the same —
// and a read of it loses one again.
func TestMigrateMaxRetriesCountsAttempts(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func sessions(retries int) {
				some := session.Must(session.NewSession(&aws.Config{MaxRetries: aws.Int(3)}))
				never := session.Must(session.NewSession(&aws.Config{MaxRetries: aws.Int(0)}))
				defaulted := session.Must(session.NewSession(&aws.Config{MaxRetries: aws.Int(-1)}))
				computed := session.Must(session.NewSession(&aws.Config{MaxRetries: aws.Int(retries * 2)}))
				_, _, _, _ = some, never, defaulted, computed
			}

			func readBack(retries int) int {
				sess := session.Must(session.NewSession())
				sess.Config.MaxRetries = aws.Int(retries)
				return *sess.Config.MaxRetries * 2
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/config"
			)

			func sessions(retries int) {
				some, err := config.LoadDefaultConfig(context.TODO(), config.WithRetryMaxAttempts(4))
				if err != nil {
					panic(err)
				}
				never, err := config.LoadDefaultConfig(context.TODO(), config.WithRetryMaxAttempts(1))
				if err != nil {
					panic(err)
				}
				defaulted, err := config.LoadDefaultConfig(context.TODO(), config.WithRetryMaxAttempts(0))
				if err != nil {
					panic(err)
				}
				computed, err := config.LoadDefaultConfig(context.TODO(), config.WithRetryMaxAttempts(retries*2+1))
				if err != nil {
					panic(err)
				}
				_, _, _, _ = some, never, defaulted, computed
			}

			func readBack(retries int) int {
				sess, err := config.LoadDefaultConfig(context.TODO())
				if err != nil {
					panic(err)
				}
				sess.RetryMaxAttempts = retries + 1
				return (sess.RetryMaxAttempts - 1) * 2
			}
		`),
	)
}

// A read becomes v2's value less one, which cannot be written through.
func TestMigrateBlockedByMaxRetriesWrittenThroughPointer(t *testing.T) {
	for name, write := range map[string]string{
		"assigned":    "*sess.Config.MaxRetries = 2",
		"incremented": "*sess.Config.MaxRetries++",
		"compound":    "*sess.Config.MaxRetries += 2",
	} {
		t.Run(name, func(t *testing.T) {
			test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
				test.Golang(`
					package main

					import (
						"github.com/aws/aws-sdk-go/aws"
						"github.com/aws/aws-sdk-go/aws/session"
					)

					func f() {
						sess := session.Must(session.NewSession(&aws.Config{MaxRetries: aws.Int(1)}))
						`+write+`
					}
				`),
			)
		})
	}
}
