/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package mapstructurev2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/mapstructurev2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func swapSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&mapstructurev2.SwapMapstructureImports{})
}

// PaddleHQ/go-aws-ssm decodes a parameter map into a caller's struct; Decode is
// unchanged in v2, so only the import moves.
func TestSwapImportsDecode(t *testing.T) {
	swapSpec().RewriteRun(t,
		test.Golang(`
			package awsssm

			import (
				"fmt"

				"github.com/mitchellh/mapstructure"
			)

			func (p *Parameters) Decode(output interface{}) error {
				if err := mapstructure.Decode(p.getKeyValueMap(), output); err != nil {
					return fmt.Errorf("decode: %w", err)
				}
				return nil
			}
		`, `
			package awsssm

			import (
				"fmt"

				"github.com/go-viper/mapstructure/v2"
			)

			func (p *Parameters) Decode(output interface{}) error {
				if err := mapstructure.Decode(p.getKeyValueMap(), output); err != nil {
					return fmt.Errorf("decode: %w", err)
				}
				return nil
			}
		`),
	)
}

// veraison/services type-asserts the decode error to *mapstructure.Error and
// ranges its Errors field. v2 has no such struct, so the file is left alone.
func TestSwapImportsBlockedByErrorTypeAssertion(t *testing.T) {
	swapSpec().RewriteRun(t,
		test.Golang(`
			package config

			import (
				"strings"

				"github.com/mitchellh/mapstructure"
			)

			func decode(decoder *mapstructure.Decoder, input map[string]interface{}) error {
				err := decoder.Decode(input)
				if err == nil {
					return nil
				}
				msErr, ok := err.(*mapstructure.Error)
				if !ok {
					return err
				}
				var messageParts []string
				for _, subError := range msErr.Errors {
					messageParts = append(messageParts, strings.Split(subError, "has invalid keys: ")[0])
				}
				return nil
			}
		`),
	)
}

// hashicorp/hcp reaches the same struct through errors.As; that blocks too.
func TestSwapImportsBlockedByErrorsAs(t *testing.T) {
	swapSpec().RewriteRun(t,
		test.Golang(`
			package profile

			import (
				"errors"

				"github.com/mitchellh/mapstructure"
			)

			func convertDecodeError(err error) error {
				mapErr := &mapstructure.Error{}
				if !errors.As(err, &mapErr) {
					return err
				}
				if len(mapErr.Errors) > 1 {
					return err
				}
				return err
			}
		`),
	)
}

// Already on the fork: nothing to do.
func TestSwapImportsAlreadyOnFork(t *testing.T) {
	swapSpec().RewriteRun(t,
		test.Golang(`
			package awsssm

			import "github.com/go-viper/mapstructure/v2"

			func decode(in, out interface{}) error {
				return mapstructure.Decode(in, out)
			}
		`),
	)
}

// The fork's path carries a /v2 suffix that names no package, so the swapped
// import still binds `mapstructure` and an existing v2 import is recognised as
// the same binding.
func TestSwapImportsBlockedWhenForkAlreadyBound(t *testing.T) {
	swapSpec().RewriteRun(t,
		test.Golang(`
			package awsssm

			import (
				"github.com/go-viper/mapstructure/v2"
				v1 "github.com/mitchellh/mapstructure"
			)

			func decode(in, out interface{}) error {
				if err := v1.Decode(in, out); err != nil {
					return err
				}
				return mapstructure.Decode(in, out)
			}
		`),
	)
}
