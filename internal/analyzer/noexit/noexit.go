// Package noexit реализует статический анализатор Go,
// запрещающий прямые вызовы os.Exit в функции main пакета main.
package noexit

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "noexit",
	Doc:  "запрещает прямой вызов os.Exit в функции main пакета main",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" ||
		strings.HasSuffix(pass.Pkg.Path(), ".test") {
		return nil, nil
	}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" || fn.Recv != nil || fn.Body == nil {
				continue
			}

			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				obj, ok := pass.TypesInfo.Uses[selector.Sel]
				if !ok {
					return true
				}

				function, ok := obj.(*types.Func)
				if !ok || function.Pkg() == nil {
					return true
				}

				if function.Pkg().Path() == "os" && function.Name() == "Exit" {
					pass.Reportf(call.Pos(),
						"прямой вызов os.Exit запрещён в функции main")
				}

				return true
			})
		}
	}

	return nil, nil
}
