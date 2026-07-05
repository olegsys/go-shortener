# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

#### Вывод команды `go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof`:

```
Showing nodes accounting for -13889.99kB, 37.44% of 37099.58kB total
Dropped 1 node (cum <= 185.50kB)
      flat  flat%   sum%        cum   cum%
-5442.97kB 14.67% 14.67% -7491.05kB 20.19%  github.com/olegsys/go-shortener/internal/repository.(*MapStorage).Set
-2048.38kB  5.52% 20.19% -2048.38kB  5.52%  net/http.Header.Clone (inline)
-2048.09kB  5.52% 25.71% -2048.09kB  5.52%  github.com/google/uuid.UUID.String (partial-inline)
 1536.07kB  4.14% 21.57% -8515.85kB 22.95%  github.com/olegsys/go-shortener/internal/handler.(*Handler).Shorten
    1028kB  2.77% 18.80%     1028kB  2.77%  bufio.NewWriterSize (inline)
-1024.50kB  2.76% 21.56% -1024.50kB  2.76%  io.ReadAll
-1024.34kB  2.76% 24.32% -1024.34kB  2.76%  net/textproto.readMIMEHeader
-1024.31kB  2.76% 27.09% -1024.31kB  2.76%  net/http.(*Request).WithContext
-1024.14kB  2.76% 29.85% -1536.52kB  4.14%  net/http.(*conn).readRequest
```