/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
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

// A delay converted from a count of milliseconds carries over as written, and
// stays the conversion the parser reads it as rather than a call.
func TestMigrateRetryerWithConvertedDelay(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package sqsevent

			import (
				"time"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/client"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/sqs"
			)

			func GetSqsClient(sess *session.Session, delayMs int) *sqs.SQS {
				return sqs.New(sess, &aws.Config{
					Retryer: client.DefaultRetryer{
						NumMaxRetries:    3,
						MaxRetryDelay:    time.Duration(delayMs) * time.Millisecond,
						MaxThrottleDelay: time.Duration(delayMs) * time.Millisecond,
					},
				})
			}
		`, `
			package sqsevent

			import (
				"time"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
				"github.com/aws/aws-sdk-go-v2/aws/retry"
				"github.com/aws/aws-sdk-go-v2/service/sqs"
			)

			func GetSqsClient(sess aws.Config, delayMs int) *sqs.Client {
				return sqs.NewFromConfig(awsCompatMergeConfig(sess, &aws.Config{
					Retryer: func() aws.Retryer {
						return retry.NewStandard(func(o *retry.StandardOptions) {
							o.MaxAttempts = 4
							o.MaxBackoff = time.Duration(delayMs) * time.Millisecond
							o.RateLimiter = ratelimit.None
						})
					},
				}))
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("sqsevent")),
	)
}

// A delay converted to a type the file declares stays the conversion the parser
// reads it as, which a file declaring nothing would read as a call.
func TestMigrateRetryerWithDelayOfTheFilesOwnType(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package sqsevent

			import (
				"time"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/client"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/sqs"
			)

			type delay = time.Duration

			func GetSqsClient(sess *session.Session, delayMs int) *sqs.SQS {
				return sqs.New(sess, &aws.Config{
					Retryer: client.DefaultRetryer{
						NumMaxRetries:    3,
						MaxRetryDelay:    delay(delayMs) * time.Millisecond,
						MaxThrottleDelay: delay(delayMs) * time.Millisecond,
					},
				})
			}
		`, `
			package sqsevent

			import (
				"time"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
				"github.com/aws/aws-sdk-go-v2/aws/retry"
				"github.com/aws/aws-sdk-go-v2/service/sqs"
			)

			type delay = time.Duration

			func GetSqsClient(sess aws.Config, delayMs int) *sqs.Client {
				return sqs.NewFromConfig(awsCompatMergeConfig(sess, &aws.Config{
					Retryer: func() aws.Retryer {
						return retry.NewStandard(func(o *retry.StandardOptions) {
							o.MaxAttempts = 4
							o.MaxBackoff = delay(delayMs) * time.Millisecond
							o.RateLimiter = ratelimit.None
						})
					},
				}))
			}
		`).WithAfterRecipe(conversionsStay("delay(delayMs)")),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("sqsevent")),
	)
}

// A delay written through a renamed import of time stays a conversion too, and
// the file gains no second import of time.
func TestMigrateRetryerWithDelayThroughARenamedTimeImport(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package sqsevent

			import (
				gotime "time"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/client"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/sqs"
			)

			func GetSqsClient(sess *session.Session, delayMs int) *sqs.SQS {
				return sqs.New(sess, &aws.Config{
					Retryer: client.DefaultRetryer{
						NumMaxRetries:    3,
						MaxRetryDelay:    gotime.Duration(delayMs) * gotime.Millisecond,
						MaxThrottleDelay: gotime.Duration(delayMs) * gotime.Millisecond,
					},
				})
			}
		`, `
			package sqsevent

			import (
				gotime "time"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
				"github.com/aws/aws-sdk-go-v2/aws/retry"
				"github.com/aws/aws-sdk-go-v2/service/sqs"
			)

			func GetSqsClient(sess aws.Config, delayMs int) *sqs.Client {
				return sqs.NewFromConfig(awsCompatMergeConfig(sess, &aws.Config{
					Retryer: func() aws.Retryer {
						return retry.NewStandard(func(o *retry.StandardOptions) {
							o.MaxAttempts = 4
							o.MaxBackoff = gotime.Duration(delayMs) * gotime.Millisecond
							o.RateLimiter = ratelimit.None
						})
					},
				}))
			}
		`).WithAfterRecipe(conversionsStay("gotime.Duration(delayMs)")),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("sqsevent")),
	)
}

// A delay v1 left at its default is written out with the name the file imports
// time under, as the file has no import of time by its own name.
func TestMigrateRetryerDefaultThroughARenamedTimeImport(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package sqsevent

			import (
				gotime "time"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/client"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/sqs"
			)

			func GetSqsClient(sess *session.Session) *sqs.SQS {
				return sqs.New(sess, &aws.Config{
					Retryer: client.DefaultRetryer{
						NumMaxRetries: 3,
						MaxRetryDelay: 5 * gotime.Second,
					},
				})
			}
		`, `
			package sqsevent

			import (
				gotime "time"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
				"github.com/aws/aws-sdk-go-v2/aws/retry"
				"github.com/aws/aws-sdk-go-v2/service/sqs"
			)

			func GetSqsClient(sess aws.Config) *sqs.Client {
				return sqs.NewFromConfig(awsCompatMergeConfig(sess, &aws.Config{
					Retryer: func() aws.Retryer {
						return retry.NewStandard(func(o *retry.StandardOptions) {
							o.MaxAttempts = 4
							o.MaxBackoff = max(5*gotime.Second, 300*gotime.Second)
							o.RateLimiter = ratelimit.None
						})
					},
				}))
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("sqsevent")),
	)
}

// conversionsStay fails when a conversion written as one of want is held as a
// call rather than the java.TypeCast the parser makes of it.
func conversionsStay(want ...string) func(t *testing.T, cu *golang.CompilationUnit) {
	return func(t *testing.T, cu *golang.CompilationUnit) {
		shapes := visitor.Init(&callShapes{})
		shapes.Visit(cu, nil)
		for _, w := range want {
			assert.Contains(t, shapes.casts, w)
			assert.NotContains(t, shapes.calls, w)
		}
	}
}

// callShapes records the printed form of each conversion and call it visits.
type callShapes struct {
	visitor.GoVisitor
	casts, calls []string
}

func (v *callShapes) VisitTypeCast(tc *java.TypeCast, p any) java.J {
	v.casts = append(v.casts, strings.TrimSpace(printer.Print(tc)))
	return v.GoVisitor.VisitTypeCast(tc, p)
}

func (v *callShapes) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	v.calls = append(v.calls, strings.TrimSpace(printer.Print(mi)))
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}
