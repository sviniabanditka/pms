//go:build integration

package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/pms/backend/internal/jobs"
)

// seedWebhook вставляет вебхук + pending-доставку (через owner, bypass RLS). Возвращает (webhookID, deliveryID).
func seedWebhook(t *testing.T, ws, owner uuid.UUID, url, secret string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	whID, did := uuid.New(), uuid.New()
	mustExec(t, `INSERT INTO webhooks (id, workspace_id, url, secret, events, active, created_by) VALUES (?, ?, ?, ?, '{issue.create}', true, ?)`,
		whID, ws, url, secret, owner)
	mustExec(t, `INSERT INTO webhook_deliveries (id, webhook_id, workspace_id, event, status) VALUES (?, ?, ?, 'issue.create', 'pending')`,
		did, whID, ws)
	return whID, did
}

func deliverJob(did, whID, ws uuid.UUID) *river.Job[jobs.WebhookDeliverArgs] {
	return &river.Job[jobs.WebhookDeliverArgs]{Args: jobs.WebhookDeliverArgs{
		DeliveryID: did, WebhookID: whID, WorkspaceID: ws, Event: "issue.create",
		EntityType: "issue", EntityID: uuid.New().String(), Op: "create", At: time.Now().UTC(),
	}}
}

func TestWebhookDeliverWorker_SignsAndSucceeds(t *testing.T) {
	ws, owner := newWS(t)
	secret := "whsec_testsecret"
	var gotSig, gotEvent, gotDelivery string
	var gotBody []byte
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-PMS-Signature")
		gotEvent = r.Header.Get("X-PMS-Event")
		gotDelivery = r.Header.Get("X-PMS-Delivery")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer recv.Close()

	whID, did := seedWebhook(t, ws, owner, recv.URL, secret)
	worker := jobs.NewWebhookDeliverWorker(jobsDB, &http.Client{Timeout: 5 * time.Second})
	if err := worker.Work(context.Background(), deliverJob(did, whID, ws)); err != nil {
		t.Fatalf("deliver: %v", err)
	}

	if gotEvent != "issue.create" {
		t.Errorf("X-PMS-Event = %q, want issue.create", gotEvent)
	}
	if gotDelivery != did.String() {
		t.Errorf("X-PMS-Delivery = %q, want %s", gotDelivery, did)
	}
	// HMAC валиден: sha256=hmac(secret, body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(gotBody)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if gotSig != want {
		t.Errorf("signature mismatch:\n got %s\nwant %s", gotSig, want)
	}
	// доставка отмечена success + код 200 + attempts=1
	if s := scalar[string](t, `SELECT status FROM webhook_deliveries WHERE id = ?`, did); s != "success" {
		t.Errorf("delivery status = %q, want success", s)
	}
	if c := scalar[int](t, `SELECT response_code FROM webhook_deliveries WHERE id = ?`, did); c != 200 {
		t.Errorf("response_code = %d, want 200", c)
	}
	if a := scalar[int](t, `SELECT attempts FROM webhook_deliveries WHERE id = ?`, did); a != 1 {
		t.Errorf("attempts = %d, want 1", a)
	}
}

func TestWebhookDeliverWorker_FailureSignalsRetry(t *testing.T) {
	ws, owner := newWS(t)
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer recv.Close()

	whID, did := seedWebhook(t, ws, owner, recv.URL, "whsec_x")
	worker := jobs.NewWebhookDeliverWorker(jobsDB, &http.Client{Timeout: 5 * time.Second})
	// 5xx → воркер возвращает ошибку (River ретраит)
	if err := worker.Work(context.Background(), deliverJob(did, whID, ws)); err == nil {
		t.Fatal("non-2xx must return an error to trigger a retry")
	}
	if s := scalar[string](t, `SELECT status FROM webhook_deliveries WHERE id = ?`, did); s != "failed" {
		t.Errorf("delivery status = %q, want failed", s)
	}
	if c := scalar[int](t, `SELECT response_code FROM webhook_deliveries WHERE id = ?`, did); c != 500 {
		t.Errorf("response_code = %d, want 500", c)
	}
}

func TestWebhookDeliverWorker_DeletedHookNoRetry(t *testing.T) {
	ws, owner := newWS(t)
	// доставка ссылается на soft-deleted вебхук → mark failed, без ретрая (nil)
	did, whID := uuid.New(), uuid.New()
	mustExec(t, `INSERT INTO webhooks (id, workspace_id, url, secret, events, active, deleted_at, created_by) VALUES (?, ?, 'http://x', 's', '{issue.create}', true, now(), ?)`,
		whID, ws, owner)
	mustExec(t, `INSERT INTO webhook_deliveries (id, webhook_id, workspace_id, event, status) VALUES (?, ?, ?, 'issue.create', 'pending')`,
		did, whID, ws)

	worker := jobs.NewWebhookDeliverWorker(jobsDB, &http.Client{Timeout: 2 * time.Second})
	if err := worker.Work(context.Background(), deliverJob(did, whID, ws)); err != nil {
		t.Fatalf("deleted hook must not retry (nil err), got %v", err)
	}
	if s := scalar[string](t, `SELECT status FROM webhook_deliveries WHERE id = ?`, did); s != "failed" {
		t.Errorf("status = %q, want failed", s)
	}
}
