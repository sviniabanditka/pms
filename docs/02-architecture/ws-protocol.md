# WS-протокол: live-дельты, подписки, presence

Code-ready формат realtime-канала (coder/websocket, ADR-0003 S4). Дополняет [realtime](realtime.md)
(инфраструктура) и [sync-protocol](sync-protocol.md) (семантика).

## Разделение каналов (важно)

| Канал | Транспорт | Что |
|---|---|---|
| **Bootstrap / догон** | ConnectRPC unary/stream (`SyncService`) | первичный срез, `Changes since=cursor` |
| **Live push + подписки + presence** | **raw WebSocket** | дельты в реальном времени, subscribe/unsubscribe, presence |
| **Команды (write)** | ConnectRPC unary | мутации (НЕ по WS) |

Причина: bidi подписки + low-latency push дёшевы на raw WS; bootstrap/догон — обычные запросы.
Фрейминг WS — **protobuf** (переиспользуем `Delta` из [proto](proto-commands.md)); JSON-режим допустим в dev для отладки.

---

## ws.proto

```proto
syntax = "proto3";
package pms.v1;
import "pms/v1/sync_service.proto"; // Delta

// Клиент → сервер
message ClientMessage {
  oneof msg {
    Subscribe   subscribe   = 1;
    Unsubscribe unsubscribe = 2;
    Ping        ping        = 3;
  }
}
message Subscribe   { repeated string channels = 1; } // 'team:{id}', 'issue:{id}', 'presence:{entity}'
message Unsubscribe { repeated string channels = 1; }
message Ping        { int64 ts = 1; }

// Сервер → клиент
message ServerMessage {
  oneof msg {
    DeltaBatch      deltas   = 1;
    PresenceUpdate  presence = 2;
    Pong            pong     = 3;
    ResyncRequired  resync   = 4;
    SubAck          sub_ack  = 5;
  }
}
message DeltaBatch     { repeated Delta deltas = 1; }          // коалесинг 50–100мс
message PresenceUpdate { string entity = 1; repeated string user_ids = 2; }
message Pong           { int64 ts = 1; }
message ResyncRequired { string reason = 1; }                 // клиент → SyncService.Changes/Bootstrap
message SubAck         { repeated string channels = 1; bool ok = 2; string error = 3; }
```

---

## Аутентификация апгрейда

- Токен (opaque, [auth](auth.md)) передаётся при WS-апгрейде (cookie или `?token=`); сервер валидирует сессию в Redis.
- При отзыве сессии (logout/бан) — соединение закрывается сервером (close code `4401`).
- Подписка на канал проверяется RBAC ([rbac](rbac.md)): нет доступа → `SubAck{ok:false}`.

## Жизненный цикл

```
WS upgrade (token) → auth ok → goroutine
C→S Subscribe{channels} → check RBAC → S→C SubAck
... мутации других клиентов → Redis pub/sub → S→C DeltaBatch (коалесинг)
C→S Ping (heartbeat ~30с) → S→C Pong;  таймаут → close
backpressure / разрыв → S→C ResyncRequired → клиент зовёт SyncService.Changes(since=cursor)
```

## Применение дельт (клиент, §sync-protocol 4)

- `DeltaBatch` → по каждой: `apply if version > local`; `delete`/`archive` → tombstone/флаг.
- Свои подтверждённые (ack по версии) изменения дельтой не откатываются.
- После применения батча — обновить локальный курсор (`max(seq)`).

## Close codes

| Код | Значение |
|---|---|
| `4401` | сессия отозвана/невалидна → реавторизация |
| `4408` | heartbeat timeout |
| `4429` | rate limit / слишком много каналов |
| `1011` | внутренняя ошибка → реконнект с backoff |

## Замечания
- Лимиты: каналов на соединение, соединений на пользователя — настраиваемые ([realtime](realtime.md)).
- permessage-deflate — включить при больших батчах, замерить (§realtime open-questions).
- Реконнект — экспоненциальный backoff + jitter; после реконнекта всегда `Changes(since=cursor)` перед доигрыванием очереди мутаций.
