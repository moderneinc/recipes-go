/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// in4it/ecs-deploy's shape: a provider held as *credentials.Credentials, whose
// Get answers the credentials.Value the code stores as JSON and reads back. The
// provider becomes v2's interface, and a stand-in keeps v1's value, and so the
// stored JSON, as it was.
func TestMigrateHeldCredentialsAndValue(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package ecs

			import (
				"encoding/json"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/aws/credentials"
				"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
				"github.com/aws/aws-sdk-go/aws/session"
				"github.com/aws/aws-sdk-go/service/sts"
			)

			type IAM struct{ stsAssumingRole *sts.STS }

			func (e *IAM) AssumeRole(roleArn, roleSessionName, prevCreds string) (*credentials.Credentials, string, error) {
				sess := session.Must(session.NewSession())
				var value credentials.Value
				var creds *credentials.Credentials
				if prevCreds != "" {
					if err := json.Unmarshal([]byte(prevCreds), &value); err == nil {
						creds = credentials.NewStaticCredentialsFromCreds(value)
						e.stsAssumingRole = sts.New(sess, &aws.Config{Credentials: creds})
					}
				}
				if creds == nil {
					creds = stscreds.NewCredentials(sess, roleArn, func(a *stscreds.AssumeRoleProvider) {
						a.RoleSessionName = roleSessionName
					})
				}
				valCreds, err := creds.Get()
				if err != nil {
					return creds, "", err
				}
				jsonCreds, err := json.Marshal(valCreds)
				if err != nil {
					return creds, "", err
				}
				return creds, string(jsonCreds), nil
			}
		`, `
			package ecs

			import (
				"context"
				"encoding/json"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/config"
				"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
				"github.com/aws/aws-sdk-go-v2/service/sts"
			)

			type IAM struct{ stsAssumingRole *sts.Client }

			func (e *IAM) AssumeRole(roleArn, roleSessionName, prevCreds string) (aws.CredentialsProvider, string, error) {
				sess, err := config.LoadDefaultConfig(context.TODO())
				if err != nil {
					panic(err)
				}
				var value awsCompatCredentialsValue
				var creds aws.CredentialsProvider
				if prevCreds != "" {
					if err := json.Unmarshal([]byte(prevCreds), &value); err == nil {
						creds = awsCompatStaticCredentials(value)
						e.stsAssumingRole = sts.NewFromConfig(awsCompatMergeConfig(sess, &aws.Config{Credentials: creds}))
					}
				}
				if creds == nil {
					creds = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(sts.NewFromConfig(sess), roleArn, func(a *stscreds.AssumeRoleOptions) {
						a.RoleSessionName = roleSessionName
					}))
				}
				valCreds, err := awsCompatGetCredentials(context.TODO(), creds)
				if err != nil {
					return creds, "", err
				}
				jsonCreds, err := json.Marshal(valCreds)
				if err != nil {
					return creds, "", err
				}
				return creds, string(jsonCreds), nil
			}
		`),
		test.Generated("zz_awssdkv2_config.go", configHelpersIn("ecs")),
		test.Generated("zz_awssdkv2_credentialsvalue.go", testdataFile(t, "credentials_value.go.txt")),
	)
}
