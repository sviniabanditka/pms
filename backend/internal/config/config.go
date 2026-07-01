package config

import (
	"os"
	"strings"
	"time"
)

// Config — конфиг приложения из окружения. Без внешних зависимостей (ponytail: stdlib хватает).
type Config struct {
	Port string

	// DatabaseURL — рабочее подключение (роль pms_app, RLS enforced).
	DatabaseURL string
	// MigrateDatabaseURL — подключение для миграций (роль postgres, владелец таблиц).
	MigrateDatabaseURL string
	// JobsDatabaseURL — роль pms_jobs (BYPASSRLS) для cross-tenant операций (напр. список воркспейсов юзера).
	JobsDatabaseURL string

	RedisURL string

	SessionTTL time.Duration

	// AllowedOrigins — белый список origin'ов для CORS+CSRF. Пусто = dev (любой origin, credentials echo).
	// Задан (прод) = только перечисленные; чужой Origin на мутациях отклоняется (CSRF-защита).
	AllowedOrigins []string
	// SecureCookies — Secure-флаг на cookie сессии (true за TLS в проде).
	SecureCookies bool

	// LiveKit (SFU для митингов). URL — клиентский ws-адрес; APIURL — http для server-API (reaper); key/secret — подпись токенов.
	LiveKitURL    string
	LiveKitAPIURL string
	LiveKitAPIKey string
	LiveKitSecret string
	// MeetingReapEvery — как часто гасить «мёртвые» live-митинги (нет комнаты в LiveKit).
	MeetingReapEvery time.Duration

	// S3 (Garage self-hosted; S3-совместимо). PublicBase — базовый URL для presigned-ссылок в браузер.
	S3Endpoint   string
	S3PublicBase string
	S3Region     string
	S3Bucket     string
	S3AccessKey  string
	S3Secret     string

	// SMTP (пусто → письма логируются, не отправляются). + интервалы фоновых задач (River).
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	DueSoonInterval time.Duration
	WebhookEvery    time.Duration
	PushEvery       time.Duration
	EmbedEvery      time.Duration
	TrainEvery      time.Duration
	ClusterEvery    time.Duration
	EmbedURL        string
	EmailSweepEvery time.Duration
	SnapshotEvery   time.Duration
}

func Load() Config {
	return Config{
		Port:               env("PORT", "8080"),
		DatabaseURL:        env("DATABASE_URL", "postgres://pms_app:pms_app@localhost:5432/pms?sslmode=disable"),
		MigrateDatabaseURL: env("MIGRATE_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pms?sslmode=disable"),
		JobsDatabaseURL:    env("JOBS_DATABASE_URL", "postgres://pms_jobs:pms_jobs@localhost:5432/pms?sslmode=disable"),
		RedisURL:           env("REDIS_URL", "redis://localhost:6379/0"),
		SessionTTL:         envDuration("SESSION_TTL", 30*24*time.Hour),
		AllowedOrigins:     envList("ALLOWED_ORIGINS"),
		SecureCookies:      env("SECURE_COOKIES", "") == "true",
		LiveKitURL:         env("LIVEKIT_URL", "ws://localhost:7880"),
		LiveKitAPIURL:      env("LIVEKIT_API_URL", "http://localhost:7880"),
		LiveKitAPIKey:      env("LIVEKIT_API_KEY", "devkey"),
		LiveKitSecret:      env("LIVEKIT_SECRET", "devsecretdevsecretdevsecretdevsecret32"),
		MeetingReapEvery:   envDuration("MEETING_REAP_EVERY", time.Minute),
		S3Endpoint:         env("S3_ENDPOINT", "http://localhost:3900"),
		S3PublicBase:       env("S3_PUBLIC_BASE", "http://localhost:3900"),
		S3Region:           env("S3_REGION", "garage"),
		S3Bucket:           env("S3_BUCKET", "pms"),
		S3AccessKey:        env("S3_ACCESS_KEY", "GK00112233445566778899aabb"),
		S3Secret:           env("S3_SECRET", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"),
		SMTPHost:           env("SMTP_HOST", "localhost"), // dev: Mailpit (docker), прод: реальный SMTP или "" → лог
		SMTPPort:           env("SMTP_PORT", "1025"),
		SMTPUser:           env("SMTP_USER", ""),
		SMTPPass:           env("SMTP_PASS", ""),
		SMTPFrom:           env("SMTP_FROM", "stdtask <no-reply@stdtask.com>"),
		DueSoonInterval:    envDuration("DUE_SOON_INTERVAL", time.Hour),
		WebhookEvery:       envDuration("WEBHOOK_DISPATCH_EVERY", 10*time.Second),
		PushEvery:          envDuration("PUSH_DISPATCH_EVERY", 15*time.Second),
		EmbedEvery:         envDuration("EMBED_SWEEP_EVERY", 10*time.Second),
		TrainEvery:         envDuration("MODEL_TRAIN_EVERY", 30*time.Minute),
		ClusterEvery:       envDuration("CLUSTER_EVERY", 30*time.Minute),
		EmbedURL:           env("EMBED_URL", "http://127.0.0.1:8100"),
		EmailSweepEvery:    envDuration("EMAIL_SWEEP_EVERY", 2*time.Minute),
		SnapshotEvery:      envDuration("SNAPSHOT_EVERY", 24*time.Hour),
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// envList парсит CSV-переменную в срез (пустые элементы отбрасываются).
func envList(k string) []string {
	v := os.Getenv(k)
	if v == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
