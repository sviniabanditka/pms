-- emailed_at: метка, что по уведомлению уже поставлена/выполнена задача отправки письма (sweep-джоб)
ALTER TABLE notifications ADD COLUMN emailed_at timestamptz;
CREATE INDEX notifications_unemailed ON notifications (at) WHERE emailed_at IS NULL;
