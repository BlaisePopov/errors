# BlaisePopov/errors

[![Go Reference](https://pkg.go.dev/badge/github.com/BlaisePopov/errors.svg)](https://pkg.go.dev/github.com/BlaisePopov/errors)

**Languages:** [English](README.md) | Русский | [Español](README.es.md) | [中文](README.zh.md)

Пакет `errors` добавляет поддержку трассировки стека к ошибкам в Go.

Он является **прямой заменой** стандартного пакета `errors`: достаточно изменить путь импорта, и все существующие вызовы `errors.New`, `errors.Is`, `errors.As`, `errors.Unwrap` и `errors.Join` продолжат работать — но теперь `errors.New` и связанные функции также захватывают трассировку стека.

## Совместимость с `errors` (stdlib)

| Функция / Переменная               | stdlib | этот пакет | Примечания                            |
|-------------------------------------|--------|------------|---------------------------------------|
| `New(text string) error`            | ✅     | ✅         | Дополнительно захватывает трассировку стека |
| `Is(err, target error) bool`        | ✅     | ✅         | Делегирует к `errors.Is`              |
| `As(err error, target any) bool`    | ✅     | ✅         | Делегирует к `errors.As`              |
| `Unwrap(err error) error`           | ✅     | ✅         | Делегирует к `errors.Unwrap`          |
| `Join(errs ...error) error`         | ✅     | ✅         | Делегирует к `errors.Join` (Go 1.20+) |
| `ErrUnsupported`                    | ✅     | ✅         | Реэкспортированный сентинел (Go 1.21+) |

### Расширенный API

| Функция                                          | Описание                                              |
|--------------------------------------------------|-------------------------------------------------------|
| `From(v any) *Error`                             | Оборачивает любое значение как `*Error` с трассировкой стека |
| `Wrap(err error, skip int) error`                | Оборачивает существующую ошибку с трассировкой стека  |
| `WrapPrefix(err error, prefix string, skip int)` | Оборачивает с описательным префиксом + трассировкой стека |
| `Errorf(format string, a ...any) error`          | Аналог `fmt.Errorf`, но с трассировкой стека          |
| `ParsePanic(text string) (*Error, error)`        | Восстанавливает `*Error` из вывода паники             |

## Установка

```bash
go get github.com/BlaisePopov/errors
```

## Быстрый старт

```go
package main

import (
    "fmt"
    "github.com/BlaisePopov/errors"
)

var ErrNotFound = errors.New("not found")

func findItem(id int) error {
    return errors.WrapPrefix(ErrNotFound, fmt.Sprintf("item %d", id), 0)
}

func main() {
    err := findItem(42)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            // Print full stack trace
            fmt.Println(err.(*errors.Error).ErrorStack())
        }
    }
}
```

## Тип `*Error`

Тип `*Error` реализует стандартный интерфейс `error` и предоставляет:

- **`Error() string`** — сообщение ошибки (с опциональным префиксом)
- **`Unwrap() error`** — возвращает обёрнутую ошибку
- **`Stack() []byte`** — форматированная трассировка стека (аналог `runtime/debug.Stack()`)
- **`StackFrames() []StackFrame`** — структурированные данные кадров стека
- **`ErrorStack() string`** — тип + сообщение + трассировка стека в одной строке
- **`TypeName() string`** — имя типа базовой ошибки
- **`Location() (file string, line int)`** — файл и строка, где была создана ошибка
- **`FuncName() string`** — имя функции, где была создана ошибка
- **`Prefix() string`** — префикс, заданный через `WrapPrefix`
- **`Callers() []uintptr`** — сырые счётчики команд

## Потокобезопасность

Объекты `*Error` безопасны для одновременного чтения после создания.
Кадры стека и форматированный вывод вычисляются лениво с помощью `sync.Once`,
что делает одновременные вызовы `StackFrames()` и `Stack()` безопасными.

## Бенчмарки

### Внутренние бенчмарки

Результаты (Windows/amd64, Intel i5-8250U):

| Операция                | ns/op | allocs | B/op |
|--------------------------|------:|-------:|-----:|
| `New()`                  |  195  |   1    |  96  |
| `Wrap()`                 |  218  |   1    |  96  |
| `WrapPrefix()`           |  209  |   1    |  96  |
| `Error()`                |    4  |   0    |   0  |
| `StackFrames()` (cached) |    5  |   0    |   0  |
| `Stack()` (cached)       |    5  |   0    |   0  |
| `ErrorStack()` (cached)  |    5  |   0    |   0  |
| `From()`                 |  219  |   1    |  96  |

### Сравнительные бенчмарки (vs. cockroachdb/errors, juju/errors)

#### New — создание листовой ошибки

| Пакет                  | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **этот пакет**         |   210 |   1    |  96  |
| juju/errors            |   689 |   3    | 328  |
| cockroachdb/errors     |  1553 |   7    | 416  |
| go-errors/errors       |   894 |   4    | 528  |

#### Single Wrap — оборачивание существующей ошибки

| Пакет                  | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **этот пакет**         |   220 |   1    |  96  |
| juju/errors            |   778 |   3    | 328  |
| cockroachdb/errors     |  1836 |   7    | 432  |
| go-errors/errors       |    81 |   1    |  80  |

#### Create + Wrap ×5 — полная цепочка ошибок

| Пакет                  | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **этот пакет**         |  1161 |   6    | 576  |
| juju/errors            |  6088 |  18    | 1968 |
| cockroachdb/errors     | 12461 |  42    | 2577 |
| go-errors/errors       |  2364 |  21    | 1224 |

#### Error() — форматирование строки цепочки из 5 обёрток

| Пакет                  | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **этот пакет**         |   3.4 |   0    |   0  |
| juju/errors            |  3059 |  15    | 408  |
| cockroachdb/errors     | 11996 |  67    | 5945 |
| go-errors/errors       |   286 |   3    | 112  |

#### Извлечение трассировки стека

| Пакет                  |     ns/op | allocs |    B/op |
|------------------------|---------:|-------:|--------:|
| **этот пакет**         |      5.6 |   0    |      0  |
| juju/errors            |     4422 |  31    |   1680  |
| cockroachdb/errors     |    56594 | 126    |  22604  |
| go-errors/errors       |   555963 |  70    |  27790  |

#### Unwrap all — полный обход цепочки

| Пакет                  | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **этот пакет**         |  35.4 |   0    |   0  |
| juju/errors            |   6.5 |   0    |   0  |
| cockroachdb/errors     |  72.9 |   0    |   0  |
| go-errors/errors       |   8.5 |   0    |   0  |

## Лицензия

Этот пакет лицензирован под лицензией MIT. Подробности см. в [LICENSE.MIT](LICENSE.MIT).

## История изменений

* v1.1.0 Обновлено для использования `errors.Is` из Go 1.13 вместо `==`
* v1.2.0 Добавлен `errors.As` из стандартной библиотеки
* v1.3.0 *КРИТИЧЕСКОЕ ИЗМЕНЕНИЕ* Методы ошибок обновлены для возврата `error` вместо `*Error`
* v1.4.0 *КРИТИЧЕСКОЕ ИЗМЕНЕНИЕ* Отменены изменения v1.3.0 (идентично v1.2.0)
* v1.4.1 Без изменений кода, удалён ненужный файл `cover.out`
* v1.4.2 Улучшение производительности `ErrorStack()`
* v1.5.0 Добавлены `errors.Join()` и `errors.Unwrap()`
* v1.5.1 Исправлена сборка на Go 1.13–1.19
* v2.0.0 Крупный рефакторинг:
  - Минимальная версия Go: 1.21
  - Добавлен сентинел `ErrUnsupported`
  - Исправлено состояние гонки в `StackFrames()` (теперь используется `sync.Once`)
  - Глобальный `stackCache` заменён на кэширование на уровне ошибки (нет утечки памяти)
  - `Wrap()` и `WrapPrefix()` теперь захватывают полные трассировки стека и информацию о местоположении
  - `Is()` теперь полностью делегирует к `errors.Is` (семантика, совместимая со stdlib)
  - Удалены файлы с тегами сборки (`error_1_13.go`, `join_unwrap_1_20.go`)
  - Улучшена производительность: меньше аллокаций при захвате стека и форматировании кадров
  - Добавлен метод `FuncName()` (псевдоним: `LocationFunc()`)
  - Исчерпывающие godoc-комментарии и тестовые функции `Example*`
