package noexit

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()

	t.Run("os.Exit inside main", func(t *testing.T) {
		t.Parallel()

		analysistest.Run(t, testdata, Analyzer, "invalid")
	})

	t.Run("no os.Exit inside main", func(t *testing.T) {
		t.Parallel()

		analysistest.Run(t, testdata, Analyzer, "valid")
	})
}
