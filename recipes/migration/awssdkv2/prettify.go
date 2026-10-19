/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"sort"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// Every v1 shape printed itself through awsutil.Prettify, so `%s` or Println
// on one wrote its set fields with sensitive ones masked. v2's types have no
// String method, and fmt would print their raw struct, secrets included. A
// shape handed to a printing call is therefore rendered by a generated copy of
// Prettify, which masks the fields v1 tagged sensitive.

const (
	prettifyHelper = "awsCompatPrettify"
	prettifyFile   = "zz_awssdkv2_prettify.go"
)

// printingNames are the calls taking `...any` that format their arguments as
// fmt does. A Stringer's String is what fmt prints, so a shape converted to the
// string v1's would have returned prints the same under any verb. Others taking
// `...any`, a mock's Called among them, compare the values and are left alone.
var printingNames = map[string]bool{
	"Print": true, "Println": true, "Sprint": true, "Sprintln": true, "Fprint": true, "Fprintln": true,
	"Fatal": true, "Fatalln": true, "Panic": true, "Panicln": true,
	"Log": true, "Error": true, "Skip": true,
	"Info": true, "Infoln": true, "Debug": true, "Debugln": true, "Warn": true, "Warnln": true,
	"Warning": true, "Warningln": true, "Trace": true, "Traceln": true,
}

// printedShapeArgs returns the positions of the shapes mi hands a printing call.
func (s *fileScan) printedShapeArgs(mi *java.MethodInvocation) map[int]shapeRef {
	if mi.Name == nil || mi.MethodType == nil || len(mi.MethodType.ParameterTypes) == 0 {
		return nil
	}
	params := mi.MethodType.ParameterTypes
	variadic, isSlice := params[len(params)-1].(*java.JavaTypeArray)
	if !isSlice || matcher.GetFullyQualifiedName(variadic.ElemType) != "any" {
		return nil
	}
	// A printf-style name ends in f by the convention vet checks it by.
	name := mi.Name.Name
	if !printingNames[name] && !(strings.HasSuffix(name, "f") && len(params) >= 2) {
		return nil
	}
	var out map[int]shapeRef
	for i, arg := range realArgs(mi) {
		if i < len(params)-1 {
			continue
		}
		if ref, ok := s.printedShape(arg); ok {
			if out == nil {
				out = map[int]shapeRef{}
			}
			out[i] = ref
		}
	}
	return out
}

// printedShape reports whether expr is a single v1 shape, as opposed to a list
// of them, an enum value, or anything v2 already typed.
func (s *fileScan) printedShape(expr java.Expression) (shapeRef, bool) {
	if strings.HasPrefix(matcher.GetFullyQualifiedName(matcher.TypeOfExpression(expr)), v2ServicePkg) {
		return shapeRef{}, false
	}
	ref, ok := s.receiverShape(expr)
	if !ok {
		return shapeRef{}, false
	}
	switch awsmanifest.Place(ref.service, ref.shape) {
	case awsmanifest.InTypes:
		return ref, true
	case awsmanifest.InService:
		return ref, isInputOrOutputShape(ref.shape)
	}
	return shapeRef{}, false
}

// shapeStringCall reports whether mi is String or GoString on a v1 shape.
func (s *fileScan) shapeStringCall(mi *java.MethodInvocation) (java.Expression, shapeRef, bool) {
	if mi.Name == nil || mi.Select == nil || (mi.Name.Name != "String" && mi.Name.Name != "GoString") || len(realArgs(mi)) != 0 {
		return nil, shapeRef{}, false
	}
	ref, ok := s.printedShape(mi.Select.Element)
	return mi.Select.Element, ref, ok
}

// wrapPrinted wraps the shapes printedShapeArgs found.
func (v *migrateVisitor) wrapPrinted(mi *java.MethodInvocation, printed map[int]shapeRef) *java.MethodInvocation {
	if len(printed) == 0 {
		return mi
	}
	elements := make([]java.RightPadded[java.Expression], len(mi.Arguments.Elements))
	copy(elements, mi.Arguments.Elements)
	for i := range elements {
		if ref, ok := printed[i]; ok {
			elements[i].Element = wrapCompat(prettifyHelper, elements[i].Element)
			if v.printedServices == nil {
				v.printedServices = map[string]bool{}
			}
			v.printedServices[ref.service] = true
		}
	}
	c := *mi
	c.Arguments.Elements = elements
	return &c
}

// prettifySource renders the helper for a package, masking the sensitive
// fields of the services it prints shapes of.
func prettifySource(pkg string, services map[string]bool) string {
	var keys []string
	sensitive := map[string][]string{}
	for service := range services {
		for _, key := range awsmanifest.SensitiveFields(service) {
			shape, field, _ := strings.Cut(key, ".")
			path := v2ServicePath(service) + "/types"
			if awsmanifest.Place(service, shape) == awsmanifest.InService {
				path = v2ServicePath(service)
			}
			typ := path + "." + awsmanifest.TypesName(service, shape)
			if sensitive[typ] == nil {
				keys = append(keys, typ)
			}
			sensitive[typ] = append(sensitive[typ], field)
		}
	}
	sort.Strings(keys)
	var table strings.Builder
	table.WriteString("{")
	if len(keys) > 0 {
		table.WriteString("\n")
	}
	for _, typ := range keys {
		fields := sensitive[typ]
		sort.Strings(fields)
		quoted := make([]string, len(fields))
		for i, f := range fields {
			quoted[i] = fmt.Sprintf("%q: true", f)
		}
		fmt.Fprintf(&table, "\t%q: {%s},\n", typ, strings.Join(quoted, ", "))
	}
	table.WriteString("}")
	return fmt.Sprintf(prettifySourceTemplate, pkg, table.String())
}

const prettifySourceTemplate = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
)

// awsCompatPrettify renders a value as aws-sdk-go v1's shapes printed
// themselves: the fields that are set, in order, with those v1 marked
// sensitive masked.
func awsCompatPrettify(i any) string {
	var buf bytes.Buffer
	awsCompatPrettifyValue(reflect.ValueOf(i), 0, &buf)
	return buf.String()
}

// awsCompatSensitive names the fields v1 tagged sensitive, which v2's types no
// longer mark.
var awsCompatSensitive = map[string]map[string]bool%s

func awsCompatByValue(k reflect.Kind) bool {
	switch k {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	}
	return false
}

func awsCompatPrettifyValue(v reflect.Value, indent int, buf *bytes.Buffer) {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		strtype := v.Type().String()
		if strtype == "time.Time" {
			fmt.Fprintf(buf, "%%s", v.Interface())
			break
		} else if strings.HasPrefix(strtype, "io.") {
			buf.WriteString("<buffer>")
			break
		}

		buf.WriteString("{\n")

		sensitive := awsCompatSensitive[v.Type().PkgPath()+"."+v.Type().Name()]
		names := []string{}
		for i := 0; i < v.Type().NumField(); i++ {
			ft := v.Type().Field(i)
			f := v.Field(i)
			if !ft.IsExported() {
				continue
			}
			if (f.Kind() == reflect.Ptr || f.Kind() == reflect.Slice || f.Kind() == reflect.Map) && f.IsNil() {
				continue
			}
			// v1 held enums and scalars by pointer, nil when unset; v2 holds some
			// by value, and sends none whose value is zero.
			if awsCompatByValue(f.Kind()) && ft.Type.Kind() != reflect.Ptr && f.IsZero() {
				continue
			}
			// v2's outputs carry response metadata v1's had no field for.
			if ft.Name == "ResultMetadata" {
				continue
			}
			names = append(names, ft.Name)
		}
		// v1 declared a shape's fields alphabetically; v2 puts required ones first.
		sort.Strings(names)

		for i, n := range names {
			val := v.FieldByName(n)
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString(n + ": ")

			if sensitive[n] {
				buf.WriteString("<sensitive>")
			} else {
				awsCompatPrettifyValue(val, indent+2, buf)
			}

			if i < len(names)-1 {
				buf.WriteString(",\n")
			}
		}

		buf.WriteString("\n" + strings.Repeat(" ", indent) + "}")
	case reflect.Slice:
		strtype := v.Type().String()
		if strtype == "[]uint8" {
			fmt.Fprintf(buf, "<binary> len %%d", v.Len())
			break
		}

		nl, id, id2 := "", "", ""
		if v.Len() > 3 {
			nl, id, id2 = "\n", strings.Repeat(" ", indent), strings.Repeat(" ", indent+2)
		}
		buf.WriteString("[" + nl)
		for i := 0; i < v.Len(); i++ {
			buf.WriteString(id2)
			awsCompatPrettifyValue(v.Index(i), indent+2, buf)

			if i < v.Len()-1 {
				buf.WriteString("," + nl)
			}
		}

		buf.WriteString(nl + id + "]")
	case reflect.Map:
		buf.WriteString("{\n")

		for i, k := range v.MapKeys() {
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString(k.String() + ": ")
			awsCompatPrettifyValue(v.MapIndex(k), indent+2, buf)

			if i < v.Len()-1 {
				buf.WriteString(",\n")
			}
		}

		buf.WriteString("\n" + strings.Repeat(" ", indent) + "}")
	default:
		if !v.IsValid() {
			fmt.Fprint(buf, "<invalid value>")
			return
		}
		format := "%%v"
		switch v.Interface().(type) {
		case io.ReadSeeker, io.Reader:
			format = "buffer(%%p)"
		default:
			// v1 held an enum as a string; v2 as a type of its own.
			if v.Kind() == reflect.String {
				format = "%%q"
			}
		}
		fmt.Fprintf(buf, format, v.Interface())
	}
}
`
