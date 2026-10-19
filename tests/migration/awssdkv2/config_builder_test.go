/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"strings"
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// willnorris/imageproxy's shape: a config built up through setters in a local,
// some only under a condition, then handed to the session. v1 applied the
// settings when the session was made, DisableSSL deciding the endpoint's scheme
// then, so the local becomes a generated builder resolving them at load.
func TestMigrateConfigBuiltThroughSetters(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"net/url"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func New(u *url.URL, region string) (*s3.S3, error) {
				config := aws.NewConfig().WithRegion(region)
				if v := u.Query().Get("endpoint"); v != "" {
					config = config.WithEndpoint(v)
				}
				if v := u.Query().Get("disableSSL"); v == "1" {
					config = config.WithDisableSSL(true)
				}
				if v := u.Query().Get("s3ForcePathStyle"); v == "1" {
					config = config.WithS3ForcePathStyle(true)
				}
				sess, err := session.NewSession(config)
				if err != nil {
					return nil, err
				}
				return s3.New(sess), nil
			}
		`, `
			package main

			import (
				"context"
				"net/url"

				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func New(u *url.URL, region string) (*s3.Client, error) {
				config := awsCompatNewConfig().WithRegion(region)
				if v := u.Query().Get("endpoint"); v != "" {
					config = config.WithEndpoint(v)
				}
				if v := u.Query().Get("disableSSL"); v == "1" {
					config = config.WithDisableSSL(true)
				}
				if v := u.Query().Get("s3ForcePathStyle"); v == "1" {
					config = config.WithS3ForcePathStyle(true)
				}
				sess, err := config.load(context.TODO())
				if err != nil {
					return nil, err
				}
				return s3.NewFromConfig(sess), nil
			}
		`),
		test.Generated("zz_awssdkv2_configbuilder.go", testdataFile(t, "config_builder.go.txt")),
		test.Generated("zz_awssdkv2_s3.go", pathStyleHelpers),
		test.Generated("zz_awssdkv2_ssl.go", strings.Replace(testdataFile(t, "disable_ssl.go.txt"), "package storage", "package main", 1)),
	)
}

// keilerkonzept/aws-secretsmanager-files' shape: a NewConfig local set once and
// handed to the session folds into the load as it stands, rather than becoming
// the builder a local set under conditions needs.
func TestMigrateNewConfigLocalUsedOnce(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func load() (*session.Session, error) {
				cfg := aws.NewConfig()
				return session.NewSession(cfg)
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
			)

			func load() (aws.Config, error) {
				return config.LoadDefaultConfig(context.TODO())
			}
		`),
	)
}

// oxyno-zeta/s3-proxy's shape: a config literal in a local, filled in under
// conditions — DisableSSL among them, which v2's config has no field for —
// then handed to the session. It becomes the builder, its fields keeping the
// pointers v1's held.
func TestMigrateConfigLiteralFilledInWithDisableSSL(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			type bucket struct {
				Region, Endpoint, AccessKey, SecretKey string
				PathStyle                              *bool
				DisableSSL                             bool
			}

			func newClient(b *bucket) (*s3.S3, error) {
				sessionConfig := &aws.Config{
					Region: aws.String(b.Region),
				}
				if b.AccessKey != "" {
					sessionConfig.Credentials = credentials.NewStaticCredentials(b.AccessKey, b.SecretKey, "")
				}
				if b.Endpoint != "" {
					sessionConfig.Endpoint = aws.String(b.Endpoint)
				}
				if b.PathStyle != nil {
					sessionConfig.S3ForcePathStyle = b.PathStyle
				} else {
					sessionConfig.S3ForcePathStyle = aws.Bool(true)
				}
				if b.DisableSSL {
					sessionConfig.DisableSSL = aws.Bool(true)
				}
				sess, err := session.NewSession(sessionConfig)
				if err != nil {
					return nil, err
				}
				return s3.New(sess), nil
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/credentials"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			type bucket struct {
				Region, Endpoint, AccessKey, SecretKey string
				PathStyle                              *bool
				DisableSSL                             bool
			}

			func newClient(b *bucket) (*s3.Client, error) {
				sessionConfig := &awsCompatConfigBuilder{
					region: aws.String(b.Region),
				}
				if b.AccessKey != "" {
					sessionConfig.credentials = credentials.NewStaticCredentialsProvider(b.AccessKey, b.SecretKey, "")
				}
				if b.Endpoint != "" {
					sessionConfig.endpoint = aws.String(b.Endpoint)
				}
				if b.PathStyle != nil {
					sessionConfig.pathStyle = b.PathStyle
				} else {
					sessionConfig.pathStyle = aws.Bool(true)
				}
				if b.DisableSSL {
					sessionConfig.disableSSL = aws.Bool(true)
				}
				sess, err := sessionConfig.load(context.TODO())
				if err != nil {
					return nil, err
				}
				return s3.NewFromConfig(sess), nil
			}
		`),
		test.Generated("zz_awssdkv2_configbuilder.go", testdataFile(t, "config_builder.go.txt")),
		test.Generated("zz_awssdkv2_s3.go", pathStyleHelpers),
		test.Generated("zz_awssdkv2_ssl.go", strings.Replace(testdataFile(t, "disable_ssl.go.txt"), "package storage", "package main", 1)),
	)
}
