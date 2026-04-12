// Staticlint — multichecker for static analysis of Go source code.
//
// # Overview
//
// staticlint aggregates multiple static-analysis passes into a single binary
// that can be run against any Go package.  It combines:
//
//   - Standard analyzers from golang.org/x/tools/go/analysis/passes
//     (assign, atomic, bools, composites, copylocks, httpresponse, loopclosure,
//     lostcancel, nilfunc, printf, shift, stringintconv, structtag,
//     tests, unmarshal, unreachable, unsafeptr).
//   - All SA-class analyzers from staticcheck.io — these cover correctness
//     checks such as misuse of sync primitives, incorrect format strings, and
//     many other common mistakes.
//   - S1003 from the "simple" (S) class of staticcheck.io — suggests replacing
//     strings.Contains-equivalent loops with direct calls.
//   - ST1005 from the "stylecheck" (ST) class of staticcheck.io — checks that
//     error strings are not capitalized.
//   - Two public analyzers:
//   - nilerr    (github.com/gostaticanalysis/nilerr) — detects returning nil
//     when an error variable is not nil.
//   - bodyclose (github.com/timakin/bodyclose) — detects unclosed HTTP response bodies.
//   - exitcheck  (project-local) — custom analyzer that forbids direct
//     os.Exit calls in the main function of the main package.
//
// # Running
//
// Build the binary and run it against packages:
//
//	go build -o staticlint ./cmd/staticlint
//	./staticlint ./...
//
// You can also pass standard analysis flags (e.g. -fix, -json):
//
//	./staticlint -json ./cmd/server/...
//
// To run a single analyzer, use the -<name> flag:
//
//	./staticlint -exitcheck ./cmd/...
//
// # Analyzers
//
// ## Standard passes (golang.org/x/tools/go/analysis/passes)
//
//   - assign:        detects useless assignments.
//   - atomic:        checks for common mistakes using sync/atomic.
//   - bools:         detects common mistakes involving boolean operators.
//   - composites:    checks for unkeyed composite literals.
//   - copylocks:     checks for locks passed by value.
//   - httpresponse:  checks for mistakes using HTTP responses.
//   - loopclosure:   checks for references to enclosing loop variables from
//     within nested functions.
//   - lostcancel:    checks for failure to call a context cancellation function.
//   - nilfunc:       checks for useless comparisons between functions and nil.
//   - printf:        checks consistency of Printf format strings and arguments.
//   - shift:         checks for shifts that equal or exceed the width of the integer.
//   - stringintconv: checks for string(int) conversions.
//   - structtag:     checks that struct field tags conform to reflect.StructTag.Get.
//   - tests:         checks for common mistaken usages of tests and examples.
//   - unmarshal:     checks for passing non-pointer or non-interface types to
//     unmarshal functions.
//   - unreachable:   checks for unreachable code.
//   - unsafeptr:     checks for invalid conversions of uintptr to unsafe.Pointer.
//
// ## Staticcheck SA class
//
// All analyzers with prefix SA from staticcheck.io are included. They provide
// hundreds of correctness checks — see https://staticcheck.dev/docs/checks/#SA
// for the full list.
//
// ## Staticcheck other classes
//
//   - S1003 (simple): suggests replacing hand-written string search loops with
//     strings.Contains.
//   - ST1005 (stylecheck): ensures error strings are not capitalized and do not
//     end with punctuation.
//
// ## Public analyzers
//
//   - nilerr:    detects returning nil when the err variable is non-nil, which
//     may cause bugs by silently discarding errors.  See
//     https://github.com/gostaticanalysis/nilerr
//   - bodyclose: detects unclosed HTTP response bodies.
//     See https://github.com/timakin/bodyclose
//
// ## Custom analyzer
//
//   - exitcheck: reports direct calls to os.Exit inside the main function of the
//     main package. Using os.Exit bypasses deferred functions, potentially
//     leaking resources. Prefer log.Fatal or returning an error code.
package main

import (
	"strings"

	"github.com/gostaticanalysis/nilerr"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"github.com/zheki1/yaprmtrc/cmd/staticlint/exitcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	checks := []*analysis.Analyzer{
		// Standard passes
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		httpresponse.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,

		// Public analyzers
		nilerr.Analyzer,
		bodyclose.Analyzer,

		// Custom analyzer
		exitcheck.Analyzer,
	}

	// All SA analyzers from staticcheck.
	for _, a := range staticcheck.Analyzers {
		if strings.HasPrefix(a.Analyzer.Name, "SA") {
			checks = append(checks, a.Analyzer)
		}
	}

	// Selected analyzers from other staticcheck classes.
	addFromLint(simple.Analyzers, "S1003", &checks)
	addFromLint(stylecheck.Analyzers, "ST1005", &checks)

	multichecker.Main(checks...)
}

// addFromLint finds an analyzer by name in a []*lint.Analyzer slice
// and appends it to the destination.
func addFromLint(src []*lint.Analyzer, name string, dst *[]*analysis.Analyzer) {
	for _, a := range src {
		if a.Analyzer.Name == name {
			*dst = append(*dst, a.Analyzer)
			return
		}
	}
}
