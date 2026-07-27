// xrlint is a small suite of repo-specific static checks, each of which
// can be individually enabled/disabled via flags (both run by default):
//
//   - "nilcheck" (--nilcheck): finds callers that compare the result of
//     a function/method whose return type is the empty interface
//     ("any"/"interface{}") directly against nil using "== nil"/"!=
//     nil", instead of using common.IsNil().
//
//   - "unused" (--unused): finds package-level funcs/methods that are
//     never referenced anywhere in the scanned packages (a simple
//     "orphaned function" check).
//
//   - "gofmt" (--gofmt): finds files that aren't gofmt-formatted, by
//     shelling out to the real "gofmt -l" rather than reimplementing
//     its formatting logic.
//
// More checks are expected to be added here over time.
//
// # nilcheck details
//
// Comparing a typed nil (e.g. a nil *SomeStruct) stored in an "any" using
// "== nil" is a classic Go footgun: the interface value itself is non-nil
// (it has a concrete type, just a nil value), so "== nil" is always false
// even though the underlying pointer/slice/map/etc. is nil. This repo's
// common.IsNil() helper uses reflection to check the underlying value
// correctly, and is meant to be used everywhere an "any"-typed value needs
// a nil check.
//
// This tool builds full type information for the module (via go/types and
// golang.org/x/tools/go/packages) so it can precisely identify:
//   - every func/method declaration with an "any"/"interface{}" result,
//     tracked per return-slot (so e.g. (*XRError, bool, any) is only
//     flagged on its LAST slot, not the first)
//   - every call site of those functions, whether compared to nil inline
//     (`if f() == nil`) or via an intermediate variable
//     (`v := f(); if v == nil`)
//
// # unused details
//
// This check reports package-level func/method declarations that have no
// references anywhere in the scanned packages. It excludes:
//   - func main, func init (entry points, never called by name)
//   - Test*/Benchmark*/Example* funcs (invoked by the "go test" runtime
//     via reflection, not by name reference)
//   - methods whose receiver type genuinely satisfies an interface (any
//     interface declared in the scanned packages, plus a short list of
//     well-known stdlib interfaces: error, fmt.Stringer, sort.Interface,
//     net/http.Handler, io.Reader/Writer/Closer,
//     encoding/json.Marshaler/Unmarshaler) - checked precisely via
//     go/types.Implements(), not by name-matching, since such methods
//     are typically invoked implicitly through interface dispatch
//     rather than by direct reference
//
// Because this check can only see references within the scanned
// packages, an exported func used solely from a package outside that
// set (e.g. some other cmds/ tool not included in the default patterns)
// will be a false positive - this is why ./tests/... is included in the
// default patterns below, so test-only usages of otherwise-"unused"
// registry/common funcs are accounted for.
//
// # gofmt details
//
// This check runs "gofmt -l" (the actual gofmt binary found on PATH)
// against every Go source file in the scanned packages, rather than
// reimplementing gofmt's own formatting rules. Any file gofmt reports as
// needing reformatting is listed as a hit.
//
// Usage:
//
//	go run ./cmds/xrlint [--nilcheck] [--unused] [--gofmt] [packages...]
//
// With no package args it checks ./registry/... ./common/... ./cmds/...
// ./tests/... (the tmp/ directory is intentionally excluded - it's a
// stale/scratch package that doesn't currently build).
//
// Exits with a non-zero status if any enabled check finds something, so
// it can be wired into CI (see "make xrlint").
package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"
)

func isEmptyInterface(t types.Type) bool {
	iface, ok := t.Underlying().(*types.Interface)
	return ok && iface.NumMethods() == 0
}

func main() {
	var nilcheckEnabled, unusedEnabled, gofmtEnabled bool

	rootCmd := &cobra.Command{
		Use:   "xrlint [packages...]",
		Short: "Repo-specific static checks (nilcheck, unused funcs, gofmt)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args, nilcheckEnabled, unusedEnabled, gofmtEnabled)
		},
	}
	rootCmd.Flags().BoolVar(&nilcheckEnabled, "nilcheck", true,
		"check for '== nil'/'!= nil' comparisons on any-typed values")
	rootCmd.Flags().BoolVar(&unusedEnabled, "unused", true,
		"check for package-level funcs/methods that are never referenced")
	rootCmd.Flags().BoolVar(&gofmtEnabled, "gofmt", true,
		"check for files that aren't gofmt-formatted (via 'gofmt -l')")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(
	patterns []string,
	nilcheckEnabled bool,
	unusedEnabled bool,
	gofmtEnabled bool,
) error {
	if len(patterns) == 0 {
		patterns = []string{
			"./registry/...",
			"./common/...",
			"./cmds/...",
			"./tests/...",
		}
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles |
			packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo | packages.NeedDeps,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return fmt.Errorf("load error: %w", err)
	}

	var fset *token.FileSet
	for _, pkg := range pkgs {
		fset = pkg.Fset
		break
	}
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			fmt.Fprintln(os.Stderr, "pkg error:", pkg.PkgPath, e)
		}
	}

	hits := 0
	if nilcheckEnabled {
		hits += runNilCheck(pkgs, fset)
	}
	if unusedEnabled {
		hits += runUnusedFuncsCheck(pkgs, fset)
	}
	if gofmtEnabled {
		hits += runGofmtCheck(pkgs)
	}

	if hits > 0 {
		os.Exit(1)
	}
	return nil
}

// runNilCheck finds callers that compare the result of a function/method
// whose return type is the empty interface ("any"/"interface{}")
// directly against nil, instead of using common.IsNil(). Returns the
// number of suspicious usages found.
func runNilCheck(pkgs []*packages.Package, fset *token.FileSet) int {
	// Map of *types.Func -> per-result-index bool: true if that specific
	// return slot's declared type is the empty interface (any/interface{}).
	// Tracking per-index (not just "func has an any somewhere") avoids
	// false positives like (*XRError, bool, any) where only the LAST slot
	// is "any" but the FIRST (an XRError) is legitimately nil-checked.
	anyFuncs := map[types.Object][]bool{}

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				fd, ok := n.(*ast.FuncDecl)
				if !ok || fd.Type.Results == nil {
					return true
				}
				var slots []bool
				anyFound := false
				for _, res := range fd.Type.Results.List {
					t := pkg.TypesInfo.TypeOf(res.Type)
					isAny := t != nil && isEmptyInterface(t)
					if isAny {
						anyFound = true
					}
					// res may declare multiple names sharing one type
					n := len(res.Names)
					if n == 0 {
						n = 1
					}
					for k := 0; k < n; k++ {
						slots = append(slots, isAny)
					}
				}
				if anyFound {
					if obj := pkg.TypesInfo.Defs[fd.Name]; obj != nil {
						anyFuncs[obj] = slots
					}
				}
				return true
			})
		}
	}

	type fn struct {
		name string
		pos  string
	}
	var fns []fn
	for obj := range anyFuncs {
		pos := fset.Position(obj.Pos())
		fns = append(
			fns,
			fn{
				name: obj.String(),
				pos:  fmt.Sprintf("%s:%d", pos.Filename, pos.Line),
			},
		)
	}
	sort.Slice(fns, func(i, j int) bool { return fns[i].pos < fns[j].pos })

	fmt.Println(
		"\n=== Suspicious '== nil' / '!= nil' usages on" +
			" any-returning calls ===",
	)
	hits := 0

	for _, pkg := range pkgs {
		info := pkg.TypesInfo
		for _, file := range pkg.Syntax {
			// Track var -> whether it was assigned (at least once) from
			// an any-typed RESULT SLOT specifically (not just "some call
			// to a func that happens to return any somewhere").
			varFromAnyCall := map[*types.Var]token.Pos{}

			ast.Inspect(file, func(n ast.Node) bool {
				stmt, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}
				if len(stmt.Rhs) == 1 && len(stmt.Lhs) > 1 {
					// multi-return single call, e.g. v, ok := Get(...)
					ce, ok := stmt.Rhs[0].(*ast.CallExpr)
					if !ok {
						return true
					}
					fobj := calleeObj(info, ce)
					slots, tracked := anyFuncs[fobj]
					if !tracked {
						return true
					}
					for idx, lhsExpr := range stmt.Lhs {
						if idx >= len(slots) || !slots[idx] {
							continue
						}
						trackVar(info, lhsExpr, ce.Pos(), varFromAnyCall)
					}
					return true
				}
				// single-value assignments, e.g. v := Get(...)
				for i, rhs := range stmt.Rhs {
					ce, ok := rhs.(*ast.CallExpr)
					if !ok {
						continue
					}
					fobj := calleeObj(info, ce)
					slots, tracked := anyFuncs[fobj]
					// single-return call: slot 0 must be any
					if !tracked || len(slots) == 0 || !slots[0] {
						continue
					}
					if i >= len(stmt.Lhs) {
						continue
					}
					trackVar(info, stmt.Lhs[i], ce.Pos(), varFromAnyCall)
				}
				return true
			})

			ast.Inspect(file, func(n ast.Node) bool {
				be, ok := n.(*ast.BinaryExpr)
				if !ok || (be.Op != token.EQL && be.Op != token.NEQ) {
					return true
				}
				var other ast.Expr
				if isNilIdent(be.X) {
					other = be.Y
				} else if isNilIdent(be.Y) {
					other = be.X
				} else {
					return true
				}
				pos := fset.Position(be.Pos())
				switch e := other.(type) {
				case *ast.CallExpr:
					fobj := calleeObj(info, e)
					slots, tracked := anyFuncs[fobj]
					if tracked && len(slots) > 0 && slots[0] {
						fmt.Printf("%s:%d: INLINE call %s %s nil  -->  %s\n",
							pos.Filename, pos.Line, fobj.Name(),
							opStr(be.Op), lineText(fset, be.Pos()))
						hits++
					}
				case *ast.Ident:
					if v, ok := info.Uses[e].(*types.Var); ok {
						if _, tracked := varFromAnyCall[v]; tracked {
							fmt.Printf(
								"%s:%d: VAR '%s' (from any-call)"+
									" %s nil  -->  %s\n",
								pos.Filename, pos.Line, e.Name,
								opStr(be.Op), lineText(fset, be.Pos()))
							hits++
						}
					}
				}
				return true
			})
		}
	}

	if hits > 0 {
		fmt.Fprintf(
			os.Stderr,
			"\nnilcheck: found %d suspicious nil comparison(s) on"+
				" 'any'-typed values; use common.IsNil() instead\n",
			hits,
		)
	} else {
		fmt.Println("\nnilcheck: no suspicious usages found")
	}
	return hits
}

// wellKnownStdlibIfaces are import-path/type-name pairs for common
// stdlib interfaces whose methods are invoked implicitly via interface
// dispatch (e.g. fmt calling String()/Error(), encoding/json calling
// MarshalJSON(), sort calling Len()/Less()/Swap(), net/http calling
// ServeHTTP()) rather than by direct reference. These are resolved via
// go/types.Implements() (real method-set matching), not name-matching,
// so a method is only excluded if its receiver type genuinely satisfies
// the interface.
var wellKnownStdlibIfaces = [][2]string{
	{"fmt", "Stringer"},
	{"sort", "Interface"},
	{"net/http", "Handler"},
	{"io", "Reader"},
	{"io", "Writer"},
	{"io", "Closer"},
	{"encoding/json", "Marshaler"},
	{"encoding/json", "Unmarshaler"},
}

// collectInterfaces gathers every non-empty interface type reachable
// from the scanned packages (their own locally-declared interfaces) plus
// a short list of well-known stdlib interfaces (see
// wellKnownStdlibIfaces). Used by runUnusedFuncsCheck to recognize
// methods that exist solely to satisfy an interface - and are therefore
// called implicitly via interface dispatch, not by direct reference.
func collectInterfaces(pkgs []*packages.Package) []*types.Interface {
	var ifaces []*types.Interface
	seen := map[*types.Interface]bool{}
	add := func(t types.Type) {
		if iface, ok := t.Underlying().(*types.Interface); ok &&
			iface.NumMethods() > 0 &&
			!seen[iface] {
			seen[iface] = true
			ifaces = append(ifaces, iface)
		}
	}

	// error is a predeclared universe type, not tied to any package.
	if errType := types.Universe.Lookup("error"); errType != nil {
		add(errType.Type())
	}

	// Local interfaces declared anywhere in the scanned packages.
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				if _, ok := ts.Type.(*ast.InterfaceType); !ok {
					return true
				}
				if obj := pkg.TypesInfo.Defs[ts.Name]; obj != nil {
					add(obj.Type())
				}
				return true
			})
		}
	}

	// Well-known stdlib interfaces, found by searching the already-loaded
	// dependency graph (via NeedDeps/NeedImports) - no extra Load() call.
	pkgByPath := map[string]*packages.Package{}
	var walk func(p *packages.Package)
	walk = func(p *packages.Package) {
		if p == nil || pkgByPath[p.PkgPath] != nil {
			return
		}
		pkgByPath[p.PkgPath] = p
		for _, imp := range p.Imports {
			walk(imp)
		}
	}
	for _, pkg := range pkgs {
		walk(pkg)
	}
	for _, pair := range wellKnownStdlibIfaces {
		if p, ok := pkgByPath[pair[0]]; ok && p.Types != nil {
			if obj := p.Types.Scope().Lookup(pair[1]); obj != nil {
				add(obj.Type())
			}
		}
	}

	return ifaces
}

// satisfiesAnyInterface reports whether the given named receiver type
// (or its pointer) implements any of the given interfaces.
func satisfiesAnyInterface(t types.Type, ifaces []*types.Interface) bool {
	ptr := types.NewPointer(t)
	for _, iface := range ifaces {
		if types.Implements(t, iface) || types.Implements(ptr, iface) {
			return true
		}
	}
	return false
}

// runUnusedFuncsCheck finds package-level funcs/methods that are never
// referenced anywhere in the scanned packages. Returns the number found.
func runUnusedFuncsCheck(pkgs []*packages.Package, fset *token.FileSet) int {
	ifaces := collectInterfaces(pkgs)

	type declInfo struct {
		name string
		pos  token.Pos
	}
	decls := map[types.Object]declInfo{}

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			isTestFile := strings.HasSuffix(
				fset.Position(file.Pos()).Filename,
				"_test.go",
			)
			for _, d := range file.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok {
					continue
				}
				name := fd.Name.Name
				if name == "main" || name == "init" {
					continue
				}
				if isTestFile && (strings.HasPrefix(name, "Test") ||
					strings.HasPrefix(name, "Benchmark") ||
					strings.HasPrefix(name, "Example")) {
					continue
				}
				if fd.Recv != nil && len(fd.Recv.List) > 0 {
					recvType := pkg.TypesInfo.TypeOf(fd.Recv.List[0].Type)
					if recvType != nil {
						if ptr, ok := recvType.(*types.Pointer); ok {
							recvType = ptr.Elem()
						}
						if satisfiesAnyInterface(recvType, ifaces) {
							continue
						}
					}
				}
				obj := pkg.TypesInfo.Defs[fd.Name]
				if obj == nil {
					continue
				}
				decls[obj] = declInfo{name: name, pos: fd.Name.Pos()}
			}
		}
	}

	used := map[types.Object]bool{}
	for _, pkg := range pkgs {
		info := pkg.TypesInfo
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok {
					if obj := info.Uses[id]; obj != nil {
						used[obj] = true
					}
				}
				if sel, ok := n.(*ast.SelectorExpr); ok {
					if s, ok := info.Selections[sel]; ok {
						used[s.Obj()] = true
					}
				}
				return true
			})
		}
	}

	type fn struct {
		name string
		pos  string
	}
	var orphans []fn
	for obj, di := range decls {
		if used[obj] {
			continue
		}
		pos := fset.Position(di.pos)
		orphans = append(
			orphans,
			fn{
				name: di.name,
				pos:  fmt.Sprintf("%s:%d", pos.Filename, pos.Line),
			},
		)
	}
	sort.Slice(
		orphans,
		func(i, j int) bool { return orphans[i].pos < orphans[j].pos },
	)

	fmt.Println(
		"\n=== Funcs/methods never referenced anywhere in scanned packages ===",
	)
	for _, f := range orphans {
		fmt.Printf("%s: %s\n", f.pos, f.name)
	}

	if len(orphans) > 0 {
		fmt.Fprintf(
			os.Stderr,
			"\nunused: found %d func(s)/method(s) with no references\n",
			len(orphans),
		)
	} else {
		fmt.Println("\nunused: no orphaned funcs found")
	}
	return len(orphans)
}

// runGofmtCheck runs the real "gofmt -l" binary (found on PATH) against
// every Go source file in the scanned packages, rather than
// reimplementing gofmt's formatting rules. Returns the number of files
// gofmt reports as needing reformatting.
func runGofmtCheck(pkgs []*packages.Package) int {
	seen := map[string]bool{}
	var files []string
	for _, pkg := range pkgs {
		for _, f := range pkg.GoFiles {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	sort.Strings(files)

	fmt.Println("\n=== Files needing 'gofmt -w' ===")

	if len(files) == 0 {
		fmt.Println("\ngofmt: no files to check")
		return 0
	}

	args := append([]string{"-l"}, files...)
	out, err := exec.Command("gofmt", args...).Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Fprintln(os.Stderr, "gofmt: failed to run gofmt:", err)
			return 0
		}
	}

	lines := strings.FieldsFunc(
		strings.TrimSpace(string(out)),
		func(r rune) bool { return r == '\n' },
	)
	for _, l := range lines {
		fmt.Println(l)
	}

	if len(lines) > 0 {
		fmt.Fprintf(
			os.Stderr,
			"\ngofmt: found %d file(s) needing 'gofmt -w'\n",
			len(lines),
		)
	} else {
		fmt.Println("\ngofmt: all files properly formatted")
	}
	return len(lines)
}

func trackVar(
	info *types.Info,
	lhsExpr ast.Expr,
	pos token.Pos,
	out map[*types.Var]token.Pos,
) {
	id, ok := lhsExpr.(*ast.Ident)
	if !ok {
		return
	}
	if v, ok := info.Defs[id].(*types.Var); ok {
		out[v] = pos
	} else if v, ok := info.Uses[id].(*types.Var); ok {
		out[v] = pos
	}
}

func opStr(op token.Token) string {
	if op == token.EQL {
		return "=="
	}
	return "!="
}

func isNilIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}

func calleeObj(info *types.Info, ce *ast.CallExpr) types.Object {
	switch fn := ce.Fun.(type) {
	case *ast.Ident:
		return info.Uses[fn]
	case *ast.SelectorExpr:
		return info.Uses[fn.Sel]
	}
	return nil
}

var lineCache = map[string][]string{}

func lineText(fset *token.FileSet, pos token.Pos) string {
	p := fset.Position(pos)
	lines, ok := lineCache[p.Filename]
	if !ok {
		data, err := os.ReadFile(p.Filename)
		if err != nil {
			return ""
		}
		lines = strings.Split(string(data), "\n")
		lineCache[p.Filename] = lines
	}
	if p.Line-1 >= 0 && p.Line-1 < len(lines) {
		return strings.TrimSpace(lines[p.Line-1])
	}
	return ""
}
