/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// attributePortGenerated asserts the port of dynamodbattribute is written,
// leaving its content to the compile test that runs it against v1's.
func attributePortGenerated() []test.Sources {
	var specs []test.Sources
	for _, name := range []string{"boundary", "converter", "decode", "encode", "field", "fields", "tag"} {
		spec := test.Generated("internal/awscompat/dynamodbattribute/zz_awssdkv2_"+name+".go", "")
		spec.After = nil
		specs = append(specs, spec)
	}
	return specs
}

// Optum/dce's shape: items written and read through dynamodbattribute. v2's
// attributevalue would store and read them differently, so the import moves
// to v1's own package ported onto v2's AttributeValue, and the calls stay.
func TestMigrateDynamodbAttributeToPort(t *testing.T) {
	specs := []test.Sources{
		test.GoMod(`
			module example.com/leases

			go 1.24

			require github.com/aws/aws-sdk-go v1.55.8
		`, `
			module example.com/leases

			go 1.24

			require (
				github.com/aws/aws-sdk-go-v2 v1.47.3
				github.com/aws/aws-sdk-go-v2/service/dynamodb v1.70.3
			)

			require (
				github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 // indirect
				github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 // indirect
				github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.21 // indirect
				github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.13.6 // indirect
				github.com/aws/smithy-go v1.28.5 // indirect
			)
		`),
		test.Golang(`
			package leases

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/dynamodb"
				"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
			)

			type Lease struct {
				ID     string `+"`json:\"id\"`"+`
				Status string `+"`json:\"status\"`"+`
			}

			func Put(db *dynamodb.DynamoDB, l Lease) error {
				item, err := dynamodbattribute.MarshalMap(l)
				if err != nil {
					return err
				}
				_, err = db.PutItem(&dynamodb.PutItemInput{TableName: aws.String("leases"), Item: item})
				return err
			}

			func Get(db *dynamodb.DynamoDB, id string) (*Lease, error) {
				key, err := dynamodbattribute.MarshalMap(map[string]string{"id": id})
				if err != nil {
					return nil, err
				}
				out, err := db.GetItem(&dynamodb.GetItemInput{TableName: aws.String("leases"), Key: key})
				if err != nil {
					return nil, err
				}
				var l Lease
				return &l, dynamodbattribute.UnmarshalMap(out.Item, &l)
			}

			func List(db *dynamodb.DynamoDB) ([]Lease, error) {
				out, err := db.Scan(&dynamodb.ScanInput{TableName: aws.String("leases")})
				if err != nil {
					return nil, err
				}
				var leases []Lease
				return leases, dynamodbattribute.UnmarshalListOfMaps(out.Items, &leases)
			}
		`, `
			package leases

			import (
				"context"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/dynamodb"
				"example.com/leases/internal/awscompat/dynamodbattribute"
			)

			type Lease struct {
				ID     string `+"`json:\"id\"`"+`
				Status string `+"`json:\"status\"`"+`
			}

			func Put(db *dynamodb.Client, l Lease) error {
				item, err := dynamodbattribute.MarshalMap(l)
				if err != nil {
					return err
				}
				_, err = db.PutItem(context.TODO(), &dynamodb.PutItemInput{TableName: aws.String("leases"), Item: item})
				return err
			}

			func Get(db *dynamodb.Client, id string) (*Lease, error) {
				key, err := dynamodbattribute.MarshalMap(map[string]string{"id": id})
				if err != nil {
					return nil, err
				}
				out, err := db.GetItem(context.TODO(), &dynamodb.GetItemInput{TableName: aws.String("leases"), Key: key})
				if err != nil {
					return nil, err
				}
				var l Lease
				return &l, dynamodbattribute.UnmarshalMap(out.Item, &l)
			}

			func List(db *dynamodb.Client) ([]Lease, error) {
				out, err := db.Scan(context.TODO(), &dynamodb.ScanInput{TableName: aws.String("leases")})
				if err != nil {
					return nil, err
				}
				var leases []Lease
				return leases, dynamodbattribute.UnmarshalListOfMaps(out.Items, &leases)
			}
		`).WithPath("leases.go"),
	}
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.GoProject("leases", append(specs, attributePortGenerated()...)...),
	)
}

// v1's Marshaler and Unmarshaler take v1's AttributeValue, which the port
// keeps internal, so a type implementing either holds its package on v1.
func TestMigrateBlockedByAttributeMarshaler(t *testing.T) {
	for name, decl := range map[string]string{
		"method": `
			func (c Cents) MarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
				av.N = aws.String("0")
				return nil
			}
		`,
		"interface": `var _ dynamodbattribute.Marshaler = nil`,
	} {
		t.Run(name, func(t *testing.T) {
			test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
				test.GoProject("ledger",
					test.GoMod(`
						module example.com/ledger

						go 1.24

						require github.com/aws/aws-sdk-go v1.55.8
					`),
					test.Golang(`
						package ledger

						import (
							"github.com/aws/aws-sdk-go/aws"
							"github.com/aws/aws-sdk-go/service/dynamodb"
							"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
						)

						type Cents int64

						var _ = aws.String
						var _ *dynamodb.AttributeValue

						`+decl+`

						func Encode(v interface{}) (map[string]*dynamodb.AttributeValue, error) {
							return dynamodbattribute.MarshalMap(v)
						}
					`).WithPath("ledger.go"),
				),
			)
		})
	}
}
