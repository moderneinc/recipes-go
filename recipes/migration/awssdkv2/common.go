/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

// Package awssdkv2 migrates github.com/aws/aws-sdk-go, which AWS ended support
// for in July 2025, to github.com/aws/aws-sdk-go-v2.
//
// This is not a path swap. v2 replaced the session with a config loaded through
// a context, gave every operation a context parameter, split each service into
// its own module, moved service enums and shapes into a `types` sub-package, and
// dropped `awserr` for smithy's typed errors. A file therefore migrates as a
// whole or not at all — half of one does not compile — but the module does not:
// a file the recipes cannot take is left as it is and listed in the blockers
// table, and the rest moves around it. What crosses the boundary between the two
// is a type error the compiler points at, which is the work that table is the
// list for.
package awssdkv2

import (
	"go/version"
	"sort"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

const (
	v1Module = "github.com/aws/aws-sdk-go"
	v2Module = "github.com/aws/aws-sdk-go-v2"

	v1Aws         = v1Module + "/aws"
	v1Session     = v1Aws + "/session"
	v1Credentials = v1Aws + "/credentials"
	v1ServicePkg  = v1Module + "/service/"

	v2Aws         = v2Module + "/aws"
	v2Config      = v2Module + "/config"
	v2Credentials = v2Module + "/credentials"
	// v2 moved the assume-role credential helpers up alongside the rest of the
	// credentials packages.
	v1Stscreds = v1Credentials + "/stscreds"
	v2Stscreds = v2Credentials + "/stscreds"
	// v1Rest is v1's REST protocol package, and smithyHTTPBinding the home of
	// the one helper of it callers reach for.
	v1Rest            = v1Module + "/private/protocol/rest"
	smithyHTTPBinding = smithyGo + "/encoding/httpbinding"
	v2ServicePkg      = v2Module + "/service/"

	contextPkg = "context"
	// The name v2's config package binds by default.
	configPkgName = "config"

	// The Go release aws-sdk-go-v2 itself requires, which a migrating module
	// has to reach whatever else it does.
	v2MinGo = "1.24"
)

// valueHelperRenames map v1's pointer-dereferencing helpers to the names v2
// settled on: <T>Value, <T>ValueSlice and <T>ValueMap became To<T>, To<T>Slice
// and To<T>Map. v1's SecondsTimeValue and MillisecondsTimeValue read an epoch
// count v2 has no helper for, so they are absent.
var valueHelperRenames = func() map[string]string {
	renames := map[string]string{}
	for _, t := range pointerTypes {
		renames[t+"Value"] = "To" + t
		renames[t+"ValueSlice"] = "To" + t + "Slice"
		renames[t+"ValueMap"] = "To" + t + "Map"
	}
	return renames
}()

// v1's aws.ReadSeekCloser passed seeks through to a reader that could. v2's
// request bodies take any io.Reader and seek one that can, so around the
// standard library's seekable readers the wrapper simply goes.

// readSeekCloserOperand returns the reader aws.ReadSeekCloser wrapped, when it
// is one that seeks.
func (s *fileScan) readSeekCloserOperand(mi *java.MethodInvocation) (java.Expression, bool) {
	helper, isAws := qualifiedCall(mi, s.awsPkg)
	args := realArgs(mi)
	if !isAws || helper != "ReadSeekCloser" || len(args) != 1 {
		return nil, false
	}
	inner, isCall := args[0].(*java.MethodInvocation)
	if !isCall || inner.Name == nil {
		return nil, false
	}
	recv, isIdent := selectIdentifier(inner)
	seekable := recv != nil && recv.Name != "" && (recv.Name == s.bytesPkg || recv.Name == s.stringsPkg)
	if !isIdent || !seekable || inner.Name.Name != "NewReader" {
		return nil, false
	}
	return inner, true
}

// pointerTypes are the types both SDKs have pointer helpers for.
var pointerTypes = []string{
	"Bool", "String", "Time", "Float32", "Float64",
	"Int", "Int8", "Int16", "Int32", "Int64",
	"Uint", "Uint8", "Uint16", "Uint32", "Uint64",
}

// awsPackageSurvivors are the `aws` package names v2 kept unchanged, so a
// reference to one needs nothing but the import swap. Anything else in that
// package — Config with its now-unpointered Region, the request plumbing, the
// endpoint resolvers — blocks its file.
var awsPackageSurvivors = func() map[string]bool {
	survivors := map[string]bool{"Duration": true}
	for _, t := range pointerTypes {
		survivors[t] = true
		survivors[t+"Slice"] = true
		survivors[t+"Map"] = true
	}
	return survivors
}()

// v1Endpoints held v1's endpoint resolution and partition data, and v1Client
// the plumbing clients were built on. Of the latter only ConfigProvider, the
// interface a session satisfied, has a counterpart: the config replacing it.
const (
	v1Client       = v1Aws + "/client"
	clientReason   = "the v1 client plumbing has no v2 counterpart beyond ConfigProvider"
	configProvider = "ConfigProvider"

	v1Endpoints     = v1Aws + "/endpoints"
	endpointsReason = "endpoint resolution moved onto the client's Options in v2, and its partition data has no public counterpart"
)

// v1OnlyPackages are the v1 sub-packages with no drop-in v2 counterpart. A file
// importing one cannot be migrated mechanically.
var v1OnlyPackages = map[string]string{
	v1Aws + "/defaults":     "v2 builds defaults through config.LoadDefaultConfig",
	v1Aws + "/awsutil":      "awsutil has no v2 counterpart",
	v1Aws + "/corehandlers": "the v1 handler stack was replaced by smithy middleware",
	// v2 moved or dropped these service sub-packages, so the generic service
	// mapping would point them at paths that do not exist. Each reason is the
	// migration path, since the blockers table is where a user reads it.
	v1ServicePkg + "s3/s3crypto": "s3crypto has no v2 package: move to github.com/aws/amazon-s3-encryption-client-go/v3, whose default " +
		"policy reads and writes the format v1's V2 client did, and set EnableLegacyUnauthenticatedModes to read objects v1's original client wrote; " +
		"move on to v4's key-committing format only once every reader of the bucket is on v3",
}

// blockedCallSuffixes are v1 operation shapes with no single-call v2 form: the
// page iterators became paginator types, and the waiters became waiter types.
var blockedCallSuffixes = []string{"Pages", "PagesWithContext"}

const waiterPrefix = "WaitUntil"

// MapPath returns the v2 import path for a v1 one, and whether there is one.
func MapPath(path string) (string, bool) {
	switch {
	case path == v1Aws:
		return v2Aws, true
	case path == v1Session:
		// The session is gone; a file using it lands on config instead.
		return v2Config, true
	case path == v1Stscreds:
		return v2Stscreds, true
	case path == v1Ec2RoleCreds:
		return v2Ec2RoleCreds, true
	case path == v1Credentials:
		return v2Credentials, true
	case path == v1Ec2Metadata:
		return v2Imds, true
	case path == v1S3Manager:
		return v2S3Manager, true
	case path == v1DynamoExpression:
		return v2DynamoExpression, true
	case path == gureguV1:
		return gureguV2, true
	case path == v1Rest:
		// Of the REST protocol helpers only EscapePath has a public v2 home,
		// unchanged in smithy's HTTP binding.
		return smithyHTTPBinding, true
	case path == v1Rdsutils:
		return v2RdsAuth, true
	case path == v1Aws+"/arn":
		// The ARN parser moved unchanged.
		return v2Aws + "/arn", true
	case strings.HasPrefix(path, v1ServicePkg):
		// An iface package has no v2 counterpart to point at; the migration
		// generates a replacement and repoints the import at that instead.
		if _, _, isIface := ifacePackageName(path); isIface {
			return "", false
		}
		service, rest, _ := strings.Cut(strings.TrimPrefix(path, v1ServicePkg), "/")
		if rest != "" {
			rest = "/" + rest
		}
		return v2ServicePath(service) + rest, true
	}
	return "", false
}

// vendored reports whether cu is a vendored dependency rather than the module's
// own source. A pre-modules repository vendors the SDK itself, and holding the
// module back over source it does not maintain would block every one of them.
func vendored(cu *golang.CompilationUnit) bool {
	if cu == nil {
		return false
	}
	path := cu.SourcePath
	return strings.HasPrefix(path, "vendor/") || strings.Contains(path, "/vendor/")
}

// importsV1 reports whether cu imports anything under the v1 SDK.
func importsV1(cu *golang.CompilationUnit) bool {
	if cu == nil || cu.Imports == nil {
		return false
	}
	for _, rp := range cu.Imports.Elements {
		if path := rp.Element.Path(); underV1(path) || path == gureguV1 {
			return true
		}
	}
	return false
}

func underV1(path string) bool {
	return path == v1Module || strings.HasPrefix(path, v1Module+"/")
}

// serviceImports returns the local name each imported v1 service package binds,
// keyed by the service's short name (`s3`, `ec2`).
func serviceImports(cu *golang.CompilationUnit) map[string]string {
	out := map[string]string{}
	if cu == nil || cu.Imports == nil {
		return out
	}
	for _, rp := range cu.Imports.Elements {
		path := rp.Element.Path()
		if !strings.HasPrefix(path, v1ServicePkg) {
			continue
		}
		// A sub-package is not the service package: it binds its own name, and
		// the migration gives each one its own destination.
		if _, _, isIface := ifacePackageName(path); isIface {
			continue
		}
		if path == v1S3Manager || path == v1DynamoExpression || path == v1Rdsutils || path == v1DynamodbAttribute {
			continue
		}
		if name := pathswap.LocalName(rp.Element); name != "" && name != "_" && name != "." {
			out[name] = strings.TrimPrefix(path, v1ServicePkg)
		}
	}
	return out
}

// qualifierFor is the local name the file binds a service to, defaulting to the
// service's own name for one it has yet to import.
func (s *fileScan) qualifierFor(service string) string {
	for local, imported := range s.services {
		if imported == service {
			return local
		}
	}
	return service
}

// moduleAcc carries the module's `go` directive, since the generics the slice
// helpers are written with need a release the module may predate.
type moduleAcc struct {
	goVersion string
}

// atLeast reports whether the module targets minVersion or newer. An unknown
// version — no go.mod in the source set — is treated as unconstrained.
func (a *moduleAcc) atLeast(minVersion string) bool {
	if a == nil || a.goVersion == "" {
		return true
	}
	return version.Compare("go"+a.goVersion, "go"+minVersion) >= 0
}

// realArgs returns the arguments of mi, skipping the Empty sentinel an empty
// argument list carries.
func realArgs(mi *java.MethodInvocation) []java.Expression {
	var out []java.Expression
	for _, rp := range mi.Arguments.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		out = append(out, rp.Element)
	}
	return out
}

// qualifiedCall reports whether mi is `<pkg>.<name>(…)`, returning the name.
func qualifiedCall(mi *java.MethodInvocation, pkg string) (string, bool) {
	if pkg == "" || mi.Name == nil || mi.Select == nil {
		return "", false
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	if !ok || recv.Name != pkg {
		return "", false
	}
	return mi.Name.Name, true
}

// qualifiedRef reports whether fa is `<pkg>.<Name>`, returning the name.
func qualifiedRef(fa *java.FieldAccess, pkg string) (string, bool) {
	if pkg == "" || fa.Name.Element == nil {
		return "", false
	}
	target, ok := fa.Target.(*java.Identifier)
	if !ok || target.Name != pkg {
		return "", false
	}
	return fa.Name.Element.Name, true
}

// isInputOrOutputShape reports whether a service-package name is an operation
// input or output struct, which v2 kept in the service package. Everything else
// a service package exports — the enums and the modelled shapes — moved to its
// types sub-package, and this recipe does not relocate those.
func isInputOrOutputShape(name string) bool {
	return strings.HasSuffix(name, "Input") || strings.HasSuffix(name, "Output")
}

// sortedKeys orders a map's keys, so a table's rows do not depend on the order
// the scan happened to see files in.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// v2ServicePath is the import path of a v1 service's v2 package.
func v2ServicePath(service string) string {
	return v2ServicePkg + awsmanifest.V2Name(service)
}

// renamedService reports whether path is a v1 service package v2 spells
// differently, whose files keep spelling it as v1 did.
func renamedService(path string) bool {
	service, isService := strings.CutPrefix(path, v1ServicePkg)
	return isService && !strings.Contains(service, "/") && awsmanifest.V2Name(service) != service
}

// v1Libraries are third-party packages built on aws-sdk-go v1, whose values a
// migrated file would hand v2's.
var v1Libraries = map[string]string{}

// shadowsPackage reports whether id, spelled as an imported package, is a
// local of some other type: in4it/ecs-deploy declares ec2 := EC2{} in a file
// importing the ec2 service. The parse-time type tells them apart where it is
// known; a package qualifier carries the package's path.
func shadowsPackage(id *java.Identifier) bool {
	class := matcher.AsClass(id.Type)
	if class == nil {
		return false
	}
	fqn := class.FullyQualifiedName
	// A type is named after its package's path; a package is the path alone.
	return strings.Contains(fqn[strings.LastIndex(fqn, "/")+1:], ".")
}

// importsAs reports whether cu imports path under the name local.
func importsAs(cu *golang.CompilationUnit, path, local string) bool {
	if cu == nil || cu.Imports == nil {
		return false
	}
	for _, rp := range cu.Imports.Elements {
		if rp.Element.Path() == path && pathswap.LocalName(rp.Element) == local {
			return true
		}
	}
	return false
}
