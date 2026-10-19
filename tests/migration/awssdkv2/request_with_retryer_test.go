/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// contentful-labs/kube-secret-syncer's shape: request.WithRetryer setting a
// default retryer on a new config, which is a setter in all but name. It
// becomes the loader's retryer option, v1's retry count as v2's attempts.
func TestMigrateRequestWithRetryer(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				awsclient "github.com/aws/aws-sdk-go/aws/client"
				"github.com/aws/aws-sdk-go/aws/request"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/secretsmanager"
			)

			func client() *secretsmanager.SecretsManager {
				cfg := request.WithRetryer(aws.NewConfig(), awsclient.DefaultRetryer{NumMaxRetries: 5})
				return secretsmanager.New(session.Must(session.NewSession(cfg)))
			}
		`, `
			package main

			import (
				"context"
				"time"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
				"github.com/aws/aws-sdk-go-v2/aws/retry"
				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
			)

			func client() *secretsmanager.Client {
				return secretsmanager.NewFromConfig(awsCompatConfig(context.TODO(), config.WithRetryer(func() aws.Retryer {
					return retry.NewStandard(func(o *retry.StandardOptions) {
						o.MaxAttempts = 6
						o.MaxBackoff = 300 * time.Second
						o.RateLimiter = ratelimit.None
					})
				})))
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("main")),
	)
}

// keikoproj/iam-manager's shape: request.WithRetryer as a statement, setting
// the retryer on a config held in a variable.
func TestMigrateRequestWithRetryerOnLocal(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"time"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/client"
				"github.com/aws/aws-sdk-go/aws/request"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/iam"
			)

			func newClient(region string) *iam.IAM {
				config := aws.NewConfig().
					WithRegion(region).
					WithCredentialsChainVerboseErrors(true)

				request.WithRetryer(config, client.DefaultRetryer{
					NumMaxRetries:    12,
					MinThrottleDelay: time.Second * 5,
					MaxThrottleDelay: time.Second * 60,
				})

				sess, err := session.NewSession(config)
				if err != nil {
					panic(err)
				}
				return iam.New(sess)
			}
		`, `
			package main

			import (
				"context"
				"time"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
				"github.com/aws/aws-sdk-go-v2/aws/retry"
				configaws "github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/iam"
			)

			func newClient(region string) *iam.Client {
				sess, err := configaws.LoadDefaultConfig(context.TODO(), configaws.WithRegion(region), configaws.WithRetryer(func() aws.Retryer {
					return retry.NewStandard(func(o *retry.StandardOptions) {
						o.MaxAttempts = 13
						o.MaxBackoff = max(300*time.Second, time.Second*60)
						o.RateLimiter = ratelimit.None
					})
				}))
				if err != nil {
					panic(err)
				}
				return iam.NewFromConfig(sess)
			}
		`),
	)
}

// contentful-labs/kube-secret-syncer's main: the retrying config held in a
// local the session is made from inside another call.
func TestMigrateRequestWithRetryerLocalInCall(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				awsclient "github.com/aws/aws-sdk-go/aws/client"
				"github.com/aws/aws-sdk-go/aws/request"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/secretsmanager"
			)

			func use(*secretsmanager.SecretsManager) {}

			func main() {
				Retry5Cfg := request.WithRetryer(aws.NewConfig(), awsclient.DefaultRetryer{NumMaxRetries: 5})
				use(secretsmanager.New(session.Must(session.NewSession(Retry5Cfg))))
				use(secretsmanager.New(session.New(Retry5Cfg)))
			}
		`, `
			package main

			import (
				"context"
				"time"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
				"github.com/aws/aws-sdk-go-v2/aws/retry"
				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
			)

			func use(*secretsmanager.Client) {}

			func main() {
				use(secretsmanager.NewFromConfig(awsCompatConfig(context.TODO(), config.WithRetryer(func() aws.Retryer {
					return retry.NewStandard(func(o *retry.StandardOptions) {
						o.MaxAttempts = 6
						o.MaxBackoff = 300 * time.Second
						o.RateLimiter = ratelimit.None
					})
				}))))
				use(secretsmanager.NewFromConfig(awsCompatConfig(context.TODO(), config.WithRetryer(func() aws.Retryer {
					return retry.NewStandard(func(o *retry.StandardOptions) {
						o.MaxAttempts = 6
						o.MaxBackoff = 300 * time.Second
						o.RateLimiter = ratelimit.None
					})
				}))))
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("main")),
	)
}
