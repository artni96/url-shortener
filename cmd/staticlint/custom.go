package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var ErrOsExitAnalyzer = &analysis.Analyzer{
	Name: "errcheck",
	Doc:  "check for unchecked errors",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		if pass.Pkg.Name() == "main" {
			ast.Inspect(file, func(n ast.Node) bool {
				funcDecl, ok := n.(*ast.FuncDecl)
				if !ok || funcDecl.Name.Name != "main" {
					return true
				}

				ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
					callExpr, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					if slc, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						if pkg, ok := slc.X.(*ast.Ident); ok && pkg.Name != "main" && slc.Sel.Name == "Exit" {
							pass.Reportf(callExpr.Pos(), "it's not allowed to call os.Exit in main function")
						}
					}
					return true
				})
				return false
			})
		}
	}
	return nil, nil
}
