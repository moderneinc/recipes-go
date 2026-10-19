/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// aws/aws-node-termination-handler's shape: v1's DefaultRetryer tuned through
// its fields, wrapped in a type whose ShouldRetry also retries a reset
// connection. v2's standard retryer takes v1's attempts and its larger maximum
// delay, with no retry budget, and consults the added condition first, so it
// retries where v1's `||` did. v1's minimum delays have no v2 counterpart.
func TestMigrateCustomRetryer(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package sqsevent

			import (
				"strings"
				"time"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/client"
				"github.com/aws/aws-sdk-go/aws/request"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/sqs"
			)

			type SqsRetryer struct {
				client.DefaultRetryer
			}

			func (r SqsRetryer) ShouldRetry(req *request.Request) bool {
				return r.DefaultRetryer.ShouldRetry(req) ||
					(req.Error != nil && strings.Contains(req.Error.Error(), "connection reset"))
			}

			func GetSqsClient(sess *session.Session) *sqs.SQS {
				return sqs.New(sess, &aws.Config{
					Retryer: SqsRetryer{
						DefaultRetryer: client.DefaultRetryer{
							NumMaxRetries:    client.DefaultRetryerMaxNumRetries,
							MinRetryDelay:    client.DefaultRetryerMinRetryDelay,
							MaxRetryDelay:    1200 * time.Millisecond,
							MinThrottleDelay: client.DefaultRetryerMinThrottleDelay,
							MaxThrottleDelay: 1200 * time.Millisecond,
						},
					},
				})
			}
		`, `
			package sqsevent

			import (
				"strings"
				"time"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
				"github.com/aws/aws-sdk-go-v2/aws/retry"
				"github.com/aws/aws-sdk-go-v2/service/sqs"
			)

			type SqsRetryer struct{}

			func (r SqsRetryer) IsErrorRetryable(err error) aws.Ternary {
				if err != nil && strings.Contains(err.Error(), "connection reset") {
					return aws.TrueTernary
				}
				return aws.UnknownTernary
			}

			func GetSqsClient(sess aws.Config) *sqs.Client {
				return sqs.NewFromConfig(awsCompatMergeConfig(sess, &aws.Config{
					Retryer: func() aws.Retryer {
						return retry.NewStandard(func(o *retry.StandardOptions) {
							o.MaxAttempts = 4
							o.MaxBackoff = 1200 * time.Millisecond
							o.RateLimiter = ratelimit.None
							o.Retryables = append([]retry.IsErrorRetryable{SqsRetryer{}}, o.Retryables...)
						})
					},
				}))
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("sqsevent")),
	)
}
