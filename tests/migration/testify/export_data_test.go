/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify/testifyexportdata"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/exportdata"
	"github.com/stretchr/testify/require"
)

func TestTestifyExportDataIsReadable(t *testing.T) {
	// Verify names an unreadable blob directly, where the attribution sweep in
	// tests/ reports it as a missing type on every emitted assertion.
	// See CLAUDE.md: Type Attribution for how to regenerate.
	require.NoError(t, exportdata.Verify(testifyexportdata.FS, testifyexportdata.Paths...))
}
