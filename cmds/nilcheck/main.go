// nilcheck finds callers that compare the result of a function/method
// whose return type is the empty interface ("any"/"interface{}") directly
// against nil using "== nil"/"!= nil", instead of using common.IsNil().
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
// Usage:
//
//	go run ./cmds/nilcheck [packages...]
//
// With no arguments it checks ./registry/... ./common/... ./cmds/...
// (the tmp/ directory is intentionally excluded — it's a stale/scratch
// package that doesn't currently build).
//
// Exits with a non-zero status if any suspicious usage is found, so it
// can be wired into CI (see "make nilcheck").
package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

func isEmptyInterface(t types.Type) bool {
	iface, ok := t.Underlying().(*types.Interface)
	return ok && iface.NumMethods() == 0
}

func main() {
	patterns := os.Args[1:]
	if len(patterns) == 0 {
		patterns = []string{"./registry/...", "./common/...", "./cmds/..."}
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo | packages.NeedDeps,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load error:", err)
		os.Exit(1)
	}

	// Map of *types.Func -> per-result-index bool: true if that specific
	// return slot's declared type is the empty interface (any/interface{}).
	// Tracking per-index (not just "func has an any somewhere") avoids
	// false positives like (*XRError, bool, any) where only the LAST slot
	// is "any" but the FIRST (an XRError) is legitimately nil-checked.
	anyFuncs := map[types.Object][]bool{}

	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			fmt.Fprintln(os.Stderr, "pkg error:", pkg.PkgPath, e)
		}
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
	var fset *token.FileSet
	for _, pkg := range pkgs {
		fset = pkg.Fset
		break
	}
	for obj := range anyFuncs {
		pos := fset.Position(obj.Pos())
		fns = append(fns, fn{name: obj.String(), pos: fmt.Sprintf("%s:%d", pos.Filename, pos.Line)})
	}
	sort.Slice(fns, func(i, j int) bool { return fns[i].pos < fns[j].pos })

	/*
		fmt.Println("=== Functions/methods returning 'any' (interface{}) ===")
		for _, f := range fns {
			fmt.Printf("%s :: %s\n", f.pos, f.name)
		}
	*/

	fmt.Println("\n=== Suspicious '== nil' / '!= nil' usages on any-returning calls ===")
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
					if slots, tracked := anyFuncs[fobj]; tracked && len(slots) > 0 && slots[0] {
						fmt.Printf("%s:%d: INLINE call %s %s nil  -->  %s\n",
							pos.Filename, pos.Line, fobj.Name(), opStr(be.Op), lineText(fset, be.Pos()))
						hits++
					}
				case *ast.Ident:
					if v, ok := info.Uses[e].(*types.Var); ok {
						if _, tracked := varFromAnyCall[v]; tracked {
							fmt.Printf("%s:%d: VAR '%s' (from any-call) %s nil  -->  %s\n",
								pos.Filename, pos.Line, e.Name, opStr(be.Op), lineText(fset, be.Pos()))
							hits++
						}
					}
				}
				return true
			})
		}
	}

	if hits > 0 {
		fmt.Fprintf(os.Stderr, "\nnilcheck: found %d suspicious nil comparison(s) on 'any'-typed values; use common.IsNil() instead\n", hits)
		os.Exit(1)
	}
	fmt.Println("\nnilcheck: no suspicious usages found")
}

func trackVar(info *types.Info, lhsExpr ast.Expr, pos token.Pos, out map[*types.Var]token.Pos) {
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
