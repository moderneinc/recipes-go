/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

// Command genmanifest derives an awsmanifest data file for each named aws-sdk-go
// v1 service by parsing the v1 package and its aws-sdk-go-v2 counterpart.
//
// Usage, from recipes/migration/awssdkv2:
//
//	go run ./internal/genmanifest sqs s3            # write awsmanifest/data/<svc>.txt
//	go run ./internal/genmanifest -check            # diff every existing data file
//	go run ./internal/genmanifest -v2version v1.2.3 acm
//	go run ./internal/genmanifest -v2module elasticloadbalancingv2 elbv2
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
)

const (
	v1Module  = "github.com/aws/aws-sdk-go"
	v1Version = "v1.55.8"
	v2Service = "github.com/aws/aws-sdk-go-v2/service/"
)

var (
	dataDir    = flag.String("data", "awsmanifest/data", "directory holding <service>.txt")
	modulesGen = flag.String("modules", "zz_modules_gen.go", "file pinning each v2 module's version")
	v2Version  = flag.String("v2version", "", "v2 module version, overriding the pinned one (single service only)")
	v2Name     = flag.String("v2module", "", "v2 service module name where it differs from the v1 package (single service only)")
	check      = flag.Bool("check", false, "compare against the existing data files instead of writing")
	out        = flag.String("o", "", "write here instead of the data directory")
)

func main() {
	log.SetFlags(0)
	flag.Parse()
	services := flag.Args()
	// With no services named, every manifest already written is regenerated,
	// which is how `go generate` keeps them at the pinned versions.
	if len(services) == 0 {
		entries, err := os.ReadDir(*dataDir)
		if err != nil {
			log.Fatal(err)
		}
		for _, e := range entries {
			if s, ok := strings.CutSuffix(e.Name(), ".txt"); ok {
				services = append(services, s)
			}
		}
	}
	if len(services) == 0 {
		log.Fatal("usage: genmanifest [-check] [-v2version v] [-v2module m] [service...]")
	}
	if (*v2Version != "" || *v2Name != "") && len(services) > 1 {
		log.Fatal("-v2version and -v2module apply to a single service")
	}
	pins := pinnedVersions(*modulesGen)
	v1Dir := download(v1Module + "@" + v1Version)

	failed := false
	for _, svc := range services {
		mod := svc
		if *v2Name != "" {
			mod = *v2Name
		} else {
			mod = awsmanifest.V2Name(svc)
		}
		version := *v2Version
		if version == "" {
			version = pins[v2Service+mod]
		}
		if version == "" {
			version = "latest"
		}
		v2Dir, resolved := downloadVersion(v2Service + mod + "@" + version)
		if version == "latest" || *v2Version != "" {
			log.Printf("%s: using %s%s@%s", svc, v2Service, mod, resolved)
		}
		text := generate(filepath.Join(v1Dir, "service", svc), v2Dir)
		path := filepath.Join(*dataDir, svc+".txt")
		if *check {
			existing, err := os.ReadFile(path)
			if err != nil {
				log.Fatal(err)
			}
			if d := diff(string(existing), text); d != "" {
				failed = true
				fmt.Printf("%s: differs (- data, + generated)\n%s", svc, d)
			} else {
				fmt.Printf("%s: identical\n", svc)
			}
			continue
		}
		if *out != "" {
			path = filepath.Join(*out, svc+".txt")
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("wrote %s (%s@%s)\n", path, mod, resolved)
	}
	if failed {
		os.Exit(1)
	}
}

// pinnedVersions reads the version each module in v2Requirements pins itself at.
func pinnedVersions(path string) map[string]string {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		log.Fatal(err)
	}
	pins := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "v2Requirements" || len(vs.Values) != 1 {
			return true
		}
		for _, elt := range vs.Values[0].(*ast.CompositeLit).Elts {
			kv := elt.(*ast.KeyValueExpr)
			key := unquote(kv.Key)
			for _, req := range kv.Value.(*ast.CompositeLit).Elts {
				pair := req.(*ast.CompositeLit).Elts
				if unquote(pair[0]) == key {
					pins[key] = unquote(pair[1])
				}
			}
		}
		return false
	})
	return pins
}

func unquote(e ast.Expr) string {
	s, _ := strconv.Unquote(e.(*ast.BasicLit).Value)
	return s
}

func download(modVersion string) string {
	dir, _ := downloadVersion(modVersion)
	return dir
}

// downloadVersion fetches a module into the cache from a scratch directory, so
// this repository's go.mod plays no part in resolving it.
func downloadVersion(modVersion string) (dir, version string) {
	scratch, err := os.MkdirTemp("", "genmanifest")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(scratch)
	cmd := exec.Command("go", "mod", "download", "-json", modVersion)
	cmd.Dir = scratch
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("go mod download %s: %v\n%s", modVersion, err, output)
	}
	var info struct{ Dir, Version string }
	if err := json.Unmarshal(output, &info); err != nil || info.Dir == "" {
		log.Fatalf("go mod download %s: no directory in %s", modVersion, output)
	}
	return info.Dir, info.Version
}

// pkg is the parsed, non-test top level of one package directory.
type pkg struct {
	files []*ast.File
	// structs maps a struct type to its exported fields, in declaration order.
	structs map[string][]field
	// exported is every exported top-level name.
	exported map[string]bool
	// aliases maps an exported alias to the import path and name it stands for.
	aliases map[string][2]string
	// interfaces is every exported interface type, which in a types package
	// is a union.
	interfaces map[string]bool
}

type field struct {
	name string
	typ  ast.Expr
	// sensitive is v1's `sensitive:"true"` tag, which v2 dropped.
	sensitive bool
}

func parseDir(dir string) *pkg {
	p := &pkg{structs: map[string][]field{}, exported: map[string]bool{}, aliases: map[string][2]string{}, interfaces: map[string]bool{}}
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			log.Fatal(err)
		}
		p.files = append(p.files, f)
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.IsExported() {
					p.exported[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if !s.Name.IsExported() {
							continue
						}
						p.exported[s.Name.Name] = true
						switch t := s.Type.(type) {
						case *ast.StructType:
							p.structs[s.Name.Name] = fieldsOf(t)
						case *ast.InterfaceType:
							p.interfaces[s.Name.Name] = true
						case *ast.SelectorExpr:
							if s.Assign.IsValid() {
								if path := importPath(f, t.X.(*ast.Ident).Name); path != "" {
									p.aliases[s.Name.Name] = [2]string{path, t.Sel.Name}
								}
							}
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.IsExported() {
								p.exported[n.Name] = true
							}
						}
					}
				}
			}
		}
	}
	return p
}

// importPath finds the path a file imports under a local name.
func importPath(f *ast.File, local string) string {
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		name := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name == local {
			return path
		}
	}
	return ""
}

func fieldsOf(st *ast.StructType) []field {
	fs := []field{}
	for _, f := range st.Fields.List {
		sensitive := false
		if f.Tag != nil {
			tag, _ := strconv.Unquote(f.Tag.Value)
			sensitive = reflect.StructTag(tag).Get("sensitive") == "true"
		}
		for _, n := range f.Names {
			if n.IsExported() {
				fs = append(fs, field{n.Name, f.Type, sensitive})
			}
		}
	}
	return fs
}

var basics = map[string]bool{
	"string": true, "bool": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "float32": true, "float64": true,
	"byte": true, "rune": true,
}

// named returns the name of the type an expression refers to, whatever its
// package, or "".
func named(e ast.Expr) string {
	if t, ok := e.(*ast.SelectorExpr); ok {
		return t.Sel.Name
	}
	return typesName(e)
}

// typesName returns the types-package name an expression refers to, or "".
func typesName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		if !basics[t.Name] && t.IsExported() {
			return t.Name
		}
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok && x.Name == "types" {
			return t.Sel.Name
		}
	}
	return ""
}

func basic(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok && basics[id.Name] {
		return id.Name
	}
	return ""
}

func pointee(e ast.Expr) (ast.Expr, bool) {
	if s, ok := e.(*ast.StarExpr); ok {
		return s.X, true
	}
	return nil, false
}

// manifest holds the records of one service, keyed by kind.
type manifest map[byte][]string

func (m manifest) add(kind byte, format string, args ...any) {
	m[kind] = append(m[kind], fmt.Sprintf(format, args...))
}

// kindOrder is the order the existing data files group their records in.
const kindOrder = "CSOXIRVQPDWLFNTEMHYUKG"

func generate(v1Dir, v2Dir string) string {
	v1 := parseDir(v1Dir)
	v2 := parseDir(v2Dir)
	v2Types := parseDir(filepath.Join(v2Dir, "types"))
	resolveAliases(v2, v2Dir)
	m := manifest{}

	if c := v1ClientType(v1); c != "" {
		m.add('C', "%s", c)
	}
	for name := range v2.exported {
		m.add('S', "%s", name)
	}
	for _, op := range operations(v2) {
		m.add('O', "%s", op)
	}
	// v2 made these shapes unions: interfaces whose members hold one field.
	for name := range v2Types.interfaces {
		m.add('U', "%s", name)
	}
	for name, value := range errCodes(v1) {
		m.add('X', "%s %s", name, value)
	}
	for op, tokens := range paginatorTokens(v2) {
		m.add('G', "%s %s %s", op, tokens[0], tokens[1])
	}
	for name, timing := range waiterTimings(v1Dir) {
		m.add('Y', "%s %d %d", name, timing[0], timing[1])
	}

	enums, consts := enumsOf(v2Types)
	for name := range v2Types.exported {
		if consts[name] == "" {
			m.add('T', "%s", name)
		}
	}
	for c, enum := range consts {
		m.add('E', "%s %s", c, enum)
	}

	// A v1 shape v2 renamed — autoscaling's InstanceDetails became
	// AutoScalingInstanceDetails — is found by the v2 shape ending in its name
	// with the same fields. The recipe infers shapes from v1 code, so every
	// record about one keeps the v1 name, and only the M record names the v2.
	renamed := renames(v1.structs, v2Types.structs)
	for v1Name, v2Name := range renamed {
		m.add('M', "%s %s", v1Name, v2Name)
	}
	v1Name := map[string]string{}
	for from, to := range renamed {
		v1Name[to] = from
	}
	asV1 := func(name string) string {
		if from, ok := v1Name[name]; ok {
			return from
		}
		return name
	}

	// Records describing a v2 field on its own. A shape both packages declare —
	// Glue's TestConnectionInput — is described by its types-package fields, as
	// the v1 shape a reference resolves to is the modelled one.
	v2Structs := map[string][]field{}
	for name, fs := range v2.structs {
		v2Structs[name] = fs
	}
	for name, fs := range v2Types.structs {
		v2Structs[name] = fs
	}
	nonEnum := map[string]bool{}
	for shape, fs := range v2.structs {
		if _, shadowed := v2Types.structs[shape]; shadowed {
			for _, f := range fs {
				if enumOfField(f.typ, enums) == "" {
					nonEnum[f.name] = true
				}
			}
		}
	}
	for shape, fs := range v2Structs {
		for _, f := range fs {
			key := asV1(shape) + "." + f.name
			enum := enumOfField(f.typ, enums)
			if enum != "" {
				m.add('F', "%s %s", key, enum)
			} else {
				nonEnum[f.name] = true
			}
			if x, ok := pointee(f.typ); ok {
				if b := basic(x); b != "" {
					m.add('V', "%s %s", key, b)
				} else if n := typesName(x); n != "" && v2Types.structs[n] != nil {
					m.add('R', "%s %s", key, asV1(n))
				}
			}
		}
	}
	for name := range nonEnum {
		m.add('N', "%s", name)
	}

	// Records describing how a field v1 and v2 share changed.
	for shape, v1Fields := range v1.structs {
		v2Fields, ok := v2Structs[shape]
		if to, isRenamed := renamed[shape]; isRenamed {
			v2Fields, ok = v2Structs[to], true
		}
		if !ok {
			continue
		}
		byName := map[string]ast.Expr{}
		for _, f := range v2Fields {
			byName[f.name] = f.typ
		}
		for _, f := range v1Fields {
			t2, ok := byName[f.name]
			if !ok {
				continue
			}
			if f.sensitive {
				m.add('H', "%s.%s", shape, f.name)
			}
			compareField(m, shape+"."+f.name, f.typ, t2, v2Types.structs, asV1)
		}
	}

	validations(m, v1, v2Structs, v2Types, enums, renamed, asV1)

	var b strings.Builder
	for i := 0; i < len(kindOrder); i++ {
		lines := m[kindOrder[i]]
		sort.Strings(lines)
		for _, l := range lines {
			b.WriteByte(kindOrder[i])
			b.WriteByte(' ')
			b.WriteString(l)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// renames pairs each v1 struct v2 has no struct of the same name for with the
// one v2 types struct, new in v2, whose name ends in it and whose fields
// include all of its own.
func renames(v1Structs, v2Structs map[string][]field) map[string]string {
	out := map[string]string{}
	for name, fs := range v1Structs {
		if _, kept := v2Structs[name]; kept {
			continue
		}
		var match string
		for candidate, cfs := range v2Structs {
			if _, inV1 := v1Structs[candidate]; inV1 {
				continue
			}
			if strings.HasSuffix(candidate, name) && candidate != name && hasFields(cfs, fs) {
				if match != "" {
					match = ""
					break
				}
				match = candidate
			}
		}
		if match != "" {
			out[name] = match
		}
	}
	return out
}

// hasFields reports whether every field of want is a field of have.
func hasFields(have, want []field) bool {
	names := map[string]bool{}
	for _, f := range have {
		names[f.name] = true
	}
	if len(want) == 0 {
		return false
	}
	for _, f := range want {
		if !names[f.name] {
			return false
		}
	}
	return true
}

// resolveAliases gives an alias of a struct from the module's own internal
// packages that struct's fields: EndpointResolverOptions is one.
func resolveAliases(p *pkg, dir string) {
	gomod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		log.Fatal(err)
	}
	module := ""
	for _, l := range strings.Split(string(gomod), "\n") {
		if m, ok := strings.CutPrefix(l, "module "); ok {
			module = strings.TrimSpace(m)
		}
	}
	for alias, target := range p.aliases {
		rel, ok := strings.CutPrefix(target[0], module+"/")
		if !ok {
			continue
		}
		if fs, ok := parseDir(filepath.Join(dir, rel)).structs[target[1]]; ok {
			p.structs[alias] = fs
		}
	}
}

// compareField records how v2 retyped a field v1 also has. structs are the v2
// types package's structs: a map whose values became enums or unions is not one
// a value conversion reaches.
func compareField(m manifest, key string, t1, t2 ast.Expr, structs map[string][]field, asV1 func(string) string) {
	switch x1 := t1.(type) {
	case *ast.StarExpr:
		b1 := basic(x1.X)
		if b1 == "" {
			return
		}
		if x2, ok := pointee(t2); ok {
			if b2 := basic(x2); b2 != "" && b2 != b1 {
				m.add('I', "%s %s %s", key, b1, b2)
			}
			return
		}
		if b2 := basic(t2); b2 == b1 {
			m.add('D', "%s %s", key, b1)
		} else if b2 != "" {
			m.add('W', "%s %s %s", key, b1, b2)
		}
	case *ast.ArrayType:
		if _, ok := pointee(x1.Elt); !ok {
			return
		}
		a2, ok := t2.(*ast.ArrayType)
		if !ok {
			return
		}
		if _, ok := pointee(a2.Elt); ok {
			return
		}
		elem := asV1(named(a2.Elt))
		if elem == "" {
			elem = "-"
		}
		m.add('L', "%s %s", key, elem)
	case *ast.MapType:
		m2, ok := t2.(*ast.MapType)
		if !ok {
			return
		}
		if _, ok := pointee(x1.Value); ok {
			if basic(m2.Value) != "" || structs[typesName(m2.Value)] != nil {
				m.add('Q', "%s", key)
			}
			return
		}
		if a1, ok := x1.Value.(*ast.ArrayType); ok {
			if _, ok := pointee(a1.Elt); !ok {
				return
			}
			if a2, ok := m2.Value.(*ast.ArrayType); ok {
				if _, ok := pointee(a2.Elt); !ok {
					m.add('P', "%s", key)
				}
			}
		}
	}
}

// enumOfField names the enum a field carries, alone or as a slice element.
func enumOfField(t ast.Expr, enums map[string]bool) string {
	if a, ok := t.(*ast.ArrayType); ok {
		t = a.Elt
	}
	if n := typesName(t); enums[n] {
		return n
	}
	return ""
}

// enumsOf finds the string types of the types package and the constants of each.
func enumsOf(p *pkg) (enums map[string]bool, consts map[string]string) {
	enums, consts = map[string]bool{}, map[string]string{}
	for _, f := range p.files {
		for _, decl := range f.Decls {
			d, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if id, ok := s.Type.(*ast.Ident); ok && id.Name == "string" && s.Name.IsExported() {
						enums[s.Name.Name] = true
					}
				case *ast.ValueSpec:
					if d.Tok != token.CONST || s.Type == nil {
						continue
					}
					for _, n := range s.Names {
						if n.IsExported() {
							consts[n.Name] = types.ExprString(s.Type)
						}
					}
				}
			}
		}
	}
	for c, e := range consts {
		if !enums[e] {
			delete(consts, c)
		}
	}
	return enums, consts
}

// v1ClientType is the struct embedding *client.Client.
func v1ClientType(p *pkg) string {
	for _, f := range p.files {
		for _, decl := range f.Decls {
			d, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range d.Specs {
				s, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := s.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, f := range st.Fields.List {
					if len(f.Names) == 0 && types.ExprString(f.Type) == "*client.Client" {
						return s.Name.Name
					}
				}
			}
		}
	}
	return ""
}

// operations lists the v2 client methods taking an <Op>Input.
func operations(p *pkg) []string {
	var ops []string
	for _, f := range p.files {
		for _, decl := range f.Decls {
			d, ok := decl.(*ast.FuncDecl)
			if !ok || d.Recv == nil || !d.Name.IsExported() || types.ExprString(d.Recv.List[0].Type) != "*Client" {
				continue
			}
			params := d.Type.Params.List
			if len(params) >= 2 && types.ExprString(params[1].Type) == "*"+d.Name.Name+"Input" {
				ops = append(ops, d.Name.Name)
			}
		}
	}
	return ops
}

// waiterPattern reads a v1 waiter's name, attempts and constant delay.
var waiterPattern = regexp.MustCompile(`Name:\s+"WaitUntil(\w+)",\s+MaxAttempts:\s+(\d+),\s+Delay:\s+request\.ConstantWaiterDelay\((\d+) \* time\.Second\)`)

// paginatorTokens returns, per operation, the input field each v2 paginator
// sends its token in and the output field it reads the next one from. A
// paginator reading its token any other way is left out.
func paginatorTokens(p *pkg) map[string][2]string {
	out := map[string][2]string{}
	for _, f := range p.files {
		for _, decl := range f.Decls {
			fd, isFunc := decl.(*ast.FuncDecl)
			if !isFunc || fd.Name.Name != "NextPage" || fd.Recv == nil || len(fd.Recv.List) != 1 || fd.Body == nil {
				continue
			}
			star, isStar := fd.Recv.List[0].Type.(*ast.StarExpr)
			if !isStar {
				continue
			}
			recv, isIdent := star.X.(*ast.Ident)
			if !isIdent {
				continue
			}
			op, isPaginator := strings.CutSuffix(recv.Name, "Paginator")
			if !isPaginator {
				continue
			}
			var in, outField []string
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				as, isAssign := n.(*ast.AssignStmt)
				if !isAssign || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
					return true
				}
				if selectorOf(as.Rhs[0], "p", "nextToken") {
					if field, ok := fieldOf(as.Lhs[0], "params"); ok {
						in = append(in, field)
					}
				}
				if selectorOf(as.Lhs[0], "p", "nextToken") {
					if field, ok := fieldOf(as.Rhs[0], "result"); ok {
						outField = append(outField, field)
					}
				}
				return true
			})
			if len(in) == 1 && len(outField) == 1 {
				out[op] = [2]string{in[0], outField[0]}
			}
		}
	}
	return out
}

// selectorOf reports whether e is x.sel.
func selectorOf(e ast.Expr, x, sel string) bool {
	s, isSel := e.(*ast.SelectorExpr)
	if !isSel || s.Sel.Name != sel {
		return false
	}
	id, isIdent := s.X.(*ast.Ident)
	return isIdent && id.Name == x
}

// fieldOf returns the field e selects directly off x.
func fieldOf(e ast.Expr, x string) (string, bool) {
	s, isSel := e.(*ast.SelectorExpr)
	if !isSel {
		return "", false
	}
	if id, isIdent := s.X.(*ast.Ident); !isIdent || id.Name != x {
		return "", false
	}
	return s.Sel.Name, true
}

// waiterTimings maps each v1 waiter to its delay in seconds and its attempts,
// which bounded a wait the way v2's maximum duration does.
func waiterTimings(v1Dir string) map[string][2]int {
	out := map[string][2]int{}
	src, err := os.ReadFile(filepath.Join(v1Dir, "waiters.go"))
	if err != nil {
		return out
	}
	for _, m := range waiterPattern.FindAllStringSubmatch(string(src), -1) {
		attempts, _ := strconv.Atoi(m[2])
		delay, _ := strconv.Atoi(m[3])
		out[m[1]] = [2]int{delay, attempts}
	}
	return out
}

// errCodes maps each v1 ErrCode constant to the code it holds.
func errCodes(p *pkg) map[string]string {
	codes := map[string]string{}
	for _, f := range p.files {
		for _, decl := range f.Decls {
			d, ok := decl.(*ast.GenDecl)
			if !ok || d.Tok != token.CONST {
				continue
			}
			for _, spec := range d.Specs {
				s := spec.(*ast.ValueSpec)
				for i, n := range s.Names {
					if strings.HasPrefix(n.Name, "ErrCode") && i < len(s.Values) {
						if lit, ok := s.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							codes[n.Name] = unquote(lit)
						}
					}
				}
			}
		}
	}
	return codes
}

// diff reports the lines only one side holds, which for sorted record sets is
// what a reviewer needs.
func diff(want, got string) string {
	count := func(s string) map[string]int {
		c := map[string]int{}
		for _, l := range strings.Split(s, "\n") {
			if l != "" {
				c[l]++
			}
		}
		return c
	}
	w, g := count(want), count(got)
	var b bytes.Buffer
	var lines []string
	for l, n := range w {
		for i := g[l]; i < n; i++ {
			lines = append(lines, "- "+l)
		}
	}
	for l, n := range g {
		for i := w[l]; i < n; i++ {
			lines = append(lines, "+ "+l)
		}
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i][2:] != lines[j][2:] {
			return lines[i][2:] < lines[j][2:]
		}
		return lines[i] < lines[j]
	})
	for _, l := range lines {
		b.WriteString(l + "\n")
	}
	if b.Len() == 0 && want != got {
		b.WriteString("  (same lines, different order)\n")
	}
	return b.String()
}

// validations records each v1 Validate method as its rules, in v1's order, with
// the form the field takes in v2: `K Shape NNN Field rule [arg] form`. A shape
// whose rules do not all carry over is recorded `K Shape 000 - unsupported`.
func validations(m manifest, v1 *pkg, v2Structs map[string][]field, v2Types *pkg, enums map[string]bool, renamed map[string]string, asV1 func(string) string) {
	for _, f := range v1.files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Name.Name != "Validate" || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			star, ok := fd.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			shape := star.X.(*ast.Ident).Name
			v2Name := shape
			if to, isRenamed := renamed[shape]; isRenamed {
				v2Name = to
			}
			v2Fields := map[string]ast.Expr{}
			for _, fl := range v2Structs[v2Name] {
				v2Fields[fl.name] = fl.typ
			}
			var rules []string
			supported := v2Structs[v2Name] != nil
			for _, st := range fd.Body.List {
				if !supported {
					break
				}
				rule, skip, ok := validationRule(st, v2Fields, v2Types, enums, asV1)
				if skip {
					continue
				}
				if !ok {
					supported = false
					break
				}
				rules = append(rules, rule)
			}
			if !supported {
				m.add('K', "%s 000 - unsupported", shape)
				continue
			}
			for i, rule := range rules {
				m.add('K', "%s %03d %s", shape, i+1, rule)
			}
			if len(rules) == 0 {
				m.add('K', "%s 000 - none", shape)
			}
		}
	}
}

var (
	requiredRule       = regexp.MustCompile(`^if s\.(\w+) == nil \{\s*invalidParams\.Add\(request\.NewErrParamRequired\("(\w+)"\)\)\s*\}$`)
	minLenRule         = regexp.MustCompile(`^if s\.(\w+) != nil && len\((\*?)s\.(\w+)\) < (\d+) \{\s*invalidParams\.Add\(request\.NewErrParamMinLen\("(\w+)", (\d+)\)\)\s*\}$`)
	minValueRule       = regexp.MustCompile(`^if s\.(\w+) != nil && \*s\.(\w+) < (\S+) \{\s*invalidParams\.Add\(request\.NewErrParamMinValue\("(\w+)", (\S+)\)\)\s*\}$`)
	nestedRule         = regexp.MustCompile(`^if s\.(\w+) != nil \{\s*if err := s\.(\w+)\.Validate\(\); err != nil \{\s*invalidParams\.AddNested\("(\w+)", err\.\(request\.ErrInvalidParams\)\)\s*\}\s*\}$`)
	eachRule           = regexp.MustCompile(`^if s\.(\w+) != nil \{\s*for i, v := range s\.(\w+) \{\s*if v == nil \{\s*continue\s*\}\s*if err := v\.Validate\(\); err != nil \{\s*invalidParams\.AddNested\(fmt\.Sprintf\("%s\[%v\]", "(\w+)", i\), err\.\(request\.ErrInvalidParams\)\)\s*\}\s*\}\s*\}$`)
	validationScaffold = regexp.MustCompile(`^(invalidParams := request\.ErrInvalidParams\{Context: "\w+"\}|if invalidParams\.Len\(\) > 0 \{\s*return invalidParams\s*\}|return nil)$`)
)

// validationRule reads one statement of a v1 Validate method as a rule on a
// field, given that field's v2 type. skip is set for the statements every
// Validate opens and closes with.
func validationRule(st ast.Stmt, v2Fields map[string]ast.Expr, v2Types *pkg, enums map[string]bool, asV1 func(string) string) (rule string, skip, ok bool) {
	var b bytes.Buffer
	if err := printer.Fprint(&b, token.NewFileSet(), st); err != nil {
		return "", false, false
	}
	src := b.String()
	if validationScaffold.MatchString(src) {
		return "", true, true
	}
	same := func(names ...string) bool {
		for _, n := range names[1:] {
			if n != names[0] {
				return false
			}
		}
		return true
	}
	var field string
	switch {
	case requiredRule.MatchString(src):
		g := requiredRule.FindStringSubmatch(src)
		if field = g[1]; !same(g[1], g[2]) {
			return "", false, false
		}
		form := requiredForm(v2Fields[field], v2Types, enums)
		return field + " req " + form, false, form != ""
	case minLenRule.MatchString(src):
		g := minLenRule.FindStringSubmatch(src)
		if field = g[1]; !same(g[1], g[3], g[5]) || g[4] != g[6] {
			return "", false, false
		}
		form := minLenForm(v2Fields[field], g[2] == "*", enums)
		return field + " len " + g[4] + " " + form, false, form != ""
	case minValueRule.MatchString(src):
		g := minValueRule.FindStringSubmatch(src)
		if field = g[1]; !same(g[1], g[2], g[4]) || g[3] != g[5] {
			return "", false, false
		}
		form := ""
		if x, isPtr := pointee(v2Fields[field]); isPtr && basic(x) != "" {
			form = "ptr"
		} else if basic(v2Fields[field]) != "" {
			form = "value"
		}
		return field + " min " + g[3] + " " + form, false, form != ""
	case nestedRule.MatchString(src):
		g := nestedRule.FindStringSubmatch(src)
		if field = g[1]; !same(g[1], g[2], g[3]) {
			return "", false, false
		}
		x, isPtr := pointee(v2Fields[field])
		elem := typesName(x)
		if !isPtr || v2Types.structs[elem] == nil {
			return "", false, false
		}
		return field + " nest " + asV1(elem), false, true
	case eachRule.MatchString(src):
		g := eachRule.FindStringSubmatch(src)
		if field = g[1]; !same(g[1], g[2], g[3]) {
			return "", false, false
		}
		var elemType ast.Expr
		form := ""
		switch t := v2Fields[field].(type) {
		case *ast.ArrayType:
			elemType, form = t.Elt, "slice"
		case *ast.MapType:
			elemType, form = t.Value, "map"
		}
		if x, isPtr := pointee(elemType); isPtr {
			elemType, form = x, form+"p"
		}
		elem := typesName(elemType)
		if form == "" || v2Types.structs[elem] == nil {
			return "", false, false
		}
		return field + " each " + asV1(elem) + " " + form, false, true
	}
	return "", false, false
}

// requiredForm is how a v2 field shows it is unset: nil, the empty enum, or
// not at all for a value v2 always holds.
func requiredForm(t ast.Expr, v2Types *pkg, enums map[string]bool) string {
	switch x := t.(type) {
	case nil:
		return ""
	case *ast.StarExpr, *ast.ArrayType, *ast.MapType, *ast.InterfaceType:
		return "nil"
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok && id.Name == "document" && x.Sel.Name == "Interface" {
			return "nil"
		}
	}
	if n := typesName(t); enums[n] {
		return "empty"
	} else if v2Types.interfaces[n] {
		return "nil"
	}
	if basic(t) != "" {
		return "none"
	}
	return ""
}

// minLenForm is how a v2 field's length is read: through the pointer v1 also
// had, off an enum v2 holds by value, or off a list, map or blob.
func minLenForm(t ast.Expr, v1Pointer bool, enums map[string]bool) string {
	if !v1Pointer {
		switch t.(type) {
		case *ast.ArrayType, *ast.MapType:
			return "container"
		}
		return ""
	}
	if x, isPtr := pointee(t); isPtr && basic(x) == "string" {
		return "ptr"
	}
	if enums[typesName(t)] {
		return "enum"
	}
	if basic(t) == "string" {
		return "value"
	}
	return ""
}
