# Task portfolio backlog contract v1

Статус: **development / open** (27 августа 2026)

`TaskPortfolioManifest` фиксирует конечную цель — 168 TaskFamily и 456
совместимых runnable revisions. Этот файл и generated backlog переводят
разницу между целью и текущим release в адресные work items. Они не являются
каталогом задач и не содержат prompt, starter, hidden test, answer или
runtime verdict.

## Что считается gap

- planned family series создаёт один `family` item на каждое новое family;
- каждый planned family создаёт `revision` item для каждой пары
  `language/profile` из compatibility set;
- существующее released family создаёт `revision` item, если совместимая
  пара отсутствует или не имеет `availability=runnable`;
- `project-book-boundary` остаётся superseded/brief-only и не превращается в
  learner Run.

Таким образом текущий pinned release даёт **153 family items** и **437
revision items**. Эти 437 включают 16 недостающих пар у уже известных семейств
и 421 revision для новых семейств. Цифры вычисляются из exact IDs и manifest,
а не из названия задачи.

## Формат и порядок

Generated JSON: `task-portfolio/backlog-2026-08-27.json`.

- `itemId` стабилен и включает group, kind, family и compatibility;
- `priority=1` — закрыть compatibility gaps существующих семейств;
- `priority=2` — author reviewed TaskFamily;
- `priority=3` — author matching TaskRevision;
- `wave` и `batchPosition` разбивают очередь на batches максимум по 100;
- `productionReady` всегда `false` до нового immutable runtime release.

Новый family/revision item закрывается только после EN/RU brief, capability
outcome, rubric, sandbox limits, immutable hash и deterministic
pass/fail/error/failure-injection evidence. Один actor не может быть author и
final reviewer. Existing revisions не переписываются — выпускается новая
revision/release.

## Reproduction

```bash
cd /Users/sergeyzhechko/developer/fluent-interview/fluent-task-runtime
docker run --rm -v "$PWD":/src -w /src golang:1.24 go run ./cmd/portfolio-backlog
docker run --rm -v "$PWD":/src -w /src golang:1.24 go run ./cmd/portfolio-backlog --check
docker run --rm -v "$PWD":/src -w /src golang:1.24 go test ./...
```

Host checkout без установленного Go использует pinned `golang:1.24` image;
runtime Compose и image-manifest остаются отдельными production boundaries.
