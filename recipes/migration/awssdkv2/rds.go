/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// v2 moved the RDS IAM token signer into its own feature module, renamed the
// package auth, and dropped the connection-string builder. The import is bound
// back to the name the file already spells. BuildAuthToken survived with a
// context ahead of its arguments and a credentials provider in place of v1's
// Credentials.
const (
	v1Rdsutils = v1ServicePkg + "rds/rdsutils"
	v2RdsAuth  = v2Module + "/feature/rds/auth"

	buildAuthToken = "BuildAuthToken"

	rdsConnectionBuilderReason = "has no v2 counterpart: build the DSN by hand, with the token from " +
		v2RdsAuth + ".BuildAuthToken as the password and the tls and allowCleartextPasswords parameters v1's builder set"
	rdsCredentialsReason = "rdsutils.BuildAuthToken with credentials v2 cannot carry over: v2's BuildAuthToken takes an " +
		"aws.CredentialsProvider, such as a loaded config's Credentials"
)

// rdsutilsBlockReason names an rdsutils call the migration cannot carry over,
// or "" when it can: BuildAuthToken handed credentials whose v2 form is a
// provider — a session's own, or a static pair.
func (s *fileScan) rdsutilsBlockReason(mi *java.MethodInvocation) string {
	call, ok := qualifiedCall(mi, s.rdsutilsPkg)
	if !ok {
		return ""
	}
	if call != buildAuthToken {
		return "rdsutils." + call + " " + rdsConnectionBuilderReason
	}
	args := realArgs(mi)
	if len(args) != 4 || !s.becomesCredentialsProvider(args[3]) {
		return rdsCredentialsReason
	}
	return ""
}

// becomesCredentialsProvider reports whether a v1 credentials expression turns
// into an aws.CredentialsProvider once migrated.
func (s *fileScan) becomesCredentialsProvider(expr java.Expression) bool {
	if _, _, target, ok := s.configAccess(expr); ok {
		return target.name == credentialsField
	}
	mi, isCall := expr.(*java.MethodInvocation)
	if !isCall {
		return false
	}
	call, ok := qualifiedCall(mi, s.credentialsPkg)
	return ok && call == "NewStaticCredentials"
}

// rdsAuthToken gives BuildAuthToken the context v2 takes ahead of its arguments.
func (v *migrateVisitor) rdsAuthToken(mi *java.MethodInvocation) (java.J, bool) {
	call, ok := qualifiedCall(mi, v.scan.rdsutilsPkg)
	if !ok || call != buildAuthToken {
		return nil, false
	}
	ctx := v.contextExpr()
	if ctx == nil {
		return nil, false
	}
	c := renameCall(mi, buildAuthToken, v2RdsAuth).(*java.MethodInvocation)
	elements := make([]java.RightPadded[java.Expression], 0, len(c.Arguments.Elements)+1)
	elements = append(elements, java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(ctx, java.EmptySpace)})
	for i, rp := range c.Arguments.Elements {
		if i == 0 {
			rp.Element = lstutil.SetExprPrefix(rp.Element, java.SingleSpace)
		}
		elements = append(elements, rp)
	}
	out := *c
	out.Arguments.Elements = elements
	return &out, true
}
