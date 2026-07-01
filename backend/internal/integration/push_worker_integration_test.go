//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/pms/backend/internal/jobs"
)

func pushJob() *river.Job[jobs.PushDispatchArgs] {
	return &river.Job[jobs.PushDispatchArgs]{Args: jobs.PushDispatchArgs{}}
}

func TestPushDispatchWorker_SendsAndMarks(t *testing.T) {
	ws, user := newWS(t)
	// device-токен + свежее непрочитанное уведомление
	mustExec(t, `INSERT INTO device_tokens (id, user_id, token, platform) VALUES (?, ?, 'ExponentPushToken[ok]', 'ios')`, uuid.New(), user)
	nid, ent := uuid.New(), uuid.New()
	mustExec(t, `INSERT INTO notifications (id, workspace_id, user_id, type, entity_type, entity_id, at) VALUES (?, ?, ?, 'assigned', 'issue', ?, now())`,
		nid, ws, user, ent)

	var gotBody []byte
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"status":"ok"}]}`))
	}))
	defer recv.Close()
	old := jobs.ExpoPushURL
	jobs.ExpoPushURL = recv.URL
	defer func() { jobs.ExpoPushURL = old }()

	worker := jobs.NewPushDispatchWorker(jobsDB, &http.Client{Timeout: 5 * time.Second})
	if err := worker.Work(context.Background(), pushJob()); err != nil {
		t.Fatalf("push: %v", err)
	}

	// Expo получил сообщение с нашим токеном + data.entityId
	var msgs []map[string]any
	if err := json.Unmarshal(gotBody, &msgs); err != nil || len(msgs) != 1 {
		t.Fatalf("expo body = %s (%v)", gotBody, err)
	}
	if msgs[0]["to"] != "ExponentPushToken[ok]" {
		t.Errorf("to = %v", msgs[0]["to"])
	}
	data, _ := msgs[0]["data"].(map[string]any)
	if data["entityId"] != ent.String() {
		t.Errorf("data.entityId = %v, want %s", data["entityId"], ent)
	}
	// уведомление помечено pushed_at
	if n := scalar[int](t, `SELECT count(*) FROM notifications WHERE id = ? AND pushed_at IS NOT NULL`, nid); n != 1 {
		t.Error("notification must be marked pushed_at")
	}
}

func TestPushDispatchWorker_PrunesDeadToken(t *testing.T) {
	ws, user := newWS(t)
	mustExec(t, `INSERT INTO device_tokens (id, user_id, token, platform) VALUES (?, ?, 'ExponentPushToken[dead]', 'android')`, uuid.New(), user)
	mustExec(t, `INSERT INTO notifications (id, workspace_id, user_id, type, entity_type, entity_id, at) VALUES (?, ?, ?, 'mentioned', 'issue', ?, now())`,
		uuid.New(), ws, user, uuid.New())

	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"status":"error","details":{"error":"DeviceNotRegistered"}}]}`))
	}))
	defer recv.Close()
	old := jobs.ExpoPushURL
	jobs.ExpoPushURL = recv.URL
	defer func() { jobs.ExpoPushURL = old }()

	worker := jobs.NewPushDispatchWorker(jobsDB, &http.Client{Timeout: 5 * time.Second})
	if err := worker.Work(context.Background(), pushJob()); err != nil {
		t.Fatalf("push: %v", err)
	}
	// мёртвый токен удалён
	if n := scalar[int](t, `SELECT count(*) FROM device_tokens WHERE token = 'ExponentPushToken[dead]'`); n != 0 {
		t.Error("DeviceNotRegistered token must be pruned")
	}
}
