# BlaisePopov/errors

[![Go Reference](https://pkg.go.dev/badge/github.com/BlaisePopov/errors.svg)](https://pkg.go.dev/github.com/BlaisePopov/errors)

**Languages:** [English](README.md) | [Русский](README.ru.md) | Español | [中文](README.zh.md)

El paquete `errors` añade soporte de trazas de pila a los errores en Go.

Es un **reemplazo directo** del paquete estándar `errors`: basta con cambiar la ruta de importación y todas las llamadas existentes a `errors.New`, `errors.Is`, `errors.As`, `errors.Unwrap` y `errors.Join` seguirán funcionando — pero ahora `errors.New` y funciones asociadas también capturan una traza de pila.

## Compatibilidad con `errors` (stdlib)

| Función / Variable                | stdlib | este paquete | Notas                                       |
|------------------------------------|--------|--------------|---------------------------------------------|
| `New(text string) error`           | ✅     | ✅           | Además captura la traza de pila             |
| `Is(err, target error) bool`       | ✅     | ✅           | Delega en `errors.Is`                       |
| `As(err error, target any) bool`   | ✅     | ✅           | Delega en `errors.As`                       |
| `Unwrap(err error) error`          | ✅     | ✅           | Delega en `errors.Unwrap`                   |
| `Join(errs ...error) error`        | ✅     | ✅           | Delega en `errors.Join` (Go 1.20+)          |
| `ErrUnsupported`                   | ✅     | ✅           | Centinela re-exportado (Go 1.21+)           |

### API extendida

| Función                                           | Descripción                                                |
|---------------------------------------------------|------------------------------------------------------------|
| `From(v any) *Error`                              | Envuelve cualquier valor como `*Error` con traza de pila  |
| `Wrap(err error, skip int) error`                 | Envuelve un error existente con traza de pila              |
| `WrapPrefix(err error, prefix string, skip int)`  | Envuelve con prefijo descriptivo + traza de pila           |
| `Errorf(format string, a ...any) error`           | Como `fmt.Errorf` pero con traza de pila                   |
| `ParsePanic(text string) (*Error, error)`         | Reconstruye `*Error` a partir de la salida de un panic     |

## Instalación

```bash
go get github.com/BlaisePopov/errors
```

## Inicio rápido

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

## El tipo `*Error`

El tipo `*Error` implementa la interfaz estándar `error` y proporciona:

- **`Error() string`** — mensaje de error (con prefijo opcional)
- **`Unwrap() error`** — devuelve el error envuelto
- **`Stack() []byte`** — traza de pila formateada (como `runtime/debug.Stack()`)
- **`StackFrames() []StackFrame`** — datos estructurados de los marcos de pila
- **`ErrorStack() string`** — tipo + mensaje + traza de pila en una sola cadena
- **`TypeName() string`** — nombre del tipo del error subyacente
- **`Location() (file string, line int)`** — archivo y línea donde se creó el error
- **`FuncName() string`** — nombre de la función donde se creó el error
- **`Prefix() string`** — prefijo establecido mediante `WrapPrefix`
- **`Callers() []uintptr`** — contadores de programa sin procesar

## Seguridad en concurrencia

Los objetos `*Error` son seguros para acceso concurrente de lectura después de su creación.
Los marcos de pila y la salida formateada se calculan de forma diferida con `sync.Once`,
lo que hace que las llamadas concurrentes a `StackFrames()` y `Stack()` sean seguras.

## Benchmarks

### Benchmarks internos

Resultados (Windows/amd64, Intel i5-8250U):

| Operación               | ns/op | allocs | B/op |
|--------------------------|------:|-------:|-----:|
| `New()`                  |  195  |   1    |  96  |
| `Wrap()`                 |  218  |   1    |  96  |
| `WrapPrefix()`           |  209  |   1    |  96  |
| `Error()`                |    4  |   0    |   0  |
| `StackFrames()` (cached) |    5  |   0    |   0  |
| `Stack()` (cached)       |    5  |   0    |   0  |
| `ErrorStack()` (cached)  |    5  |   0    |   0  |
| `From()`                 |  219  |   1    |  96  |

### Benchmarks comparativos (vs. cockroachdb/errors, juju/errors)

#### New — creación de error hoja

| Paquete                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **este paquete**       |   210 |   1    |  96  |
| juju/errors            |   689 |   3    | 328  |
| cockroachdb/errors     |  1553 |   7    | 416  |
| go-errors/errors       |   894 |   4    | 528  |

#### Single Wrap — envolver un error preexistente

| Paquete                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **este paquete**       |   220 |   1    |  96  |
| juju/errors            |   778 |   3    | 328  |
| cockroachdb/errors     |  1836 |   7    | 432  |
| go-errors/errors       |    81 |   1    |  80  |

#### Create + Wrap ×5 — cadena completa de errores

| Paquete                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **este paquete**       |  1161 |   6    | 576  |
| juju/errors            |  6088 |  18    | 1968 |
| cockroachdb/errors     | 12461 |  42    | 2577 |
| go-errors/errors       |  2364 |  21    | 1224 |

#### Error() — formateo de cadena de cadena de 5 envolturas

| Paquete                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **este paquete**       |   3.4 |   0    |   0  |
| juju/errors            |  3059 |  15    | 408  |
| cockroachdb/errors     | 11996 |  67    | 5945 |
| go-errors/errors       |   286 |   3    | 112  |

#### Extracción de traza de pila

| Paquete                |     ns/op | allocs |    B/op |
|------------------------|---------:|-------:|--------:|
| **este paquete**       |      5.6 |   0    |      0  |
| juju/errors            |     4422 |  31    |   1680  |
| cockroachdb/errors     |    56594 | 126    |  22604  |
| go-errors/errors       |   555963 |  70    |  27790  |

#### Unwrap all — recorrido completo de la cadena

| Paquete                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **este paquete**       |  35.4 |   0    |   0  |
| juju/errors            |   6.5 |   0    |   0  |
| cockroachdb/errors     |  72.9 |   0    |   0  |
| go-errors/errors       |   8.5 |   0    |   0  |

### Conclusiones

1. **La extracción de traza de pila es la ventaja principal.** Este paquete es **790× más rápido** que juju/errors, **10 100× más rápido** que cockroachdb/errors y **99 300× más rápido** que BlaisePopov/errors en el renderizado de trazas de pila — gracias a la caché con `sync.Once` y la resolución diferida de marcos.

2. **WrapPrefix es eficiente en asignaciones.** Una sola envoltura produce solo **1 asignación / 96 B**, superando a juju (3/328) y cockroachdb (7/432). La cadena de 5 envolturas usa **6 asignaciones / 576 B** — menos de un tercio de las asignaciones de cualquier competidor (18–42).

3. **El formateo Error() es casi gratuito con caché.** Con **3.4 ns** para una cadena de 5 envolturas (con `sync.Once`), supera a juju (900×) y cockroachdb (3 500×). Incluso la primera llamada es rápida gracias a la caché de concatenación de prefijos.

4. **New() tiene una huella mínima.** La creación de error hoja usa **96 B / 1 asignación** — la menor huella de memoria y menos asignaciones de todos los paquetes probados.

## Licencia

Este paquete está licenciado bajo la licencia MIT. Consulte [LICENSE.MIT](LICENSE.MIT) para más detalles.

## Registro de cambios

* v1.1.0 Actualizado para usar `errors.Is` de Go 1.13 en lugar de `==`
* v1.2.0 Añadido `errors.As` de la biblioteca estándar
* v1.3.0 *CAMBIO ROTURO* Actualizados los métodos de error para devolver `error` en lugar de `*Error`
* v1.4.0 *CAMBIO ROTURO* Revertidos los cambios de v1.3.0 (idéntico a v1.2.0)
* v1.4.1 Sin cambios de código, eliminado el archivo innecesario `cover.out`
* v1.4.2 Mejora de rendimiento en `ErrorStack()`
* v1.5.0 Añadidos `errors.Join()` y `errors.Unwrap()`
* v1.5.1 Corregida la compilación en Go 1.13–1.19
* v2.0.0 Refactorización importante:
  - Versión mínima de Go: 1.21
  - Añadido centinela `ErrUnsupported`
  - Corregida condición de carrera en `StackFrames()` (ahora usa `sync.Once`)
  - Reemplazado `stackCache` global por caché por error (sin fuga de memoria)
  - `Wrap()` y `WrapPrefix()` ahora capturan trazas de pila completas e información de ubicación
  - `Is()` ahora delega completamente en `errors.Is` (semántica compatible con stdlib)
  - Eliminados archivos con etiquetas de compilación (`error_1_13.go`, `join_unwrap_1_20.go`)
  - Mejorada la rendimiento: menos asignaciones en captura de pila y formateo de marcos
  - Añadido método `FuncName()` (alias: `LocationFunc()`)
  - Comentarios godoc exhaustivos y funciones de prueba `Example*`
