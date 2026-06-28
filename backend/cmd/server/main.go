package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pms/backend/internal/auth"
	"github.com/pms/backend/internal/config"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/domain/attachments"
	"github.com/pms/backend/internal/domain/chats"
	"github.com/pms/backend/internal/domain/comments"
	"github.com/pms/backend/internal/domain/customfields"
	"github.com/pms/backend/internal/domain/cycles"
	"github.com/pms/backend/internal/domain/dashboards"
	"github.com/pms/backend/internal/domain/docs"
	"github.com/pms/backend/internal/domain/issues"
	"github.com/pms/backend/internal/domain/meetings"
	"github.com/pms/backend/internal/domain/milestones"
	"github.com/pms/backend/internal/domain/notifications"
	"github.com/pms/backend/internal/domain/search"
	syncsvc "github.com/pms/backend/internal/domain/sync"
	"github.com/pms/backend/internal/domain/teams"
	"github.com/pms/backend/internal/domain/views"
	"github.com/pms/backend/internal/domain/workspaces"
	"github.com/pms/backend/internal/httpx"
	"github.com/pms/backend/internal/jobs"
	"github.com/pms/backend/internal/mailer"
	"github.com/pms/backend/internal/realtime"
	"github.com/pms/backend/internal/rpc"
	"github.com/pms/backend/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

func main() {
	cfg := config.Load()

	bdb, err := db.OpenBun(cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer bdb.Close()

	jobsDB, err := db.OpenBun(cfg.JobsDatabaseURL)
	if err != nil {
		slog.Error("jobs db connect failed", "err", err)
		os.Exit(1)
	}
	defer jobsDB.Close()

	rdb, err := db.OpenRedis(cfg.RedisURL)
	if err != nil {
		slog.Error("redis connect failed", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	sessions := auth.NewSessionStore(rdb, cfg.SessionTTL)
	authSvc := auth.NewService(bdb, sessions, cfg.SecureCookies)

	pub := realtime.NewPublisher(rdb)
	gw := realtime.NewGateway(rdb, bdb)
	store := storage.New(cfg)

	// Фоновые задачи (River): пул на роли-владельце (postgres) для DDL/DML очереди.
	pool, err := pgxpool.New(context.Background(), cfg.MigrateDatabaseURL)
	if err != nil {
		slog.Error("river pool connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	riverClient, err := jobs.Start(context.Background(), jobs.Deps{
		Pool: pool, JobsDB: jobsDB, Pub: pub, Mailer: mailer.New(cfg),
		DueSoonInterval: river.PeriodicInterval(cfg.DueSoonInterval),
		EmailSweepEvery: river.PeriodicInterval(cfg.EmailSweepEvery),
		SnapshotEvery:   river.PeriodicInterval(cfg.SnapshotEvery),
	})
	if err != nil {
		slog.Error("jobs start failed", "err", err)
		os.Exit(1)
	}

	svcs := rpc.Services{
		Workspaces: workspaces.NewService(bdb, pub, jobsDB),
		Teams:      teams.NewService(bdb, pub),
		Issues:     issues.NewService(bdb, pub),
		Views:      views.NewService(bdb, pub),
		Cycles:       cycles.NewService(bdb, pub),
		CustomFields:  customfields.NewService(bdb, pub),
		Dashboards:    dashboards.NewService(bdb, pub),
		Notifications: notifications.NewService(bdb, pub),
		Docs:          docs.NewService(bdb, pub),
		Chats:         chats.NewService(bdb, pub),
		Comments:      comments.NewService(bdb, pub),
		Milestones:    milestones.NewService(bdb, pub),
		Meetings:      meetings.NewService(bdb, pub, meetings.LiveKit{URL: cfg.LiveKitURL, APIKey: cfg.LiveKitAPIKey, Secret: cfg.LiveKitSecret}),
		Attachments:   attachments.NewService(bdb, pub, store),
		Search:        search.NewService(bdb),
		Sync:          syncsvc.NewService(bdb),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpx.NewRouter(bdb, rdb, authSvc, gw, svcs, store, cfg.AllowedOrigins),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := riverClient.Stop(ctx); err != nil {
		slog.Error("river stop error", "err", err)
	}
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}
