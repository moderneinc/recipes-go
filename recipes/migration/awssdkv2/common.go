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

	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
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
	v1Stscreds   = v1Credentials + "/stscreds"
	v2Stscreds   = v2Credentials + "/stscreds"
	v2ServicePkg = v2Module + "/service/"

	// Versions the go.mod rewrite pins. Every service is its own module,
	// versioned independently, so those requires are left for `go mod tidy` to
	// add from the imports rather than guessed at here.
	v2CoreVersion        = "v1.47.0"
	v2ConfigVersion      = "v1.33.5"
	v2CredentialsVersion = "v1.20.5"
	smithyVersion        = "v1.28.1"

	contextPkg = "context"
	// The name v2's config package binds by default.
	configPkgName = "config"

	// The Go release aws-sdk-go-v2 itself requires, which a migrating module
	// has to reach whatever else it does.
	v2MinGo = "1.24"
)

// valueHelperRenames map v1's pointer-dereferencing helpers to the names v2
// settled on. v2 dropped the `Value` spelling outright, so a call that kept it
// does not compile after the swap.
var valueHelperRenames = map[string]string{
	"BoolValue":             "ToBool",
	"IntValue":              "ToInt",
	"Int64Value":            "ToInt64",
	"StringValue":           "ToString",
	"TimeValue":             "ToTime",
	"UintValue":             "ToUint",
	"Float64Value":          "ToFloat64",
	"SecondsTimeValue":      "ToTime",
	"MillisecondsTimeValue": "ToTime",
}

// awsPackageSurvivors are the `aws` package names v2 kept unchanged, so a
// reference to one needs nothing but the import swap. Anything else in that
// package — Config with its now-unpointered Region, the request plumbing, the
// endpoint resolvers — blocks its file.
var awsPackageSurvivors = map[string]bool{
	"Bool": true, "Bool_": true, "BoolMap": true, "BoolSlice": true,
	"Float64": true, "Float64Map": true, "Float64Slice": true,
	"Int": true, "Int32": true, "Int64": true, "Int64Map": true, "Int64Slice": true,
	"String": true, "StringMap": true, "StringSlice": true,
	"Time": true, "TimeSlice": true, "TimeUnixMilli": true,
	"Uint": true, "Uint64": true,
	"Duration": true, "JSONValue": true,
}

// v1OnlyPackages are the v1 sub-packages with no drop-in v2 counterpart. A file
// importing one cannot be migrated mechanically.
var v1OnlyPackages = map[string]string{
	v1Aws + "/request":      "the v1 request plumbing has no v2 counterpart; per-operation behaviour is configured through functional options on the client",
	v1Aws + "/endpoints":    "endpoint resolution moved onto the client's Options in v2",
	v1Aws + "/defaults":     "v2 builds defaults through config.LoadDefaultConfig",
	v1Aws + "/client":       "the v1 client plumbing has no v2 counterpart",
	v1Aws + "/awsutil":      "awsutil has no v2 counterpart",
	v1Aws + "/corehandlers": "the v1 handler stack was replaced by smithy middleware",
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
	case path == v1Credentials:
		return v2Credentials, true
	case path == v1Ec2Metadata:
		return v2Imds, true
	case path == v1S3Manager:
		return v2S3Manager, true
	case strings.HasPrefix(path, v1ServicePkg):
		// An iface package has no v2 counterpart to point at; the migration
		// generates a replacement and repoints the import at that instead.
		if _, _, isIface := ifacePackageName(path); isIface {
			return "", false
		}
		return v2ServicePkg + strings.TrimPrefix(path, v1ServicePkg), true
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
		if underV1(pathswap.Path(rp.Element)) {
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
		path := pathswap.Path(rp.Element)
		if !strings.HasPrefix(path, v1ServicePkg) {
			continue
		}
		// A sub-package is not the service package: it binds its own name, and
		// the migration gives each one its own destination.
		if _, _, isIface := ifacePackageName(path); isIface {
			continue
		}
		if path == v1S3Manager {
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
