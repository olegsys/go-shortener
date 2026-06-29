// Package noosexit содержит анализатор, который запрещает использование
// прямого вызова os.Exit в функции main пакета main
//
// Прямой вызов os.Exit в функции main приводит к немедленному завершению
// процесса без выполнения defer-функций. Это может стать причиной:
//   - потери данных в буферах (например, логи zap.Logger могут не успеть
//     записаться на диск);
//   - незакрытых соединений с БД, сетью или файлов;
//   - невыполнения graceful shutdown логики, зарегистрированной через defer.
//
// Анализатор срабатывает только на пакете main и только внутри функции main.
// Использование os.Exit в других пакетах и других функциях остаётся разрешённым.
package noosexit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// Analyzer - экземпляр анализатора для подключения в multichecker
var Analyzer = &analysis.Analyzer{
	Name:     "noosexit",
	Doc:      "запрещает прямой вызов os.Exit в функции main пакета main",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Name.Name != "main" {
				continue
			}

			ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
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
