package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// WithWorkspace выполняет fn в транзакции с установленным RLS-контекстом тенанта.
// set_config(..., is_local=true) эквивалентен SET LOCAL → действует до конца транзакции.
// Забытый скоуп в запросах внутри fn не утечёт между тенантами (RLS-политики).
func WithWorkspace(ctx context.Context, bdb *bun.DB, wsID uuid.UUID, fn func(ctx context.Context, tx bun.Tx) error) error {
	return bdb.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "SELECT set_config('app.workspace_id', ?, true)", wsID.String()); err != nil {
			return err
		}
		return fn(ctx, tx)
	})
}
