package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/pms/backend/internal/auth"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

type clientMsg struct {
	Type     string   `json:"type"` // subscribe | unsubscribe | ping
	Channels []string `json:"channels,omitempty"`
}

// Gateway — WS-точка realtime: подписки + поток дельт из Redis pub/sub.
// Аутентификация — middleware (RequireAuth) выше по стеку.
type Gateway struct {
	rdb *redis.Client
	bdb *bun.DB
}

func NewGateway(rdb *redis.Client, bdb *bun.DB) *Gateway {
	return &Gateway{rdb: rdb, bdb: bdb}
}

// Handle обслуживает одно WS-соединение (goroutine на соединение, §realtime).
func (g *Gateway) Handle(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// InsecureSkipVerify: dev — без проверки Origin; в проде задать OriginPatterns.
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusInternalError, "bye")

	ctx := r.Context()
	pubsub := g.rdb.Subscribe(ctx)
	defer pubsub.Close()

	// Персональный канал юзера: таргетная доставка WebRTC-сигналов (D56).
	_ = pubsub.Subscribe(ctx, UserChannel(uid.String()))

	// Redis → WS с коалесингом дельт (окно 80мс, дедуп по entity_id → последняя версия).
	go g.forward(ctx, c, pubsub)

	// WS → сервер: subscribe/unsubscribe/ping.
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		var m clientMsg
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		switch m.Type {
		case "subscribe":
			valid := g.authorizeChannels(ctx, uid, m.Channels)
			if len(valid) > 0 {
				_ = pubsub.Subscribe(ctx, valid...)
			}
			g.write(ctx, c, serverMsg{Type: "sub_ack", Channels: valid})
		case "unsubscribe":
			if len(m.Channels) > 0 {
				_ = pubsub.Unsubscribe(ctx, m.Channels...)
			}
		case "ping":
			g.write(ctx, c, serverMsg{Type: "pong"})
		}
	}
}

// forward коалесит дельты из Redis: накапливает по entity_id в окне 80мс и шлёт пачкой,
// схлопывая множественные апдейты одной сущности (bulk-операции не штормят клиента, §realtime).
func (g *Gateway) forward(ctx context.Context, c *websocket.Conn, pubsub *redis.PubSub) {
	ch := pubsub.Channel()
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	buf := map[string][]byte{}
	flush := func() {
		for _, p := range buf {
			if err := c.Write(ctx, websocket.MessageText, p); err != nil {
				return
			}
		}
		if len(buf) > 0 {
			buf = map[string][]byte{}
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var sm serverMsg
			if json.Unmarshal([]byte(msg.Payload), &sm) == nil && sm.Type == "delta" && sm.Delta != nil {
				buf[sm.Delta.EntityID] = []byte(msg.Payload) // последняя по entity побеждает в окне
			} else {
				_ = c.Write(ctx, websocket.MessageText, []byte(msg.Payload)) // presence/прочее — сразу
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (g *Gateway) write(ctx context.Context, c *websocket.Conn, m serverMsg) {
	payload, _ := json.Marshal(m)
	_ = c.Write(ctx, websocket.MessageText, payload)
}

// authorizeChannels оставляет только каналы вида ws:{uuid}, где user — член воркспейса (RBAC).
func (g *Gateway) authorizeChannels(ctx context.Context, uid uuid.UUID, channels []string) []string {
	out := make([]string, 0, len(channels))
	for _, ch := range channels {
		rest, ok := strings.CutPrefix(ch, "ws:")
		if !ok {
			continue
		}
		wsID, err := uuid.Parse(rest)
		if err != nil {
			continue
		}
		if g.isMember(ctx, uid, wsID) {
			out = append(out, ch)
		}
	}
	return out
}

func (g *Gateway) isMember(ctx context.Context, uid, wsID uuid.UUID) bool {
	var member bool
	_ = db.WithWorkspace(ctx, g.bdb, wsID, func(ctx context.Context, tx bun.Tx) error {
		exists, err := tx.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("user_id = ?", uid).Where("workspace_id = ?", wsID).Exists(ctx)
		if err != nil {
			return err
		}
		member = exists
		return nil
	})
	return member
}
