/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// kayac/sqsjkr's shape: a client takes a config whose settings depend on a
// branch. v1 merged whichever literal it held over the session's config, and
// the generated merge the v2 client is built from does the same.
func TestMigrateClientConfigLocalMerged(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package lock

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/dynamodb"
			)

			func New(profile, region string) *dynamodb.DynamoDB {
				var conf *aws.Config
				if profile != "" {
					conf = &aws.Config{
						Region:      aws.String(region),
						Credentials: credentials.NewSharedCredentials("", profile),
					}
				} else {
					conf = &aws.Config{
						Region: aws.String(region),
					}
				}
				return dynamodb.New(session.New(), conf)
			}
		`, `
			package lock

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/dynamodb"
			)

			func New(profile, region string) *dynamodb.Client {
				var conf *aws.Config
				if profile != "" {
					conf = &aws.Config{
						Region:      region,
						Credentials: awsCompatSharedCredentials(context.TODO(), "", profile),
					}
				} else {
					conf = &aws.Config{
						Region: region,
					}
				}
				return dynamodb.NewFromConfig(awsCompatMergeConfig(awsCompatConfig(context.TODO()), conf))
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("lock")),
	)
}

// cmattoon/aws-ssm's shape: a client assumes a role for a call, through a
// config literal naming the credentials and the session's region.
func TestMigrateClientConfigAssumingRole(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package provider

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/ssm"
			)

			type Provider struct {
				Session *session.Session
				Service *ssm.SSM
			}

			func (p Provider) assume(roleArn string) {
				creds := stscreds.NewCredentials(p.Session, roleArn)
				p.Service = ssm.New(p.Session, &aws.Config{
					Credentials: creds,
					Region:      aws.String(*p.Session.Config.Region),
				})
			}
		`, `
			package provider

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
				"github.com/aws/aws-sdk-go-v2/service/ssm"
				"github.com/aws/aws-sdk-go-v2/service/sts"
			)

			type Provider struct {
				Session aws.Config
				Service *ssm.Client
			}

			func (p Provider) assume(roleArn string) {
				creds := aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(sts.NewFromConfig(p.Session), roleArn))
				p.Service = ssm.NewFromConfig(awsCompatMergeConfig(p.Session, &aws.Config{
					Credentials: creds,
					Region:      p.Session.Region,
				}))
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("provider")),
	)
}

// An option v1's NewCredentials took that v2's assume-role options do not hold,
// such as the expiry window, has no counterpart the rewrite writes.
func TestMigrateBlockedByAssumeRoleOptions(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package provider

			import (
				"time"

				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func creds(sess *session.Session, role string) *credentials.Credentials {
				return stscreds.NewCredentials(sess, role, func(p *stscreds.AssumeRoleProvider) { p.ExpiryWindow = time.Minute })
			}
		`),
	)
}

// rakyll/goproxy-s3's shape: a config built field by field, each setting only
// where the caller gave one, then handed to NewSession. The local stays as v2's
// config, and the settings it ends up holding become the loader's options, as
// v1 applied only the settings the config carried.
func TestMigrateConfigBuiltFieldByField(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func load(region, endpoint string) (*session.Session, error) {
				cfg := &aws.Config{}
				if region != "" {
					cfg.Region = aws.String(region)
				}
				if endpoint != "" {
					cfg.Endpoint = aws.String(endpoint)
				}
				sess, err := session.NewSession(cfg)
				return sess, err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
			)

			func load(region, endpoint string) (aws.Config, error) {
				cfg := &aws.Config{}
				if region != "" {
					cfg.Region = region
				}
				if endpoint != "" {
					cfg.BaseEndpoint = aws.String(awsCompatEndpoint(endpoint))
				}
				sess, err := config.LoadDefaultConfig(context.TODO(), awsCompatConfigOptions(cfg)...)
				return sess, err
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("main")),
	)
}

// sourcegraph/checkup's shape: a seam that builds the client from anything a
// session satisfied, merging the configs it is handed. ConfigProvider becomes
// v2's config, which is what replaced the session, and the configs it is handed
// stay v2 configs.
func TestMigrateConfigProviderSeam(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package s3store

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/client"
				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			var newS3 = func(p client.ConfigProvider, cfgs ...*aws.Config) *s3.S3 {
				return s3.New(p, cfgs...)
			}

			func store(region, id, secret string) *s3.S3 {
				config := &aws.Config{
					Region: &region,
				}
				if id != "" {
					config.Credentials = credentials.NewStaticCredentials(id, secret, "")
				}
				return newS3(session.Must(session.NewSession()), config)
			}
		`, `
			package s3store

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/credentials"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			var newS3 = func(p aws.Config, cfgs ...*aws.Config) *s3.Client {
				return s3.NewFromConfig(awsCompatMergeConfig(p, cfgs...))
			}

			func store(region, id, secret string) *s3.Client {
				config := &aws.Config{
					Region: region,
				}
				if id != "" {
					config.Credentials = credentials.NewStaticCredentialsProvider(id, secret, "")
				}
				return newS3(awsCompatConfig(context.TODO()), config)
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("s3store")),
	)
}

// kayac/sqsjkr's shape: the client and the input it is called with are both a
// struct's fields. Each is still what its declared type says it is, so the call
// is an operation, and what it returns is a shape the range reads.
func TestMigrateOperationThroughStructFields(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package sqsjkr

			import "github.com/aws/aws-sdk-go/service/sqs"

			type jkr struct {
				SQS        *sqs.SQS
				recvParams *sqs.ReceiveMessageInput
			}

			func handle(msg *sqs.Message) {}

			func (j *jkr) poll() error {
				resp, err := j.SQS.ReceiveMessage(j.recvParams)
				if err != nil {
					return err
				}
				for _, msg := range resp.Messages {
					handle(msg)
				}
				return nil
			}
		`, `
			package sqsjkr

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/sqs"
				sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
			)

			type jkr struct {
				SQS        *sqs.Client
				recvParams *sqs.ReceiveMessageInput
			}

			func handle(msg *sqstypes.Message) {}

			func (j *jkr) poll() error {
				resp, err := j.SQS.ReceiveMessage(context.TODO(), j.recvParams)
				if err != nil {
					return err
				}
				for _, msg := range awsCompatPtrs(resp.Messages) {
					handle(msg)
				}
				return nil
			}
		`),
		test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("sqsjkr")),
	)
}

// sourcegraph/checkup's test: a stand-in for the seam reads the credentials and
// the region off the configs it is handed. v2's provider answers Retrieve with
// the same key and secret, and its region is held by value.
func TestMigrateConfigReadsInSeamStandIn(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package s3store

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/client"
			)

			var check = func(p client.ConfigProvider, cfgs ...*aws.Config) (string, string) {
				creds, err := cfgs[0].Credentials.Get()
				if err != nil {
					return "", ""
				}
				return creds.AccessKeyID, *cfgs[0].Region
			}
		`, `
			package s3store

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
			)

			var check = func(p aws.Config, cfgs ...*aws.Config) (string, string) {
				creds, err := cfgs[0].Credentials.Retrieve(context.TODO())
				if err != nil {
					return "", ""
				}
				return creds.AccessKeyID, cfgs[0].Region
			}
		`),
	)
}

// gilbertchen/duplicacy's shape: credentials built into a local and handed to
// two config literals, and an enum output field tested against nil. v2 holds
// the enum by value, and empty is what v1's nil was, since no enum has the
// empty string as a value.
func TestMigrateCredentialsLocalAndEnumNilCheck(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package storage

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func region(bucket, accessKey, secretKey string) (string, error) {
				auth := credentials.NewStaticCredentials(accessKey, secretKey, "")
				client := s3.New(session.New(&aws.Config{
					Region:      aws.String("us-east-1"),
					Credentials: auth,
				}))
				response, err := client.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: aws.String(bucket)})
				if err != nil {
					return "", err
				}
				region := "us-east-1"
				if response.LocationConstraint != nil {
					region = *response.LocationConstraint
				}
				return region, nil
			}
		`, `
			package storage

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/credentials"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func region(bucket, accessKey, secretKey string) (string, error) {
				auth := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
				client := s3.NewFromConfig(awsCompatConfig(context.TODO(), config.WithRegion("us-east-1"), config.WithCredentialsProvider(auth)))
				response, err := client.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{Bucket: aws.String(bucket)})
				if err != nil {
					return "", err
				}
				region := "us-east-1"
				if response.LocationConstraint != "" {
					region = string(response.LocationConstraint)
				}
				return region, nil
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("storage")),
	)
}

// willnorris/imageproxy's shape: a struct embedding the client, whose field Go
// names after the type. v2's client type is Client, so the literal's key and a
// selector naming the field follow it.
func TestMigrateEmbeddedClientField(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package s3cache

			import (
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			type cache struct {
				*s3.S3
				bucket string
			}

			func New(sess *session.Session, bucket string) *cache {
				return &cache{
					S3:     s3.New(sess),
					bucket: bucket,
				}
			}

			func (c *cache) client() *s3.S3 {
				return c.S3
			}
		`, `
			package s3cache

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			type cache struct {
				*s3.Client
				bucket string
			}

			func New(sess aws.Config, bucket string) *cache {
				return &cache{
					Client: s3.NewFromConfig(sess),
					bucket: bucket,
				}
			}

			func (c *cache) client() *s3.Client {
				return c.Client
			}
		`),
	)
}

// oxyno-zeta/s3-proxy's shape: Go 1.26's new(v), pointing to a copy of a value
// as aws.String does, filling the config the session takes.
func TestMigrateNewOfValue(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package s3client

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func open(region string, retries int) (*session.Session, error) {
				return session.NewSession(&aws.Config{Region: new(region), MaxRetries: new(retries)})
			}
		`, `
			package s3client

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
			)

			func open(region string, retries int) (aws.Config, error) {
				return config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithRetryMaxAttempts(retries+1))
			}
		`),
	)
}

// oxyno-zeta/s3-proxy's listing: new(v) filling a field v2 narrowed, which
// converts as the address of a value does.
func TestMigrateNewOfValueNarrowed(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package s3client

			import "github.com/aws/aws-sdk-go/service/s3"

			func input(bucket string, limit int64) *s3.ListObjectsV2Input {
				maxKeys := min(limit, 1000)
				return &s3.ListObjectsV2Input{Bucket: new(bucket), MaxKeys: new(maxKeys)}
			}
		`, `
			package s3client

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func input(bucket string, limit int64) *s3.ListObjectsV2Input {
				maxKeys := min(limit, 1000)
				return &s3.ListObjectsV2Input{Bucket: new(bucket), MaxKeys: aws.Int32(int32(maxKeys))}
			}
		`),
	)
}
