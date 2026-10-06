## Оптимизация использования памяти

Для поиска участков кода с избыточными аллокациями был снят профиль памяти приложения под нагрузкой.

В `Repository.Save` было обнаружено большое количество аллокаций при построении SQL-запроса через последовательную конкатенацию строк:

```go
sql += fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4)

if i != len(chunk)-1 {
    sql += ","
}
```

Профилирование до оптимизации:

```text
ROUTINE ======================== github.com/bazueva/metrics/internal/repository/db/metrics.(*Repository).Save
   12.51MB    26.64MB (flat, cum)

    6.50MB     7.01MB     sql += fmt.Sprintf(...)
       5MB        5MB     sql += ","
  512.75kB   512.75kB     sql += ` ON CONFLICT ...`
```

Для уменьшения количества аллокаций код был переписан с использованием `strings.Builder`.

Сравнение профилей:

```bash
go tool pprof -top \
  -diff_base=profiles/base.pprof \
  profiles/result.pprof
```

Оптимизация заключалась в уменьшении аллокаций при построении SQL-запроса, дополнительно профили были сравнены по `alloc_space`:

```bash
go tool pprof \
  -sample_index=alloc_space \
  -top \
  -diff_base=profiles/base.pprof \
  profiles/result.pprof
```

Фрагмент результата:

```text
File: server
Type: alloc_space
Time: 2026-10-03 08:30:31 +05

      flat  flat%   sum%        cum   cum%
  -12.51MB  0.04%  0.98%   -16.56MB 0.052%  github.com/bazueva/metrics/internal/repository/db/metrics.(*Repository).Save
```

В итоге в `Repository.Save` объём аллокаций уменьшился на `12.51 MB`, а с учётом вызываемых функций (`cum`) — на `16.56 MB`.


### Оптимизация gzip-сжатия

После оптимизации формирования SQL-запроса повторное профилирование показало, что основная часть аллокаций приходится на создание `gzip.Writer`:

```text
25421.37MB  80.31%  compress/flate.NewWriter
 5352.28MB  16.91%  compress/flate.(*compressor).initDeflate
```

Причиной являлось создание нового `gzip.Writer` для каждого HTTP-запроса:

```go
gzipWriter := gzip.NewWriter(w)
```

Для уменьшения количества аллокаций был добавлен `sync.Pool`, позволяющий переиспользовать `gzip.Writer` между запросами.

После изменения был снят новый профиль `profiles/gzip_pool.pprof`.

Сравнение профилей:

```bash
go tool pprof \
  -sample_index=alloc_space \
  -top \
  -diff_base=profiles/result.pprof \
  profiles/gzip_pool.pprof
```

Результат:

```text
      flat       flat%        cum          cum%
-25546.54MB      79.88%   -31118.51MB     97.30%  compress/flate.NewWriter
 -5432.87MB      16.99%    -5432.87MB     16.99%  compress/flate.(*compressor).initDeflate
  -192.92MB       0.60%     -192.92MB      0.60%  compress/flate.(*huffmanEncoder).generate
```

Общий объём `alloc_space` в новом профиле составил:

```text
369.59MB
```

В исходном профиле объём аллокаций составлял:

```text
31653.97MB
```

Таким образом, переиспользование `gzip.Writer` через `sync.Pool` позволило устранить основную часть аллокаций, связанных с созданием gzip-компрессора для каждого HTTP-запроса.

### Оптимизация пула соединений с PostgreSQL

В профиле `alloc_space` значительная часть аллокаций приходилась на создание новых соединений с PostgreSQL:

```text
57.93MB  github.com/jackc/pgx/v5/internal/stmtcache.NewLRUCache
80.02MB  github.com/jackc/pgx/v5.connect
80.52MB  github.com/jackc/pgx/v5.ConnectConfig
85.52MB  github.com/jackc/pgx/v5/stdlib.(*driverConnector).Connect
```

Для проверки поведения пула соединений была добавлена временная диагностика `database/sql` через `db.Stats()`.

Под нагрузкой наблюдалась следующая картина:

```text
DB STATS: open=22 in_use=20 idle=2 wait_count=0 wait_duration=0s max_open=0
DB STATS: open=22 in_use=22 idle=0 wait_count=0 wait_duration=0s max_open=0
DB STATS: open=3  in_use=2  idle=1 wait_count=0 wait_duration=0s max_open=0
DB STATS: open=3  in_use=1  idle=2 wait_count=0 wait_duration=0s max_open=0
```

Во время всплеска нагрузки `database/sql` открывал около 20 соединений, но потом они закрывались.

Для переиспользования уже созданных соединений было увеличено максимальное количество idle-соединений:

```go
db.SetMaxIdleConns(20)
```

После изменения статистика пула стала выглядеть следующим образом:

```text
DB STATS: open=2  in_use=1  idle=1  wait_count=0 wait_duration=0s max_open=0
DB STATS: open=21 in_use=16 idle=5  wait_count=0 wait_duration=0s max_open=0
DB STATS: open=21 in_use=1  idle=20 wait_count=0 wait_duration=0s max_open=0
```

Теперь после завершения нагрузки соединения не закрываются сразу, а остаются в пуле и могут быть повторно использованы последующими запросами.

Повторное профилирование показало существенное снижение аллокаций, связанных с созданием соединений:

```text
до:
80.52MB  github.com/jackc/pgx/v5.ConnectConfig
80.02MB  github.com/jackc/pgx/v5.connect
85.52MB  github.com/jackc/pgx/v5/stdlib.(*driverConnector).Connect

после:
2.55MB   github.com/jackc/pgx/v5.ConnectConfig
2.55MB   github.com/jackc/pgx/v5.connect
2.55MB   github.com/jackc/pgx/v5/stdlib.(*driverConnector).Connect
```

Аллокации `pgx.ConnectConfig` уменьшились:

```text
80.52 MB → 2.55 MB
```

`pgx/internal/stmtcache.NewLRUCache`, ранее занимавший около `57.93 MB`, также исчез из верхней части профиля.

Таким образом, увеличение `MaxIdleConns` позволило переиспользовать уже установленные соединения с PostgreSQL вместо их постоянного создания и закрытия при всплесках нагрузки.

### Оптимизация разбиения метрик на batch

При профилировании `Repository.Save` была дополнительная аллокация памяти при использовании `lo.Chunk` для разбиения метрик на группы перед сохранением в PostgreSQL.

Исходная реализация:

```go
chunks := lo.Chunk(data, 100)

for _, chunk := range chunks {
    // сохранение batch
}
```

`lo.Chunk` создаёт дополнительную структуру с наборами слайсов. Поскольку в данном случае группы используются последовательно и не требуется хранить их все одновременно, разбиение было заменено на проход по исходному слайсу по индексам:

```go
for start := 0; start < len(data); start += 100 {
    end := min(start+100, len(data))
    chunk := data[start:end]

    // сохранение batch
}
```

Сравнение `alloc_space` профилей показало:

```text
-8.03MB  github.com/samber/lo.Chunk
```

Таким образом, аллокации, связанные с `lo.Chunk`, были полностью устранены — примерно **8 MB за 10 секунд нагрузки**.

После этой оптимизации основная часть аллокаций внутри сохранения метрик приходится уже на формирование и передачу аргументов SQL-запроса:

```text
+48.43MB  database/sql.driverArgsConnLocked
+32.16MB  github.com/jackc/pgx/v5.(*ExtendedQueryBuilder).appendParam
+17.54MB  github.com/bazueva/metrics/internal/repository/db/metrics.(*Repository).Save
```
