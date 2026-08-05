/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Flags racy channel length checks like `len(ch) == 0`, since the length can
// change between the check and the next send or receive.
type AvoidChannelLenCheck struct {
	recipe.Base
}

func (r *AvoidChannelLenCheck) Name() string {
	return "org.openrewrite.golang.codequality.AvoidChannelLenCheck"
}
func (r *AvoidChannelLenCheck) DisplayName() string { return "Avoid channel length check" }
func (r *AvoidChannelLenCheck) Description() string {
	return "Find comparisons on channel length such as `len(ch) == 0`. These are almost always racy because the length can change between the check and the next operation."
}
func (r *AvoidChannelLenCheck) Tags() []string { return []string{"simplification", "concurrency"} }

func (r *AvoidChannelLenCheck) Editor() recipe.TreeVisitor {
	return visitor.Init(&findChannelLenCheckVisitor{})
}

type findChannelLenCheckVisitor struct {
	visitor.GoVisitor

	// Fully-qualified names of same-file types whose underlying type is a channel,
	// collected once per compilation unit before the body is visited.
	namedChannelTypes map[string]bool
}

func (v *findChannelLenCheckVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.namedChannelTypes = collectNamedChannelTypes(cu)
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *findChannelLenCheckVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	// Match patterns like: len(ch) == 0, len(ch) > 0, len(ch) != 0, etc.
	if !isComparisonOp(bin.Operator.Element) {
		return bin
	}

	if v.isChannelLenCall(bin.Left) || v.isChannelLenCall(bin.Right) {
		bin = bin.WithMarkers(
			java.MarkupWarn(bin.Markers, "channel length check is racy; the value can change between check and send/receive"),
		)
	}

	return bin
}

// Reports whether the expression is a `len` call whose argument is a channel,
// checking the type since `len` also accepts slices, maps, arrays, and strings.
func (v *findChannelLenCheckVisitor) isChannelLenCall(expr java.Expression) bool {
	mi, ok := expr.(*java.MethodInvocation)
	if !ok {
		return false
	}
	if mi.Select != nil || mi.Name.Name != "len" || len(mi.Arguments.Elements) != 1 {
		return false
	}
	return v.isChannelType(matcher.TypeOfExpression(mi.Arguments.Elements[0].Element))
}

// Reports whether the type is a channel, matching direct channels by FQN
// (`chan`, `chan<-`, `<-chan`) and same-file named channel types by name.
func (v *findChannelLenCheckVisitor) isChannelType(t java.JavaType) bool {
	switch fqn := matcher.GetFullyQualifiedName(t); fqn {
	case "chan", "chan<-", "<-chan":
		return true
	default:
		return v.namedChannelTypes[fqn]
	}
}

// Returns the FQNs of same-file types whose underlying type is a channel,
// following indirection through named types like `type D C`.
func collectNamedChannelTypes(cu *golang.CompilationUnit) map[string]bool {
	// Map each declared type's FQN to its definition expression.
	defs := map[string]java.Expression{}
	var record func(stmt java.Statement)
	record = func(stmt java.Statement) {
		td, ok := stmt.(*golang.TypeDecl)
		if !ok {
			return
		}
		if td.Specs != nil { // grouped `type ( ... )`
			for _, spec := range td.Specs.Elements {
				record(spec.Element)
			}
			return
		}
		if td.Name != nil && td.Definition != nil {
			if fqn := matcher.GetFullyQualifiedName(td.Name.Type); fqn != "" {
				defs[fqn] = td.Definition
			}
		}
	}
	for _, stmt := range cu.Statements {
		record(stmt.Element)
	}

	// Seed with types defined directly as a channel, then propagate through
	// named-type references to a fixpoint (`type D C` where C is a channel).
	channels := map[string]bool{}
	for fqn, defn := range defs {
		if _, ok := defn.(*golang.Channel); ok {
			channels[fqn] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for fqn, defn := range defs {
			if channels[fqn] {
				continue
			}
			if id, ok := defn.(*java.Identifier); ok && channels[matcher.GetFullyQualifiedName(id.Type)] {
				channels[fqn] = true
				changed = true
			}
		}
	}
	return channels
}

// isComparisonOp returns true for ==, !=, <, >, <=, >=.
func isComparisonOp(op java.BinaryOperator) bool {
	switch op {
	case java.Equal, java.NotEqual,
		java.LessThan, java.GreaterThan,
		java.LessThanOrEqual, java.GreaterThanOrEqual:
		return true
	default:
		return false
	}
}
