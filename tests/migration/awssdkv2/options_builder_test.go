/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"strings"
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// orgrim/pg_back's shape: a config built through setters under conditions,
// carried whole as a session.Options' Config, whose profile is set after. The
// builder answers v2's options value, which the later write and the load take.
func TestMigrateOptionsCarryingBuiltConfig(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package storage

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func open(region, keyID, secret, endpoint, profile string, forcePath, disableSSL bool) (*session.Session, error) {
				conf := aws.NewConfig()
				if region != "" {
					conf = conf.WithRegion(region)
				}
				if keyID != "" {
					conf = conf.WithCredentials(credentials.NewStaticCredentials(keyID, secret, ""))
				}
				if endpoint != "" {
					conf = conf.WithEndpoint(endpoint)
				}
				if forcePath {
					conf = conf.WithS3ForcePathStyle(true)
				}
				if disableSSL {
					conf = conf.WithDisableSSL(true)
				}
				sopts := session.Options{
					Config:            *conf,
					SharedConfigState: session.SharedConfigEnable,
				}
				if profile != "" {
					sopts.Profile = profile
				}
				return session.NewSessionWithOptions(sopts)
			}
		`, `
			package storage

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/credentials"
			)

			func open(region, keyID, secret, endpoint, profile string, forcePath, disableSSL bool) (aws.Config, error) {
				conf := awsCompatNewConfig()
				if region != "" {
					conf = conf.WithRegion(region)
				}
				if keyID != "" {
					conf = conf.WithCredentials(credentials.NewStaticCredentialsProvider(keyID, secret, ""))
				}
				if endpoint != "" {
					conf = conf.WithEndpoint(endpoint)
				}
				if forcePath {
					conf = conf.WithS3ForcePathStyle(true)
				}
				if disableSSL {
					conf = conf.WithDisableSSL(true)
				}
				sopts := conf.options()
				if profile != "" {
					sopts.SharedConfigProfile = profile
				}
				return config.LoadDefaultConfig(context.TODO(), awsCompatLoadOptions(sopts))
			}
		`),
		test.Generated("zz_awssdkv2_configbuilder.go", strings.Replace(testdataFile(t, "config_builder.go.txt"), "package main", "package storage", 1)),
		test.Generated("zz_awssdkv2_loadoptions.go", strings.Replace(testdataFile(t, "load_options.go.txt"), "package main", "package storage", 1)),
		test.Generated("zz_awssdkv2_s3.go", strings.Replace(pathStyleHelpers, "package main", "package storage", 1)),
		test.Generated("zz_awssdkv2_ssl.go", testdataFile(t, "disable_ssl_no_clients.go.txt")),
	)
}
