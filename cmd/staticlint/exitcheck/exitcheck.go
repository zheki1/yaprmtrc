// Package exitcheck defines an analyzer that reports dangerous direct calls
// in the main package: os.Exit, log.Fatal (and variants), and panic.
//
// # Analyzer exitcheck
//
// exitcheck inspects every function in a main package (not just main()) and
// reports:
//   - os.Exit          — terminates the process immediately, skipping defers.
//   - log.Fatal/Fatalf/Fatalln — calls os.Exit(1) after logging.
//   - panic            — unwinds the stack; in production code a controlled
//     shutdown is preferable.
package exitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer is the exitcheck analyzer.
var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "reports os.Exit, log.Fatal*, and panic calls in the main package",
	Run:  run,
}

// isFatalSelector returns true for os.Exit, log.Fatal, log.Fatalf, log.Fatalln.
func isFatalSelector(pkg, name string) bool {
	if pkg == "os" && name == "Exit" {
		return true
	}
	if pkg == "log" && (name == "Fatal" || name == "Fatalf" || name == "Fatalln") {
		return true
	}
	return false
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// Skip generated files (e.g., test main files produced by go test).
		if ast.IsGenerated(file) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr:
				// Matches pkg.Func() calls: os.Exit, log.Fatal, etc.
				ident, ok := fn.X.(*ast.Ident)
				if !ok {
					return true
				}
				if isFatalSelector(ident.Name, fn.Sel.Name) {
					pass.Reportf(call.Pos(), "call to %s.%s in the main package is not allowed", ident.Name, fn.Sel.Name)
				}
			case *ast.Ident:
				// Matches bare panic() call.
				if fn.Name == "panic" {
					pass.Reportf(call.Pos(), "call to panic in the main package is not allowed")
				}
			}
			return true
		})
	}
	return nil, nil
}
