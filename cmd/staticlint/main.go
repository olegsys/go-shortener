// Package main реализует multichecker — единый инструмент статического анализа,
// объединяющий набор стандартных и сторонних анализаторов Go.
//
// # Механизм запуска
//
// Multichecker построен на базе пакета golang.org/x/tools/go/analysis/multichecker,
// который предоставляет единый CLI-интерфейс для запуска сразу нескольких
// анализаторов. Он поддерживает все стандартные флаги утилиты go vet:
//
//	# Анализ всего проекта:
//	go run ./cmd/staticlint ./...
//
//	# Анализ конкретного пакета:
//	go run ./cmd/staticlint ./internal/service/...
//
//	# Сборка бинарника и запуск:
//	go build -o staticlint ./cmd/staticlint
//	./staticlint ./...
//
//	# Вывод в формате JSON (удобно для CI):
//	./staticlint -json ./...
//
// multichecker автоматически возвращает ненулевой exit-код, если хотя бы один
// из анализаторов нашёл проблему, что позволяет использовать его в CI/CD пайплайнах.
//
// # Состав multichecker
//
// Ниже подробно описаны все анализаторы, подключённые к multichecker, сгруппированные
// по источнику и классу.
//
// ## 1. Стандартные анализаторы (golang.org/x/tools/go/analysis/passes)
//
// Это «официальный» набор анализаторов от команды Go. Они покрывают наиболее
// критичные классы ошибок: гонки данных, блокировки, printf-несоответствия и т.д.
// Обратите внимание, что в современных версиях x/tools некоторые пакеты были
// переименованы (например, copylocks -> copylock, composites -> composite).
//
// ## 2. Анализаторы класса SA пакета staticcheck.io
//
// Класс SA (Static Analysis) — это наиболее строгие проверки staticcheck,
// которые обнаруживают реальные баги в коде: паники, утечки памяти, гонки
// и другие серьёзные ошибки.
// Подключаются все анализаторы из статического массива staticcheck.Analyzers
// (SA1000–SA9999).
//
// ## 3. Анализаторы остальных классов staticcheck.io
//
// Помимо класса SA, staticcheck предоставляет три других класса:
//
//   - simple (S):    упрощения кода (S1000–S1039).
//   - stylecheck (ST): стиль кода и соглашения (ST1000–ST1023).
//   - quickfix (QF):  автоматические рефакторинги (QF1001–QF1011).
//
// Все они также подключены к multichecker.
//
// ## 4. Публичные сторонние анализаторы
//
//   - errcheck (github.com/kisielk/errcheck) — находит места, где возвращаемое
//     значение error не проверяется. Классическая проблема Go-кода.
//   - bodyclose (github.com/timakin/bodyclose) — проверяет, что Body HTTP-ответа
//     закрывается после использования. Предотвращает утечки сетевых сокетов.
//
// ## 5. Собственный анализатор
//
//   - noosexit (cmd/staticlint/noosexit) — запрещает прямой вызов os.Exit
//     в функции main пакета main. Подробнее — в документации пакета noosexit.
package main

import (
	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/olegsys/go-shortener/cmd/staticlint/noosexit"
)

func main() {
	analyzers := []*analysis.Analyzer{
		// Стандартные анализаторы (golang.org/x/tools/go/analysis/passes)
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		copylock.Analyzer,
		errorsas.Analyzer,
		httpresponse.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,

		// Сторонние публичные анализаторы
		errcheck.Analyzer,
		bodyclose.Analyzer,

		// Собственный анализатор
		noosexit.Analyzer,
	}

	// Все анализаторы класса SA (staticcheck)
	for _, a := range staticcheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	// Остальные классы staticcheck (S, ST, QF)
	for _, a := range simple.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}
	for _, a := range stylecheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}
	for _, a := range quickfix.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	multichecker.Main(analyzers...)
}
