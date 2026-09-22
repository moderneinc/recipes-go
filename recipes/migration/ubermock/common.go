/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

// Package ubermock migrates github.com/golang/mock, archived by Google in 2023,
// to its maintained fork go.uber.org/mock.
//
// The fork's gomock API is a strict superset of the original's — every name
// keeps its signature, with `Satisfied`, `Cond`, `Regex`, `AnyOf` and the
// ControllerOption variadic added — so the source migration is a path swap
// rather than an API rewrite.
package ubermock

import (
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
)

const (
	oldModule = "github.com/golang/mock"
	newModule = "go.uber.org/mock"

	oldGomock = oldModule + "/gomock"
	newGomock = newModule + "/gomock"

	// The fork's current release. A module already requiring go.uber.org/mock
	// keeps whatever version it pinned.
	newVersion = "v0.6.0"
)

// swapRules moves the whole module, so gomock, mockgen and mockgen/model all
// travel with it.
var swapRules = []pathswap.Rule{{Old: oldModule, New: newModule}}
