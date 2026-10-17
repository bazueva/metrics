package main

import (
	"github.com/bazueva/metrics/internal/analyzer/noexit"
	"github.com/timakin/bodyclose/passes/bodyclose"

	"github.com/gostaticanalysis/nilerr"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"

	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	analyzers := []*analysis.Analyzer{
		// Стандартные анализаторы
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		copylock.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,

		// Публичные анализаторы
		bodyclose.Analyzer,
		nilerr.Analyzer,

		// Свой анализатор
		noexit.Analyzer,
	}

	// Все анализаторы класса SA
	for _, analyzer := range staticcheck.Analyzers {
		analyzers = append(analyzers, analyzer.Analyzer)
	}

	// Анализатор ST1005 из класса ST.
	for _, analyzer := range stylecheck.Analyzers {
		if analyzer.Analyzer.Name == "ST1005" {
			analyzers = append(analyzers, analyzer.Analyzer)
			break
		}
	}

	multichecker.Main(analyzers...)
}
