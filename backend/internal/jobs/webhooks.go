package jobs

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/domain/webhooks"
	"github.com/pms/backend/internal/models"
)

// --- Dispatcher: читает change_log по seq-курсору, ставит delivery-задачи ---

type WebhookDispatchArgs struct{}

func (WebhookDispatchArgs) Kind() string { return "webhook_dispatch" }

type WebhookDispatchWorker struct {
	river.WorkerDefaults[WebhookDispatchArgs]
	db *bun.DB // pms_jobs (BYPASSRLS) — cross-tenant скан
}

func (w *WebhookDispatchWorker) Work(ctx context.Context, _ *river.Job[WebhookDispatchArgs]) error {
	var last int64
	if err := w.db.NewSelect().Table("webhook_cursor").Column("last_seq").Where("only_one").Scan(ctx, &last); err != nil {
		return err
	}
	var changes []struct {
		Seq         int64      `bun:"seq"`
		WorkspaceID uuid.UUID  `bun:"workspace_id"`
		EntityType  string     `bun:"entity_type"`
		EntityID    uuid.UUID  `bun:"entity_id"`
		Op          string     `bun:"op"`
		ActorID     *uuid.UUID `bun:"actor_id"`
		At          time.Time  `bun:"at"`
	}
	if err := w.db.NewSelect().TableExpr("change_log AS cl").
		ColumnExpr("cl.seq, cl.workspace_id, cl.entity_type, cl.entity_id, cl.op, cl.actor_id, cl.at").
		Where("cl.seq > ?", last).OrderExpr("cl.seq").Limit(500).Scan(ctx, &changes); err != nil {
		return err
	}
	if len(changes) == 0 {
		return nil
	}
	client := river.ClientFromContext[pgx.Tx](ctx)
	hooksByWS := map[uuid.UUID][]models.Webhook{} // кэш активных вебхуков по воркспейсу
	maxSeq := last
	for _, c := range changes {
		if c.Seq > maxSeq {
			maxSeq = c.Seq
		}
		hooks, ok := hooksByWS[c.WorkspaceID]
		if !ok {
			_ = w.db.NewSelect().Model(&hooks).Where("workspace_id = ?", c.WorkspaceID).
				Where("active").Where("deleted_at IS NULL").Scan(ctx)
			hooksByWS[c.WorkspaceID] = hooks
		}
		ev := c.EntityType + "." + c.Op
		for _, h := range hooks {
			if !webhooks.Matches(h.Events, c.EntityType, c.Op) {
				continue
			}
			did := uuid.New()
			if _, err := w.db.NewInsert().Model(&models.WebhookDelivery{
				ID: did, WebhookID: h.ID, WorkspaceID: c.WorkspaceID, Event: ev, Status: "pending", CreatedAt: time.Now().UTC(),
			}).Exec(ctx); err != nil {
				continue
			}
			actor := ""
			if c.ActorID != nil {
				actor = c.ActorID.String()
			}
			_, _ = client.Insert(ctx, WebhookDeliverArgs{
				DeliveryID: did, WebhookID: h.ID, WorkspaceID: c.WorkspaceID,
				Event: ev, EntityType: c.EntityType, EntityID: c.EntityID.String(), Op: c.Op, Actor: actor, At: c.At,
			}, &river.InsertOpts{MaxAttempts: 5})
		}
	}
	_, err := w.db.NewUpdate().Table("webhook_cursor").Set("last_seq = ?", maxSeq).Where("only_one").Exec(ctx)
	return err
}

// --- Delivery: POST с HMAC-подписью; ошибка → River-ретрай ---

type WebhookDeliverArgs struct {
	DeliveryID  uuid.UUID `json:"delivery_id"`
	WebhookID   uuid.UUID `json:"webhook_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Event       string    `json:"event"`
	EntityType  string    `json:"entity_type"`
	EntityID    string    `json:"entity_id"`
	Op          string    `json:"op"`
	Actor       string    `json:"actor"`
	At          time.Time `json:"at"`
}

func (WebhookDeliverArgs) Kind() string { return "webhook_deliver" }

type WebhookDeliverWorker struct {
	river.WorkerDefaults[WebhookDeliverArgs]
	db   *bun.DB
	http *http.Client
}

func (w *WebhookDeliverWorker) Work(ctx context.Context, job *river.Job[WebhookDeliverArgs]) error {
	a := job.Args
	var wh models.Webhook
	if err := w.db.NewSelect().Model(&wh).Where("id = ?", a.WebhookID).Where("deleted_at IS NULL").Scan(ctx); err != nil {
		w.mark(ctx, a.DeliveryID, "failed", 0) // вебхук удалён — без ретрая
		return nil
	}
	payload := map[string]any{
		"id": a.DeliveryID, "event": a.Event, "workspaceId": a.WorkspaceID, "op": a.Op,
		"actor": a.Actor, "at": a.At, "entity": map[string]string{"type": a.EntityType, "id": a.EntityID},
	}
	body, _ := json.Marshal(payload)
	mac := hmac.New(sha256.New, []byte(wh.Secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PMS-Event", a.Event)
	req.Header.Set("X-PMS-Signature", "sha256="+sig)
	req.Header.Set("X-PMS-Delivery", a.DeliveryID.String())

	resp, err := w.http.Do(req)
	code := 0
	if resp != nil {
		code = resp.StatusCode
		_ = resp.Body.Close()
	}
	if err != nil || code < 200 || code >= 300 {
		w.mark(ctx, a.DeliveryID, "failed", code)
		if err != nil {
			return err
		}
		return fmt.Errorf("webhook %s returned %d", a.WebhookID, code) // → ретрай
	}
	w.markSuccess(ctx, a.DeliveryID, code)
	return nil
}

func (w *WebhookDeliverWorker) mark(ctx context.Context, id uuid.UUID, status string, code int) {
	q := w.db.NewUpdate().Table("webhook_deliveries").
		Set("attempts = attempts + 1").Set("status = ?", status).Where("id = ?", id)
	if code > 0 {
		q.Set("response_code = ?", code)
	}
	_, _ = q.Exec(ctx)
}

func (w *WebhookDeliverWorker) markSuccess(ctx context.Context, id uuid.UUID, code int) {
	_, _ = w.db.NewUpdate().Table("webhook_deliveries").
		Set("attempts = attempts + 1").Set("status = 'success'").Set("response_code = ?", code).
		Set("delivered_at = now()").Where("id = ?", id).Exec(ctx)
}
