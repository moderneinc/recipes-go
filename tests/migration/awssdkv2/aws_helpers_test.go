/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"os"
	"strings"
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// v1's <T>ValueSlice and <T>ValueMap helpers became v2's To<T>Slice and
// To<T>Map, alongside To<T> for the scalars.
func TestMigrateAwsValueHelpers(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import "github.com/aws/aws-sdk-go/aws"

			func values(s []*string, m map[string]*int64, n *int32) ([]string, map[string]int64, int32) {
				return aws.StringValueSlice(s), aws.Int64ValueMap(m), aws.Int32Value(n)
			}
		`, `
			package main

			import "github.com/aws/aws-sdk-go-v2/aws"

			func values(s []*string, m map[string]*int64, n *int32) ([]string, map[string]int64, int32) {
				return aws.ToStringSlice(s), aws.ToInt64Map(m), aws.ToInt32(n)
			}
		`),
	)
}

// tsub/s3-edit's shape: an aws helper v2 dropped, which no rewrite stands in
// for, holds its file back.
func TestMigrateBlockedByAwsHelpersV2Dropped(t *testing.T) {
	for name, call := range map[string]string{
		"ReadSeekCloser":   "aws.ReadSeekCloser(io.MultiReader())",
		"SecondsTimeValue": "aws.SecondsTimeValue(nil)",
		"TimeUnixMilli":    "aws.TimeUnixMilli(time.Time{})",
	} {
		t.Run(name, func(t *testing.T) {
			test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
				test.Golang(`
					package main

					import (
						"io"
						"time"

						"github.com/aws/aws-sdk-go/aws"
						"github.com/aws/aws-sdk-go/service/s3"
					)

					var _ = io.MultiReader
					var _ time.Time

					func put(svc *s3.S3) error {
						_, err := svc.PutObject(&s3.PutObjectInput{Bucket: aws.String("b")})
						_ = `+call+`
						return err
					}
				`),
			)
		})
	}
}

// rakyll/goproxy-s3's shape: a call through the uploader's S3 client. v2
// narrowed that field to the calls the uploader makes, but it still holds the
// client the uploader was built from, which a session-built one makes S3's own.
func TestMigrateCallThroughManagerS3(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package proxy

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
				"github.com/aws/aws-sdk-go/service/s3/s3manager"
			)

			type cache struct {
				uploader *s3manager.Uploader
			}

			func newCache() *cache {
				return &cache{uploader: s3manager.NewUploader(session.Must(session.NewSession()))}
			}

			func (c *cache) exists(key string) error {
				_, err := c.uploader.S3.HeadObject(&s3.HeadObjectInput{Key: aws.String(key)})
				return err
			}
		`, `
			package proxy

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			type cache struct {
				uploader *s3manager.Uploader
			}

			func newCache() *cache {
				return &cache{uploader: s3manager.NewUploader(s3.NewFromConfig(awsCompatConfig(context.TODO())))}
			}

			func (c *cache) exists(key string) error {
				_, err := c.uploader.S3.(*s3.Client).HeadObject(context.TODO(), &s3.HeadObjectInput{Key: aws.String(key)})
				return err
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("proxy")),
	)
}

// An uploader built over any S3API may hold a client that is not S3's own, so
// a call through its S3 has nothing certain to reach.
func TestMigrateBlockedByManagerS3FieldOfUnknownClient(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package proxy

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
				"github.com/aws/aws-sdk-go/service/s3/s3iface"
				"github.com/aws/aws-sdk-go/service/s3/s3manager"
			)

			type cache struct {
				uploader *s3manager.Uploader
			}

			func newCache(api s3iface.S3API) *cache {
				return &cache{uploader: s3manager.NewUploaderWithClient(api)}
			}

			func (c *cache) exists(key string) error {
				_, err := c.uploader.S3.HeadObject(&s3.HeadObjectInput{Key: aws.String(key)})
				return err
			}
		`),
	)
}

// The proof spans the package: another file setting the field from an unknown
// client holds back the file calling through it.
func TestMigrateBlockedByManagerS3FieldSetElsewhere(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package proxy

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
				"github.com/aws/aws-sdk-go/service/s3/s3manager"
			)

			type cache struct {
				uploader *s3manager.Uploader
			}

			func (c *cache) exists(key string) error {
				_, err := c.uploader.S3.HeadObject(&s3.HeadObjectInput{Key: aws.String(key)})
				return err
			}
		`).WithPath("proxy/cache.go"),
		test.Golang(`
			package proxy

			import "github.com/aws/aws-sdk-go/service/s3/s3manager"

			func newCache(u func() *s3manager.Uploader) *cache {
				return &cache{uploader: u()}
			}
		`).WithPath("proxy/new.go"),
	)
}

// The fields v2 kept as they were carry over. The load inside the constructor
// names nothing from the config package, so its import goes.
func TestMigrateManagerFieldsV2Kept(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package proxy

			import (
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3/s3manager"
			)

			func newUploader() *s3manager.Uploader {
				u := s3manager.NewUploader(session.Must(session.NewSession()))
				u.PartSize = 10 << 20
				return u
			}
		`, `
			package proxy

			import (
				"context"

				s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func newUploader() *s3manager.Uploader {
				u := s3manager.NewUploader(s3.NewFromConfig(awsCompatConfig(context.TODO())))
				u.PartSize = 10 << 20
				return u
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("proxy")),
	)
}

// glassechidna/lambdalite3's shape: a download into aws.WriteAtBuffer. v2 moved
// the type into the transfer manager unchanged, reached through the file's own
// s3manager import where it has one.
func TestMigrateWriteAtBufferMovesToManager(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
				"github.com/aws/aws-sdk-go/service/s3/s3manager"
			)

			func get(d *s3manager.Downloader, bucket, key string) ([]byte, error) {
				buf := aws.NewWriteAtBuffer(nil)
				_, err := d.Download(buf, &s3.GetObjectInput{Bucket: &bucket, Key: &key})
				return buf.Bytes(), err
			}
		`, `
			package main

			import (
				"context"

				s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func get(d *s3manager.Downloader, bucket, key string) ([]byte, error) {
				buf := s3manager.NewWriteAtBuffer(nil)
				_, err := d.Download(context.TODO(), buf, &s3.GetObjectInput{Bucket: &bucket, Key: &key})
				return buf.Bytes(), err
			}
		`),
	)
}

// Without an s3manager import the transfer manager is imported under its own
// name.
func TestMigrateWriteAtBufferAddsManagerImport(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import "github.com/aws/aws-sdk-go/aws"

			var buf *aws.WriteAtBuffer = aws.NewWriteAtBuffer(nil)
		`, `
			package main

			import "github.com/aws/aws-sdk-go-v2/feature/s3/manager"

			var buf *manager.WriteAtBuffer = manager.NewWriteAtBuffer(nil)
		`),
	)
}

// tsub/s3-edit's shape: v1 needed a ReadSeekCloser for a body; v2 takes any
// io.Reader and seeks one that can, so around a reader that seeks the wrapper
// goes.
func TestMigrateReadSeekCloserAroundSeekableReader(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"bytes"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func put(svc *s3.S3, body []byte) error {
				_, err := svc.PutObject(&s3.PutObjectInput{
					Body:   aws.ReadSeekCloser(bytes.NewReader(body)),
					Bucket: aws.String("b"),
				})
				return err
			}
		`, `
			package main

			import (
				"bytes"
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func put(svc *s3.Client, body []byte) error {
				_, err := svc.PutObject(context.TODO(), &s3.PutObjectInput{
					Body:   bytes.NewReader(body),
					Bucket: aws.String("b"),
				})
				return err
			}
		`),
	)
}

// aidansteele/jwtex's shape: a session from a NewConfig chain asking for STS's
// regional endpoints, which v2 always uses, so the chain leaves no option
// behind and the endpoints import goes with it.
func TestMigrateConfigChainWithRegionalSTS(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/endpoints"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func load(region string) (*session.Session, error) {
				return session.NewSessionWithOptions(session.Options{
					SharedConfigState: session.SharedConfigEnable,
					Config:            *aws.NewConfig().WithRegion(region).WithSTSRegionalEndpoint(endpoints.RegionalSTSEndpoint),
				})
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
			)

			func load(region string) (aws.Config, error) {
				return config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
			}
		`),
	)
}

// The partition data and the resolvers have no public v2 counterpart.
func TestMigrateBlockedByEndpointsPartitions(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws/endpoints"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			var sess = session.Must(session.NewSession())

			func regions() []string {
				var ids []string
				for id := range endpoints.AwsPartition().Regions() {
					ids = append(ids, id)
				}
				return ids
			}
		`),
	)
}

// aws/aws-node-termination-handler's shape: a queue URL's region found by
// looking through v1's partitions. v2 keeps that data internal; aws-sdk-go is
// frozen, so a generated copy of the list v1's final release held answers the
// same lookup.
func TestMigrateDefaultPartitions(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"strings"

				"github.com/aws/aws-sdk-go/aws/endpoints"
			)

			func regionOf(queueURL string) string {
				for _, partition := range endpoints.DefaultPartitions() {
					for regionID := range partition.Regions() {
						if strings.Contains(queueURL, regionID) {
							return regionID
						}
					}
				}
				return ""
			}
		`, `
			package main

			import (
				"strings"
			)

			func regionOf(queueURL string) string {
				for _, partition := range awsCompatPartitions() {
					for regionID := range partition.Regions() {
						if strings.Contains(queueURL, regionID) {
							return regionID
						}
					}
				}
				return ""
			}
		`),
		test.Generated("zz_awssdkv2_partitions.go", partitionsHelpers(t)),
	)
}

// partitionsHelpers is the generated copy of v1's partition data. aws-sdk-go is
// frozen, so it does not change.
func partitionsHelpers(t *testing.T) string {
	b, err := os.ReadFile("testdata/partitions.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// An external test package's helpers go in a _test.go file: Go rejects a
// directory whose non-test files declare two packages.
func TestMigrateExternalTestPackageHelpers(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package regions

			func Known() []string { return nil }
		`).WithPath("regions/regions.go"),
		test.Golang(`
			package regions_test

			import (
				"testing"

				"github.com/aws/aws-sdk-go/aws/endpoints"
			)

			func TestPartitions(t *testing.T) {
				if len(endpoints.DefaultPartitions()) == 0 {
					t.Fatal("no partitions")
				}
			}
		`, `
			package regions_test

			import (
				"testing"
			)

			func TestPartitions(t *testing.T) {
				if len(awsCompatPartitions()) == 0 {
					t.Fatal("no partitions")
				}
			}
		`).WithPath("regions/regions_test.go"),
		test.Generated("regions/zz_awssdkv2_partitions_test.go", strings.Replace(partitionsHelpers(t), "package main", "package regions_test", 1)),
	)
}

// hypnoglow/helm-s3's shape: v1's REST path escaper, which v2 keeps unchanged
// in smithy's HTTP binding, bound back to the name the file spells.
func TestMigrateRestEscapePath(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package awsutil

			import "github.com/aws/aws-sdk-go/private/protocol/rest"

			func EscapePath(path string) string {
				return rest.EscapePath(path, true)
			}
		`, `
			package awsutil

			import rest "github.com/aws/smithy-go/encoding/httpbinding"

			func EscapePath(path string) string {
				return rest.EscapePath(path, true)
			}
		`),
	)
}
