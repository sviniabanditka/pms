package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/models"
)

// Push-диспетчер: сканит новые непрочитанные уведомления (pushed_at IS NULL), шлёт
// Expo push на устройства пользователя, помечает pushed_at. Best-effort; мёртвые
// токены (DeviceNotRegistered) удаляет. Зеркало NotifySweepWorker, но для push.

// ExpoPushURL — переопределяется в тестах (httptest-ресивер).
var ExpoPushURL = "https://exp.host/--/api/v2/push/send"

type PushDispatchArgs struct{}

func (PushDispatchArgs) Kind() string { return "push_dispatch" }

type PushDispatchWorker struct {
	river.WorkerDefaults[PushDispatchArgs]
	db   *bun.DB // pms_jobs (BYPASSRLS)
	http *http.Client
}

// человекочитаемый текст пуша по типу уведомления
var pushBody = map[string]string{
	"due_soon":      "An issue is due soon",
	"assigned":      "You were assigned an issue",
	"mentioned":     "You were mentioned",
	"commented":     "New comment on your issue",
	"state_changed": "Issue status changed",
}

type expoMessage struct {
	To    string            `json:"to"`
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data,omitempty"`
}

type expoResponse struct {
	Data []struct {
		Status  string `json:"status"`
		Details struct {
			Error string `json:"error"`
		} `json:"details"`
	} `json:"data"`
}

func (w *PushDispatchWorker) Work(ctx context.Context, _ *river.Job[PushDispatchArgs]) error {
	var notes []models.Notification
	if err := w.db.NewSelect().Model(&notes).
		Where("pushed_at IS NULL AND read_at IS NULL").
		Where("at > now() - interval '1 day'").
		OrderExpr("at").Limit(200).Scan(ctx); err != nil {
		return err
	}
	if len(notes) == 0 {
		return nil
	}

	// device-токены получателей
	userSet := map[uuid.UUID]struct{}{}
	for _, n := range notes {
		userSet[n.UserID] = struct{}{}
	}
	userIDs := make([]uuid.UUID, 0, len(userSet))
	for u := range userSet {
		userIDs = append(userIDs, u)
	}
	var devices []models.DeviceToken
	_ = w.db.NewSelect().Model(&devices).Where("user_id IN (?)", bun.In(userIDs)).Scan(ctx)
	tokensByUser := map[uuid.UUID][]string{}
	for _, d := range devices {
		tokensByUser[d.UserID] = append(tokensByUser[d.UserID], d.Token)
	}

	var messages []expoMessage
	var msgTokens []string // токен, использованный для messages[i] — для прунинга
	for _, n := range notes {
		body := pushBody[n.Type]
		if body == "" {
			body = "New notification"
		}
		for _, tok := range tokensByUser[n.UserID] {
			messages = append(messages, expoMessage{
				To: tok, Title: "stdtask", Body: body,
				Data: map[string]string{"type": n.Type, "entityType": n.EntityType, "entityId": n.EntityID.String(), "workspaceId": n.WorkspaceID.String()},
			})
			msgTokens = append(msgTokens, tok)
		}
	}

	if len(messages) > 0 {
		w.send(ctx, messages, msgTokens)
	}

	// помечаем ВСЕ выбранные (даже без устройств) — чтобы не зацикливаться
	ids := make([]uuid.UUID, len(notes))
	for i, n := range notes {
		ids[i] = n.ID
	}
	_, err := w.db.NewUpdate().Table("notifications").Set("pushed_at = now()").Where("id IN (?)", bun.In(ids)).Exec(ctx)
	return err
}

// send шлёт батч в Expo (≤100 сообщений/запрос) и удаляет мёртвые токены.
func (w *PushDispatchWorker) send(ctx context.Context, messages []expoMessage, tokens []string) {
	for start := 0; start < len(messages); start += 100 {
		end := start + 100
		if end > len(messages) {
			end = len(messages)
		}
		batch, batchTokens := messages[start:end], tokens[start:end]
		body, _ := json.Marshal(batch)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ExpoPushURL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := w.http.Do(req)
		if err != nil || resp == nil {
			continue
		}
		var er expoResponse
		_ = json.NewDecoder(resp.Body).Decode(&er)
		_ = resp.Body.Close()
		var dead []string
		for i, r := range er.Data {
			if i < len(batchTokens) && r.Status == "error" && r.Details.Error == "DeviceNotRegistered" {
				dead = append(dead, batchTokens[i])
			}
		}
		if len(dead) > 0 {
			_, _ = w.db.NewDelete().Model((*models.DeviceToken)(nil)).Where("token IN (?)", bun.In(dead)).Exec(ctx)
		}
	}
}

// NewPushDispatchWorker — конструктор для тестов/обвязки.
func NewPushDispatchWorker(db *bun.DB, httpc *http.Client) *PushDispatchWorker {
	return &PushDispatchWorker{db: db, http: httpc}
}
