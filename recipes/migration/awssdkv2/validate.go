/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// v1 shapes validated themselves; v2 checks only the required fields, inside
// the operation. A shape's Validate becomes a generated copy of v1's: the same
// rules, minimum lengths and values included, reporting the same error text.

const validateFile = "zz_awssdkv2_validate.go"

// Whether a shape variable holds a pointer or a value decides whether the
// helper, which takes a pointer as v1's method did, is handed it or its
// address.
const (
	shapePointer byte = 'p'
	shapeValue   byte = 'v'
	shapeEither  byte = '?'
)

// noteShapeAddress records how name holds its shape; one bound both ways in
// the file is known as neither.
func (s *fileScan) noteShapeAddress(name string, how byte) {
	if s.shapeAddress == nil {
		s.shapeAddress = map[string]byte{}
	}
	if prior, seen := s.shapeAddress[name]; seen && prior != how {
		how = shapeEither
	}
	s.shapeAddress[name] = how
}

// addressOfShapeExpr is how an assigned value holds a shape: a literal by
// value, its address or an operation's output by pointer.
func addressOfShapeExpr(expr java.Expression) byte {
	switch e := expr.(type) {
	case *golang.Composite:
		return shapeValue
	case *golang.Unary:
		if e.Operator.Element == golang.AddressOf {
			return shapePointer
		}
	case *java.MethodInvocation:
		return shapePointer
	}
	return shapeEither
}

// validatable reports whether a shape, and every shape it validates in turn,
// has a Validate v2 can express.
func validatable(service, shape string) bool {
	return validatableFrom(service, shape, map[string]bool{})
}

func validatableFrom(service, shape string, seen map[string]bool) bool {
	if seen[shape] {
		return true
	}
	seen[shape] = true
	rules, ok := awsmanifest.Validation(service, shape)
	if !ok {
		return false
	}
	for _, rule := range rules {
		if (rule.Kind == "nest" || rule.Kind == "each") && !validatableFrom(service, rule.Arg, seen) {
			return false
		}
	}
	return true
}

// shapeValidateCall reports whether mi is a shape's Validate the helper can
// stand in for, and whether its receiver is a value whose address it takes.
func (s *fileScan) shapeValidateCall(mi *java.MethodInvocation) (shapeRef, bool, bool) {
	if mi.Name == nil || mi.Name.Name != "Validate" || mi.Select == nil || len(realArgs(mi)) != 0 {
		return shapeRef{}, false, false
	}
	owner, isShape := s.receiverShape(mi.Select.Element)
	if !isShape || !validatable(owner.service, owner.shape) {
		return shapeRef{}, false, false
	}
	switch recv := mi.Select.Element.(type) {
	case *java.Identifier:
		switch s.shapeAddress[recv.Name] {
		case shapePointer:
			return owner, false, true
		case shapeValue:
			return owner, true, true
		}
	case *java.FieldAccess:
		// A shape held in another's field is a pointer in v2 as in v1.
		if _, nested := s.receiverShape(recv.Target); nested {
			return owner, false, true
		}
	}
	return shapeRef{}, false, false
}

// validateHelper names the generated function validating a shape.
func validateHelper(service, shape string) string {
	pkg := []rune(awsmanifest.V2Name(service))
	pkg[0] = unicode.ToUpper(pkg[0])
	return "awsCompatValidate" + string(pkg) + awsmanifest.TypesName(service, shape)
}

// validateCall stands the helper in for recv.Validate().
func (v *migrateVisitor) validateCall(mi *java.MethodInvocation, owner shapeRef, byValue bool) java.Expression {
	recv := lstutil.SetExprPrefix(mi.Select.Element, java.EmptySpace)
	if byValue {
		recv = &golang.Unary{ID: uuid.New(), Operator: java.LeftPadded[golang.UnaryOperator]{Element: golang.AddressOf}, Expression: recv}
	}
	if v.validatedShapes == nil {
		v.validatedShapes = map[string]bool{}
	}
	v.validatedShapes[owner.service+"."+owner.shape] = true
	helper := validateHelper(owner.service, owner.shape)
	return &java.MethodInvocation{
		ID:         uuid.New(),
		Prefix:     mi.Prefix,
		Name:       &java.Identifier{ID: uuid.New(), Name: helper},
		Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: recv}}},
		MethodType: lstutil.FuncType("", helper, nil),
	}
}

// validateSource renders the validators for shapes, each `service.Shape`, and
// for every shape they validate in turn.
func validateSource(pkg string, shapes map[string]bool) string {
	type entry struct{ service, shape string }
	pending := make([]entry, 0, len(shapes))
	for key := range shapes {
		service, shape, _ := strings.Cut(key, ".")
		pending = append(pending, entry{service, shape})
	}
	done := map[entry]bool{}
	services := map[string]bool{}
	usesTypes := map[string]bool{}
	var funcs []string
	byName := map[string]string{}
	for len(pending) > 0 {
		e := pending[0]
		pending = pending[1:]
		if done[e] {
			continue
		}
		done[e] = true
		rules, _ := awsmanifest.Validation(e.service, e.shape)
		qualifier := awsmanifest.V2Name(e.service)
		if awsmanifest.Place(e.service, e.shape) != awsmanifest.InService {
			qualifier = awsmanifest.TypesAlias(e.service)
			usesTypes[e.service] = true
		} else {
			services[e.service] = true
		}
		var body strings.Builder
		for _, rule := range rules {
			if rule.Kind == "nest" || rule.Kind == "each" {
				pending = append(pending, entry{e.service, rule.Arg})
			}
			body.WriteString(validationCheck(e.service, rule))
		}
		name := validateHelper(e.service, e.shape)
		funcs = append(funcs, name)
		byName[name] = fmt.Sprintf(`
func %s(s *%s.%s) error {
	invalid := awsCompatInvalidParams{Context: %q}
%s	if len(invalid.errs) > 0 {
		return invalid
	}
	return nil
}
`, name, qualifier, awsmanifest.TypesName(e.service, e.shape), awsmanifest.TypesName(e.service, e.shape), body.String())
	}
	sort.Strings(funcs)
	imports := []string{`"fmt"`, `"strings"`}
	var paths []string
	for service := range services {
		paths = append(paths, fmt.Sprintf("%q", v2ServicePath(service)))
	}
	for service := range usesTypes {
		paths = append(paths, fmt.Sprintf("%s %q", awsmanifest.TypesAlias(service), v2ServicePath(service)+"/types"))
	}
	sort.Slice(paths, func(i, j int) bool { return importPathOf(paths[i]) < importPathOf(paths[j]) })
	var b strings.Builder
	fmt.Fprintf(&b, validateHeader, pkg, strings.Join(imports, "\n\t"), strings.Join(paths, "\n\t"))
	for _, name := range funcs {
		b.WriteString(byName[name])
	}
	return b.String()
}

// importPathOf is the quoted path of an import spec, past any alias.
func importPathOf(spec string) string {
	return spec[strings.Index(spec, `"`):]
}

// validationCheck renders one rule as the statement v1's Validate held.
func validationCheck(service string, rule awsmanifest.ValidationRule) string {
	f := "s." + rule.Field
	switch rule.Kind {
	case "req":
		switch rule.Form {
		case "nil":
			return fmt.Sprintf("\tif %s == nil {\n\t\tinvalid.add(awsCompatParamRequired(%q))\n\t}\n", f, rule.Field)
		case "empty":
			return fmt.Sprintf("\tif len(%s) == 0 {\n\t\tinvalid.add(awsCompatParamRequired(%q))\n\t}\n", f, rule.Field)
		}
		// A value v2 always holds is never missing.
		return ""
	case "len":
		cond := ""
		switch rule.Form {
		case "ptr":
			cond = fmt.Sprintf("%s != nil && len(*%s) < %s", f, f, rule.Arg)
		case "enum", "value":
			cond = fmt.Sprintf("%s != \"\" && len(%s) < %s", f, f, rule.Arg)
		default:
			cond = fmt.Sprintf("%s != nil && len(%s) < %s", f, f, rule.Arg)
		}
		return fmt.Sprintf("\tif %s {\n\t\tinvalid.add(awsCompatParamMinLen(%q, %s))\n\t}\n", cond, rule.Field, rule.Arg)
	case "min":
		value := f
		if rule.Form == "ptr" {
			value = "*" + f
		}
		// A fractional or exponent minimum may not be representable in the
		// field's integer type, where v1's comparison was in float64 terms.
		if strings.ContainsAny(rule.Arg, ".e") {
			value = "float64(" + value + ")"
		}
		set := f + " != nil"
		if rule.Form == "value" {
			// v2 holds an unset value as zero, which v1 held as nil and skipped.
			set = f + " != 0"
		}
		return fmt.Sprintf("\tif %s && %s < %s {\n\t\tinvalid.add(awsCompatParamMinValue(%q, %s))\n\t}\n", set, value, rule.Arg, rule.Field, rule.Arg)
	case "nest":
		return fmt.Sprintf("\tif %s != nil {\n\t\tif err := %s(%s); err != nil {\n\t\t\tinvalid.addNested(%q, err)\n\t\t}\n\t}\n", f, validateHelper(service, rule.Arg), f, rule.Field)
	case "each":
		helper := validateHelper(service, rule.Arg)
		key, elem := "i", "&"+f+"[i]"
		loop := fmt.Sprintf("for i := range %s {", f)
		skipNil := ""
		switch rule.Form {
		case "slicep":
			loop, elem = fmt.Sprintf("for i, v := range %s {", f), "v"
			skipNil = "\t\tif v == nil {\n\t\t\tcontinue\n\t\t}\n"
		case "map":
			loop, key, elem = fmt.Sprintf("for k, v := range %s {", f), "k", "&v"
		case "mapp":
			loop, key, elem = fmt.Sprintf("for k, v := range %s {", f), "k", "v"
			skipNil = "\t\tif v == nil {\n\t\t\tcontinue\n\t\t}\n"
		}
		return fmt.Sprintf("\t%s\n%s\t\tif err := %s(%s); err != nil {\n\t\t\tinvalid.addNested(fmt.Sprintf(\"%%s[%%v]\", %q, %s), err)\n\t\t}\n\t}\n", loop, skipNil, helper, elem, rule.Field, key)
	}
	return ""
}

const validateHeader = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import (
	%s

	%s
)

// awsCompatInvalidParams is aws-sdk-go v1's request.ErrInvalidParams: every
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
	return fmt.Sprintf("%%d validation error(s) found.", len(e.errs))
}

func (e awsCompatInvalidParams) Error() string {
	var w strings.Builder
	fmt.Fprintf(&w, "%%s: %%s\n", e.Code(), e.Message())
	for _, err := range e.errs {
		fmt.Fprintf(&w, "- %%s\n", err.Message())
	}
	return w.String()
}

// awsCompatInvalidParam is one broken rule, as v1 reported it.
type awsCompatInvalidParam struct {
	context, nestedContext, field, code, msg string
}

func (e *awsCompatInvalidParam) Code() string { return e.code }

func (e *awsCompatInvalidParam) Message() string {
	return fmt.Sprintf("%%s, %%s.", e.msg, e.Field())
}

func (e *awsCompatInvalidParam) Error() string {
	return fmt.Sprintf("%%s: %%s", e.code, e.Message())
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
	return &awsCompatInvalidParam{code: "ParamMinLenError", field: field, msg: fmt.Sprintf("minimum field size of %%v", min)}
}

func awsCompatParamMinValue(field string, min float64) *awsCompatInvalidParam {
	return &awsCompatInvalidParam{code: "ParamMinValueError", field: field, msg: fmt.Sprintf("minimum field value of %%v", min)}
}
`
