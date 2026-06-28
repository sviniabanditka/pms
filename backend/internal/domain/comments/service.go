package comments

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/domain/notifications"
	"github.com/pms/backend/internal/mentions"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var (
	ErrEmpty     = errors.New("empty comment")
	ErrForbidden = errors.New("not your comment")
)

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

func (s *Service) AddComment(ctx context.Context, actor, wsID, id, issueID uuid.UUID, body string) error {
	// пустое тело допускается, если коммент несёт вложения
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		c := &models.Comment{ID: id, WorkspaceID: wsID, IssueID: issueID, AuthorID: actor, Body: body, CreatedAt: now}
		if _, err := tx.NewInsert().Model(c).Exec(ctx); err != nil {
			return err
		}
		// получатели: создатель + ассайни + прежние комментаторы, кроме актора
		var iss models.Issue
		if err := tx.NewSelect().Model(&iss).Where("id = ?", issueID).Scan(ctx); err != nil {
			return err
		}
		recipients := map[uuid.UUID]bool{}
		if iss.CreatedBy != uuid.Nil {
			recipients[iss.CreatedBy] = true
		}
		if iss.AssigneeID != nil {
			recipients[*iss.AssigneeID] = true
		}
		var prior []uuid.UUID
		_ = tx.NewSelect().Model((*models.Comment)(nil)).ColumnExpr("DISTINCT author_id").
			Where("issue_id = ?", issueID).Scan(ctx, &prior)
		for _, u := range prior {
			recipients[u] = true
		}
		delete(recipients, actor)
		payload, _ := json.Marshal(map[string]any{"issueId": issueID.String()})
		// упомянутые → тип mention (и не дублируем им commented)
		mentioned := map[uuid.UUID]bool{}
		for _, u := range mentions.Parse(body) {
			if u == actor {
				continue
			}
			mentioned[u] = true
			delete(recipients, u)
			if err := notifications.Record(ctx, tx, wsID, u, actor, "mention", "issue", issueID, payload); err != nil {
				return err
			}
		}
		for r := range recipients {
			if err := notifications.Record(ctx, tx, wsID, r, actor, "commented", "issue", issueID, payload); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "comment", EntityID: issueID.String(), Op: "create"})
	return nil
}

func (s *Service) EditComment(ctx context.Context, actor, wsID, id uuid.UUID, body string) error {
	if strings.TrimSpace(body) == "" {
		return ErrEmpty
	}
	var issueID uuid.UUID
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var c models.Comment
		if err := tx.NewSelect().Model(&c).Where("id = ?", id).Scan(ctx); err != nil {
			return err
		}
		if c.AuthorID != actor {
			return ErrForbidden
		}
		issueID = c.IssueID
		_, err := tx.NewUpdate().Model((*models.Comment)(nil)).
			Set("body = ?", body).Set("edited_at = now()").Where("id = ?", id).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "comment", EntityID: issueID.String(), Op: "update"})
	return nil
}

func (s *Service) DeleteComment(ctx context.Context, actor, wsID, id uuid.UUID) error {
	var issueID uuid.UUID
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var c models.Comment
		if err := tx.NewSelect().Model(&c).Where("id = ?", id).Scan(ctx); err != nil {
			return err
		}
		if c.AuthorID != actor {
			return ErrForbidden
		}
		issueID = c.IssueID
		_, err := tx.NewUpdate().Model((*models.Comment)(nil)).
			Set("deleted_at = now()").Set("body = ''").Where("id = ?", id).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "comment", EntityID: issueID.String(), Op: "delete"})
	return nil
}

// ToggleReaction — переключить эмодзи-реакцию текущего юзера на комментарий (паритет с чатом).
func (s *Service) ToggleReaction(ctx context.Context, actor, wsID, commentID uuid.UUID, emoji string) error {
	emoji = strings.TrimSpace(emoji)
	if emoji == "" {
		return ErrEmpty
	}
	var issueID uuid.UUID
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var c models.Comment
		if err := tx.NewSelect().Model(&c).Column("issue_id").Where("id = ?", commentID).Scan(ctx); err != nil {
			return err
		}
		issueID = c.IssueID
		res, err := tx.NewDelete().Model((*models.CommentReaction)(nil)).
			Where("comment_id = ?", commentID).Where("user_id = ?", actor).Where("emoji = ?", emoji).Exec(ctx)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return nil // была — сняли (toggle off)
		}
		rid, _ := uuid.NewV7()
		_, err = tx.NewInsert().Model(&models.CommentReaction{ID: rid, WorkspaceID: wsID, CommentID: commentID, UserID: actor, Emoji: emoji, CreatedAt: time.Now().UTC()}).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "comment", EntityID: issueID.String(), Op: "update"})
	return nil
}

type reactionDTO struct {
	Emoji  string    `json:"emoji"`
	UserID uuid.UUID `json:"userId"`
}
type commentDTO struct {
	models.Comment
	Reactions []reactionDTO `json:"reactions"`
}

func (s *Service) GetComments(ctx context.Context, actor, wsID, issueID uuid.UUID) ([]byte, error) {
	var list []models.Comment
	var out []byte
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewSelect().Model(&list).Where("issue_id = ?", issueID).Order("created_at ASC").Scan(ctx); err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, len(list))
		for i := range list {
			ids[i] = list[i].ID
		}
		var reactions []models.CommentReaction
		if err := tx.NewSelect().Model(&reactions).Where("comment_id IN (?)", bun.In(ids)).Scan(ctx); err != nil {
			return err
		}
		reByCmt := map[uuid.UUID][]reactionDTO{}
		for _, r := range reactions {
			reByCmt[r.CommentID] = append(reByCmt[r.CommentID], reactionDTO{Emoji: r.Emoji, UserID: r.UserID})
		}
		dtos := make([]commentDTO, len(list))
		for i := range list {
			rs := reByCmt[list[i].ID]
			if rs == nil {
				rs = []reactionDTO{}
			}
			dtos[i] = commentDTO{Comment: list[i], Reactions: rs}
		}
		out, _ = json.Marshal(dtos)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if out == nil || string(out) == "null" {
		return []byte("[]"), nil
	}
	return out, nil
}
