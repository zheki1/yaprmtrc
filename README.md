# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
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

## Бенчмарки

Бенчмарки измеряют скорость выполнения ключевых компонентов системы.

### Запуск бенчмарков

```bash
# Repository (in-memory storage)
cd internal/repository && go test -bench . -benchmem

# Server handlers
cd cmd/server && go test -bench . -benchmem

# Security (hash)
cd internal/security && go test -bench . -benchmem

# Agent (runtime metrics collection)
cd cmd/agent && go test -bench . -benchmem
```

### Результаты бенчмарков

#### Repository (in-memory storage)
```
BenchmarkMemRepository_UpdateGauge-22           38582851                29.73 ns/op            0 B/op          0 allocs/op
BenchmarkMemRepository_UpdateCounter-22         45839492                31.17 ns/op            0 B/op          0 allocs/op
BenchmarkMemRepository_GetGauge-22              74923552                18.11 ns/op            0 B/op          0 allocs/op
BenchmarkMemRepository_GetCounter-22            54374260                23.68 ns/op            0 B/op          0 allocs/op
BenchmarkMemRepository_GetAll-22                  126432              9761 ns/op   11328 B/op        201 allocs/op
BenchmarkMemRepository_UpdateBatch-22             405140              3780 ns/op       0 B/op          0 allocs/op
```

#### Server handlers
```
BenchmarkUpdateHandlerJSON_Gauge-22               279198              5330 ns/op    7617 B/op         34 allocs/op
BenchmarkUpdateHandlerJSON_Counter-22             238893              5332 ns/op    7625 B/op         34 allocs/op
BenchmarkValueHandlerJSON-22                      219756              5044 ns/op    7593 B/op         33 allocs/op
BenchmarkBatchUpdateHandler-22                      5073            293085 ns/op   89231 B/op        646 allocs/op
BenchmarkPageHandler-22                             3518            358024 ns/op   79551 B/op       1970 allocs/op
```

#### Security (hash)
```
BenchmarkCalcHash-22                     8679318               149.0 ns/op          160 B/op           3 allocs/op
BenchmarkCalcHash_LargeBody-22            505092              2366 ns/op            160 B/op           3 allocs/op
```

## Профилирование памяти (pprof)

### Методология

1. Создаётся тестовый сервер с in-memory хранилищем
2. Хранилище предзаполняется 500 gauge + 500 counter метриками
3. Симулируется нагрузка: 500 раундов, каждый из которых включает:
   - одиночное обновление gauge и counter (`/update`)
   - запрос значения метрики (`/value`)
   - batch-обновление 200 метрик (`/updates`)
   - рендер HTML-страницы со всеми метриками (`/`)
4. Профиль сохраняется после `runtime.GC()`

### Снятие профилей

```bash
# Базовый профиль (до оптимизаций)
go test -run TestProfileMemory -count=1 ./cmd/server/

# Результирующий профиль (после оптимизаций)
PPROF_OUTPUT=../../profiles/result.pprof go test -run TestProfileMemory -count=1 ./cmd/server/
```

### Проведённые оптимизации

1. **pageHandler**: HTML-шаблон парсится один раз при инициализации (`template.Must`), вместо создания нового шаблона на каждый запрос. Замена `fmt.Sprintf` на `strconv.FormatFloat`/`strconv.FormatInt`. Предаллокация среза `rows` с известной ёмкостью.

2. **updateHandlerJSON**: Замена `io.ReadAll` + `json.Unmarshal` на потоковый `json.NewDecoder().Decode()` — убирается промежуточный буфер.

3. **batchUpdateHandler**: Аналогичная замена `io.ReadAll` + `json.Unmarshal` на `json.NewDecoder().Decode()`.

4. **MemRepository.GetAll**: Предаллокация результирующего среза `make([]Metrics, 0, len(gauges)+len(counters))` вместо `var res []Metrics` с последующими `append`.

5. **GzipMiddleware**: Использование `sync.Pool` для переиспользования `gzip.Writer`, вместо создания нового на каждый запрос.

### Результат сравнения профилей

```
$ go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof

Type: inuse_space
Showing nodes accounting for 513kB, 33.33% of 1539kB total
      flat  flat%   sum%        cum   cum%
     513kB 33.33% 33.33%      513kB 33.33%  runtime.allocm
         0     0% 33.33%      513kB 33.33%  runtime.mcall
         0     0% 33.33%      513kB 33.33%  runtime.newm
         0     0% 33.33%      513kB 33.33%  runtime.park_m
         0     0% 33.33%      513kB 33.33%  runtime.resetspinning
         0     0% 33.33%      513kB 33.33%  runtime.schedule
         0     0% 33.33%      513kB 33.33%  runtime.startm
         0     0% 33.33%      513kB 33.33%  runtime.wakep
```

```
$ go tool pprof -top -alloc_space -diff_base=profiles/base.pprof profiles/result.pprof

Type: alloc_space
Showing nodes accounting for -3.49MB, 0.77% of 454.02MB total
Dropped 2 nodes (cum <= 2.27MB)
      flat  flat%   sum%        cum   cum%
  -10.50MB  2.31%  2.31%    -5.50MB  1.21%  reflect.Value.call
   10.48MB  2.31% 0.0048%    10.48MB  2.31%  github.com/zheki1/yaprmtrc/internal/repository.(*MemRepository).GetAll
  -10.02MB  2.21%  2.21%   -10.02MB  2.21%  bytes.growSlice
   -8.09MB  1.78%  3.99%    -8.09MB  1.78%  encoding/json.(*Decoder).refill
    4.50MB  0.99%  3.00%     4.50MB  0.99%  html/template.htmlReplacer
      -4MB  0.88%  3.88%       -4MB  0.88%  reflect.unsafe_NewArray
       3MB  0.66%  3.22%        3MB  0.66%  reflect.packEface
    1.60MB  0.35%  2.87%     1.09MB  0.24%  github.com/zheki1/yaprmtrc/cmd/server.(*Server).pageHandler
    1.50MB  0.33%  2.54%    -2.50MB  0.55%  reflect.MakeSlice
    1.03MB  0.23%  2.31%     1.03MB  0.23%  reflect.growslice
       1MB  0.22%  2.09%        1MB  0.22%  bufio.NewReaderSize
       1MB  0.22%  1.87%        1MB  0.22%  sync.(*Pool).pinSlow
       1MB  0.22%  1.65%        1MB  0.22%  net/textproto.MIMEHeader.Set
       1MB  0.22%  1.43%        1MB  0.22%  net/url.parse
       1MB  0.22%  1.21%     3.51MB  0.77%  net/http/httptest.NewRequestWithContext
       1MB  0.22%  0.99%        1MB  0.22%  strconv.FormatFloat
       1MB  0.22%  0.77%        2MB  0.44%  encoding/json.(*decodeState).literalStore
       1MB  0.22%  0.55%        1MB  0.22%  reflect.New
      -1MB  0.22%  0.77%   -12.48MB  2.75%  text/template.(*state).walkRange
    0.50MB  0.11%  0.66%    -8.09MB  1.78%  github.com/zheki1/yaprmtrc/cmd/server.(*Server).batchUpdateHandler
```

Отрицательные значения показывают снижение потребления памяти после оптимизаций:
- `bytes.growSlice`: **-10.02MB** — снижение за счёт потокового декодирования JSON
- `encoding/json.(*Decoder).refill`: **-8.09MB** — потоковый `json.NewDecoder` вместо `io.ReadAll`
- `reflect.Value.call`: **-10.50MB** — сокращение вызовов рефлексии при рендере шаблона
- `text/template.(*state).walkRange`: **-12.48MB** (cum) — шаблон парсится один раз
- `batchUpdateHandler`: **-8.09MB** (cum) — устранение промежуточного буфера
