# ADR-0005 — Сквозные конвенции данных

- Статус: accepted
- Дата: 2026-06-27
- Связано: §3.3, §7.4, §7.8 спеки; [ADR-0001](ADR-0001-deploy-model.md), [ADR-0002](ADR-0002-local-first-strategy.md)
- Практический справочник: [`../01-domain/00-conventions.md`](../01-domain/00-conventions.md)

## Решения

### PK — UUIDv7

Все доменные таблицы: `id uuid` PK, генерация **UUIDv7** (k-sortable → локальность по индексу,
дружелюбно к синку и пагинации по `id`). Внешний идентификатор issue `KEY-N` — отдельная пара
колонок (`team_id`, `number`), не PK. Client-generated UUID для оптимистичных мутаций (идемпотентность, §7.4).

### Изоляция арендаторов — app-level + Postgres RLS

- **App-level** (основной путь): каждая доменная таблица несёт `workspace_id`; единый query-guard в bun-слое всегда добавляет скоуп.
- **RLS** (сетка безопасности): политики на каждую таблицу по `current_setting('app.workspace_id')`; транзакция выставляет `SET LOCAL app.workspace_id`. Забытый `WHERE` физически не утечёт между тенантами.
- Исключение: глобальные таблицы без тенанта — `users` (учётка живёт в нескольких воркспейсах). Принадлежность — через `workspace_members`.

### Удаление — archive + soft-delete + async purge

- **archive** (`archived_at`): обратимое пользовательское состояние; объект жив, уходит из активных view (если view не просит archived). Undo мгновенный.
- **soft-delete** (`deleted_at`): удаление с окном Undo (toast). По истечении окна — async purge-джоба (River) физически удаляет.
- **hard-delete**: только из purge-джобы или GDPR-erasure (§9.4).
- Каскад: на soft-delete/archive — приложение каскадит на owned-детей (sub-issues, comments, attachments) пометкой; на purge — FK `ON DELETE CASCADE` для истинно owned.

### Версионирование и синк

- Каждая синкуемая таблица: `version bigint NOT NULL DEFAULT 0`, бампается при каждой мутации.
- Глобальный `change_log` (append-only): `(workspace_id, entity_type, entity_id, version, op, changed_fields jsonb, actor_id, at)` — источник дельт и догона `since=<cursor>` (§7.4). Курсор = монотонный `seq` в change_log.
- Конфликты: per-field LWW по `version`/`updated_at` (структурные поля); rich text — LWW по полю в MVP (ADR-0004 #9).

### Ручной порядок — fractional indexing

- Колонка `position text` (лексикографически сортируемый ключ, алгоритм fractional-indexing: `keyBetween(a,b)`).
- Перестановка элемента = генерация ключа между соседями, без перенумерации (§3.3).
- Скоуп уникальности порядка — по контексту (колонка board = `(view-grouping-value)`, backlog, `parent_id` для sub-issues).

### Время и типы

- Все метки времени — `timestamptz`, хранение в UTC.
- Rich text — `jsonb` (TipTap/ProseMirror JSON). Custom field values — `jsonb` (типизированы схемой поля). Настройки — `jsonb`.

## Последствия

- Пул соединений работает с `SET LOCAL` внутри транзакции (RLS) — учитываем в bun-обвязке.
- Каждая сущность в `01-domain` наследует стандартный набор колонок (см. справочник), описывает лишь свои.
- `change_log` + `version` обязательны с M0 (скелет синка).

## Открытые подвопросы

- Длина окна Undo и retention purge — в `02-architecture` (нотификации/джобы). Дефолт-черновик: Undo 10с (UI), purge через 30 дней.
