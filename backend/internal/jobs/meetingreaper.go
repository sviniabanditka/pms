package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	"github.com/pms/backend/internal/realtime"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"
)

type MeetingReapArgs struct{}

func (MeetingReapArgs) Kind() string { return "meeting_reap" }

// MeetingReaperWorker гасит «мёртвые» live-митинги: LiveKit — источник правды о занятости.
// Митинг live, но его комнаты нет в SFU (никто не дозвонился / все упали / вкладку закрыли) →
// после grace помечаем ended. Решает залипшие huddle'ы (LeaveMeeting не вызывается при краше/закрытии).
type MeetingReaperWorker struct {
	river.WorkerDefaults[MeetingReapArgs]
	db     *bun.DB
	pub    *realtime.Publisher
	apiURL string
	key    string
	secret string
}

func (w *MeetingReaperWorker) Work(ctx context.Context, _ *river.Job[MeetingReapArgs]) error {
	active, err := listActiveRooms(ctx, w.apiURL, w.key, w.secret)
	if err != nil {
		return err // LiveKit недоступен → не гасим (повтор позже), без ложных завершений
	}
	var rows []struct {
		ID          uuid.UUID `bun:"id"`
		WorkspaceID uuid.UUID `bun:"workspace_id"`
	}
	// grace 90с: свежесозданный huddle успевает получить комнату (room создаётся при первом join)
	if err := w.db.NewSelect().TableExpr("meetings AS m").ColumnExpr("m.id, m.workspace_id").
		Where("m.status = 'live'").Where("m.deleted_at IS NULL").
		Where("m.started_at < now() - interval '90 seconds'").Scan(ctx, &rows); err != nil {
		return err
	}
	for _, r := range rows {
		if active[r.ID.String()] {
			continue // живёт в SFU
		}
		res, err := w.db.NewUpdate().TableExpr("meetings").
			Set("status = 'ended'").Set("ended_at = now()").Set("updated_at = now()").
			Where("id = ?", r.ID).Where("status = 'live'").Exec(ctx)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			_ = w.pub.PublishDelta(ctx, r.WorkspaceID, realtime.Delta{EntityType: "meeting", EntityID: r.ID.String(), Op: "update"})
		}
	}
	return nil
}

// listActiveRooms — имена активных комнат LiveKit (= meetingID) через Twirp RoomService.ListRooms.
func listActiveRooms(ctx context.Context, apiURL, key, secret string) (map[string]bool, error) {
	tok, err := auth.NewAccessToken(key, secret).SetValidFor(time.Minute).
		AddGrant(&auth.VideoGrant{RoomList: true}).ToJWT()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL+"/twirp/livekit.RoomService/ListRooms", bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("livekit ListRooms: status %d", resp.StatusCode)
	}
	var out struct {
		Rooms []struct {
			Name string `json:"name"`
		} `json:"rooms"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(out.Rooms))
	for _, r := range out.Rooms {
		m[r.Name] = true
	}
	return m, nil
}
