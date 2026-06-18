package main

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	mychecks := []*analysis.Analyzer{
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
		ErrOsExitAnalyzer,
	}
	saAnalyzers := staticcheck.Analyzers
	additionalChecks := map[string]bool{
		"ST1000": true,
		"ST1003": true,
		"ST1004": true,
		"ST1005": true,
		"ST1006": true,
		"ST1007": true,
	}

	for _, sa := range saAnalyzers {
		if strings.HasPrefix(sa.Analyzer.Name, "SA") {
			mychecks = append(mychecks, sa.Analyzer)
		} else if additionalChecks[sa.Analyzer.Name] {
			mychecks = append(mychecks, sa.Analyzer)
		}
	}

	multichecker.Main(
		mychecks...,
	)
}
