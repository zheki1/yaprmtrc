// Package exitcheck defines an analyzer that reports direct calls to os.Exit
// in the main function of the main package.
//
// Using os.Exit in main bypasses all deferred functions, which can lead to
// resource leaks (unclosed files, unflushed buffers, etc.). It is recommended
// to use log.Fatal, return an error code, or restructure the code to let
// deferred cleanup run before the process terminates.
//
// # Analyzer exitcheck
//
// exitcheck checks that os.Exit is not called directly inside the main()
// function of a main package. The diagnostic is reported at the position of
// every os.Exit call expression found in such a function.
package exitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer is the exitcheck analyzer.
var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "reports direct os.Exit calls in the main function of the main package",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// Only inspect main packages.
		if pass.Pkg.Name() != "main" {
			continue
		}

		// Skip generated files (e.g., test main files produced by go test).
		if ast.IsGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" || fn.Recv != nil {
				continue
			}
			// Walk the body of the main function.
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if ident.Name == "os" && sel.Sel.Name == "Exit" {
					pass.Reportf(call.Pos(), "direct call to os.Exit in main function of main package is not allowed")
				}
				return true
			})
		}
	}
	return nil, nil
}
