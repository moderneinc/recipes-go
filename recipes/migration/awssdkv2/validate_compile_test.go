/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"golang.org/x/mod/semver"
)

// Every validator the manifests describe compiles against the v2 types at the
// pinned versions, which is what proves each rule's v2 form was read right,
// and one reports a nested, a length and a value rule as v1's own Validate
// did. It fetches modules, so it runs only with COMPILE_TESTS set.
func TestEveryValidatorCompiles(t *testing.T) {
	if os.Getenv("COMPILE_TESTS") == "" {
		t.Skip("set COMPILE_TESTS=1 to build every generated validator")
	}
	shapes := map[string]bool{}
	versions := map[string]string{}
	for _, service := range awsmanifest.Services() {
		for _, mv := range v2Requirements[v2ServicePath(service)] {
			if semver.Compare(mv.version, versions[mv.path]) > 0 {
				versions[mv.path] = mv.version
			}
		}
		for _, shape := range awsmanifest.ValidatedShapes(service) {
			if validatable(service, shape) {
				shapes[service+"."+shape] = true
			}
		}
	}
	if len(shapes) == 0 {
		t.Fatal("no validatable shapes")
	}
	dir := t.TempDir()
	var gomod strings.Builder
	gomod.WriteString("module validators\n\ngo 1.24\n\nrequire (\n")
	for path, version := range versions {
		gomod.WriteString("\t" + path + " " + version + "\n")
	}
	gomod.WriteString("\tgithub.com/aws/aws-sdk-go v1.55.8\n)\n")
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", gomod.String())
	write(validateFile, validateSource("main", shapes))
	write("main.go", validatorParity)
	for _, args := range [][]string{{"mod", "tidy"}, {"vet", "./..."}, {"run", "."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}

const validatorParity = `package main

import (
	"fmt"
	"os"

	v1aws "github.com/aws/aws-sdk-go/aws"
	v1iam "github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func main() {
	v1 := (&v1iam.CreateRoleInput{
		RoleName:           v1aws.String(""),
		MaxSessionDuration: v1aws.Int64(5),
		Tags:               []*v1iam.Tag{{Key: v1aws.String("k"), Value: v1aws.String("v")}, {Key: v1aws.String("")}},
	}).Validate()
	v2 := awsCompatValidateIamCreateRoleInput(&iam.CreateRoleInput{
		RoleName:           aws.String(""),
		MaxSessionDuration: aws.Int32(5),
		Tags:               []iamtypes.Tag{{Key: aws.String("k"), Value: aws.String("v")}, {Key: aws.String("")}},
	})
	if v1 == nil || v2 == nil || v1.Error() != v2.Error() {
		fmt.Printf("v1: %v\nv2: %v\n", v1, v2)
		os.Exit(1)
	}
	if awsCompatValidateIamCreateRoleInput(&iam.CreateRoleInput{RoleName: aws.String("r"), AssumeRolePolicyDocument: aws.String("{}")}) != nil {
		fmt.Println("a valid input failed")
		os.Exit(1)
	}
}
`
