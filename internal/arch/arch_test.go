// Package arch enforces TeamWERK's dependency-layering invariants as a test.
//
// This is the project's "ArchUnit" equivalent (Harness-Engineering, Säule 2 —
// Architectural Constraints): it locks in the clean structure mechanically so
// agent- or human-written code cannot quietly erode it. It uses only the
// standard library (go/parser) — no extra dependency.
//
// Layering model:
//
//	COMPOSITION (app)        ← darf alles verdrahten
//	    │
//	DOMAIN (members, …)      ← importiert nur FOUNDATION, nie andere DOMAIN
//	    │
//	FOUNDATION (auth, db, …) ← importiert nie DOMAIN/COMPOSITION
//
// A new package under internal/ MUST be added to one of the maps below;
// otherwise TestArchitecture_AllPackagesClassified fails on purpose. That
// forced decision is the entropy brake.
package arch

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePrefix = "github.com/teamstuttgart/teamwerk/internal/"

// foundation = shared lower layer. Importable by everyone; may not import DOMAIN/COMPOSITION.
var foundation = map[string]bool{
	"auth": true, "config": true, "db": true, "hub": true, "mailer": true,
	"notify": true, "notifications": true, "push": true, "policy": true,
	"sepa": true, "upload": true, "files": true, "scheduler": true, "permissions": true,
	"health": true, "metrics": true, "crypto": true, "timez": true,
}

// domain = one HTTP handler package per business domain. May import FOUNDATION only.
var domain = map[string]bool{
	"members": true, "duties": true, "games": true, "kader": true, "teams": true,
	"trainings": true, "venues": true, "beitragslauf": true, "beitragssaetze": true,
	"chat": true, "carpooling": true, "absences": true, "dashboard": true,
	"calendar": true, "stammvereine": true, "attendance": true,
}

// composition = the wiring root. Allowed to import any internal package.
var composition = map[string]bool{"app": true}

// exempt = packages excluded from layering rules. testutil builds the full
// server for tests and therefore legitimately imports everything.
var exempt = map[string]bool{"testutil": true, "arch": true}

func classify(pkg string) string {
	switch {
	case foundation[pkg]:
		return "foundation"
	case domain[pkg]:
		return "domain"
	case composition[pkg]:
		return "composition"
	case exempt[pkg]:
		return "exempt"
	default:
		return ""
	}
}

// internalImports maps each top-level internal package to the set of other
// internal packages it imports (production code only; _test.go excluded).
func internalImports(t *testing.T) map[string]map[string]bool {
	t.Helper()
	root := internalRoot(t)
	result := map[string]map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		pkg := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0] // top-level segment under internal/

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		if result[pkg] == nil {
			result[pkg] = map[string]bool{}
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(p, modulePrefix) {
				continue
			}
			dep := strings.SplitN(strings.TrimPrefix(p, modulePrefix), "/", 2)[0]
			if dep != pkg {
				result[pkg][dep] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking internal/: %v", err)
	}
	return result
}

func internalRoot(t *testing.T) string {
	t.Helper()
	// This test lives in internal/arch/, so internal/ is one level up.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Dir(wd)
}

func TestArchitecture_AllPackagesClassified(t *testing.T) {
	for pkg := range internalImports(t) {
		if classify(pkg) == "" {
			t.Errorf("package internal/%s is not classified in arch_test.go — "+
				"add it to foundation, domain, composition, or exempt", pkg)
		}
	}
}

func TestArchitecture_NoCrossDomainImports(t *testing.T) {
	for pkg, imps := range internalImports(t) {
		if classify(pkg) != "domain" {
			continue
		}
		for dep := range imps {
			if classify(dep) == "domain" {
				t.Errorf("domain package internal/%s imports domain package internal/%s — "+
					"domain packages must not depend on each other (extract shared logic into a FOUNDATION package)", pkg, dep)
			}
		}
	}
}

func TestArchitecture_FoundationDoesNotImportDomain(t *testing.T) {
	for pkg, imps := range internalImports(t) {
		if classify(pkg) != "foundation" {
			continue
		}
		for dep := range imps {
			if c := classify(dep); c == "domain" || c == "composition" {
				t.Errorf("foundation package internal/%s imports %s package internal/%s — "+
					"the foundation layer must not depend on higher layers", pkg, c, dep)
			}
		}
	}
}

func TestArchitecture_DomainDoesNotImportComposition(t *testing.T) {
	for pkg, imps := range internalImports(t) {
		if classify(pkg) != "domain" {
			continue
		}
		for dep := range imps {
			if classify(dep) == "composition" {
				t.Errorf("domain package internal/%s imports composition package internal/%s — "+
					"only main.go and internal/app may wire the composition root", pkg, dep)
			}
		}
	}
}
