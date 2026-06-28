# Домен: Platform

**Notification · NotificationPreference**. Наследует [конвенции](00-conventions.md).
Связано: §3.2, §8.1.8 (Inbox). Генерация/доставка — в `02-architecture/notifications.md`.

---

## Notification (Inbox)

Персональный поток уведомлений (§4.2 Inbox).

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `user_id` | uuid FK→users NOT NULL | получатель |
| `type` | text NOT NULL | `assigned \| mentioned \| state_changed \| commented \| due_soon \| subscribed_update` |
| `entity_type` | text NOT NULL | `issue \| doc \| project \| comment` |
| `entity_id` | uuid NOT NULL | |
| `actor_id` | uuid FK→users NULL | кто инициировал |
| `payload` | jsonb | денорм. для рендера строки Inbox без доп. запросов |
| `read_at` | timestamptz NULL | |
| `snoozed_until` | timestamptz NULL | отложить (опц. MVP) |
| `at` | timestamptz NOT NULL | |
| стандартные | | `version` (read-статус синкается); без archive (есть read/delete) |

**Инварианты**
- Скоупится по `(user_id, workspace_id)`; Inbox = фильтр по непрочитанным/всем.
- Генерация — из доменных событий (Activity/Subscription/mention) через River-джобы; дедуп по `(user, entity, type)` в окне.
- `due_soon` — джоба по расписанию сканирует `due_date` подписанных/назначенных issues.
- Прочтение/«прочитать всё» — мутации, отражаются в `change_log` (синк между клиентами).

---

## NotificationPreference

Настройки доставки per-user (§8.1.8).

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `user_id` | uuid FK→users NOT NULL | |
| `type` | text NOT NULL | тип нотификации (как выше) |
| `channel` | text NOT NULL | `in_app \| email` (push — позже) |
| `enabled` | boolean NOT NULL DEFAULT true | |
| стандартные | | `version` |

**Инварианты**
- UNIQUE `(user_id, type, channel)`; отсутствие строки = дефолт (enabled).
- `in_app` всегда формирует Notification; `channel=email` управляет доставкой email-джобой.
- Email-доставка в MVP — минимум (assigned/mentioned/due_soon); расширяется позже.

---

## Связи кластера

```
notifications        *──1 users(recipient) / users(actor) ── polymorphic entity
notification_prefs   *──1 users
```

## Открытые микро-вопросы

- Snooze/«later» в MVP — опционально (поле есть, UI можно отложить).
- Группировка нотификаций (digest) — после MVP.
