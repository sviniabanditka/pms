ALTER TABLE calendar_events ADD COLUMN recurrence text NOT NULL DEFAULT 'none'; -- none|daily|weekly|monthly
ALTER TABLE calendar_events ADD COLUMN recurrence_until timestamptz;
