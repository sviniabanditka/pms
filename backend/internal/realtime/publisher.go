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

// Signal — WebRTC-сигналинг peer→peer (offer/answer/ice/bye), доставка через WS воркспейса.
type Signal struct {
	MeetingID  string `json:"meetingId"`
	FromUserID string `json:"fromUserId"`
	ToUserID   string `json:"toUserId"`
	Kind       string `json:"kind"`
	Payload    string `json:"payload"`
}

// serverMsg — конверт сервер→клиент.
type serverMsg struct {
	Type     string   `json:"type"`               // delta | signal | pong | sub_ack
	Delta    *Delta   `json:"delta,omitempty"`
	Signal   *Signal  `json:"signal,omitempty"`
	Channels []string `json:"channels,omitempty"`
}

// WorkspaceChannel — Redis/WS-канал воркспейса.
func WorkspaceChannel(wsID uuid.UUID) string { return "ws:" + wsID.String() }

// UserChannel — персональный Redis/WS-канал юзера (таргетная доставка сигналов, D56).
func UserChannel(userID string) string { return "user:" + userID }

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

// PublishSignal публикует WebRTC-сигнал в ПЕРСОНАЛЬНЫЙ канал получателя (D56): только адресат
// получает фрейм, без широковещания на весь воркспейс. gateway шлёт не-дельты сразу.
func (p *Publisher) PublishSignal(ctx context.Context, sig Signal) error {
	payload, err := json.Marshal(serverMsg{Type: "signal", Signal: &sig})
	if err != nil {
		return err
	}
	return p.rdb.Publish(ctx, UserChannel(sig.ToUserID), payload).Err()
}
