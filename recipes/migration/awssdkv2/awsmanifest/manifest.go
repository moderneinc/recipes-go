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
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
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
	// unions are the shapes v2 made union interfaces, and unionVariants the v1
	// field names their members stand for.
	unions        map[string]bool
	unionVariants map[string]bool
	// depointeredFields maps `Shape.Field` to the basic type v2 holds by value
	// where v1 held a pointer to it.
	depointeredFields map[string]string
	// narrowedFields maps a depointered `Shape.Field` to the v1 pointee type,
	// where v2 also narrowed it: v1's *int64 became an int32.
	narrowedFields map[string]string
	// nonEnumFieldNames is the set of field names that are something other than
	// an enum somewhere in the service, which makes the name alone no evidence
	// of an enum.
	nonEnumFieldNames map[string]bool
	// renames maps a v1 shape to the name v2 gave it.
	renames map[string]string
	// sensitive lists the `Shape.Field` v1 tagged sensitive and v2 does not.
	sensitive []string
	// waiters maps a v1 waiter to its delay in seconds and its attempts.
	waiters map[string][2]int
	// interfaces are the types-package interfaces, every one a union.
	interfaces map[string]bool
	// validations maps a v1 shape with a Validate method to its rules, in
	// v1's order; unvalidatable holds those whose rules v2 cannot express.
	validations   map[string][]ValidationRule
	unvalidatable map[string]bool
	// pageTokens maps an operation to the input and output fields its v2
	// paginator carries the page token in.
	pageTokens map[string][2]string
}

// ValidationRule is one check of a v1 Validate method. Kind is req, len, min,
// nest or each; Arg the length, the minimum or the nested shape; Form how the
// field is read in v2.
type ValidationRule struct {
	Field, Kind, Arg, Form string
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
		narrowedFields:      map[string]string{},
		nonEnumFieldNames:   map[string]bool{},
		renames:             map[string]string{},
		waiters:             map[string][2]int{},
		interfaces:          map[string]bool{},
		validations:         map[string][]ValidationRule{},
		unvalidatable:       map[string]bool{},
		pageTokens:          map[string][2]string{},
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
		case 'W':
			parts := strings.Fields(name)
			if len(parts) == 3 {
				m.depointeredFields[parts[0]] = parts[2]
				m.narrowedFields[parts[0]] = parts[1]
			}
		case 'N':
			m.nonEnumFieldNames[name] = true
		case 'M':
			from, to, _ := strings.Cut(name, " ")
			m.renames[from] = to
		case 'H':
			m.sensitive = append(m.sensitive, name)
		case 'U':
			m.interfaces[name] = true
		case 'K':
			parts := strings.Fields(name)
			if len(parts) < 4 {
				continue
			}
			shape := parts[0]
			if parts[2] == "-" {
				if parts[3] == "unsupported" {
					m.unvalidatable[shape] = true
				} else {
					m.validations[shape] = []ValidationRule{}
				}
				continue
			}
			rule := ValidationRule{Field: parts[2], Kind: parts[3]}
			switch rest := parts[4:]; {
			case rule.Kind == "req" && len(rest) == 1:
				rule.Form = rest[0]
			case rule.Kind == "nest" && len(rest) == 1:
				rule.Arg = rest[0]
			case len(rest) == 2:
				rule.Arg, rule.Form = rest[0], rest[1]
			default:
				m.unvalidatable[shape] = true
				continue
			}
			m.validations[shape] = append(m.validations[shape], rule)
		case 'G':
			parts := strings.Fields(name)
			if len(parts) == 3 {
				m.pageTokens[parts[0]] = [2]string{parts[1], parts[2]}
			}
		case 'Y':
			parts := strings.Fields(name)
			if len(parts) == 3 {
				delay, _ := strconv.Atoi(parts[1])
				attempts, _ := strconv.Atoi(parts[2])
				m.waiters[parts[0]] = [2]int{delay, attempts}
			}
		}
	}
	m.unions, m.unionVariants = unionsOf(m.types)
	// A union with a single member is a union all the same, which only the
	// interface record says.
	for name := range m.interfaces {
		if !m.unions[name] {
			m.unions[name] = true
			for t := range m.types {
				if variant, ok := strings.CutPrefix(t, name+"Member"); ok && variant != "" {
					m.unionVariants[variant] = true
				}
			}
		}
	}
	// A list of a union holds v2's interface, not a value of a shape, so it
	// takes no conversion between v1's pointers and v2's values.
	for field, elem := range m.listFields {
		if m.unions[elem] {
			delete(m.listFields, field)
		}
	}
	// A variant spelled like another shape's field cannot tell a union read
	// from an ordinary one: S3's MetricsFilter has a Prefix member, and half of
	// S3's shapes a Prefix field.
	for _, fields := range []map[string]bool{keysOf(m.enumFields), keysOf(m.scalarFields), keysOf(m.listFields),
		keysOf(m.shapeFields), keysOf(m.pointerScalarFields), m.mapValueFields, m.mapSliceFields, keysOf(m.depointeredFields)} {
		for key := range fields {
			if shape, field, ok := strings.Cut(key, "."); ok && !m.unions[shape] {
				delete(m.unionVariants, field)
			}
		}
	}
	loaded[service] = m
	return m
}

func keysOf[V any](m map[string]V) map[string]bool {
	keys := make(map[string]bool, len(m))
	for k := range m {
		keys[k] = true
	}
	return keys
}

// unionsOf finds the shapes v2 turned into union interfaces, which it spells as
// the union plus one `<Union>Member<Variant>` type per v1 field.
func unionsOf(types map[string]bool) (unions, variants map[string]bool) {
	members := map[string][]string{}
	for name := range types {
		base, variant, found := strings.Cut(name, "Member")
		if found && types[base] && variant != "" && unicode.IsUpper(rune(variant[0])) {
			members[base] = append(members[base], variant)
		}
	}
	unions, variants = map[string]bool{}, map[string]bool{}
	for base, vs := range members {
		if len(vs) < 2 {
			continue
		}
		unions[base] = true
		// v1 read an event stream through an event channel rather than off the
		// shape's fields, so its variant names say nothing about a field read.
		if strings.Contains(base, "Stream") || strings.HasSuffix(base, "Event") {
			continue
		}
		for _, v := range vs {
			variants[v] = true
		}
	}
	return unions, variants
}

// Known reports whether a manifest covers the service at all.
func Known(service string) bool { return load(service) != nil }

// IsUnion reports whether v2 turned the v1 shape into a union interface, whose
// value is one member type per field v1 could set. Nothing about building or
// reading one carries over by renaming.
func IsUnion(service, shape string) bool {
	m := load(service)
	return m != nil && m.unions[shape]
}

// IsUnionVariant reports whether field is one a union shape of the service
// held in v1, and so a read of it may be a read of a union member.
func IsUnionVariant(service, field string) bool {
	m := load(service)
	return m != nil && m.unionVariants[field]
}

// Services lists every service a manifest covers, in name order.
func Services() []string {
	entries, err := data.ReadDir("data")
	if err != nil {
		return nil
	}
	var services []string
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".txt"); ok {
			services = append(services, name)
		}
	}
	return services
}

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
	case m.types[name], m.renames[name] != "":
		return InTypes
	case m.service[name]:
		return InService
	}
	return Unknown
}

// TypesName returns the name v2 gives a v1 shape in its types package.
func TypesName(service, name string) string {
	if m := load(service); m != nil && m.renames[name] != "" {
		return m.renames[name]
	}
	return name
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

// NarrowedFrom names the v1 pointee type of a depointered field v2 also
// narrowed, or "" where the two hold the same basic type.
func NarrowedFrom(service, shape, field string) string {
	m := load(service)
	if m == nil {
		return ""
	}
	return m.narrowedFields[shape+"."+field]
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

// SensitiveFields lists the `Shape.Field` v1 printed masked, by v1 shape name.
func SensitiveFields(service string) []string {
	if m := load(service); m != nil {
		return m.sensitive
	}
	return nil
}

// v2PackageNames are the v1 service packages v2 spells differently.
var v2PackageNames = map[string]string{
	"elb":   "elasticloadbalancing",
	"elbv2": "elasticloadbalancingv2",
}

// V2Name is the name of a v1 service's package in v2.
func V2Name(service string) string {
	if name, renamed := v2PackageNames[service]; renamed {
		return name
	}
	return service
}

// V1Name is the v1 service a v2 package name stands for.
func V1Name(v2Name string) string {
	for v1, v2 := range v2PackageNames {
		if v2 == v2Name {
			return v1
		}
	}
	return v2Name
}

// UnionMember names the v2 type standing for a v1 union shape's field set, as
// ServiceDiscoveryMemberDns stands for ServiceDiscovery's Dns.
func UnionMember(service, union, field string) (string, bool) {
	m := load(service)
	if m == nil || !m.unions[union] {
		return "", false
	}
	member := union + "Member" + field
	return member, m.types[member]
}

// WaiterTiming answers a v1 waiter's delay in seconds and its attempts.
func WaiterTiming(service, waiter string) (delay, attempts int, ok bool) {
	m := load(service)
	if m == nil {
		return 0, 0, false
	}
	t, ok := m.waiters[waiter]
	return t[0], t[1], ok
}

// V1ShapeName is the v1 name of a shape known by the name v2 gave it.
func V1ShapeName(service, name string) string {
	if m := load(service); m != nil {
		for v1, v2 := range m.renames {
			if v2 == name {
				return v1
			}
		}
	}
	return name
}

// PageTokens returns the input field an operation's v2 paginator sends its
// token in and the output field it reads the next one from.
func PageTokens(service, operation string) (input, output string, ok bool) {
	m := load(service)
	if m == nil {
		return "", "", false
	}
	tokens, ok := m.pageTokens[operation]
	return tokens[0], tokens[1], ok
}

// Validation returns the rules of a v1 shape's Validate method, and whether it
// has one whose rules v2 can express.
func Validation(service, shape string) ([]ValidationRule, bool) {
	m := load(service)
	if m == nil || m.unvalidatable[shape] {
		return nil, false
	}
	rules, ok := m.validations[shape]
	return rules, ok
}

// ValidatedShapes lists the v1 shapes of a service with a Validate method v2
// can express.
func ValidatedShapes(service string) []string {
	m := load(service)
	if m == nil {
		return nil
	}
	shapes := make([]string, 0, len(m.validations))
	for shape := range m.validations {
		if !m.unvalidatable[shape] {
			shapes = append(shapes, shape)
		}
	}
	sort.Strings(shapes)
	return shapes
}
