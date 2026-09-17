/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
)

func graphWithStatus(mod test.SourceSpec, resolved []golang.GoResolvedDependency, pkgs []golang.GoPackageModule, status golang.GoResolutionStatus) test.SourceSpec {
	mod = test.GoModGraph(mod, resolved, pkgs)
	for i, m := range mod.Markers {
		if mrr, ok := m.(golang.GoResolutionResult); ok {
			mrr.ResolutionStatus = status
			mod.Markers[i] = mrr
			break
		}
	}
	return mod
}

func resolvedGraph(mod test.SourceSpec, resolved []golang.GoResolvedDependency, pkgs []golang.GoPackageModule) test.SourceSpec {
	return graphWithStatus(mod, resolved, pkgs, golang.GoResolutionResolved)
}
