/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// isInitWrappedIf reports whether the If at the cursor is the inner statement of
// a golang.StatementWithInit — i.e. it carried an `if init; cond` init clause.
func isInitWrappedIf(c *visitor.Cursor) bool {
	parent := c.Parent()
	if parent == nil {
		return false
	}
	_, ok := parent.Value().(*golang.StatementWithInit)
	return ok
}
