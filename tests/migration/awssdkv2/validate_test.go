/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// keikoproj/iam-manager's shape: an input validated before the call. v2
// shapes have no Validate, and v2's own check skips v1's length rules, so a
// generated copy of v1's stands in, nested shapes and all.
func TestMigrateShapeValidate(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/iam"
			)

			func check(name, doc string) error {
				input := &iam.CreateRoleInput{RoleName: aws.String(name), AssumeRolePolicyDocument: aws.String(doc)}
				return input.Validate()
			}
		`, `
			package main

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/iam"
			)

			func check(name, doc string) error {
				input := &iam.CreateRoleInput{RoleName: aws.String(name), AssumeRolePolicyDocument: aws.String(doc)}
				return awsCompatValidateIamCreateRoleInput(input)
			}
		`),
		test.Generated("zz_awssdkv2_validate.go", validateHelpersIn("main", iamValidateImports, createRoleValidators)),
	)
}

const iamValidateImports = "\t\"github.com/aws/aws-sdk-go-v2/service/iam\"\n\tiamtypes \"github.com/aws/aws-sdk-go-v2/service/iam/types\"\n"

const createRoleValidators = `
func awsCompatValidateIamCreateRoleInput(s *iam.CreateRoleInput) error {
	invalid := awsCompatInvalidParams{Context: "CreateRoleInput"}
	if s.AssumeRolePolicyDocument == nil {
		invalid.add(awsCompatParamRequired("AssumeRolePolicyDocument"))
	}
	if s.AssumeRolePolicyDocument != nil && len(*s.AssumeRolePolicyDocument) < 1 {
		invalid.add(awsCompatParamMinLen("AssumeRolePolicyDocument", 1))
	}
	if s.MaxSessionDuration != nil && *s.MaxSessionDuration < 3600 {
		invalid.add(awsCompatParamMinValue("MaxSessionDuration", 3600))
	}
	if s.Path != nil && len(*s.Path) < 1 {
		invalid.add(awsCompatParamMinLen("Path", 1))
	}
	if s.PermissionsBoundary != nil && len(*s.PermissionsBoundary) < 20 {
		invalid.add(awsCompatParamMinLen("PermissionsBoundary", 20))
	}
	if s.RoleName == nil {
		invalid.add(awsCompatParamRequired("RoleName"))
	}
	if s.RoleName != nil && len(*s.RoleName) < 1 {
		invalid.add(awsCompatParamMinLen("RoleName", 1))
	}
	for i := range s.Tags {
		if err := awsCompatValidateIamTag(&s.Tags[i]); err != nil {
			invalid.addNested(fmt.Sprintf("%s[%v]", "Tags", i), err)
		}
	}
	if len(invalid.errs) > 0 {
		return invalid
	}
	return nil
}

func awsCompatValidateIamTag(s *iamtypes.Tag) error {
	invalid := awsCompatInvalidParams{Context: "Tag"}
	if s.Key == nil {
		invalid.add(awsCompatParamRequired("Key"))
	}
	if s.Key != nil && len(*s.Key) < 1 {
		invalid.add(awsCompatParamMinLen("Key", 1))
	}
	if s.Value == nil {
		invalid.add(awsCompatParamRequired("Value"))
	}
	if len(invalid.errs) > 0 {
		return invalid
	}
	return nil
}
`

// A shape held by value is validated through its address, as v1's pointer
// method was called on it.
func TestMigrateShapeValidateOnValue(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/iam"
			)

			func check(name string) error {
				var input iam.GetRoleInput
				input.RoleName = aws.String(name)
				return input.Validate()
			}
		`, `
			package main

			import (
				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/iam"
			)

			func check(name string) error {
				var input iam.GetRoleInput
				input.RoleName = aws.String(name)
				return awsCompatValidateIamGetRoleInput(&input)
			}
		`),
		test.Generated("zz_awssdkv2_validate.go", validateHelpersIn("main", "\t\"github.com/aws/aws-sdk-go-v2/service/iam\"\n", `
func awsCompatValidateIamGetRoleInput(s *iam.GetRoleInput) error {
	invalid := awsCompatInvalidParams{Context: "GetRoleInput"}
	if s.RoleName == nil {
		invalid.add(awsCompatParamRequired("RoleName"))
	}
	if s.RoleName != nil && len(*s.RoleName) < 1 {
		invalid.add(awsCompatParamMinLen("RoleName", 1))
	}
	if len(invalid.errs) > 0 {
		return invalid
	}
	return nil
}
`)),
	)
}

// The scan knows shape variables by name, so one name holding a pointer in
// one function and a value in another could be either at a call, and the file
// stays on v1.
func TestMigrateBlockedByShapeValidateOfAmbiguousReceiver(t *testing.T) {
	migrateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/iam"
			)

			func byPointer(name string) error {
				input := &iam.GetRoleInput{RoleName: aws.String(name)}
				return input.Validate()
			}

			func byValue(name string) error {
				input := iam.GetRoleInput{RoleName: aws.String(name)}
				return input.Validate()
			}
		`),
	)
}

// validateHelpersIn is the generated validator file for pkg: v1's error types,
// the imports given and the validators.
func validateHelpersIn(pkg, imports, validators string) string {
	return "// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.\n\npackage " + pkg + "\n\nimport (\n\t\"fmt\"\n\t\"strings\"\n\n" + imports + ")\n\n" + validateErrorTypes + validators
}

const validateErrorTypes = `// awsCompatInvalidParams is aws-sdk-go v1's request.ErrInvalidParams: every
// rule a shape broke, each named by its path from the shape validated.
type awsCompatInvalidParams struct {
	Context string
	errs    []*awsCompatInvalidParam
}

func (e *awsCompatInvalidParams) add(err *awsCompatInvalidParam) {
	err.context = e.Context
	e.errs = append(e.errs, err)
}

func (e *awsCompatInvalidParams) addNested(nestedCtx string, nested error) {
	for _, err := range nested.(awsCompatInvalidParams).errs {
		err.context = e.Context
		if err.nestedContext == "" {
			err.nestedContext = nestedCtx
		} else {
			err.nestedContext = nestedCtx + "." + err.nestedContext
		}
		e.errs = append(e.errs, err)
	}
}

func (e awsCompatInvalidParams) Len() int { return len(e.errs) }

func (e awsCompatInvalidParams) Code() string { return "InvalidParameter" }

func (e awsCompatInvalidParams) Message() string {
	return fmt.Sprintf("%d validation error(s) found.", len(e.errs))
}

func (e awsCompatInvalidParams) Error() string {
	var w strings.Builder
	fmt.Fprintf(&w, "%s: %s\n", e.Code(), e.Message())
	for _, err := range e.errs {
		fmt.Fprintf(&w, "- %s\n", err.Message())
	}
	return w.String()
}

// awsCompatInvalidParam is one broken rule, as v1 reported it.
type awsCompatInvalidParam struct {
	context, nestedContext, field, code, msg string
}

func (e *awsCompatInvalidParam) Code() string { return e.code }

func (e *awsCompatInvalidParam) Message() string {
	return fmt.Sprintf("%s, %s.", e.msg, e.Field())
}

func (e *awsCompatInvalidParam) Error() string {
	return fmt.Sprintf("%s: %s", e.code, e.Message())
}

func (e *awsCompatInvalidParam) Field() string {
	field := e.context
	if len(field) > 0 {
		field += "."
	}
	if len(e.nestedContext) > 0 {
		field += e.nestedContext + "."
	}
	return field + e.field
}

func awsCompatParamRequired(field string) *awsCompatInvalidParam {
	return &awsCompatInvalidParam{code: "ParamRequiredError", field: field, msg: "missing required field"}
}

func awsCompatParamMinLen(field string, min int) *awsCompatInvalidParam {
	return &awsCompatInvalidParam{code: "ParamMinLenError", field: field, msg: fmt.Sprintf("minimum field size of %v", min)}
}

func awsCompatParamMinValue(field string, min float64) *awsCompatInvalidParam {
	return &awsCompatInvalidParam{code: "ParamMinValueError", field: field, msg: fmt.Sprintf("minimum field value of %v", min)}
}
`
