// Package architecture holds repository-wide architecture gate tests.
package architecture_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const (
	modulePath     = "github.com/flowline-io/flowbot"
	internalPrefix = modulePath + "/internal/"
)

// pkgInternalImportAllowlist is empty after L1–L4: no pkg package may import internal.
// Never grow this list without an explicit architecture decision.
var pkgInternalImportAllowlist = map[string]struct{}{}

func TestPkgMustNotImportInternal(t *testing.T) {
	t.Parallel()

	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedImports | packages.NeedFiles,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, modulePath+"/pkg/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		t.Fatal("packages.Load reported errors")
	}

	var violations []string
	for _, pkg := range pkgs {
		if pkg.PkgPath == "" || !strings.HasPrefix(pkg.PkgPath, modulePath+"/pkg/") {
			continue
		}
		// External test packages use the "_test" suffix; allowlist the base path.
		basePath := strings.TrimSuffix(pkg.PkgPath, "_test")
		if _, ok := pkgInternalImportAllowlist[basePath]; ok {
			continue
		}
		for importPath := range pkg.Imports {
			if strings.HasPrefix(importPath, internalPrefix) || importPath == modulePath+"/internal" {
				violations = append(violations, fmt.Sprintf("%s -> %s", pkg.PkgPath, importPath))
			}
		}
	}

	if len(violations) == 0 {
		return
	}
	slices.Sort(violations)
	t.Fatalf("pkg packages must not import internal (except allowlisted migration packages):\n  %s",
		strings.Join(violations, "\n  "))
}
