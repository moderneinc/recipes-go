/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v2 moved the transfer manager out of the s3 service into its own feature
// module and renamed the package, so the import is aliased back to the name the
// file already spells. Its own API survived: the constructors take an S3 client
// where v1 took a session, the upload input became the plain PutObjectInput,
// and Upload and Download take a context.
const (
	v1S3Manager = v1Module + "/service/s3/s3manager"
	v2S3Manager = v2Module + "/feature/s3/manager"

	s3Service = "s3"
	// v1's UploadInput was a near-copy of PutObjectInput that v2 dropped.
	v1UploadInput = "UploadInput"
	v2UploadInput = "PutObjectInput"
)

// managerConstructors map v1's constructors to v2's. The WithClient pair is
// gone because taking a client is now the only form.
var managerConstructors = map[string]string{
	"NewUploader":              "NewUploader",
	"NewDownloader":            "NewDownloader",
	"NewUploaderWithClient":    "NewUploader",
	"NewDownloaderWithClient":  "NewDownloader",
	"NewUploaderWithIface":     "",
	"NewDownloaderWithIface":   "",
	"NewBatchDelete":           "",
	"NewBatchDeleteWithClient": "",
}

// managerTakesClient reports whether the v1 constructor already took a client
// rather than a session, in which case its argument carries over unchanged.
func managerTakesClient(name string) bool {
	return name == "NewUploaderWithClient" || name == "NewDownloaderWithClient"
}

// managerNames are the s3manager references the migration carries over. Any
// other one blocks the file, since the package has no home in v2 otherwise.
var managerNames = map[string]bool{
	"Uploader": true, "Downloader": true, "UploadInput": true, "UploadOutput": true,
	"NewUploader": true, "NewDownloader": true,
	"NewUploaderWithClient": true, "NewDownloaderWithClient": true,
	"DefaultUploadPartSize": true, "DefaultUploadConcurrency": true,
	"DefaultDownloadPartSize": true, "DefaultDownloadConcurrency": true,
	"MaxUploadParts": true, "MinUploadPartSize": true,
}

// managerUploadInput reports whether fa is the s3manager upload input, which
// becomes s3.PutObjectInput.
func (s *fileScan) managerUploadInput(fa *java.FieldAccess) bool {
	if s.managerPkg == "" || fa.Name.Element == nil || fa.Name.Element.Name != v1UploadInput {
		return false
	}
	target, ok := fa.Target.(*java.Identifier)
	return ok && target.Name == s.managerPkg
}

// s3Qualifier is the name the file binds the s3 service to once migrated. The
// transfer manager's input lands in that package, so a file using it names s3
// whether or not it did before.
func (s *fileScan) s3Qualifier() string { return s.qualifierFor(s3Service) }

// retargetToS3 rewrites `s3manager.UploadInput` as `s3.PutObjectInput`.
func (v *migrateVisitor) retargetToS3(fa *java.FieldAccess) java.J {
	local := v.scan.s3Qualifier()
	v.serviceUsed[s3Service] = true
	v.needsS3 = true
	c := *fa
	c.Target = &java.Identifier{
		Prefix: fa.Target.(*java.Identifier).Prefix,
		Name:   local,
		Type:   lstutil.NamedType(v2ServicePkg + s3Service),
	}
	c.Name = java.LeftPadded[*java.Identifier]{
		Before:  fa.Name.Before,
		Element: &java.Identifier{Prefix: fa.Name.Element.Prefix, Name: v2UploadInput},
	}
	c.Type = lstutil.NamedType(v2ServicePkg + s3Service + "." + v2UploadInput)
	return &c
}

// managerConstructor rewrites a transfer-manager constructor. v2 builds both
// from an S3 client, so the session a v1 call passed becomes one.
func (v *migrateVisitor) managerConstructor(mi *java.MethodInvocation) (java.J, bool) {
	call, ok := qualifiedCall(mi, v.scan.managerPkg)
	if !ok {
		return nil, false
	}
	renamed, known := managerConstructors[call]
	if !known || renamed == "" {
		return nil, false
	}
	c := renameCall(mi, renamed, v2S3Manager).(*java.MethodInvocation)
	if managerTakesClient(call) {
		return c, true
	}
	args := realArgs(c)
	if len(args) != 1 {
		return nil, false
	}
	local := v.scan.s3Qualifier()
	v.serviceUsed[s3Service] = true
	v.needsS3 = true
	client := &java.MethodInvocation{
		Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: local, Type: lstutil.NamedType(v2ServicePkg + s3Service)}},
		Name:       &java.Identifier{Name: "NewFromConfig"},
		Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(args[0], java.EmptySpace)}}},
		MethodType: lstutil.FuncType(v2ServicePkg+s3Service, "NewFromConfig", nil),
	}
	out := *c
	out.Arguments = c.Arguments
	out.Arguments.Elements = []java.RightPadded[java.Expression]{{Element: client}}
	return &out, true
}

// managerBlockReason names an s3manager reference the migration cannot carry
// over, or "" when it can.
func (s *fileScan) managerBlockReason(name string) string {
	if managerNames[name] {
		return ""
	}
	return "s3manager." + name
}

// managerShapeOf reports the shape an s3manager composite constructs, so the
// enum and value-slice machinery sees the PutObjectInput it becomes.
func (s *fileScan) managerShapeOf(comp *golang.Composite) (service, shape string, ok bool) {
	fa, isField := comp.TypeExpr.(*java.FieldAccess)
	if !isField || !s.managerUploadInput(fa) {
		return "", "", false
	}
	return s3Service, v2UploadInput, true
}

// managerMethods are the transfer manager's methods, and the position the
// context takes in the v2 form. v1 took none; v2 takes one first.
var managerMethods = map[string]bool{"Upload": true, "Download": true}

// managerVarScan records the names holding an Uploader or a Downloader, so
// their calls can be told apart from any other method call in the file.
type managerVarScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *managerVarScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	if kind, isManager := v.scan.declaresManager(vd.TypeExpr); isManager {
		for _, decl := range vd.Variables {
			if d := decl.Element; d != nil && d.Name != nil {
				v.scan.managerVars[d.Name.Name] = kind
			}
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

func (v *managerVarScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if target, isIdent := a.Variable.(*java.Identifier); isIdent {
		if mi, isCall := a.Value.Element.(*java.MethodInvocation); isCall {
			if call, isManager := qualifiedCall(mi, v.scan.managerPkg); isManager {
				if renamed, known := managerConstructors[call]; known && renamed != "" {
					v.scan.managerVars[target.Name] = strings.TrimPrefix(renamed, "New")
				}
			}
		}
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

// declaresManager reports whether a type expression is one of v1's transfer
// manager types, naming which.
func (s *fileScan) declaresManager(expr java.Expression) (string, bool) {
	if s.managerPkg == "" {
		return "", false
	}
	if ptr, isPointer := expr.(*golang.PointerType); isPointer {
		expr = ptr.Elem
	}
	fa, isField := expr.(*java.FieldAccess)
	if !isField {
		return "", false
	}
	name, isManager := qualifiedRef(fa, s.managerPkg)
	if !isManager || (name != "Uploader" && name != "Downloader") {
		return "", false
	}
	return name, true
}

// managerMethod names the transfer-manager method a call invokes, when the
// receiver is one of the names the file holds a manager in.
func (s *fileScan) managerMethod(mi *java.MethodInvocation) (string, bool) {
	if s.managerPkg == "" || mi.Name == nil || mi.Select == nil {
		return "", false
	}
	var recv string
	switch target := mi.Select.Element.(type) {
	case *java.Identifier:
		recv = target.Name
	case *java.FieldAccess:
		if target.Name.Element == nil {
			return "", false
		}
		recv = target.Name.Element.Name
	default:
		return "", false
	}
	if s.managerVars[recv] == "" && !s.attributedManager(mi.Select.Element) && !s.managerInputCall(mi) {
		return "", false
	}
	return mi.Name.Name, true
}

// managerInputCall recognises a transfer-manager call by the shape it takes,
// for a receiver held in a struct another file declares. The file importing the
// transfer manager is what makes the shape unambiguous.
func (s *fileScan) managerInputCall(mi *java.MethodInvocation) bool {
	if s.managerPkg == "" || mi.Name == nil {
		return false
	}
	var want map[string]bool
	switch strings.TrimSuffix(mi.Name.Name, "WithContext") {
	case "Upload":
		// The upload input is the manager's own before the rewrite and s3's
		// after it, and the receiver may be visited either side of that.
		want = map[string]bool{v1UploadInput: true, v2UploadInput: true}
	case "Download":
		want = map[string]bool{"GetObjectInput": true}
	default:
		return false
	}
	for _, arg := range realArgs(mi) {
		expr := arg
		if unary, isUnary := expr.(*golang.Unary); isUnary {
			expr = unary.Expression
		}
		comp, isComposite := expr.(*golang.Composite)
		if !isComposite {
			continue
		}
		if fa, isField := comp.TypeExpr.(*java.FieldAccess); isField && fa.Name.Element != nil && want[fa.Name.Element.Name] {
			return true
		}
	}
	return false
}

// attributedManager reports whether an expression's parse-time type is one of
// the transfer managers, for a receiver declared in another file than the one
// calling it — a struct field, most often.
func (s *fileScan) attributedManager(expr java.Expression) bool {
	fqn := matcher.GetFullyQualifiedName(matcher.TypeOfExpression(expr))
	for _, pkg := range []string{v1S3Manager, v2S3Manager} {
		name, under := strings.CutPrefix(fqn, pkg+".")
		if under && (name == "Uploader" || name == "Downloader") {
			return true
		}
	}
	return false
}

// managerCall gives a transfer-manager call the context v2 takes, dropping the
// WithContext suffix where the caller already passed one.
func (v *migrateVisitor) managerCall(mi *java.MethodInvocation) (java.J, bool) {
	method, isManager := v.scan.managerMethod(mi)
	if !isManager {
		return nil, false
	}
	if trimmed := strings.TrimSuffix(method, "WithContext"); trimmed != method && managerMethods[trimmed] {
		return renameCall(mi, trimmed, v2S3Manager), true
	}
	if !managerMethods[method] {
		return nil, false
	}
	ctx := v.contextExpr()
	if ctx == nil {
		return nil, false
	}
	args := mi.Arguments.Elements
	elements := make([]java.RightPadded[java.Expression], 0, len(args)+1)
	elements = append(elements, java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(ctx, java.EmptySpace)})
	for i, rp := range args {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		if i == 0 {
			rp.Element = lstutil.SetExprPrefix(rp.Element, java.SingleSpace)
		}
		elements = append(elements, rp)
	}
	c := *mi
	c.Arguments = mi.Arguments
	c.Arguments.Elements = elements
	kind := v.scan.managerVars[receiverName(mi)]
	if kind == "" {
		kind = "Uploader"
	}
	c.MethodType = lstutil.FuncType(v2S3Manager+"."+kind, method, nil)
	return &c, true
}

// receiverName is the trailing name of a call's receiver, however it is reached.
func receiverName(mi *java.MethodInvocation) string {
	switch target := mi.Select.Element.(type) {
	case *java.Identifier:
		return target.Name
	case *java.FieldAccess:
		if target.Name.Element != nil {
			return target.Name.Element.Name
		}
	}
	return ""
}
