package db

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// OpenBun открывает bun.DB поверх pgdriver (роль pms_app, RLS enforced).
func OpenBun(dsn string) (*bun.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	bdb := bun.NewDB(sqldb, pgdialect.New())
	if err := bdb.PingContext(context.Background()); err != nil {
		return nil, err
	}
	return bdb, nil
}

// OpenRedis открывает клиент Redis.
func OpenRedis(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return rdb, nil
}
