/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"os"
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// aws/aws-node-termination-handler's shape: a v1 shape printed with %s, which
// called its String method. v2's types have none, so the shape is rendered by
// the generated copy of v1's printer instead.
func TestMigratePrintedShape(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"
				"fmt"
				"log"

				"github.com/aws/aws-sdk-go/service/sqs"
			)

			func drain(ctx context.Context, client *sqs.SQS, url string) error {
				out, err := client.ReceiveMessage(&sqs.ReceiveMessageInput{QueueUrl: &url})
				if err != nil {
					return err
				}
				log.Printf("received %d", len(out.Messages))
				for _, message := range out.Messages {
					fmt.Printf("message: %s\n", message)
				}
				fmt.Println(out)
				return nil
			}
		`, `
			package main

			import (
				"context"
				"fmt"
				"log"

				"github.com/aws/aws-sdk-go-v2/service/sqs"
			)

			func drain(ctx context.Context, client *sqs.Client, url string) error {
				out, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{QueueUrl: &url})
				if err != nil {
					return err
				}
				log.Printf("received %d", len(awsCompatPtrs(out.Messages)))
				for _, message := range awsCompatPtrs(out.Messages) {
					fmt.Printf("message: %s\n", awsCompatPrettify(message))
				}
				fmt.Println(awsCompatPrettify(out))
				return nil
			}
		`),
		test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("main")),
		test.Generated("zz_awssdkv2_prettify.go", testdataFile(t, "prettify.go.txt")),
	)
}

func testdataFile(t *testing.T, name string) string {
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Credentials' secret key was tagged sensitive in v1 and printed masked; the
// generated printer masks it too, which v2's own types no longer would.
func TestMigratePrintedSensitiveShape(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"
				"log"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/sts"
			)

			func assume(ctx context.Context, client *sts.STS, role string) error {
				out, err := client.AssumeRole(&sts.AssumeRoleInput{RoleArn: aws.String(role)})
				if err != nil {
					return err
				}
				log.Printf("assumed: %v", out.Credentials)
				return nil
			}
		`, `
			package main

			import (
				"context"
				"log"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/sts"
			)

			func assume(ctx context.Context, client *sts.Client, role string) error {
				out, err := client.AssumeRole(ctx, &sts.AssumeRoleInput{RoleArn: aws.String(role)})
				if err != nil {
					return err
				}
				log.Printf("assumed: %v", awsCompatPrettify(out.Credentials))
				return nil
			}
		`),
		test.Generated("zz_awssdkv2_prettify.go", testdataFile(t, "prettify_sts.go.txt")),
	)
}

// A mock compares the values it is handed, so a shape passed to Called stays
// the shape.
func TestMigrateShapeHandedToMockStaysShape(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "github.com/aws/aws-sdk-go/service/sqs"

			type recorder struct{ calls []any }

			func (r *recorder) Called(args ...any) { r.calls = append(r.calls, args...) }

			func (r *recorder) SendMessage(in *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
				r.Called(in)
				return nil, nil
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/service/sqs"
			)

			type recorder struct{ calls []any }

			func (r *recorder) Called(args ...any) { r.calls = append(r.calls, args...) }

			func (r *recorder) SendMessage(ctx context.Context, in *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
				r.Called(in)
				return nil, nil
			}
		`),
	)
}

// v1's String and GoString on a shape both returned what Prettify did.
func TestMigrateShapeStringCall(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "github.com/aws/aws-sdk-go/service/sqs"

			func describe(m *sqs.Message) string {
				return m.String()
			}
		`, `
			package main

			import sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

			func describe(m *sqstypes.Message) string {
				return awsCompatPrettify(m)
			}
		`),
		test.Generated("zz_awssdkv2_prettify.go", testdataFile(t, "prettify.go.txt")),
	)
}
