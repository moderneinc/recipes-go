/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"strings"
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// v2 made DynamoDB's AttributeValue an interface with a struct per member. A
// value setting one scalar member becomes that member's struct, the pointer the
// field needed coming off.
func TestMigrateAttributeValueLiteral(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/dynamodb"
			)

			func get(ctx context.Context, svc *dynamodb.DynamoDB, id string) error {
				_, err := svc.GetItem(&dynamodb.GetItemInput{
					TableName: aws.String("t"),
					Key:       map[string]*dynamodb.AttributeValue{"id": {S: aws.String(id)}},
				})
				return err
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/dynamodb"
				dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
			)

			func get(ctx context.Context, svc *dynamodb.Client, id string) error {
				_, err := svc.GetItem(ctx, &dynamodb.GetItemInput{
					TableName: aws.String("t"),
					Key:       map[string]dynamodbtypes.AttributeValue{"id": &dynamodbtypes.AttributeValueMemberS{Value: id}},
				})
				return err
			}
		`),
	)
}

// A member read goes through a getter answering nil where the value holds
// another member, as v1's unset field did, so a dereference of it fails where
// v1's did.
func TestMigrateAttributeValueMemberRead(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/dynamodb"
			)

			func names(ctx context.Context, svc *dynamodb.DynamoDB) ([]string, error) {
				out, err := svc.Scan(&dynamodb.ScanInput{TableName: aws.String("t")})
				if err != nil {
					return nil, err
				}
				var names []string
				for _, item := range out.Items {
					names = append(names, *item["name"].S)
				}
				return names, nil
			}
		`, `
			package main

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/dynamodb"
			)

			func names(ctx context.Context, svc *dynamodb.Client) ([]string, error) {
				out, err := svc.Scan(ctx, &dynamodb.ScanInput{TableName: aws.String("t")})
				if err != nil {
					return nil, err
				}
				var names []string
				for _, item := range out.Items {
					names = append(names, *awsCompatAttributeS(item["name"]))
				}
				return names, nil
			}
		`),
		test.Generated("zz_awssdkv2_attributevalue.go", attributeValueHelpersIn("main")),
	)
}

// glassechidna/lambdalite3's shape: a key taken from a pointer the enclosing if
// has tested, a body io.ReadAll returned, and both read back. Neither can be
// nil, so neither reaches v1's empty value.
func TestMigrateAttributeValueProvenValues(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package changesets

			import (
				"context"
				"io"
				"io/ioutil"

				"github.com/aws/aws-sdk-go/service/dynamodb"
			)

			type store struct {
				svc      *dynamodb.DynamoDB
				startKey *string
			}

			func (d *store) put(ctx context.Context, r io.Reader) error {
				body, err := ioutil.ReadAll(r)
				if err != nil {
					return err
				}
				pk := "changesets"
				_, err = d.svc.PutItemWithContext(ctx, &dynamodb.PutItemInput{
					Item: map[string]*dynamodb.AttributeValue{
						"pk": {S: &pk},
						"cs": {B: body},
					},
				})
				return err
			}

			func (d *store) key() map[string]*dynamodb.AttributeValue {
				var esk map[string]*dynamodb.AttributeValue
				if d.startKey != nil {
					esk = map[string]*dynamodb.AttributeValue{"sk": {S: d.startKey}}
				}
				return esk
			}

			func (d *store) read(item map[string]*dynamodb.AttributeValue) []byte {
				d.startKey = item["sk"].S
				return item["cs"].B
			}
		`, `
			package changesets

			import (
				"context"
				"io"
				"io/ioutil"

				"github.com/aws/aws-sdk-go-v2/service/dynamodb"
				dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
			)

			type store struct {
				svc      *dynamodb.Client
				startKey *string
			}

			func (d *store) put(ctx context.Context, r io.Reader) error {
				body, err := ioutil.ReadAll(r)
				if err != nil {
					return err
				}
				pk := "changesets"
				_, err = d.svc.PutItem(ctx, &dynamodb.PutItemInput{
					Item: map[string]dynamodbtypes.AttributeValue{
						"pk": &dynamodbtypes.AttributeValueMemberS{Value: pk},
						"cs": &dynamodbtypes.AttributeValueMemberB{Value: body},
					},
				})
				return err
			}

			func (d *store) key() map[string]dynamodbtypes.AttributeValue {
				var esk map[string]dynamodbtypes.AttributeValue
				if d.startKey != nil {
					esk = map[string]dynamodbtypes.AttributeValue{"sk": &dynamodbtypes.AttributeValueMemberS{Value: *d.startKey}}
				}
				return esk
			}

			func (d *store) read(item map[string]dynamodbtypes.AttributeValue) []byte {
				d.startKey = awsCompatAttributeS(item["sk"])
				return awsCompatAttributeB(item["cs"])
			}
		`),
		test.Generated("zz_awssdkv2_attributevalue.go", attributeValueHelpersIn("changesets")),
	)
}

// What has no exact v2 form holds the file back: a member v2 holds in a
// different shape, a pointer that may be nil — which v1 sent as an empty value
// the service rejected — a member written in place, and a value held by value.
func TestMigrateBlockedByAttributeValueShapes(t *testing.T) {
	for name, body := range map[string]string{
		"map member":      `_ = &dynamodb.AttributeValue{M: map[string]*dynamodb.AttributeValue{}}`,
		"unproven member": `_ = &dynamodb.AttributeValue{S: p}`,
		"member write":    `av := &dynamodb.AttributeValue{S: aws.String("x")}; av.S = p`,
		"held by value":   `var av dynamodb.AttributeValue; _ = av`,
	} {
		t.Run(name, func(t *testing.T) {
			migrateSpec().RewriteRun(t,
				test.Golang(`
					package main

					import (
						"github.com/aws/aws-sdk-go/aws"
						"github.com/aws/aws-sdk-go/service/dynamodb"
					)

					var _ = aws.String

					func f(p *string) {
						`+body+`
					}
				`),
			)
		})
	}
}

func attributeValueHelpersIn(pkg string) string {
	return strings.Replace(attributeValueHelpers, "package main", "package "+pkg, 1)
}

const attributeValueHelpers = `
	// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

	package main

	import "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	// v1's AttributeValue held a pointer field per member, nil unless that member
	// was the one set. v2 made each member a type of its own; these read one back
	// the way v1's field did.

	func awsCompatAttributeS(av types.AttributeValue) *string {
		if m, ok := av.(*types.AttributeValueMemberS); ok {
			return &m.Value
		}
		return nil
	}

	func awsCompatAttributeN(av types.AttributeValue) *string {
		if m, ok := av.(*types.AttributeValueMemberN); ok {
			return &m.Value
		}
		return nil
	}

	func awsCompatAttributeB(av types.AttributeValue) []byte {
		if m, ok := av.(*types.AttributeValueMemberB); ok {
			return m.Value
		}
		return nil
	}

	func awsCompatAttributeBOOL(av types.AttributeValue) *bool {
		if m, ok := av.(*types.AttributeValueMemberBOOL); ok {
			return &m.Value
		}
		return nil
	}

	func awsCompatAttributeNULL(av types.AttributeValue) *bool {
		if m, ok := av.(*types.AttributeValueMemberNULL); ok {
			return &m.Value
		}
		return nil
	}
`
