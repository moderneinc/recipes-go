/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"strings"

	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Reports every aws-sdk-go v1 construct with its v2 counterpart.
type FindAwsSdkGoV1Usage struct {
	recipe.Base
}

func (r *FindAwsSdkGoV1Usage) Name() string {
	return "org.openrewrite.golang.migration.FindAwsSdkGoV1Usage"
}
func (r *FindAwsSdkGoV1Usage) DisplayName() string {
	return "Find `aws-sdk-go` v1 usage"
}
func (r *FindAwsSdkGoV1Usage) Description() string {
	return "Mark every `github.com/aws/aws-sdk-go` construct with the `aws-sdk-go-v2` shape that replaces it. AWS ended support for v1 in July 2025. The rewrite recipes cover the constructs with a faithful one-to-one v2 form; this reports those alongside the ones that need a hand migration — the `awserr` error matching, the page iterators and waiters that became types, and the service enums that moved to a `types` sub-package."
}
func (r *FindAwsSdkGoV1Usage) Tags() []string {
	return []string{"search", "migration", "aws"}
}

func (r *FindAwsSdkGoV1Usage) Editor() recipe.TreeVisitor {
	return visitor.Init(&findUsageVisitor{})
}

type findUsageVisitor struct {
	visitor.GoVisitor
	awsPkg         string
	sessionPkg     string
	credentialsPkg string
	services       map[string]string
}

func (v *findUsageVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	if !importsV1(cu) {
		return cu
	}
	v.awsPkg = pathswap.Qualifier(cu, v1Aws)
	v.sessionPkg = pathswap.Qualifier(cu, v1Session)
	v.credentialsPkg = pathswap.Qualifier(cu, v1Credentials)
	v.services = serviceImports(cu)
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *findUsageVisitor) VisitImport(imp *java.Import, p any) java.J {
	imp = v.GoVisitor.VisitImport(imp, p).(*java.Import)
	path := pathswap.Path(imp)
	if reason, blocked := v1OnlyPackages[path]; blocked {
		return imp.WithMarkers(java.MarkupWarn(imp.Markers, reason))
	}
	if path == v1Awserr {
		return imp.WithMarkers(java.MarkupInfo(imp.Markers,
			"awserr was replaced by smithy's typed errors: an assertion to awserr.Error becomes errors.As against smithy.APIError"))
	}
	if newPath, ok := MapPath(path); ok {
		if path == v1Session {
			return imp.WithMarkers(java.MarkupWarn(imp.Markers,
				"the v1 session is gone: load a config with "+v2Config+".LoadDefaultConfig(ctx) and build clients from it"))
		}
		return imp.WithMarkers(java.MarkupInfo(imp.Markers, "migrates to "+newPath))
	}
	if underV1(path) {
		return imp.WithMarkers(java.MarkupWarn(imp.Markers,
			"no aws-sdk-go-v2 counterpart is known for this package"))
	}
	return imp
}

func (v *findUsageVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Name == nil {
		return mi
	}
	name := mi.Name.Name

	if helper, ok := qualifiedCall(mi, v.awsPkg); ok {
		if renamed, isHelper := valueHelperRenames[helper]; isHelper {
			return mi.WithMarkers(java.MarkupInfo(mi.Markers, "aws."+helper+" became aws."+renamed))
		}
		return mi
	}

	if sessionCall, ok := qualifiedCall(mi, v.sessionPkg); ok {
		return mi.WithMarkers(java.MarkupWarn(mi.Markers, sessionGuidance(sessionCall)))
	}

	if credCall, ok := qualifiedCall(mi, v.credentialsPkg); ok && credCall == "NewStaticCredentials" {
		return mi.WithMarkers(java.MarkupInfo(mi.Markers,
			"credentials.NewStaticCredentials became credentials.NewStaticCredentialsProvider"))
	}

	// A client constructor on a service package.
	if _, ok := v.serviceCall(mi); ok && name == "New" {
		return mi.WithMarkers(java.MarkupInfo(mi.Markers,
			"a v2 client is built from a config: New(sess) became NewFromConfig(cfg)"))
	}

	// An operation on a client value.
	if guidance := operationGuidance(name); guidance != "" {
		return mi.WithMarkers(java.MarkupWarn(mi.Markers, guidance))
	}
	return mi
}

func (v *findUsageVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)

	if name, ok := qualifiedRef(fa, v.awsPkg); ok {
		if name == "Config" {
			return fa.WithMarkers(java.MarkupWarn(fa.Markers,
				"v2's aws.Config is loaded rather than constructed, and its Region is a string where v1's was a *string"))
		}
		if !awsPackageSurvivors[name] && valueHelperRenames[name] == "" {
			return fa.WithMarkers(java.MarkupWarn(fa.Markers, "aws."+name+" has no unchanged v2 counterpart"))
		}
		return fa
	}

	for local, service := range v.services {
		name, ok := qualifiedRef(fa, local)
		if !ok {
			continue
		}
		if isInputOrOutputShape(name) || name == "New" {
			return fa
		}
		return fa.WithMarkers(java.MarkupWarn(fa.Markers,
			service+"."+name+" moved to "+v2ServicePkg+service+"/types in v2"))
	}
	return fa
}

// serviceCall reports whether mi is called on an imported v1 service package,
// returning the service's short name.
func (v *findUsageVisitor) serviceCall(mi *java.MethodInvocation) (string, bool) {
	if mi.Select == nil {
		return "", false
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	if !ok {
		return "", false
	}
	service, ok := v.services[recv.Name]
	return service, ok
}

func sessionGuidance(call string) string {
	switch call {
	case "Must":
		return "session.Must has no v2 counterpart: config.LoadDefaultConfig returns an error that has to be handled rather than panicked on"
	case "NewSession", "New", "NewSessionWithOptions":
		return "build a config instead: config.LoadDefaultConfig(ctx, config.WithRegion(region)), then pass it to each client's NewFromConfig"
	}
	return "the v1 session has no v2 counterpart; load a config with config.LoadDefaultConfig(ctx) instead"
}

// operationGuidance describes the v2 form of a v1 operation-shaped call.
func operationGuidance(name string) string {
	if strings.HasPrefix(name, waiterPrefix) {
		return name + " became a waiter type: build one with " + strings.TrimPrefix(name, waiterPrefix) + "Waiter and call Wait(ctx, params, maxWait)"
	}
	for _, suffix := range blockedCallSuffixes {
		if strings.HasSuffix(name, suffix) && name != suffix {
			base := strings.TrimSuffix(name, suffix)
			return name + " became a paginator: build one with New" + base + "Paginator(client, params) and loop while HasMorePages()"
		}
	}
	if strings.HasSuffix(name, "WithContext") && name != "WithContext" {
		return name + " became " + strings.TrimSuffix(name, "WithContext") + ", since every v2 operation takes a context"
	}
	return ""
}
