-- AI петля фидбэка: лог accept/dismiss подсказок → сигнал для ретрейна.
CREATE TABLE ai_feedback (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL,
    actor_id     uuid,
    kind         text NOT NULL,   -- priority | estimate | assignee | label | slip | similar
    entity_id    uuid,            -- issue
    suggested    text,            -- предложенное значение (как текст)
    action       text NOT NULL,   -- applied | dismissed
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ai_feedback_ws_kind ON ai_feedback (workspace_id, kind, action);
ALTER TABLE ai_feedback ENABLE ROW LEVEL SECURITY;
CREATE POLICY ai_feedback_tenant ON ai_feedback
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);
