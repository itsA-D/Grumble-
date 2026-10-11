// Command checkimports enforces the module's import boundaries
// (ARCHITECTURE.md section 2). domain and scoring stay pure; the
// transport → service → domain direction is never reversed; and adapters
// (store, platform) are only imported by the composition root.
//
// Run from the module root: `go run ./scripts/checkimports`.
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// forbidden maps a package group (the first path segment under internal/) to
// the module-relative prefixes it must not import. A group may import within
// itself unless listed (domain sub-packages share value types and rules).
var forbidden = map[string][]string{
	"domain":    {"internal/service", "internal/transport", "internal/store", "internal/platform"},
	"scoring":   {"internal/"}, // pure: no internal imports at all, no I/O
	"service":   {"internal/store", "internal/transport", "internal/platform"},
	"transport": {"internal/store"},
	"store":     {"internal/service", "internal/transport"},
	"platform":  {"internal/service", "internal/transport", "internal/store"},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "checkimports:", err)
		os.Exit(1)
	}
	fmt.Println("checkimports: ok")
}

func run() error {
	module, err := modulePath("go.mod")
	if err != nil {
		return err
	}

	pkgs, err := collect("internal")
	if err != nil {
		return err
	}

	var violations []string
	for _, p := range sortedPackages(pkgs) {
		group := groupOf(p.rel)
		rules, ok := forbidden[group]
		if !ok {
			continue // e.g. internal/app, the composition root
		}
		for _, imp := range p.imports {
			for _, bad := range rules {
				if strings.HasPrefix(imp, module+"/"+bad) {
					violations = append(violations,
						fmt.Sprintf("%s must not import %s", p.rel, imp))
				}
			}
		}
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		return fmt.Errorf("import boundary violations:\n  %s", strings.Join(violations, "\n  "))
	}
	return nil
}

func modulePath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("module path not found in %s", path)
}

type pkg struct {
	rel     string
	imports []string
}

func collect(root string) (map[string]*pkg, error) {
	seen := make(map[string]map[string]bool)
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		rel := filepath.ToSlash(filepath.Dir(path))
		if seen[rel] == nil {
			seen[rel] = make(map[string]bool)
		}
		for _, spec := range f.Imports {
			seen[rel][strings.Trim(spec.Path.Value, `"`)] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	pkgs := make(map[string]*pkg, len(seen))
	for rel, set := range seen {
		imports := make([]string, 0, len(set))
		for imp := range set {
			imports = append(imports, imp)
		}
		pkgs[rel] = &pkg{rel: rel, imports: imports}
	}
	return pkgs, nil
}

func groupOf(rel string) string {
	rest, ok := strings.CutPrefix(rel, "internal/")
	if !ok {
		return ""
	}
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		return rest[:i]
	}
	return rest
}

func sortedPackages(pkgs map[string]*pkg) []*pkg {
	out := make([]*pkg, 0, len(pkgs))
	for _, p := range pkgs {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out
}
