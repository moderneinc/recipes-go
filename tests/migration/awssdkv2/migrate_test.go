/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsexportdata"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/exportdata"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// Verify names an unreadable blob directly, where the attribution sweep in
// tests/ reports it as a missing type on every emitted config load.
// See CLAUDE.md: Type Attribution for how to regenerate.
func TestAwsExportDataIsReadable(t *testing.T) {
	require.NoError(t, exportdata.Verify(awsexportdata.FS, awsexportdata.Paths...))
}

func migrateSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{})
}

// dmfutcher/git-s3-push's shape, reduced to one function: a session built from a
// region, a client built from the session, and an operation that gains the
// context v2 takes.
func TestMigrateSessionClientAndOperation(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func load(ctx context.Context, region, bucket, key string) error {
				sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
				if err != nil {
					return err
				}
				svc := s3.New(sess)
				_, err = svc.GetObject(&s3.GetObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func load(ctx context.Context, region, bucket, key string) error {
				sess, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
				if err != nil {
					return err
				}
				svc := s3.NewFromConfig(sess)
				_, err = svc.GetObject(ctx, &s3.GetObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				})
				return err
			}
		`),
	)
}

// v2 dropped the Value spelling of the pointer helpers outright, and takes a
// context on every operation, so a WithContext call only loses its suffix.
func TestMigrateValueHelpersAndWithContext(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func head(ctx context.Context, bucket string) (string, error) {
				sess, err := session.NewSession()
				if err != nil {
					return "", err
				}
				svc := s3.New(sess)
				out, err := svc.HeadBucketWithContext(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
				if err != nil {
					return "", err
				}
				return aws.StringValue(out.BucketRegion), nil
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func head(ctx context.Context, bucket string) (string, error) {
				sess, err := config.LoadDefaultConfig(ctx)
				if err != nil {
					return "", err
				}
				svc := s3.NewFromConfig(sess)
				out, err := svc.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
				if err != nil {
					return "", err
				}
				return aws.ToString(out.BucketRegion), nil
			}
		`),
	)
}

// The static credentials constructor gained a Provider suffix in v2.
func TestMigrateStaticCredentials(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func creds(ctx context.Context, id, secret string) error {
				_ = credentials.NewStaticCredentials(id, secret, "")
				_, err := session.NewSession()
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/credentials"
			)

			func creds(ctx context.Context, id, secret string) error {
				_ = credentials.NewStaticCredentialsProvider(id, secret, "")
				_, err := config.LoadDefaultConfig(ctx)
				return err
			}
		`),
	)
}

// cultureamp/s3dotenv wraps the constructor in session.Must, which panics where
// the v2 loader returns an error — so one statement becomes two, the load and
// the panic Must stood for.
func TestMigrateExpandsSessionMust(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func load(ctx context.Context, region string) error {
				sess := session.Must(session.NewSession(&aws.Config{Region: &region}))
				svc := s3.New(sess)
				_, err := svc.ListBuckets(&s3.ListBucketsInput{})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func load(ctx context.Context, region string) error {
				sess, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
				if err != nil {
					panic(err)
				}
				svc := s3.NewFromConfig(sess)
				_, err := svc.ListBuckets(ctx, &s3.ListBucketsInput{})
				return err
			}
		`),
	)
}

// v1 took no context anywhere v2 takes one, and almost no v1 code has one in
// scope. A function without one gets context.TODO(), the placeholder the
// standard library provides for exactly this, and the import that reads it.
func TestMigrateUsesTodoContextWhenNoneInScope(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func load() error {
				sess, err := session.NewSession()
				if err != nil {
					return err
				}
				svc := s3.New(sess)
				_, err = svc.ListBuckets(&s3.ListBucketsInput{})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func load() error {
				sess, err := config.LoadDefaultConfig(context.TODO())
				if err != nil {
					return err
				}
				svc := s3.NewFromConfig(sess)
				_, err = svc.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
				return err
			}
		`),
	)
}

// The page iterators became paginator types, which is not a call-site rewrite.
func TestMigrateBlockedByPaginator(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func list(ctx context.Context) error {
				sess, err := session.NewSession()
				if err != nil {
					return err
				}
				svc := s3.New(sess)
				return svc.ListObjectsPages(&s3.ListObjectsInput{}, func(page *s3.ListObjectsOutput, last bool) bool {
					return true
				})
			}
		`),
	)
}

// The constant names carry over unchanged, but each belongs to an enum: v1's
// *string field took any of them, v2's typed field takes only its own.
// ObjectStorageClass is not StorageClass, so this file waits.
func TestMigrateBlockedByMismatchedEnumConstant(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func put(ctx context.Context, bucket string) error {
				sess, err := session.NewSession()
				if err != nil {
					return err
				}
				svc := s3.New(sess)
				_, err = svc.PutObject(&s3.PutObjectInput{
					Bucket:       aws.String(bucket),
					StorageClass: aws.String(s3.ObjectStorageClassStandard),
				})
				return err
			}
		`),
	)
}

// dmfutcher/git-s3-push builds the config in a variable first. The local has no
// v2 counterpart — v2's own aws.Config is what the loader returns — so its
// region is read off and the declaration goes with it.
func TestMigrateConfigBuiltInAVariable(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func load(ctx context.Context, region string) error {
				s3config := aws.Config{Region: aws.String(region)}
				_, err := session.NewSession(&s3config)
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/config"
			)

			func load(ctx context.Context, region string) error {
				_, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
				return err
			}
		`),
	)
}

// A config local something else reads, or writes a second field to, is not one
// the region can simply be lifted out of.
func TestMigrateBlockedByReusedConfigVariable(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func load(ctx context.Context, region, endpoint string) error {
				s3config := aws.Config{Region: aws.String(region)}
				s3config.Endpoint = aws.String(endpoint)
				_, err := session.NewSession(&s3config)
				return err
			}
		`),
	)
}

// No AWS SDK import at all.
func TestMigrateUnrelatedFile(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func run(ctx context.Context) {}
		`),
	)
}

// The whole module: the migratable file moves and go.mod gains the v2 requires
// and loses the v1 one.
func TestMigrateWholeModule(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoModuleToV2{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.23

				require (
					github.com/aws/aws-sdk-go v1.55.8
				)
			`, `
				module example.com/app

				go 1.24

				require (
					github.com/aws/aws-sdk-go-v2 v1.47.0
					github.com/aws/aws-sdk-go-v2/config v1.33.5
				)
			`),
			test.Golang(`
				package app

				import (
					"context"

					"github.com/aws/aws-sdk-go/aws"
					"github.com/aws/aws-sdk-go/aws/session"
					"github.com/aws/aws-sdk-go/service/s3"
				)

				func head(ctx context.Context, region, bucket string) error {
					sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
					if err != nil {
						return err
					}
					svc := s3.New(sess)
					_, err = svc.HeadBucket(&s3.HeadBucketInput{Bucket: aws.String(bucket)})
					return err
				}
			`, `
				package app

				import (
					"context"

					"github.com/aws/aws-sdk-go-v2/aws"
					"github.com/aws/aws-sdk-go-v2/config"
					"github.com/aws/aws-sdk-go-v2/service/s3"
				)

				func head(ctx context.Context, region, bucket string) error {
					sess, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
					if err != nil {
						return err
					}
					svc := s3.NewFromConfig(sess)
					_, err = svc.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
					return err
				}
			`),
		),
	)
}

// A file the migration cannot take is left alone and reported, and the rest of
// the module moves around it. The two requires then sit side by side, since one
// file still imports v1 — a client and a shape cross file boundaries, so callers
// of what stayed behind will not compile until it is seen to by hand, which is
// what the blockers table is the list for.
func TestMigrateLeavesBehindWhatItCannotTake(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoModuleToV2{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.23

				require (
					github.com/aws/aws-sdk-go v1.55.8
				)
			`, `
				module example.com/app

				go 1.24

				require (
					github.com/aws/aws-sdk-go v1.55.8
					github.com/aws/aws-sdk-go-v2 v1.47.0
					github.com/aws/aws-sdk-go-v2/config v1.33.5
				)
			`),
			test.Golang(`
				package clean

				import (
					"context"

					"github.com/aws/aws-sdk-go/aws"
					"github.com/aws/aws-sdk-go/aws/session"
					"github.com/aws/aws-sdk-go/service/s3"
				)

				func head(ctx context.Context, bucket string) error {
					sess, err := session.NewSession()
					if err != nil {
						return err
					}
					svc := s3.New(sess)
					_, err = svc.HeadBucket(&s3.HeadBucketInput{Bucket: aws.String(bucket)})
					return err
				}
			`, `
				package clean

				import (
					"context"

					"github.com/aws/aws-sdk-go-v2/aws"
					"github.com/aws/aws-sdk-go-v2/config"
					"github.com/aws/aws-sdk-go-v2/service/s3"
				)

				func head(ctx context.Context, bucket string) error {
					sess, err := config.LoadDefaultConfig(ctx)
					if err != nil {
						return err
					}
					svc := s3.NewFromConfig(sess)
					_, err = svc.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
					return err
				}
			`),
			test.Golang(`
				package legacy

				import (
					"context"

					"github.com/aws/aws-sdk-go/aws/request"
					"github.com/aws/aws-sdk-go/aws/session"
				)

				func instrument(ctx context.Context, sess *session.Session) {
					sess.Handlers.Send.PushFront(func(r *request.Request) {})
				}
			`),
		),
	)
}

// 99designs/iamy passes the session around as a parameter. v2 has no session:
// the config replaced it, and it is a value rather than a pointer, so the
// indirection goes with the type.
func TestMigrateSessionTypeBecomesConfig(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package iamy

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/sts"
			)

			func accountID(ctx context.Context, sess *session.Session) (string, error) {
				svc := sts.New(sess)
				resp, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
				if err != nil {
					return "", err
				}
				return *resp.Account, nil
			}
		`, `
			package iamy

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/sts"
			)

			func accountID(ctx context.Context, sess aws.Config) (string, error) {
				svc := sts.NewFromConfig(sess)
				resp, err := svc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
				if err != nil {
					return "", err
				}
				return *resp.Account, nil
			}
		`),
	)
}

// iamy also builds the client inline. A constructor chained into an operation is
// still a client operation, so the context lands on the operation rather than on
// the constructor.
func TestMigrateChainedClientWithoutContext(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package iamy

			import (
				"github.com/aws/aws-sdk-go/service/sts"
			)

			func accountID(cfg stsConfig) (string, error) {
				resp, err := sts.New(cfg).GetCallerIdentity(&sts.GetCallerIdentityInput{})
				if err != nil {
					return "", err
				}
				return *resp.Account, nil
			}
		`, `
			package iamy

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/sts"
			)

			func accountID(cfg stsConfig) (string, error) {
				resp, err := sts.NewFromConfig(cfg).GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
				if err != nil {
					return "", err
				}
				return *resp.Account, nil
			}
		`),
	)
}

// The same chain does migrate where a context is in scope, gaining it on the
// operation rather than on the constructor.
func TestMigrateChainedClientWithContext(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package iamy

			import (
				"context"

				"github.com/aws/aws-sdk-go/service/sts"
			)

			func accountID(ctx context.Context, cfg stsConfig) (string, error) {
				resp, err := sts.New(cfg).GetCallerIdentity(&sts.GetCallerIdentityInput{})
				if err != nil {
					return "", err
				}
				return *resp.Account, nil
			}
		`, `
			package iamy

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/sts"
			)

			func accountID(ctx context.Context, cfg stsConfig) (string, error) {
				resp, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
				if err != nil {
					return "", err
				}
				return *resp.Account, nil
			}
		`),
	)
}

// codahale/sneaker reaches S3 through an interface it declares. The interface
// is retyped to the v2 signature alongside the call, which is what keeps the v2
// client satisfying it — and the interface declares its parameters unnamed, so
// the replacement does too.
func TestMigrateRetypesDeclaredInterface(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package sneaker

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			type ObjectStorage interface {
				ListObjects(*s3.ListObjectsInput) (*s3.ListObjectsOutput, error)
			}

			type Manager struct {
				Objects ObjectStorage
				Bucket  string
			}

			func (m *Manager) List() error {
				_, err := m.Objects.ListObjects(&s3.ListObjectsInput{
					Bucket: aws.String(m.Bucket),
				})
				return err
			}
		`, `
			package sneaker

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			type ObjectStorage interface {
				ListObjects(context.Context, *s3.ListObjectsInput, ...func(*s3.Options)) (*s3.ListObjectsOutput, error)
			}

			type Manager struct {
				Objects ObjectStorage
				Bucket  string
			}

			func (m *Manager) List() error {
				_, err := m.Objects.ListObjects(context.TODO(), &s3.ListObjectsInput{
					Bucket: aws.String(m.Bucket),
				})
				return err
			}
		`),
	)
}

// ZipRecruiter/cloudwatching's shape: the modelled shapes moved to the types
// sub-package, which is imported under an alias so a file touching two services
// stays unambiguous. The client type v1 named for its service becomes Client,
// and the operation gains its context.
func TestMigrateRelocatesShapesAndClientType(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package exportcloudwatch

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/cloudwatch"
			)

			func read(ctx context.Context, cw *cloudwatch.CloudWatch) error {
				stat := &cloudwatch.MetricStat{Period: aws.Int64(60)}
				_ = stat
				_, err := cw.GetMetricData(&cloudwatch.GetMetricDataInput{})
				return err
			}
		`, `
			package exportcloudwatch

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
				cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
			)

			func read(ctx context.Context, cw *cloudwatch.Client) error {
				stat := &cloudwatchtypes.MetricStat{Period: aws.Int32(60)}
				_ = stat
				_, err := cw.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{})
				return err
			}
		`),
	)
}

// An enum constant belonging to the field's own enum loses the aws.String that
// v1's *string field required.
func TestMigrateUnwrapsMatchingEnumConstant(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func put(ctx context.Context, svc *s3.S3, bucket string) error {
				_, err := svc.PutObject(&s3.PutObjectInput{
					Bucket:       aws.String(bucket),
					StorageClass: aws.String(s3.StorageClassStandard),
				})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
				s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
			)

			func put(ctx context.Context, svc *s3.Client, bucket string) error {
				_, err := svc.PutObject(ctx, &s3.PutObjectInput{
					Bucket:       aws.String(bucket),
					StorageClass: s3types.StorageClassStandard,
				})
				return err
			}
		`),
	)
}

// A value computed at runtime becomes a conversion into the enum.
func TestMigrateConvertsDynamicEnumValue(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func put(ctx context.Context, svc *s3.S3, bucket, class string) error {
				_, err := svc.PutObject(&s3.PutObjectInput{
					Bucket:       aws.String(bucket),
					StorageClass: aws.String(class),
				})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
				s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
			)

			func put(ctx context.Context, svc *s3.Client, bucket, class string) error {
				_, err := svc.PutObject(ctx, &s3.PutObjectInput{
					Bucket:       aws.String(bucket),
					StorageClass: s3types.StorageClass(class),
				})
				return err
			}
		`),
	)
}

// A service with no manifest cannot be classified, so the file is left alone.
func TestMigrateBlockedByUnknownService(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/service/glacier"
			)

			func vaults(ctx context.Context, svc *glacier.Glacier) error {
				_, err := svc.ListVaults(&glacier.ListVaultsInput{})
				return err
			}
		`),
	)
}

// v1 made an enum field a pointer and v2 holds it by value, so the dereference
// the reader wrote comes off. The shape is known from the operation that
// produced it, which is what makes the field's enum-ness knowable.
func TestMigrateDropsEnumFieldDereference(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/service/s3"
			)

			func class(ctx context.Context, svc *s3.S3) (string, error) {
				out, err := svc.HeadObject(&s3.HeadObjectInput{})
				if err != nil {
					return "", err
				}
				return *out.StorageClass, nil
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func class(ctx context.Context, svc *s3.Client) (string, error) {
				out, err := svc.HeadObject(ctx, &s3.HeadObjectInput{})
				if err != nil {
					return "", err
				}
				return string(out.StorageClass), nil
			}
		`),
	)
}

// ZipRecruiter/cloudwatching builds a []*MetricDataQuery and reads a
// []*MetricDataResult back. v2 holds both by value, so the list is converted
// where it crosses the API and the code around it keeps its v1 shape. The
// helpers land once in the package.
func TestMigrateConvertsValueSlicesAtTheBoundary(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.GoProject("cw",
			test.GoMod(`
				module example.com/cw

				go 1.21

				require github.com/aws/aws-sdk-go v1.55.8
			`, `
				module example.com/cw

				go 1.24

				require (
					github.com/aws/aws-sdk-go-v2 v1.47.0
				)
			`),
			test.Golang(`
				package exportcloudwatch

				import (
					"context"

					"github.com/aws/aws-sdk-go/aws"
					"github.com/aws/aws-sdk-go/service/cloudwatch"
				)

				func read(ctx context.Context, cw *cloudwatch.CloudWatch, id string) error {
					mdq := make([]*cloudwatch.MetricDataQuery, 0, 100)
					mdq = append(mdq, &cloudwatch.MetricDataQuery{Id: aws.String(id)})

					gmdo, err := cw.GetMetricData(&cloudwatch.GetMetricDataInput{
						MetricDataQueries: mdq,
					})
					if err != nil {
						return err
					}
					for _, v := range gmdo.MetricDataResults {
						_ = *v.Id
					}
					return nil
				}
			`, `
				package exportcloudwatch

				import (
					"context"

					"github.com/aws/aws-sdk-go-v2/aws"
					"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
					cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
				)

				func read(ctx context.Context, cw *cloudwatch.Client, id string) error {
					mdq := make([]*cloudwatchtypes.MetricDataQuery, 0, 100)
					mdq = append(mdq, &cloudwatchtypes.MetricDataQuery{Id: aws.String(id)})

					gmdo, err := cw.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
						MetricDataQueries: awsCompatVals(mdq),
					})
					if err != nil {
						return err
					}
					for _, v := range awsCompatPtrs(gmdo.MetricDataResults) {
						_ = *v.Id
					}
					return nil
				}
			`),
			test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("exportcloudwatch")),
		),
	)
}

// The awserr idiom every caller writes: assert the error, then compare its
// code. v2 reports failures through smithy's interface, which errors.As matches
// — so the assertion becomes a declaration and a match, the accessors take
// smithy's names, and the ErrCode constant becomes the wire string it held.
func TestMigrateAwserrToSmithy(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws/awserr"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func exists(ctx context.Context, svc *s3.S3) (bool, error) {
				_, err := svc.HeadObject(&s3.HeadObjectInput{})
				if err != nil {
					if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
						return false, nil
					}
					return false, err
				}
				return true, nil
			}
		`, `
			package main

			import (
				"context"
				"errors"

				"github.com/aws/aws-sdk-go-v2/service/s3"
				"github.com/aws/smithy-go"
			)

			func exists(ctx context.Context, svc *s3.Client) (bool, error) {
				_, err := svc.HeadObject(ctx, &s3.HeadObjectInput{})
				if err != nil {
					var aerr smithy.APIError
					if errors.As(err, &aerr) && aerr.ErrorCode() == "NoSuchKey" {
						return false, nil
					}
					return false, err
				}
				return true, nil
			}
		`),
	)
}

// v2 deleted the per-service iface packages. A caller holding the interface —
// the shape mocks embed so they only implement what they use — gets a
// replacement generated into its own module, since the v2 SDK has nowhere to
// point the import at.
func TestMigrateGeneratesIfacePackage(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.GoProject("who",
			test.GoMod(`
				module example.com/who

				go 1.21

				require github.com/aws/aws-sdk-go v1.55.8
			`, `
				module example.com/who

				go 1.24
			`),
			test.Golang(`
				package who

				import (
					"context"

					"github.com/aws/aws-sdk-go/service/sts"
					"github.com/aws/aws-sdk-go/service/sts/stsiface"
				)

				func account(ctx context.Context, svc stsiface.STSAPI) (string, error) {
					out, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
					if err != nil {
						return "", err
					}
					return *out.Account, nil
				}
			`, `
				package who

				import (
					"context"

					"github.com/aws/aws-sdk-go-v2/service/sts"
					"example.com/who/internal/awsiface/stsiface"
				)

				func account(ctx context.Context, svc stsiface.STSAPI) (string, error) {
					out, err := svc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
					if err != nil {
						return "", err
					}
					return *out.Account, nil
				}
			`),
			test.Generated("internal/awsiface/stsiface/zz_awssdkv2_iface.go", `
				// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

				package stsiface

				import (
					"context"

					"github.com/aws/aws-sdk-go-v2/service/sts"
				)

				// STSAPI is the sts client's interface, regenerated against aws-sdk-go-v2.
				// Embedding it keeps a partial implementation partial, which is what the v1
				// package this replaces was for.
				type STSAPI interface {
					AssumeRole(context.Context, *sts.AssumeRoleInput, ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
					AssumeRoleWithSAML(context.Context, *sts.AssumeRoleWithSAMLInput, ...func(*sts.Options)) (*sts.AssumeRoleWithSAMLOutput, error)
					AssumeRoleWithWebIdentity(context.Context, *sts.AssumeRoleWithWebIdentityInput, ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error)
					AssumeRoot(context.Context, *sts.AssumeRootInput, ...func(*sts.Options)) (*sts.AssumeRootOutput, error)
					DecodeAuthorizationMessage(context.Context, *sts.DecodeAuthorizationMessageInput, ...func(*sts.Options)) (*sts.DecodeAuthorizationMessageOutput, error)
					GetAccessKeyInfo(context.Context, *sts.GetAccessKeyInfoInput, ...func(*sts.Options)) (*sts.GetAccessKeyInfoOutput, error)
					GetCallerIdentity(context.Context, *sts.GetCallerIdentityInput, ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
					GetDelegatedAccessToken(context.Context, *sts.GetDelegatedAccessTokenInput, ...func(*sts.Options)) (*sts.GetDelegatedAccessTokenOutput, error)
					GetFederationToken(context.Context, *sts.GetFederationTokenInput, ...func(*sts.Options)) (*sts.GetFederationTokenOutput, error)
					GetSessionToken(context.Context, *sts.GetSessionTokenInput, ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error)
					GetWebIdentityToken(context.Context, *sts.GetWebIdentityTokenInput, ...func(*sts.Options)) (*sts.GetWebIdentityTokenOutput, error)
				}
			`),
		),
	)
}

// v2 moved the transfer manager into its own feature module: the constructors
// take an S3 client where v1 took a session, the upload input became the plain
// PutObjectInput, and Upload takes a context. The import is aliased back so the
// references the file already spells keep compiling.
func TestMigrateS3TransferManager(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"
				"io"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
				"github.com/aws/aws-sdk-go/service/s3/s3manager"
			)

			func put(ctx context.Context, svc *s3.S3, bucket, key string, body io.Reader) error {
				uploader := s3manager.NewUploaderWithClient(svc)
				_, err := uploader.Upload(&s3manager.UploadInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
					Body:   body,
				})
				return err
			}
		`, `
			package main

			import (
				"context"
				"io"

				"github.com/aws/aws-sdk-go-v2/aws"
				s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func put(ctx context.Context, svc *s3.Client, bucket, key string, body io.Reader) error {
				uploader := s3manager.NewUploader(svc)
				_, err := uploader.Upload(ctx, &s3.PutObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
					Body:   body,
				})
				return err
			}
		`),
	)
}

// v1's session.New built a session in an expression and deferred a load failure
// to the first call. v2's loader returns that failure and an expression has
// nowhere to put it, so the load moves into a generated helper.
func TestMigrateInlineSessionNew(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.GoProject("inline",
			test.GoMod(`
				module example.com/inline

				go 1.24
			`, `
				module example.com/inline

				go 1.24

				require (
					github.com/aws/aws-sdk-go-v2 v1.47.0
					github.com/aws/aws-sdk-go-v2/config v1.33.5
				)
			`),
			test.Golang(`
				package inline

				import (
					"context"

					"github.com/aws/aws-sdk-go/aws/session"
					"github.com/aws/aws-sdk-go/service/s3"
				)

				type deps struct {
					Objects *s3.S3
				}

				func newDeps(ctx context.Context) *deps {
					return &deps{Objects: s3.New(session.New())}
				}
			`, `
				package inline

				import (
					"context"

					"github.com/aws/aws-sdk-go-v2/service/s3"
				)

				type deps struct {
					Objects *s3.Client
				}

				func newDeps(ctx context.Context) *deps {
					return &deps{Objects: s3.NewFromConfig(awsCompatConfig(ctx))}
				}
			`),
			test.Generated("zz_awssdkv2_config.go", configHelpersIn("inline")),
		),
	)
}

// v1 baked a waiter's attempt count and delay into the client method. v2 makes
// the waiter a type built from the client and takes the bound from the caller,
// which the v1 call site never named — so it comes from a generated constant.
func TestMigrateWaiter(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func gone(ctx context.Context, svc *s3.S3, bucket, key string) error {
				return svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				})
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func gone(ctx context.Context, svc *s3.Client, bucket, key string) error {
				return s3.NewObjectNotExistsWaiter(svc).Wait(ctx, &s3.HeadObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				}, awsCompatWaitDuration)
			}
		`),
		test.Generated("zz_awssdkv2_wait.go", `
			// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

			package main

			import "time"

			// awsCompatWaitDuration bounds a wait that aws-sdk-go v1 bounded with a
			// per-waiter attempt count and delay. v2 takes the bound from the caller and
			// there was nothing at the v1 call site to read it from, so this is a ceiling
			// rather than the v1 default — review any wait that has to fail sooner.
			const awsCompatWaitDuration = 5 * time.Minute
		`),
	)
}

// An enum field was a *string in v1, so a call site could set it from the
// address of a local as readily as from aws.String. Both carry the same value
// into v2's typed field.
func TestMigrateEnumFieldFromAddressOfLocal(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func put(ctx context.Context, svc *s3.S3, bucket, key, acl string) error {
				_, err := svc.PutObject(&s3.PutObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
					ACL:    &acl,
				})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
				s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
			)

			func put(ctx context.Context, svc *s3.Client, bucket, key, acl string) error {
				_, err := svc.PutObject(ctx, &s3.PutObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
					ACL:    s3types.ObjectCannedACL(acl),
				})
				return err
			}
		`),
	)
}

// v2's loader lives in a package called config, which Go code often uses for a
// local of its own. The import is aliased rather than left to be shadowed.
func TestMigrateAliasesConfigAroundALocal(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			type settings struct{ Region string }

			func load(ctx context.Context, config settings) error {
				_, err := session.NewSession(&aws.Config{Region: aws.String(config.Region)})
				return err
			}
		`, `
			package main

			import (
				"context"

				configaws "github.com/aws/aws-sdk-go-v2/config"
			)

			type settings struct{ Region string }

			func load(ctx context.Context, config settings) error {
				_, err := configaws.LoadDefaultConfig(ctx, configaws.WithRegion(config.Region))
				return err
			}
		`),
	)
}

// treeder/ecs-gen overrides the session's region per client. v1 took a trailing
// *aws.Config for that; v2 takes a functional option over the client's own
// Options.
func TestMigrateClientConstructorRegionOverride(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/ecs"
			)

			func newECS(ctx context.Context, region string, sess *session.Session) *ecs.ECS {
				return ecs.New(sess, aws.NewConfig().WithRegion(region))
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/ecs"
			)

			func newECS(ctx context.Context, region string, sess aws.Config) *ecs.Client {
				return ecs.NewFromConfig(sess, func(o *ecs.Options) { o.Region = region })
			}
		`),
	)
}

// v2 moved the instance metadata client into its own feature module and made
// every method answer with an output struct. Region keeps its v1 call shape
// through a generated helper.
func TestMigrateEc2MetadataRegion(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws/ec2metadata"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func region(ctx context.Context, sess *session.Session) (string, error) {
				metadata := ec2metadata.New(sess)
				return metadata.Region()
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				ec2metadata "github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
			)

			func region(ctx context.Context, sess aws.Config) (string, error) {
				metadata := ec2metadata.NewFromConfig(sess)
				return awsCompatIMDSRegion(ctx, metadata)
			}
		`),
		test.Generated("zz_awssdkv2_imds.go", `
			// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
			)

			// awsCompatIMDSRegion stands in for aws-sdk-go v1's EC2Metadata.Region. v2 takes
			// a context and answers with an output struct, so the call site keeps the shape
			// it was written in by going through here.
			func awsCompatIMDSRegion(ctx context.Context, c *imds.Client) (string, error) {
				out, err := c.GetRegion(ctx, &imds.GetRegionInput{})
				if err != nil {
					return "", err
				}
				return out.Region, nil
			}
		`),
	)
}

// v1 iterated pages by handing the client a callback; v2 hands out a paginator
// the caller drives. The loop is wrapped around the callback rather than the
// callback turned into a loop, which would mean rewriting every return in its
// body.
func TestMigratePaginator(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"
				"fmt"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func list(ctx context.Context, svc *s3.S3, bucket string) error {
				return svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
					Bucket: aws.String(bucket),
				}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
					fmt.Println(*page.Name, lastPage)
					return true
				})
			}
		`, `
			package main

			import (
				"context"
				"fmt"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func list(ctx context.Context, svc *s3.Client, bucket string) error {
				return func() error {
					awsFn := func(page *s3.ListObjectsV2Output, lastPage bool) bool {
						fmt.Println(*page.Name, lastPage)
						return true
					}
					awsPager := s3.NewListObjectsV2Paginator(svc, &s3.ListObjectsV2Input{
						Bucket: aws.String(bucket),
					})
					for awsPager.HasMorePages() {
						awsPage, awsErr := awsPager.NextPage(ctx)
						if awsErr != nil {
							return awsErr
						}
						if !awsFn(awsPage, !awsPager.HasMorePages()) {
							return nil
						}
					}
					return nil
				}()
			}
		`),
	)
}

// 99designs/iamy filters a list with enum values. v2 changed the list twice —
// pointers to values, and *string to the named enum — so neither the value-slice
// helpers nor the scalar enum rewrite reaches it on its own.
func TestMigrateEnumList(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/cloudformation"
			)

			func stacks(ctx context.Context, svc *cloudformation.CloudFormation) error {
				_, err := svc.ListStacks(&cloudformation.ListStacksInput{
					StackStatusFilter: []*string{
						aws.String("CREATE_COMPLETE"),
						aws.String("ROLLBACK_COMPLETE"),
					},
				})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/cloudformation"
				cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
			)

			func stacks(ctx context.Context, svc *cloudformation.Client) error {
				_, err := svc.ListStacks(ctx, &cloudformation.ListStacksInput{
					StackStatusFilter: []cloudformationtypes.StackStatus{
						cloudformationtypes.StackStatus("CREATE_COMPLETE"),
						cloudformationtypes.StackStatus("ROLLBACK_COMPLETE"),
					},
				})
				return err
			}
		`),
	)
}

// v1 reported a missing region through a sentinel error; v2 returns a typed one.
func TestMigrateMissingRegionSentinel(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"
				"fmt"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func head(ctx context.Context, svc *s3.S3) error {
				_, err := svc.ListBuckets(&s3.ListBucketsInput{})
				if err == aws.ErrMissingRegion {
					return fmt.Errorf("set AWS_REGION")
				}
				return err
			}
		`, `
			package main

			import (
				"context"
				"errors"
				"fmt"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func head(ctx context.Context, svc *s3.Client) error {
				_, err := svc.ListBuckets(ctx, &s3.ListBucketsInput{})
				if errors.As(err, new(*aws.MissingRegionError)) {
					return fmt.Errorf("set AWS_REGION")
				}
				return err
			}
		`),
	)
}

// A field v1 held as *bool and v2 holds as bool loses both the dereference that
// read it and the aws.Bool that set it. The shape is inferred from the slice the
// parameter declares, which is how the elements of a range over it are known.
func TestMigrateDepointeredField(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/iam"
			)

			func defaultVersion(ctx context.Context, versions []*iam.PolicyVersion) *iam.PolicyVersion {
				for _, version := range versions {
					if *version.IsDefaultVersion {
						return version
					}
				}
				return &iam.PolicyVersion{IsDefaultVersion: aws.Bool(true)}
			}
		`, `
			package main

			import (
				"context"

				iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
			)

			func defaultVersion(ctx context.Context, versions []*iamtypes.PolicyVersion) *iamtypes.PolicyVersion {
				for _, version := range versions {
					if version.IsDefaultVersion {
						return version
					}
				}
				return &iamtypes.PolicyVersion{IsDefaultVersion: true}
			}
		`),
	)
}

// v1's client carried the v1 client plumbing on itself; v2's carries none of
// it, so a read off a client value has nothing to resolve to.
func TestMigrateBlockedByClientField(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/service/s3"
			)

			func name(ctx context.Context, svc *s3.S3) string {
				return svc.ClientInfo.ServiceName
			}
		`),
	)
}

// A call on a service package the manifest cannot account for — a v1-only
// helper with no stand-in — has nowhere to land in v2.
func TestMigrateBlockedByV1OnlyHelper(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/service/s3"
			)

			func acls(ctx context.Context) []string {
				return s3.BucketCannedACL_Values()
			}
		`),
	)
}

// 99designs/iamy normalises a bucket's location constraint with a v1 helper v2
// dropped. It is small enough to carry over rather than leave the file behind
// for.
func TestMigrateBucketLocationHelper(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/service/s3"
			)

			func normalise(ctx context.Context, loc string) string {
				return s3.NormalizeBucketLocation(loc)
			}
		`, `
			package main

			import (
				"context"
			)

			func normalise(ctx context.Context, loc string) string {
				return awsCompatBucketLocation(loc)
			}
		`),
		test.Generated("zz_awssdkv2_helpers.go", `
			// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

			package main

			// awsCompatBucketLocation is aws-sdk-go v1's s3.NormalizeBucketLocation, which
			// v2 dropped. It reports the region a bucket's location constraint names, for
			// the two values that are not region IDs.
			func awsCompatBucketLocation(loc string) string {
				switch loc {
				case "":
					return "us-east-1"
				case "EU":
					return "eu-west-1"
				}
				return loc
			}
		`),
	)
}

// v2 has no shared-credentials provider: the loader reads the file and the
// profile itself. Where the loader options do not reach it — assigned to a
// config's field rather than set in its literal — a generated helper does.
func TestMigrateSharedCredentialsAssignedToAField(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func load(ctx context.Context, region, file, profile string) *s3.S3 {
				cfg := &aws.Config{Region: aws.String(region)}
				if profile != "" {
					cfg.Credentials = credentials.NewSharedCredentials(file, profile)
				}
				return s3.New(session.New(cfg))
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func load(ctx context.Context, region, file, profile string) *s3.Client {
				cfg := awsCompatConfig(ctx, config.WithRegion(region))
				if profile != "" {
					cfg.Credentials = awsCompatSharedCredentials(ctx, file, profile)
				}
				return s3.NewFromConfig(cfg)
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("main")),
	)
}

// v1 held an enum field as a *string, and a read that passed it on was written
// against that pointer. There is no telling from the call site what it expected,
// only that v1 gave it a *string, so the pointer is put back at the read.
func TestMigrateRestoresPointerOnEnumRead(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/service/s3"
			)

			func deref(s *string) string { return *s }

			func class(ctx context.Context, svc *s3.S3) (string, error) {
				out, err := svc.HeadObject(&s3.HeadObjectInput{})
				if err != nil {
					return "", err
				}
				return deref(out.StorageClass), nil
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func deref(s *string) string { return *s }

			func class(ctx context.Context, svc *s3.Client) (string, error) {
				out, err := svc.HeadObject(ctx, &s3.HeadObjectInput{})
				if err != nil {
					return "", err
				}
				return deref(aws.String(string(out.StorageClass))), nil
			}
		`),
	)
}

// A read feeding a field of the same kind on another shape needs nothing: v2
// lines both ends up.
func TestMigrateEnumReadIntoMatchingField(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/service/s3"
			)

			func copyClass(ctx context.Context, svc *s3.S3, out *s3.HeadObjectOutput) error {
				_, err := svc.PutObject(&s3.PutObjectInput{
					StorageClass: out.StorageClass,
				})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func copyClass(ctx context.Context, svc *s3.Client, out *s3.HeadObjectOutput) error {
				_, err := svc.PutObject(ctx, &s3.PutObjectInput{
					StorageClass: out.StorageClass,
				})
				return err
			}
		`),
	)
}

// v2 replaced the session pointer with a config value, so a nil test on one has
// nothing left to compare against.
func TestMigrateBlockedBySessionComparedToNil(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws/session"
			)

			var sess *session.Session

			func current(ctx context.Context) bool {
				return sess == nil
			}
		`),
	)
}

// v1 configured a session by filling in an aws.Config; v2 passes the loader
// functional options, one per field that has a counterpart.
func TestMigrateConfigOptionsBeyondRegion(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func load(ctx context.Context, region, endpoint, id, secret string) error {
				_, err := session.NewSession(&aws.Config{
					Region:      aws.String(region),
					Endpoint:    aws.String(endpoint),
					Credentials: credentials.NewStaticCredentials(id, secret, ""),
				})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/credentials"
			)

			func load(ctx context.Context, region, endpoint, id, secret string) error {
				_, err := config.LoadDefaultConfig(ctx, config.WithRegion(region), config.WithBaseEndpoint(endpoint), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(id, secret, "")))
				return err
			}
		`),
	)
}

// v2 has no shared-credentials provider: the loader reads the file and the
// profile itself.
func TestMigrateSharedCredentials(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func load(ctx context.Context, file, profile string) error {
				_, err := session.NewSession(&aws.Config{
					Credentials: credentials.NewSharedCredentials(file, profile),
				})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/config"
			)

			func load(ctx context.Context, file, profile string) error {
				_, err := config.LoadDefaultConfig(ctx, config.WithSharedCredentialsFiles([]string{file}), config.WithSharedConfigProfile(profile))
				return err
			}
		`),
	)
}

// andreimarcu/linx-server holds the upload input in a variable, which is not
// the shape that identifies an operation by its argument. The receiver is what
// says the call is a transfer-manager one.
func TestMigrateS3ManagerCallOnAVariable(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"
				"io"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
				"github.com/aws/aws-sdk-go/service/s3/s3manager"
			)

			func put(ctx context.Context, svc *s3.S3, bucket, key string, body io.Reader) error {
				uploader := s3manager.NewUploaderWithClient(svc)
				input := &s3manager.UploadInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
					Body:   body,
				}
				_, err := uploader.Upload(input)
				return err
			}
		`, `
			package main

			import (
				"context"
				"io"

				"github.com/aws/aws-sdk-go-v2/aws"
				s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func put(ctx context.Context, svc *s3.Client, bucket, key string, body io.Reader) error {
				uploader := s3manager.NewUploader(svc)
				input := &s3.PutObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
					Body:   body,
				}
				_, err := uploader.Upload(ctx, input)
				return err
			}
		`),
	)
}

// v1 let a caller reach the settings a client was built with through a Config
// field; v2's client answers with Options, whose Region is a plain string.
func TestMigrateClientConfigRegion(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/service/s3"
			)

			func region(ctx context.Context, svc *s3.S3) string {
				return *svc.Config.Region
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func region(ctx context.Context, svc *s3.Client) string {
				return svc.Options().Region
			}
		`),
	)
}

// A session's settings were reached the same way. The config value that
// replaced the session holds them directly.
func TestMigrateSessionConfigWrite(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
			)

			func region(ctx context.Context, sess *session.Session, region string) string {
				sess.Config.Region = aws.String(region)
				return *sess.Config.Region
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
			)

			func region(ctx context.Context, sess aws.Config, region string) string {
				sess.Region = region
				return sess.Region
			}
		`),
	)
}

// A read of a nested shape resolves as far as the enum at the end of it, and
// two functions using the same name for different things each see their own.
func TestMigrateNestedShapeRead(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"
				"fmt"

				"github.com/aws/aws-sdk-go/service/ec2"
				"github.com/aws/aws-sdk-go/service/ssm"
			)

			type row struct{ State string }

			func format(rows []row) {
				for _, i := range rows {
					fmt.Println(i.State)
				}
			}

			func managed(ctx context.Context, client *ssm.SSM) error {
				_, err := client.DescribeInstanceInformation(&ssm.DescribeInstanceInformationInput{})
				return err
			}

			func list(ctx context.Context, client *ec2.EC2) error {
				response, err := client.DescribeInstances(&ec2.DescribeInstancesInput{})
				if err != nil {
					return err
				}
				for _, reservation := range response.Reservations {
					for _, i := range reservation.Instances {
						fmt.Println(*i.State.Name)
					}
				}
				return nil
			}
		`, `
			package main

			import (
				"context"
				"fmt"

				"github.com/aws/aws-sdk-go-v2/service/ec2"
				"github.com/aws/aws-sdk-go-v2/service/ssm"
			)

			type row struct{ State string }

			func format(rows []row) {
				for _, i := range rows {
					fmt.Println(i.State)
				}
			}

			func managed(ctx context.Context, client *ssm.Client) error {
				_, err := client.DescribeInstanceInformation(ctx, &ssm.DescribeInstanceInformationInput{})
				return err
			}

			func list(ctx context.Context, client *ec2.Client) error {
				response, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
				if err != nil {
					return err
				}
				for _, reservation := range awsCompatPtrs(response.Reservations) {
					for _, i := range awsCompatPtrs(reservation.Instances) {
						fmt.Println(string(i.State.Name))
					}
				}
				return nil
			}
		`),
		test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("main")),
	)
}

// v1's fluent setters are gone in v2; the assignment that replaces one puts the
// value into whatever v2 holds the field in.
func TestMigrateFluentSetter(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/service/ssm"
			)

			func page(ctx context.Context, client *ssm.SSM, token string) error {
				input := &ssm.DescribeInstanceInformationInput{}
				input.SetNextToken(token)
				_, err := client.DescribeInstanceInformation(input)
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/ssm"
			)

			func page(ctx context.Context, client *ssm.Client, token string) error {
				input := &ssm.DescribeInstanceInformationInput{}
				input.NextToken = aws.String(token)
				_, err := client.DescribeInstanceInformation(ctx, input)
				return err
			}
		`),
	)
}

// compatHelpersIn is the slice-conversion file the migration writes into any
// package that needs it.
func compatHelpersIn(pkg string) string {
	return strings.Replace(compatHelpers, "package exportcloudwatch", "package "+pkg, 1)
}

const compatHelpers = `
				// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

				package exportcloudwatch

				// awsCompatPtrs views a value slice as the pointer slice aws-sdk-go v1
				// modelled it as, aliasing the original so writes through an element still
				// land. aws-sdk-go-v2 holds a list of shapes by value, and this keeps code
				// written against v1 working unchanged.
				func awsCompatPtrs[T any](vs []T) []*T {
					if vs == nil {
						return nil
					}
					ps := make([]*T, len(vs))
					for i := range vs {
						ps[i] = &vs[i]
					}
					return ps
				}

				// awsCompatVals is the inverse, for handing a v1-shaped list back to v2. A nil
				// element has no representation in a value slice and becomes the zero value.
				func awsCompatVals[T any](ps []*T) []T {
					if ps == nil {
						return nil
					}
					vs := make([]T, len(ps))
					for i, p := range ps {
						if p != nil {
							vs[i] = *p
						}
					}
					return vs
				}

				// awsCompatPtrsMap and awsCompatValsMap are the same pair for a map, which v2
				// likewise holds by value.
				func awsCompatPtrsMap[T any](m map[string]T) map[string]*T {
					if m == nil {
						return nil
					}
					out := make(map[string]*T, len(m))
					for k, v := range m {
						out[k] = &v
					}
					return out
				}

				func awsCompatValsMap[T any](m map[string]*T) map[string]T {
					if m == nil {
						return nil
					}
					out := make(map[string]T, len(m))
					for k, p := range m {
						if p != nil {
							out[k] = *p
						}
					}
					return out
				}

				// awsCompatPtrsSliceMap and awsCompatValsSliceMap are the same conversion one
				// level deeper, for a map whose values are lists.
				func awsCompatPtrsSliceMap[T any](m map[string][]T) map[string][]*T {
					if m == nil {
						return nil
					}
					out := make(map[string][]*T, len(m))
					for k, v := range m {
						out[k] = awsCompatPtrs(v)
					}
					return out
				}

				func awsCompatValsSliceMap[T any](m map[string][]*T) map[string][]T {
					if m == nil {
						return nil
					}
					out := make(map[string][]T, len(m))
					for k, v := range m {
						out[k] = awsCompatVals(v)
					}
					return out
				}
			`

// andreimarcu/linx-server builds its config field by field under conditionals,
// which does not reduce to one LoadDefaultConfig call. The load moves to the
// declaration instead, and the writes that follow retarget onto v2's own config
// — except the S3 addressing style, which v2 holds on the client's options, so
// it is hoisted into a local the constructor reads.
func TestMigrateImperativeConfigWithPathStyle(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package s3backend

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func New(region string, endpoint string, forcePathStyle bool) *s3.S3 {
				awsConfig := &aws.Config{}
				if region != "" {
					awsConfig.Region = aws.String(region)
				}
				if endpoint != "" {
					awsConfig.Endpoint = aws.String(endpoint)
				}
				if forcePathStyle {
					awsConfig.S3ForcePathStyle = aws.Bool(true)
				}

				sess := session.Must(session.NewSession(awsConfig))
				return s3.New(sess)
			}
		`, `
			package s3backend

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func New(region string, endpoint string, forcePathStyle bool) *s3.Client {
				awsConfig := awsCompatConfig(context.TODO())
				awsUsePathStyle := false
				if region != "" {
					awsConfig.Region = region
				}
				if endpoint != "" {
					awsConfig.BaseEndpoint = aws.String(endpoint)
				}
				if forcePathStyle {
					awsUsePathStyle = true
				}

				sess := awsConfig
				return s3.NewFromConfig(sess, func(o *s3.Options) { o.UsePathStyle = awsUsePathStyle })
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("s3backend")),
	)
}

// A map field v2 holds by value is converted where it crosses the API, the same
// way a list is. The operation's input reaches it through a variable, which is
// as common as writing the literal inline.
func TestMigrateValueMapAndInputVariable(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/s3"
			)

			func meta(ctx context.Context, svc *s3.S3, bucket, key string) (map[string]*string, error) {
				input := &s3.HeadObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				}
				out, err := svc.HeadObject(input)
				if err != nil {
					return nil, err
				}
				return out.Metadata, nil
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			func meta(ctx context.Context, svc *s3.Client, bucket, key string) (map[string]*string, error) {
				input := &s3.HeadObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				}
				out, err := svc.HeadObject(ctx, input)
				if err != nil {
					return nil, err
				}
				return awsCompatPtrsMap(out.Metadata), nil
			}
		`),
		test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("main")),
	)
}

// A call taking an operation input is not an operation on it. claranet/sshm
// hands one to json.Marshal, which must not collect a context.
func TestMigrateLeavesNonOperationTakingAnInput(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"
				"encoding/json"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/ssm"
			)

			func start(ctx context.Context, svc *ssm.SSM, target string) ([]byte, error) {
				input := &ssm.StartSessionInput{Target: aws.String(target)}
				if _, err := svc.StartSession(input); err != nil {
					return nil, err
				}
				return json.Marshal(input)
			}
		`, `
			package main

			import (
				"context"
				"encoding/json"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/ssm"
			)

			func start(ctx context.Context, svc *ssm.Client, target string) ([]byte, error) {
				input := &ssm.StartSessionInput{Target: aws.String(target)}
				if _, err := svc.StartSession(ctx, input); err != nil {
					return nil, err
				}
				return json.Marshal(input)
			}
		`),
	)
}

// A method implementing an operation names its input, and its body refers to
// that name. ZipRecruiter/cloudwatching stubs GetMetricData with a `gmdi`
// parameter, which the v2 signature has to keep.
func TestMigrateOperationDeclarationKeepsParameterName(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/service/cloudwatch"
			)

			type stub struct{}

			func (s stub) GetMetricData(gmdi *cloudwatch.GetMetricDataInput) (*cloudwatch.GetMetricDataOutput, error) {
				if gmdi.NextToken != nil {
					return nil, nil
				}
				return &cloudwatch.GetMetricDataOutput{}, nil
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
			)

			type stub struct{}

			func (s stub) GetMetricData(ctx context.Context, gmdi *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
				if gmdi.NextToken != nil {
					return nil, nil
				}
				return &cloudwatch.GetMetricDataOutput{}, nil
			}
		`),
	)
}

// codahale/sneaker declares its interfaces in a file with no call sites of its
// own. The signatures name context.Context whether or not they name the
// parameter, so the import comes in on their account alone.
func TestMigrateInterfaceOnlyFileImportsContext(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package sneaker

			import (
				"github.com/aws/aws-sdk-go/service/s3"
			)

			type ObjectStorage interface {
				ListObjects(*s3.ListObjectsInput) (*s3.ListObjectsOutput, error)
				DeleteObject(*s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
			}
		`, `
			package sneaker

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			type ObjectStorage interface {
				ListObjects(context.Context, *s3.ListObjectsInput, ...func(*s3.Options)) (*s3.ListObjectsOutput, error)
				DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
			}
		`),
	)
}

// configHelpersIn is the loader file the migration writes into any package that
// needs it.
func configHelpersIn(pkg string) string {
	return strings.Replace(configHelpers, "package awssdkv2config", "package "+pkg, 1)
}

const configHelpers = `
				// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

				package awssdkv2config

				import (
					"context"

					"github.com/aws/aws-sdk-go-v2/aws"
					"github.com/aws/aws-sdk-go-v2/config"
				)

				// awsCompatConfig stands in for aws-sdk-go v1's session.New, which built a
				// session in an expression and deferred a load failure to the first call. v2
				// returns that failure, and an expression cannot, so it panics the way
				// session.Must did.
				func awsCompatConfig(ctx context.Context, optFns ...func(*config.LoadOptions) error) aws.Config {
					cfg, err := config.LoadDefaultConfig(ctx, optFns...)
					if err != nil {
						panic(err)
					}
					return cfg
				}

				// awsCompatSharedCredentials stands in for aws-sdk-go v1's
				// credentials.NewSharedCredentials. v2 has no shared-credentials provider: the
				// loader reads the file and the profile itself, so the provider is taken off a
				// config loaded for exactly that.
				func awsCompatSharedCredentials(ctx context.Context, file, profile string) aws.CredentialsProvider {
					cfg, err := config.LoadDefaultConfig(ctx,
						config.WithSharedCredentialsFiles([]string{file}),
						config.WithSharedConfigProfile(profile))
					if err != nil {
						panic(err)
					}
					return cfg.Credentials
				}

				// awsCompatSessionConfig stands in for a session built from a config the caller
				// already had. v2's aws.Config is that config, so there is nothing left to
				// build and nothing left to fail; the error is kept so the call site reads as
				// it did.
				func awsCompatSessionConfig(cfg aws.Config) (aws.Config, error) {
					return cfg, nil
				}
			`

// aws-cloudmap-prometheus-sd keeps an aws.Config of its own and hands it to the
// session when it comes to build a client. v2's aws.Config is that config, so it
// is carried whole — and the settings inside it are brought to the v2 shape
// where they are written.
func TestMigrateCarriedConfig(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package discovery

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/servicediscovery"
			)

			type discovery struct {
				aws *aws.Config
			}

			func New(region string) *discovery {
				return &discovery{aws: &aws.Config{Region: &region}}
			}

			func (d *discovery) refresh(ctx context.Context) error {
				sess, err := session.NewSessionWithOptions(session.Options{
					Config: *d.aws,
				})
				if err != nil {
					return err
				}
				_, err = servicediscovery.New(sess).ListNamespaces(&servicediscovery.ListNamespacesInput{})
				return err
			}
		`, `
			package discovery

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
			)

			type discovery struct {
				aws *aws.Config
			}

			func New(region string) *discovery {
				return &discovery{aws: &aws.Config{Region: region}}
			}

			func (d *discovery) refresh(ctx context.Context) error {
				sess, err := awsCompatSessionConfig(*d.aws)
				if err != nil {
					return err
				}
				_, err = servicediscovery.NewFromConfig(sess).ListNamespaces(ctx, &servicediscovery.ListNamespacesInput{})
				return err
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("discovery")),
	)
}

// rgeorgiev583/s3secrets formats a list of objects in a file that never names
// the SDK — it is handed them by the file that fetched them. There are no
// imports to swap, but the fields it reads changed shape all the same, so what
// the parse-time type says about the values is what guides the rewrite. The
// go.mod is what makes that type available: the test framework registers each
// require against the module cache, which is where the type comes from.
func TestMigrateAdjunctFileFollowsShapes(t *testing.T) {
	requireModuleCache(t, "github.com/aws/aws-sdk-go@v1.55.8")
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.24

				require github.com/aws/aws-sdk-go v1.55.8
			`, `
				module example.com/app

				go 1.24
			`),
			test.Golang(`
				package main

				import (
					"context"

					"github.com/aws/aws-sdk-go/service/s3"
				)

				func keys(ctx context.Context, svc *s3.S3, bucket string) ([]*s3.Object, error) {
					out, err := svc.ListObjects(&s3.ListObjectsInput{Bucket: &bucket})
					if err != nil {
						return nil, err
					}
					return out.Contents, nil
				}
			`, `
				package main

				import (
					"context"

					"github.com/aws/aws-sdk-go-v2/service/s3"
					s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
				)

				func keys(ctx context.Context, svc *s3.Client, bucket string) ([]*s3types.Object, error) {
					out, err := svc.ListObjects(ctx, &s3.ListObjectsInput{Bucket: &bucket})
					if err != nil {
						return nil, err
					}
					return awsCompatPtrs(out.Contents), nil
				}
			`),
			test.Golang(`
				package main

				import (
					"context"
					"fmt"
				)

				func describe(ctx context.Context) {
					objects, _ := keys(ctx, nil, "b")
					for _, k := range objects {
						fmt.Println(*k.Key, *k.StorageClass)
					}
				}
			`, `
				package main

				import (
					"context"
					"fmt"
				)

				func describe(ctx context.Context) {
					objects, _ := keys(ctx, nil, "b")
					for _, k := range objects {
						fmt.Println(*k.Key, string(k.StorageClass))
					}
				}
			`),
			test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("main")),
		),
	)
}

// A file that never names the SDK is visited all the same — it may hold values
// of its shapes, handed to it by the file that fetched them — but nothing in it
// is rewritten on a guess. Without a type to go on, it is left exactly as it is.
func TestMigrateLeavesUnrelatedFilesAlone(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			type record struct {
				Key          *string
				StorageClass *string
				Contents     []*record
			}

			func describe(rs []*record) {
				for _, r := range rs {
					fmt.Println(*r.Key, *r.StorageClass, len(r.Contents))
				}
			}
		`),
	)
}

// moduleCacheEnv gates the tests whose fixtures type against a module rather
// than against the sources in the fixture itself. The parser reads such a module
// out of $GOMODCACHE, which a clean CI checkout has nothing in, so they are off
// by default and run on demand.
const moduleCacheEnv = "MODULE_CACHE_TESTS"

// requireModuleCache skips a test whose fixture needs a real type from a module
// the parser reads out of the module cache. Every other test in this file types
// its fixture from the sources in it, and needs nothing fetched.
func requireModuleCache(t *testing.T, module string) {
	t.Helper()
	if os.Getenv(moduleCacheEnv) == "" {
		t.Skipf("set %s=1 to run the tests that type their fixtures against %s", moduleCacheEnv, module)
	}
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Skipf("cannot locate the module cache: %v", err)
	}
	dir := filepath.Join(strings.TrimSpace(string(out)), filepath.FromSlash(module))
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("%s is not in the module cache; run `go mod download %s` in a scratch module to prime it",
			module, module)
	}
}
