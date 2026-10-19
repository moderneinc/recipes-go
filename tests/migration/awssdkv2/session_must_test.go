/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// Only `x := session.Must(…)` has statements to expand into. Anywhere else the
// load has nowhere to return its error to, so Must becomes the generated loader,
// which panics on a failure as Must did.
func TestMigrateSessionMustInExpression(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/sqs"
			)

			var shared = session.Must(session.NewSession())

			func returned(region string) *session.Session {
				return session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
			}

			func passed() *sqs.SQS {
				return sqs.New(session.Must(session.NewSession()))
			}

			func reassigned() *session.Session {
				var sess *session.Session
				sess = session.Must(session.NewSession())
				return sess
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/sqs"
			)

			var shared = awsCompatConfig(context.TODO())

			func returned(region string) aws.Config {
				return awsCompatConfig(context.TODO(), config.WithRegion(region))
			}

			func passed() *sqs.Client {
				return sqs.NewFromConfig(awsCompatConfig(context.TODO()))
			}

			func reassigned() aws.Config {
				var sess aws.Config
				sess = awsCompatConfig(context.TODO())
				return sess
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("main")),
	)
}

// A session built from a config the caller already held cannot fail to load,
// so Must around it is that config.
func TestMigrateSessionMustAroundCarriedConfig(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package discovery

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/servicediscovery"
			)

			func client(cfg *aws.Config) *servicediscovery.ServiceDiscovery {
				return servicediscovery.New(session.Must(session.NewSessionWithOptions(session.Options{
					Config: *cfg,
				})))
			}
		`, `
			package discovery

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
			)

			func client(cfg *aws.Config) *servicediscovery.Client {
				return servicediscovery.NewFromConfig(*cfg)
			}
		`),
	)
}
