/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// A *credentials.Credentials held and passed around is v2's
// aws.CredentialsProvider. Its Get answered a credentials.Value, which
// in4it/ecs-deploy marshals to JSON and stores, reading it back on the next
// call; v2's Credentials has other fields, so a stand-in keeps v1's shape, and
// with it the stored JSON.

const (
	credentialsValueType   = "awsCompatCredentialsValue"
	getCredentialsHelper   = "awsCompatGetCredentials"
	staticCredentialHelper = "awsCompatStaticCredentials"
	credentialsValueFile   = "zz_awssdkv2_credentialsvalue.go"
)

// assumeRoleFields are the AssumeRoleProvider settings v2's AssumeRoleOptions
// holds under the same names and types.
var assumeRoleFields = map[string]bool{
	"RoleSessionName": true, "Duration": true, "ExternalID": true,
	"Policy": true, "SerialNumber": true, "TokenProvider": true,
}

// assumeRoleOptionsFunc reports whether expr is a function setting only the
// AssumeRoleProvider fields v2's options hold.
func (s *fileScan) assumeRoleOptionsFunc(expr java.Expression) bool {
	se, isWrapped := expr.(*golang.StatementExpression)
	if !isWrapped {
		return false
	}
	md, isFunc := se.Statement.(*java.MethodDeclaration)
	if !isFunc || md.Body == nil || len(md.Parameters.Elements) != 1 {
		return false
	}
	vd, isDecl := md.Parameters.Elements[0].Element.(*java.VariableDeclarations)
	if !isDecl || len(vd.Variables) != 1 || vd.Variables[0].Element == nil || vd.Variables[0].Element.Name == nil || !s.isAssumeRoleProvider(vd.TypeExpr) {
		return false
	}
	param := vd.Variables[0].Element.Name.Name
	for _, rp := range md.Body.Statements {
		a, isAssign := rp.Element.(*java.Assignment)
		if !isAssign {
			return false
		}
		root, field, ok := writePath(a.Variable)
		if !ok || root.Name != param || !assumeRoleFields[field] {
			return false
		}
	}
	return true
}

// isAssumeRoleProvider reports whether expr is *stscreds.AssumeRoleProvider.
func (s *fileScan) isAssumeRoleProvider(expr java.Expression) bool {
	ptr, isPointer := expr.(*golang.PointerType)
	if !isPointer {
		return false
	}
	fa, isField := ptr.Elem.(*java.FieldAccess)
	if !isField {
		return false
	}
	// The editor sees it after the rename to v2's options type.
	name, isStscreds := qualifiedRef(fa, s.stscredsPkg)
	return isStscreds && (name == "AssumeRoleProvider" || name == "AssumeRoleOptions")
}

// isCredentialsPointer reports whether expr is *credentials.Credentials.
func (s *fileScan) isCredentialsPointer(expr java.Expression) bool {
	ptr, isPointer := expr.(*golang.PointerType)
	if !isPointer {
		return false
	}
	fa, isField := ptr.Elem.(*java.FieldAccess)
	if !isField {
		return false
	}
	name, isCredentials := qualifiedRef(fa, s.credentialsPkg)
	return isCredentials && name == "Credentials"
}

// credentialsReceiver reports whether expr holds a *credentials.Credentials,
// by its declaration in the file or its parse-time type.
func (s *fileScan) credentialsReceiver(expr java.Expression) bool {
	if id, isIdent := expr.(*java.Identifier); isIdent && s.credentialsVars[id.Name] {
		return true
	}
	return matcher.GetFullyQualifiedName(matcher.TypeOfExpression(expr)) == v1Credentials+".Credentials"
}

// credentialsVarScan records the names declared as *credentials.Credentials.
type credentialsVarScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *credentialsVarScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	if v.scan.isCredentialsPointer(vd.TypeExpr) {
		for _, rp := range vd.Variables {
			if d := rp.Element; d != nil && d.Name != nil {
				v.scan.credentialsVars[d.Name.Name] = true
			}
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

func (v *credentialsVarScan) VisitAssignment(a *java.Assignment, p any) java.J {
	// A local assigned a constructor's provider holds one too.
	if target, isIdent := a.Variable.(*java.Identifier); isIdent && java.HasMarker[golang.ShortVarDecl](a.Markers) {
		if mi, isCall := a.Value.Element.(*java.MethodInvocation); isCall {
			if call, isCredentials := qualifiedCall(mi, v.scan.credentialsPkg); isCredentials && call == "NewStaticCredentialsFromCreds" {
				v.scan.credentialsVars[target.Name] = true
			}
		}
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

func (v *credentialsVarScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	// The options type an assume-role call's function takes is consumed by
	// the rewrite, recorded before the guard reaches it.
	if _, _, opts, ok := v.scan.assumeRoleCredentials(mi); ok && opts != nil {
		md := opts.(*golang.StatementExpression).Statement.(*java.MethodDeclaration)
		vd := md.Parameters.Elements[0].Element.(*java.VariableDeclarations)
		v.scan.consumedAssumeRoleRefs[vd.TypeExpr.(*golang.PointerType).Elem.(*java.FieldAccess)] = true
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

func credentialsProviderType(prefix java.Space, awsLocal string) java.Expression {
	return &java.FieldAccess{
		Prefix: prefix,
		Target: &java.Identifier{Name: awsLocal, Type: lstutil.NamedType(v2Aws)},
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: "CredentialsProvider"}},
		Type:   lstutil.NamedType(v2Aws + ".CredentialsProvider"),
	}
}

const credentialsValueSource = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// awsCompatCredentialsValue stands in for aws-sdk-go v1's credentials.Value,
// whose fields, and so whose JSON, v2's Credentials does not share.
type awsCompatCredentialsValue struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	ProviderName    string
}

// HasKeys reports whether both keys are set, as v1's did.
func (v awsCompatCredentialsValue) HasKeys() bool {
	return len(v.AccessKeyID) != 0 && len(v.SecretAccessKey) != 0
}

// awsCompatProviderNames are the providers v2 names differently from v1.
var awsCompatProviderNames = map[string]string{
	"StaticCredentials": "StaticProvider",
}

// awsCompatGetCredentials stands in for v1's Credentials.Get.
func awsCompatGetCredentials(ctx context.Context, provider aws.CredentialsProvider) (awsCompatCredentialsValue, error) {
	creds, err := provider.Retrieve(ctx)
	if err != nil {
		return awsCompatCredentialsValue{}, err
	}
	name := creds.Source
	if v1, renamed := awsCompatProviderNames[name]; renamed {
		name = v1
	}
	return awsCompatCredentialsValue{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		ProviderName:    name,
	}, nil
}

// awsCompatStaticCredentials stands in for v1's NewStaticCredentialsFromCreds.
func awsCompatStaticCredentials(value awsCompatCredentialsValue) aws.CredentialsProvider {
	return aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
		if value.AccessKeyID == "" || value.SecretAccessKey == "" {
			return aws.Credentials{Source: "StaticProvider"}, errors.New("EmptyStaticCreds: static credentials are empty")
		}
		source := value.ProviderName
		if source == "" {
			source = "StaticProvider"
		}
		return aws.Credentials{
			AccessKeyID:     value.AccessKeyID,
			SecretAccessKey: value.SecretAccessKey,
			SessionToken:    value.SessionToken,
			Source:          source,
		}, nil
	})
}
`
