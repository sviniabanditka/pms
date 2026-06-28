package realtime

import (
	"context"
	"encoding/json"

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

type Publisher struct{ rdb *redis.Client }

func NewPublisher(rdb *redis.Client) *Publisher { return &Publisher{rdb: rdb} }

// PublishDelta публикует дельту в канал воркспейса (fan-out на все WS-инстансы).
func (p *Publisher) PublishDelta(ctx context.Context, wsID uuid.UUID, d Delta) error {
	payload, err := json.Marshal(serverMsg{Type: "delta", Delta: &d})
	if err != nil {
		return err
	}
	return p.rdb.Publish(ctx, WorkspaceChannel(wsID), payload).Err()
}

