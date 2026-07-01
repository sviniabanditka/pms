-- Push: device-токены (Expo push tokens) + отметка отправки на уведомлениях.
-- БЕЗ RLS: user-scoped (по user_id), читается воркером кросс-тенантно (pms_jobs).
CREATE TABLE device_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      text NOT NULL UNIQUE,          -- ExponentPushToken[...]
    platform   text NOT NULL DEFAULT '',      -- ios | android
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX device_tokens_user ON device_tokens (user_id);

-- когда по уведомлению отправлен push (NULL = ещё нет). Диспетчер сканит IS NULL.
ALTER TABLE notifications ADD COLUMN pushed_at timestamptz;
