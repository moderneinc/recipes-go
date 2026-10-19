/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// profefe/profefe's shape: DisableSSL beside an endpoint that may be empty.
// The endpoint takes v1's scheme from the setting, and the service option
// covers the endpoint v2 resolves when none was given.
func TestMigrateDisableSSL(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package storage

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			type Config struct {
				EndpointURL string
				DisableSSL  bool
				Region      string
			}

			func New(conf Config) (*s3.S3, error) {
				sess, err := session.NewSession(&aws.Config{
					Endpoint:   aws.String(conf.EndpointURL),
					DisableSSL: aws.Bool(conf.DisableSSL),
					Region:     aws.String(conf.Region),
				})
				if err != nil {
					return nil, err
				}
				return s3.New(sess), nil
			}
		`, `
			package storage

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			type Config struct {
				EndpointURL string
				DisableSSL  bool
				Region      string
			}

			func New(conf Config) (*s3.Client, error) {
				sess, err := config.LoadDefaultConfig(context.TODO(), config.WithBaseEndpoint(awsCompatEndpointScheme(conf.EndpointURL, conf.DisableSSL)), config.WithRegion(conf.Region), config.WithServiceOptions(awsCompatDisableSSL(conf.DisableSSL)))
				if err != nil {
					return nil, err
				}
				return s3.NewFromConfig(sess), nil
			}
		`),
		test.Generated("zz_awssdkv2_ssl.go", testdataFile(t, "disable_ssl.go.txt")),
	)
}
