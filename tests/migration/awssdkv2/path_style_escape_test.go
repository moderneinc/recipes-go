/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"strings"
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// pshima/consul-snapshot's shape: the addressing style set under a condition
// on a config that leaves the function, so no S3 client here could read a
// hoisted local. The setting travels with the config as a service option.
func TestMigratePathStyleOnEscapingConfig(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package adapters

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			type Adapter struct{ session *session.Session }

			func New(region, endpoint string) *Adapter {
				awsConfig := &aws.Config{Region: aws.String(region)}
				if endpoint != "" {
					awsConfig.Endpoint = aws.String(endpoint)
					awsConfig.S3ForcePathStyle = aws.Bool(true)
				}
				sess := session.New(awsConfig)
				return &Adapter{session: sess}
			}
		`, `
			package adapters

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
			)

			type Adapter struct{ session aws.Config }

			func New(region, endpoint string) *Adapter {
				awsConfig := awsCompatConfig(context.TODO(), config.WithRegion(region))
				if endpoint != "" {
					awsConfig.BaseEndpoint = aws.String(awsCompatEndpoint(endpoint))
					awsConfig.ServiceOptions = append(awsConfig.ServiceOptions, awsCompatS3PathStyle(true))
				}
				sess := awsConfig
				return &Adapter{session: sess}
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("adapters")),
		test.Generated("zz_awssdkv2_s3.go", strings.Replace(pathStyleHelpers, "package main", "package adapters", 1)),
	)
}
