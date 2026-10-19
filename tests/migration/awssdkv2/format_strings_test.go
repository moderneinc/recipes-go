/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// Raising the go directive to 1.24 turns on vet's check for a printf call
// whose format is not a constant, so code that vetted clean would no longer.
// Each such call passes its format through "%s", as vet suggests; a constant
// format, or one with arguments to format, is left alone.
func TestMigrateFixesNonConstantFormatStrings(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.GoProject("cw",
			test.GoMod(`
				module example.com/cw

				go 1.22

				require github.com/aws/aws-sdk-go v1.55.8

				require github.com/jmespath/go-jmespath v0.4.0 // indirect
			`, `
				module example.com/cw

				go 1.24

				require github.com/jmespath/go-jmespath v0.4.0 // indirect

				require (
					github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.73.3
				)

				require (
					github.com/aws/aws-sdk-go-v2 v1.47.3 // indirect
					github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 // indirect
					github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 // indirect
					github.com/aws/smithy-go v1.28.5 // indirect
				)
			`),
			test.GoSum(cloudwatchV1Sum, `
				github.com/aws/aws-sdk-go v1.55.8 h1:JRmEUbU52aJQZ2AjX4q4Wu7t4uZjOu71uyNmaWlUkJQ=
				github.com/aws/aws-sdk-go v1.55.8/go.mod h1:ZkViS9AqA6otK+JBBNH2++sx1sgxrPKcSzPPvQkUtXk=
				github.com/aws/aws-sdk-go-v2 v1.47.3 h1:HRoT/MkNi3bjO5xF4dCzLzvVmkvRGHj3/cLINDnJejk=
				github.com/aws/aws-sdk-go-v2 v1.47.3/go.mod h1:xSl+OSI5ZUw6bSMupWc218zE5Pl+65MCCu3ySWFY99A=
				github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 h1:CqL8NN3p4I0JTwwV5MEoatIl4Y2amZ2X9dRD+0hbRL0=
				github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6/go.mod h1:exdJ32hMqXl0Wyo6NwqEC5yIhtua3+PDaKE8P29m4qE=
				github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 h1:FpBq24TG6WVSObTNhQOv7Q0088rounUUlwn3zlNItkM=
				github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6/go.mod h1:ONzdPVYC95oBOu0VWNEsKEntcoemP3xVKjT+qWml1s4=
				github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.73.3 h1:VmR1+YQW2vc3Sy2fLGSFtijUq9bAOMzzFxNC1zmHiA0=
				github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.73.3/go.mod h1:Q6H59zrEAgECYo4bjSh3IAvr+A+KklmThMoZjqnXbHY=
				github.com/aws/smithy-go v1.28.5 h1:OqbqKtzY9NUnr2yAe9+hiTdFb20tNaK/lI0PMOGXX9k=
				github.com/aws/smithy-go v1.28.5/go.mod h1:YE2RhdIuDbA5E5bTdciG9KrW3+TiEONeUWCqxX9i1Fc=
				github.com/jmespath/go-jmespath v0.4.0 h1:BEgLn5cpjn8UN1mAw4NjwDrS35OdebyEtFe+9YPoQUg=
				github.com/jmespath/go-jmespath v0.4.0/go.mod h1:T8mJZnbsbmF+m6zOOFylbeCJqk5+pHWvzYPziyZiYoo=
			`),
			test.Golang(cloudwatchV1Source, `
				package main

				import "github.com/aws/aws-sdk-go-v2/service/cloudwatch"

				var _ *cloudwatch.Client
			`),
			test.Golang(`
				package main

				import (
					"fmt"
					"log"
					"testing"
				)

				const banner = "ready"

				type event struct{ msg string }

				func report(t *testing.T, msg string, e event, parts []string) error {
					log.Printf(msg)
					fmt.Printf(e.msg + "\\n")
					t.Fatalf(parts[0])
					fmt.Println(fmt.Sprintf(msg))
					fmt.Printf(banner)
					fmt.Printf("ok")
					fmt.Printf(msg, 1)
					raise(msg)
					return fmt.Errorf(msg)
				}

				func raise(format string, args ...interface{}) {
					fmt.Println(fmt.Sprintf(format, args...))
				}
			`, `
				package main

				import (
					"fmt"
					"log"
					"testing"
				)

				const banner = "ready"

				type event struct{ msg string }

				func report(t *testing.T, msg string, e event, parts []string) error {
					log.Printf("%s", msg)
					fmt.Printf("%s", e.msg + "\\n")
					t.Fatalf("%s", parts[0])
					fmt.Println(fmt.Sprintf("%s", msg))
					fmt.Printf(banner)
					fmt.Printf("ok")
					fmt.Printf(msg, 1)
					raise("%s", msg)
					return fmt.Errorf("%s", msg)
				}

				func raise(format string, args ...interface{}) {
					fmt.Println(fmt.Sprintf(format, args...))
				}
			`),
		),
	)
}

// With no go.mod there is no directive being raised, so nothing turns the
// check on.
func TestMigrateLeavesFormatStringsWithoutGoMod(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(cloudwatchV1Source, `
			package main

			import "github.com/aws/aws-sdk-go-v2/service/cloudwatch"

			var _ *cloudwatch.Client
		`),
		test.Golang(`
			package main

			import "log"

			func report(msg string) {
				log.Printf(msg)
			}
		`),
	)
}
