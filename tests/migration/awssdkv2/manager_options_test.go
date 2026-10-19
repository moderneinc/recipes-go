/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// oxyno-zeta/s3-proxy's shape: an uploader built with options setting its
// part limit, and the default limit named as a constant. v2 builds the
// uploader from an S3 client and types the limit int32, which the write
// converts to; v1's constant was untyped, which its value still is.
func TestMigrateUploaderPartLimit(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package s3client

			import (
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/s3/s3manager"
			)

			const DefaultMaxUploadParts = s3manager.MaxUploadParts

			type Bucket struct{ MaxUploadParts int }

			func uploader(sess *session.Session, b Bucket) *s3manager.Uploader {
				return s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
					u.MaxUploadParts = b.MaxUploadParts
				})
			}
		`, `
			package s3client

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
				"github.com/aws/aws-sdk-go-v2/service/s3"
			)

			const DefaultMaxUploadParts = 10000

			type Bucket struct{ MaxUploadParts int }

			func uploader(sess aws.Config, b Bucket) *s3manager.Uploader {
				return s3manager.NewUploader(s3.NewFromConfig(sess), func(u *s3manager.Uploader) {
					u.MaxUploadParts = int32(b.MaxUploadParts)
				})
			}
		`),
	)
}
