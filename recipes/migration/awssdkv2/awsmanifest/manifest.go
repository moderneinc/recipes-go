/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

// Package awsmanifest answers where an aws-sdk-go v1 name lands in v2.
//
// v1 kept every shape, enum and client for a service in one flat package. v2
// split them: the service package holds the client, its options, the operation
// inputs and outputs, the paginators and the waiters, while every modelled shape
// and every enum constant moved to a types sub-package. Which side a given name
// falls on is not derivable from its spelling — S3 alone keeps fourteen names
// that match no pattern, among them PresignClient and ResponseError — so the
// answer is generated from the v2 modules rather than guessed.
package awsmanifest

import (
	"embed"
	"strings"
	"sync"
)

//go:embed all:data
var data embed.FS

// Placement is where a v1 name lives in v2.
type Placement int

const (
	// Unknown means the manifest cannot account for the name, which is a
	// signal to leave the file alone rather than guess.
	Unknown Placement = iota
	// InService means v2 kept the name in the service package.
	InService
	// InTypes means the name moved to the service's types sub-package.
	InTypes
	// InTypesConst is an enum value, which also moved to types. v2 gives the
	// enum a named type where v1 left it a bare string, so a constant loses the
	// aws.String wrapper v1 needed around it.
	InTypesConst
	// IsClient means the name is v1's client type, which v2 calls Client.
	IsClient
)

type manifest struct {
	clientType string
	service    map[string]bool
	types      map[string]bool
	// consts maps an enum value to the enum it belongs to, which is what decides
	// whether it fits a given field.
	consts map[string]string
	// enumFields maps `Shape.Field` to the enum that field carries.
	enumFields map[string]string
	// listFields is the set of `Shape.Field` v2 holds by value where v1 held
	// pointers.
	// operations lists the operations the v2 client exposes, which is what a
	// generated replacement for a v1 iface package has to declare.
	operations []string
	// errCodes maps a v1 ErrCode constant to the wire string it holds.
	errCodes map[string]string
	// scalarFields maps `Shape.Field` to the v1 and v2 pointee types of a
	// pointer field whose width or kind v2 changed.
	scalarFields map[string][2]string
	// listFields maps `Shape.Field` to that list's element shape, or "-" where
	// the elements are scalars.
	listFields map[string]string
	// enumFieldNames is the set of field names any shape carries an enum in,
	// for the coarser question of whether a bare field name might be one.
	enumFieldNames map[string]bool
	// shapeFields maps `Shape.Field` to the shape that field holds.
	shapeFields map[string]string
	// pointerScalarFields maps `Shape.Field` to the basic type v2 holds a
	// pointer to it in.
	pointerScalarFields map[string]string
	// mapValueFields is the set of `Shape.Field` holding a map whose values v2
	// holds by value where v1 held pointers.
	mapValueFields map[string]bool
	// mapSliceFields is the set of `Shape.Field` holding a map whose slice values
	// v2 holds by value where v1 held pointers.
	mapSliceFields map[string]bool
	// depointeredFields maps `Shape.Field` to the basic type v2 holds by value
	// where v1 held a pointer to it.
	depointeredFields map[string]string
	// nonEnumFieldNames is the set of field names that are something other than
	// an enum somewhere in the service, which makes the name alone no evidence
	// of an enum.
	nonEnumFieldNames map[string]bool
}

var (
	mu     sync.Mutex
	loaded = map[string]*manifest{}
)

// load reads a service's manifest, caching it. A service with no manifest
// resolves to nil, and every name in it is Unknown.
func load(service string) *manifest {
	mu.Lock()
	defer mu.Unlock()
	if m, ok := loaded[service]; ok {
		return m
	}
	loaded[service] = nil

	raw, err := data.ReadFile("data/" + service + ".txt")
	if err != nil {
		return nil
	}
	m := &manifest{
		service:             map[string]bool{},
		types:               map[string]bool{},
		consts:              map[string]string{},
		enumFields:          map[string]string{},
		errCodes:            map[string]string{},
		scalarFields:        map[string][2]string{},
		listFields:          map[string]string{},
		enumFieldNames:      map[string]bool{},
		shapeFields:         map[string]string{},
		pointerScalarFields: map[string]string{},
		mapValueFields:      map[string]bool{},
		mapSliceFields:      map[string]bool{},
		depointeredFields:   map[string]string{},
		nonEnumFieldNames:   map[string]bool{},
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if len(line) < 3 {
			continue
		}
		name := line[2:]
		switch line[0] {
		case 'C':
			m.clientType = name
		case 'S':
			m.service[name] = true
		case 'T':
			m.types[name] = true
		case 'E':
			value, enum, _ := strings.Cut(name, " ")
			m.consts[value] = enum
		case 'O':
			m.operations = append(m.operations, name)
		case 'X':
			constant, value, _ := strings.Cut(name, " ")
			m.errCodes[constant] = value
		case 'I':
			parts := strings.Fields(name)
			if len(parts) == 3 {
				m.scalarFields[parts[0]] = [2]string{parts[1], parts[2]}
			}
		case 'L':
			field, elem, _ := strings.Cut(name, " ")
			m.listFields[field] = elem
		case 'F':
			field, enum, _ := strings.Cut(name, " ")
			m.enumFields[field] = enum
			if _, after, ok := strings.Cut(field, "."); ok {
				m.enumFieldNames[after] = true
			}
		case 'R':
			field, elem, _ := strings.Cut(name, " ")
			m.shapeFields[field] = elem
		case 'V':
			field, basic, _ := strings.Cut(name, " ")
			m.pointerScalarFields[field] = basic
		case 'Q':
			m.mapValueFields[name] = true
		case 'P':
			m.mapSliceFields[name] = true
		case 'D':
			field, basic, _ := strings.Cut(name, " ")
			m.depointeredFields[field] = basic
		case 'N':
			m.nonEnumFieldNames[name] = true
		}
	}
	loaded[service] = m
	return m
}

// Known reports whether a manifest covers the service at all.
func Known(service string) bool { return load(service) != nil }

// Place answers where the v1 name `<service>.<name>` lands in v2.
func Place(service, name string) Placement {
	m := load(service)
	if m == nil {
		return Unknown
	}
	switch {
	case name == m.clientType:
		return IsClient
	case m.consts[name] != "":
		return InTypesConst
	case m.types[name]:
		return InTypes
	case m.service[name]:
		return InService
	}
	return Unknown
}

// TypesAlias is the local name an added types import binds. Aliasing every one
// keeps a file that touches two services unambiguous, which a bare `types`
// would not be.
func TypesAlias(service string) string { return service + "types" }

// EnumOf returns the enum an enum value belongs to, or "".
func EnumOf(service, constant string) string {
	m := load(service)
	if m == nil {
		return ""
	}
	return m.consts[constant]
}

// FieldEnum returns the enum the named field of the named shape carries, or "".
func FieldEnum(service, shape, field string) string {
	m := load(service)
	if m == nil {
		return ""
	}
	return m.enumFields[shape+"."+field]
}

// HasEnumFieldNamed reports whether any shape in the service carries an enum in
// a field of this name. It answers the coarse question — is this field name ever
// an enum — where the shape it belongs to is not known, which is how a read
// through a pointer is recognised without resolving the receiver.
func HasEnumFieldNamed(service, field string) bool {
	m := load(service)
	if m == nil {
		return false
	}
	return m.enumFieldNames[field]
}

// IsOnlyEnumFieldNamed reports whether every field of this name in the service
// is an enum. A name that is an ordinary scalar on some other shape — `Name`,
// `Status` — says nothing about the receiver it was read from.
func IsOnlyEnumFieldNamed(service, field string) bool {
	m := load(service)
	if m == nil {
		return false
	}
	return m.enumFieldNames[field] && !m.nonEnumFieldNames[field]
}

// FieldShape names the shape a field holds, or "". It is what carries the
// inference through a read of a nested shape.
func FieldShape(service, shape, field string) string {
	m := load(service)
	if m == nil {
		return ""
	}
	return m.shapeFields[shape+"."+field]
}

// PointerScalar names the basic type v2 holds a field as a pointer to, or "".
// A fluent setter took that basic value and wrapped it, which is what an
// assignment replacing the setter has to do.
func PointerScalar(service, shape, field string) string {
	m := load(service)
	if m == nil {
		return ""
	}
	return m.pointerScalarFields[shape+"."+field]
}

// IsMapOfValues reports whether v2 holds a map field's values by value where v1
// held pointers.
func IsMapOfValues(service, shape, field string) bool {
	m := load(service)
	if m == nil {
		return false
	}
	return m.mapValueFields[shape+"."+field]
}

// IsMapOfValueSlices reports whether v2 holds a map field's slice values by
// value where v1 held pointers. Both the key and the element type change, which
// the value-slice helpers do not reach.
func IsMapOfValueSlices(service, shape, field string) bool {
	m := load(service)
	if m == nil {
		return false
	}
	return m.mapSliceFields[shape+"."+field]
}

// Depointered names the basic type v2 holds a field by value where v1 held a
// pointer to it, or "". Reading one no longer takes a dereference, and setting
// one no longer takes an aws helper.
func Depointered(service, shape, field string) string {
	m := load(service)
	if m == nil {
		return ""
	}
	return m.depointeredFields[shape+"."+field]
}

// IsValueSlice reports whether v2 holds the named field of the named shape by
// value where v1 held pointers. Those are the boundaries a migration has to
// convert across.
func IsValueSlice(service, shape, field string) bool {
	m := load(service)
	if m == nil {
		return false
	}
	return m.listFields[shape+"."+field] != ""
}

// ValueSliceElement names the shape a value slice holds, or "" where the field
// is not one and "-" where its elements are scalars.
func ValueSliceElement(service, shape, field string) string {
	m := load(service)
	if m == nil {
		return ""
	}
	return m.listFields[shape+"."+field]
}

// OperationShapes names the input and output shapes of an operation, and
// whether the service declares them. v2 keeps both in the service package under
// the operation's own name.
func OperationShapes(service, operation string) (input, output string, ok bool) {
	input, output = operation+"Input", operation+"Output"
	if Place(service, input) != InService || Place(service, output) != InService {
		return "", "", false
	}
	return input, output, true
}

// ScalarChange reports the v1 and v2 pointee types of a pointer field v2
// retyped, and whether it retyped one at all. v2 narrowed most counts from
// *int64 to *int32.
func ScalarChange(service, shape, field string) (from, to string, ok bool) {
	m := load(service)
	if m == nil {
		return "", "", false
	}
	change, found := m.scalarFields[shape+"."+field]
	if !found {
		return "", "", false
	}
	return change[0], change[1], true
}

// ErrCodeValue returns the wire string a v1 ErrCode constant holds, and whether
// the name is one. v2 dropped the constants for typed errors, and a caller
// comparing a code compares against this string either way.
func ErrCodeValue(service, constant string) (string, bool) {
	m := load(service)
	if m == nil {
		return "", false
	}
	value, ok := m.errCodes[constant]
	return value, ok
}

// Operations lists the operations a v2 client exposes.
func Operations(service string) []string {
	m := load(service)
	if m == nil {
		return nil
	}
	return m.operations
}

// ClientType names the type v1 called the service's client, which the iface
// package derived its interface name from.
func ClientType(service string) (string, bool) {
	m := load(service)
	if m == nil || m.clientType == "" {
		return "", false
	}
	return m.clientType, true
}
