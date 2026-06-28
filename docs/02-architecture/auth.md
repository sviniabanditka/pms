# Архитектура: Аутентификация и сессии

Решение — opaque-токены в Redis ([ADR-0006](../decisions/ADR-0006-auth.md)).
Связано: §7.2, §9.3, [identity-access](../01-domain/identity-access.md), [realtime](realtime.md).

---

## 1. Токены и сессии

- **Access-токен** — непрозрачная случайная строка (≥256 бит энтропии), отдаётся в httpOnly secure cookie (web) или заголовке.
- **Сессия в Redis**: ключ `session:{token}` → `{ user_id, workspace_ids, created_at, expires_at, ua/ip }`, TTL = время жизни сессии.
- **Refresh**: скользящее продление TTL при активности; абсолютный лимит (re-login по истечении).
- Проверка запроса = lookup `session:{token}`; короткий in-memory кэш на инстансе (TTL ~секунды) против частых lookups.

## 2. Поток

```
login (email+argon2id) → создать session:{token} в Redis → set cookie
request → middleware: lookup session → inject user/ctx → (RBAC далее)
WS upgrade → тот же токен в query/cookie → проверка сессии → goroutine
logout / ban / смена пароля → DEL session(s) → разрыв WS
```

## 3. Безопасность (§9.3)

- Пароли — **argon2id** (`users.password_hash`).
- Rate limiting на login/команды (защита от перебора); CSRF-защита для cookie-flow.
- Отзыв всех сессий пользователя — скан/индекс `user_sessions:{user_id}` → set токенов.
- 2FA, OAuth/SAML/SCIM — после MVP, поверх этой модели.

## 4. Связь с tenancy

- Сессия не привязана к одному воркспейсу: `workspace_ids` — где пользователь состоит.
- Активный воркспейс — из URL (`/w/:slug`); RLS-контекст `app.workspace_id` выставляется per-request (ADR-0005) после проверки членства.

## Открытые микро-вопросы

- TTL сессии (черновик: 30 дней скользящих, абсолют 90) — подтвердить.
- Отдельный refresh-токен vs только продление access — для MVP достаточно скользящего access; refresh при добавлении мобильных клиентов.
