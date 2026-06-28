# Реестр технического долга

Сознательные упрощения. Сверяемся каждую веху; закрытое → в «Закрыто».

## A. Реально закрываемые сейчас (план добивания)

| ID | Долг | Приоритет |
|----|------|-----------|

## B. Отложены по триггеру (не «скелет-долг», ждут события)

| ID | Долг | Триггер |
|----|------|---------|
| D21 | Серверный Filter AST → SQL/OpenSearch компилятор. (Базовый `QueryService.Search` ILIKE по issues/docs/channels — есть; полный Filter-AST компилятор/контент-индекс — нет) | масштаб/экспорт |
| D2 | CSRF-токен (сейчас SameSite=Lax) | прод-деплой (same-origin) |
| D3 | Токен в localStorage (не httpOnly) | прод |
| D4 | CORS `*` | прод (строгий origin) |
| D16 | Refresh-токен | мобильные клиенты |
| D19 | Нативный DnD (не pragmatic-drag-and-drop) | touch/a11y |
| D20 | Float-midpoint позиции | исчерпание точности → fractional-строки |
| D14 | UI на Nuxt UI; скелетоны/стабильный gutter добавлены. Остаток: полный a11y/perf-аудит, пустые состояния местами | итеративно |
| D13 | Интеграционные тесты per-field LWW/конфликтов | при testcontainers-инфре (с DB-тестами) |
| D41 | **Chats**: edit/delete/threads/reactions/typing/пагинация — есть. Остаток: md-рендер свой (не полный CommonMark, нет картинок/таблиц/код-подсветки), нет read-receipts/вложений/поиска/упоминаний, realtime точечный рефетч; @mentions есть. typing throttle 2.5с | при доводке |
| D42 | **Comments**: add/edit/delete/нотиф/@mentions есть. Нет реакций/вложений/тредов в комментах, пагинации | при доводке |
| D51 | Workspace Settings modal (General/Members/Teams/Labels/Sidebar). Rename ws (UpdateWorkspaceSettings). Нет: удаление ws, billing, удаление лейблов/команд, аватар/лого | функционал |
| D55 | WebRTC mesh деградирует на >4 участниках (N² соединений). SFU (mediasoup/LiveKit/Pion) | когда нужны крупные митинги |
| D56 | Митинг-сигналинг шлётся в весь ws-канал (клиент фильтрует toUserId). Таргетная доставка (per-user pub/sub канал) | масштаб/приватность |
| D17 | Репозиторий не закоммичен | по команде пользователя |

## Закрыто

| D63 | **Burndown snapshots (D29) + Mailpit**. Migration 000026 cycle_snapshots (cycle_id, day text, remaining/total points+count, PK cycle_id+day). River CycleSnapshotWorker (периодик SNAPSHOT_EVERY=24ч, RunOnStart): по активным циклам (CURRENT_DATE in [starts_on,ends_on]) считает остаток (estimate/count, статус не completed/canceled) → upsert слепок дня. Bootstrap-чанк cycle_snapshot. Frontend: store.cycleBurndown getter, Burndown.vue (SVG: actual + ideal линии, «копится ≥2 дня»), заменил синтетический график в cycle-панели. **Mailpit** в docker-compose (SMTP :1025, UI :8025), dev-дефолт SMTP_HOST=localhost:1025. Фикс mailer: envelope MAIL FROM = голый адрес (было «Имя <email>» → 501). E2E: Mailpit получил 3 письма (вкл due_soon), snapshot создан для активного цикла. | функционал | snapshot по дню (не intraday); ideal — линейная от первого слепка; нет бэкфилла истории |
| D62 | **River async (D1+D38 закрыты)** (migration 000025 notifications.emailed_at). River-очередь на Postgres (pgxpool на postgres-роли, миграции River на старте сервера). Воркеры: EmailWorker (mailer: SMTP если SMTP_HOST, иначе dev-лог), DueSoonWorker (периодик: задачи со сроком ≤24ч + assignee + не completed → in-app due_soon + realtime-дельта; dedupe 20ч), NotifySweepWorker (периодик: un-emailed уведомления ЛЮБОГО типа → email-задачи + emailed_at). Конфиг SMTP_*/DUE_SOON_INTERVAL/EMAIL_SWEEP_EVERY. jobsDB (BYPASSRLS) для кросс-тенантного скана. Graceful river.Stop. E2E: sweep→8 email-задач completed (dev-лог alice/bob/eve), due_soon_scan создал due-soon-уведомление, 8/8 emailed. | функционал | server-side prefs не учитываются (клиентские localStorage); email-тело generic; SMTP не настроен (dev-лог); burndown-снапшоты (D29) не сделаны |
| D61 | **Chat unread** (migration 000024 channel_reads). Bootstrap-чанк channel_unread [{channelId,count,lastReadAt}] для actor (count = чужие сообщения после last_read, с visibility-фильтром public/member). ChatService.MarkChannelRead (upsert read-маркера). Message-delta теперь с Actor=author → клиент не считает своё непрочитанным. Frontend: бейдж総 в рейле (totalUnread), жирный+бейдж непрочитанных каналов/DM (dmChannelWith), разделитель «New messages» + автоскролл на первый непрочитанный при открытии, mark-read при открытии/получении в активном, bumpUnread в фоне по WS. | функционал | счётчик клиентский между bootstrap'ами (bump на message-delta); divider по строковому compare ISO; нет «отметить все прочитанным» |
| D59 | **Аттачменты UX**: (1) превью в композере — PendingAttachments.vue (картинки=тумбы, файлы=плашки) в чате+комментах; вставка/пик → превью, шлётся ПРИКРЕПЛённым (chat sendMessageAttached, comment addCommentAttached; пустое тело разрешено). (2) Файлы=плашки с иконкой по типу (video/audio/pdf/zip/doc/xls/...) + размер + кнопка скачать (`/files/view?dl=1` → Content-Disposition attachment). (3) Лайтбокс: useLightbox+ImageLightbox.vue — клик по картинке (рендер md или превью аттача) → fullscreen-модалка с отступом, object-contain (маленькая картинка = маленькая модалка). Фикс рендера: markdown stash картинок/ссылок/чипов/кода в плейсхолдеры до bold/italic — `_` в URL ломал src. | функционал | comment attach рендерится readonly. D60: описание+доки переведены на attachment-модель (paste/скрепка→вложение, не inline-md); доки получили секцию Attachments (entityType doc). Аттачменты теперь: issue/comment/message/doc |
| D58 | **Inline @/# + унфёрл + inline-картинки + превью-фикс**. Стабильный роут `GET /files/view/<key>` (бэкенд **стримит** байты из S3 — один origin, без redirect; <img> кросс-портовый 302 не грузил; без auth — ключ uuid-непредсказуем) → чинит превью (presigned истекал) + даёт вечные URL для эмбедов. Avatar/Attachments → fileViewUrl. Чат: вложения теперь **прикрепляются к сообщению** (pendingFiles-чипы, пустое тело разрешено бэкендом), картинки (paste/pick) — **в тело** `![]()`. Inline-картинки в description+docs (paste/toolbar). `useMention` composable + MentionMenu: `@`-пикер юзеров (аватар+фильтр) и `#`-пикер сущностей (issue→`#KEY-N`, doc/meeting/channel→`#[label](type:id)`) в чате/описании/доках/комментах (↑↓/Enter/Tab/Esc). Markdown: картинки, mention-чипы, `#KEY-N` чипы, generic ref-чипы, унфёрл голых внутренних ссылок (/issue//doc//chat/); клик по чипу открывает сущность. | функционал | унфёрл markdown-link формы `[t](.../issue/x)` не идеален (редко); mention в description не шлёт нотификацию (нет parse в SetDescription); #-чип без title/статуса (только identifier) |
| D57 | **S3 + attachments + avatars/logos** (migration 000023). Self-hosted **Garage** в docker-compose (distroless → bootstrap `make s3-init` с хоста: layout+key import детерминированные креды+bucket; CORS не нужен). Backend: internal/storage (aws-sdk-go-v2, path-style, presign GET с подменой endpoint→publicBase), HTTP `POST /files/upload` (multipart через бэкенд → нет CORS-префлайта на Garage), AttachmentService (Create/Delete/GetURL/SetUserAvatar/SetWorkspaceLogo), attachments-чанк в bootstrap, users.avatar_key/workspaces.logo_key. Frontend: lib/upload, useFileUrl (кэш presigned 1ч), Avatar.vue/Attachments.vue, аттачменты в задаче+чате (paperclip→sendFileMessage), аватарки/лого в настройках + отрисовка (чат/DM/комменты/топбар). E2E curl: upload→attach→presign→fetch контент→avatar все 200. | функционал | presign на каждый рендер (кэш сессии); upload через бэкенд (не прямой PUT); нет thumbnail-генерации/антивируса/лимита типов |
| D54 | **Meetings** (полный pipeline, migration 000022): сущность meeting (kind scheduled\|huddle, status scheduled→live→ended) + meeting_participants (role/state). RPC MeetingService: CreateMeeting (круг юзеров), StartHuddle (из чата — весь канал/DM 1-1, реюз live-huddle канала), JoinMeeting (→ peer_ids для mesh), LeaveMeeting (last leave→ended), EndMeeting, SendSignal. WebRTC **audio+video mesh**: joiner-initiates offer (анти-glare), сигналинг offer/answer/ice/bye через WS воркспейса (publisher.PublishSignal, gateway пробрасывает не-дельты сразу; клиент фильтрует по toUserId). Frontend: composable useMeeting (singleton, RTCPeerConnection per peer, STUN google), MeetingCall.vue (floating video grid + mute/cam/leave), VideoTile.vue. UI: раздел Meetings (рейл+страница, scheduled/live карточки), huddle-кнопка в шапке чата, create-модалка (title/datetime/participants), глобальный «+». E2E curl: register→ws→channel→huddle→join→scheduled→signal→leave все 200; huddle авто-ended при последнем leave. | функционал | Mesh ~4 чел (без SFU — D55); signaling broadcast на всех клиентов воркспейса (фильтр на клиенте); нет screen-share/записи/ack-presence speaking-detection (мок) |
| D53 | **Issue description** (markdown): proto UpdateIssueRequest.description=9 → SetDescription (jsonb {"md":...}) → bootstrap (model тег `json:"-"`→`description,omitempty`) → store.setIssueDescription → UI-блок под заголовком (пусто/edit-textarea/rendered md). Detail-панель: ghost-контролы свойств просторнее (min-h-8), «⋯»-дропдаун (copy link/id, mark complete, duplicate, archive) | функционал/UI |
| D52 | **Bugfix**: UInput `@change` в Nuxt UI v2 эмитит **значение**, не нативный Event → `e.target.value`=undefined. Due date/Estimate/title/doc-title/state-name/team-name/label-name молча не сохранялись. Исправлено на `(v) => ...v`. + капитализация значений селектов (labelOpts: статусы/роли/категории/типы связей/тип виджета/group-by) + размер селектов xs→sm. Labels/Relations — теперь модалки, не инлайн-формы. Multiselect custom field: UButton soft (светлый фон на тёмном) → нативные тёмные кнопки. Select custom field: пустая опция теперь «—», не пустая строка. Проверено: бэкенд нормализует estimate≤0/пустой assignee/milestone/date → NULL, поэтому optimistic `||null` vs RPC `||''` консистентны | багфикс/UI |
| ID | Долг | Закрыт |
|----|------|--------|
| — | json-теги, MoveTeam, коалесинг дельт, IndexedDB-персист | долги после M0 |
| D8 | Endpoint «мои воркспейсы» + свитчер | после M3 |
| D12 | Даты issue правятся в UI (detail Date + UpdateIssue) | M5 |
| D7 | applyChunk-merge (пишет только изменённые по JSON-diff, чистит удалённые) → нет re-render churn на дельтах | закрытие долгов |
| D6/D18 | сняты вместе с D7 (полный re-bootstrap остаётся, но без лишнего ререндера) | закрытие долгов |
| D22 | Группировка List по status/priority/assignee (issueGroups + селектор) | закрытие долгов |
| D34 | cfv-изменения детектятся JSON-diff merge (version-инкремент не нужен для синка) | закрытие долгов |
| D31 | Calendar week-view (тоггл Month/Week) | закрытие долгов |
| D36 | ChartWidget (бар-чарт issues по категориям статусов) | закрытие долгов |
| D35 | Dashboard виджеты: drag-reorder (y) + widen/shrink (нативный DnD; 2D-резайза нет) | закрытие долгов |
| D39 | Docs editor: md-тулбар + Edit/Split/Preview + автосейв (без TipTap) | закрытие долгов |
| D33 | Custom field people/relationship пикеры (юзер/issue селекты) | закрытие долгов |
| D30 | Milestones: CRUD + список + страница вехи (задачи/прогресс/overdue) + привязка issue + группировка списка по вехе | закрытие долгов |
| D15 | synccheck удалён | добивание долгов |
| D24 | Цель связи через datalist-поиск | добивание долгов |
| D25 | childProgress по категории `completed` (не completedAt) | добивание долгов |
| D26/27/28 | project-долги — устарели (projects удалены, D40) | добивание долгов |
| D37 | `[]byte` jsonb-поля → `json.RawMessage` (нет base64) + сброс старых строк | добивание долгов |
| D40 | Остатки projects в store/index убраны (таблица в БД дремлет) | добивание долгов |
| D5 | CI: .github/workflows/ci.yml (go vet/build/test, buf lint, vitest) | добивание долгов |
| D23 | Vitest + tests/filter.test.ts (7 кейсов Filter AST) | добивание долгов |
| D9 | per-field LWW реализован (merge присланных полей; strict-reject — non-goal) | by design |
| D50 | Маршрутизация: catch-all `[...slug].vue`, путь↔состояние (issue/team/cycle/doc/chat/dashboard/inbox/my), deep-link + copy-link. Доп: query-параметры фильтров/layout в URL не сохраняются | функционал |
| D48 | Bulk-действия в List (статус/ассайни/лейбл/архив через цикл клиентом; batch-RPC нет) | функционал |
| D49 | Cycle points-rollup + velocity (клиентский расчёт по estimate; снапшотов нет — D29) | функционал |
| D46 | Workflow statuses CRUD (create/rename/recategory/delete-если-пуст) + UI-модалка; reorder статусов не сделан (position append-only) | функционал |
| D47 | Estimate (SetEstimate + поле в detail); cycle velocity/points-rollup не считается | функционал |
| D43 | @mentions (токен @[Name](id), нотиф mention, рендер; picker в чате) | функционал |
| D44 | Members: InviteMember (существующий→участник, новый→pending invite) + UpdateMemberRole + UI-модалка; AcceptInvitation не реализован (нет email/токен-флоу) | функционал |
| D45 | Activity feed задачи из change_log (op/actor/время; changedFields не разворачиваются в человекочитаемый дифф) | функционал |
| D32 | Фильтр по лейблам в FilterBar (по custom field — всё ещё нет) | частично |
| D11 | Assignee-пикер (detail-селект + аватары в строках/картах; users в bootstrap) | функционал |
| D10 | Labels: CreateLabel (TeamService) + Add/RemoveLabel + UI (чипы/пикер/создание, пилюли в списках) | функционал |
