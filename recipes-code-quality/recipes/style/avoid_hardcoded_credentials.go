/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// credentialKeywords lists substrings that indicate a variable may hold a credential.
var credentialKeywords = []string{"password", "secret", "token", "apikey", "api_key"}

// AvoidHardcodedCredentials replaces string literals assigned to variables whose
// names suggest they hold credentials with `os.Getenv("VAR_NAME")` calls.
// Hardcoded credentials are a security risk and should be loaded from the
// environment or a secrets manager instead.
type AvoidHardcodedCredentials struct {
	recipe.Base
}

func (r *AvoidHardcodedCredentials) Name() string {
	return "org.openrewrite.golang.codequality.AvoidHardcodedCredentials"
}
func (r *AvoidHardcodedCredentials) DisplayName() string { return "Avoid hardcoded credentials" }
func (r *AvoidHardcodedCredentials) Description() string {
	return "Replace hardcoded credential string literals with `os.Getenv(\"VAR_NAME\")` calls."
}
func (r *AvoidHardcodedCredentials) Tags() []string { return []string{"security"} }

func (r *AvoidHardcodedCredentials) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidHardcodedCredentialsVisitor{})
}

type avoidHardcodedCredentialsVisitor struct {
	visitor.GoVisitor
}

// envVarName converts a Go variable name to an environment variable name.
// e.g., "dbPassword" -> "DB_PASSWORD", "apiKey" -> "API_KEY"
func envVarName(name string) string {
	var result strings.Builder
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(r)
		} else if r >= 'a' && r <= 'z' {
			result.WriteRune(r - 'a' + 'A')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func (v *avoidHardcodedCredentialsVisitor) VisitVariableDeclarator(vd *java.VariableDeclarator, p any) java.J {
	vd = v.GoVisitor.VisitVariableDeclarator(vd, p).(*java.VariableDeclarator)

	if vd.Initializer == nil {
		return vd
	}

	lit, ok := vd.Initializer.Element.(*java.Literal)
	if !ok || lit.Kind != java.StringLiteral {
		return vd
	}

	varName := strings.ToLower(vd.Name.Name)
	matched := false
	for _, keyword := range credentialKeywords {
		if strings.Contains(varName, keyword) {
			matched = true
			break
		}
	}
	if !matched {
		return vd
	}

	// Build os.Getenv("VAR_NAME") to replace the string literal.
	envName := envVarName(vd.Name.Name)

	osIdent := &java.Identifier{
		Prefix: lit.Prefix,
		Name:   "os",
	}

	getenvIdent := &java.Identifier{
		Name: "Getenv",
	}

	envLit := &java.Literal{
		Kind:   java.StringLiteral,
		Source: `"` + envName + `"`,
	}

	getenvCall := &java.MethodInvocation{
		Select: &java.RightPadded[java.Expression]{Element: osIdent},
		Name:   getenvIdent,
		Arguments: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{Element: envLit},
			},
		},
	}

	c := *vd
	c.Initializer = &java.LeftPadded[java.Expression]{
		Before:  vd.Initializer.Before,
		Element: getenvCall,
	}
	return &c
}
