package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Delta — изменение сущности (wire-формат WS, JSON-режим для M0; protobuf — позже).
type Delta struct {
	Seq        int64  `json:"seq"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Version    int64  `json:"version"`
	Op         string `json:"op"`
	Actor      string `json:"actor,omitempty"` // userId инициатора (typing/авторство)
}

// serverMsg — конверт сервер→клиент.
type serverMsg struct {
	Type     string   `json:"type"` // delta | pong | sub_ack
	Delta    *Delta   `json:"delta,omitempty"`
	Channels []string `json:"channels,omitempty"`
}

// WorkspaceChannel — Redis/WS-канал воркспейса.
func WorkspaceChannel(wsID uuid.UUID) string { return "ws:" + wsID.String() }

type Publisher struct {
	rdb     *redis.Client
	lastLog atomic.Int64 // unix-время последнего warn (троттлинг спама при падении redis)
}

func NewPublisher(rdb *redis.Client) *Publisher { return &Publisher{rdb: rdb} }

// PublishDelta публикует дельту в канал воркспейса (fan-out на все WS-инстансы).
// Ошибку логирует сам (не чаще раза в 10с): все вызовы best-effort (`_ =`), и без
// этого падение redis = тихий десинк всех клиентов.
func (p *Publisher) PublishDelta(ctx context.Context, wsID uuid.UUID, d Delta) error {
	payload, err := json.Marshal(serverMsg{Type: "delta", Delta: &d})
	if err != nil {
		return err
	}
	if err := p.rdb.Publish(ctx, WorkspaceChannel(wsID), payload).Err(); err != nil {
		now := time.Now().Unix()
		if last := p.lastLog.Load(); now-last >= 10 && p.lastLog.CompareAndSwap(last, now) {
			slog.Warn("realtime: delta publish failed (clients will desync until reload)", "err", err, "entity", d.EntityType)
		}
		return err
	}
	return nil
}

