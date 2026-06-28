package storage

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pms/backend/internal/config"
)

// Store — обёртка над S3-совместимым хранилищем (Garage). Path-style, статичные креды.
type Store struct {
	client     *s3.Client
	presign    *s3.PresignClient
	bucket     string
	endpoint   string
	publicBase string
}

func New(cfg config.Config) *Store {
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(cfg.S3Endpoint),
		Region:       cfg.S3Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3Secret, ""),
		UsePathStyle: true,
	})
	return &Store{
		client:     client,
		presign:    s3.NewPresignClient(client),
		bucket:     cfg.S3Bucket,
		endpoint:   strings.TrimRight(cfg.S3Endpoint, "/"),
		publicBase: strings.TrimRight(cfg.S3PublicBase, "/"),
	}
}

// Put заливает объект (upload через бэкенд → нет CORS-префлайта на Garage).
func (s *Store) Put(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket, Key: &key, Body: body, ContentType: &contentType, ContentLength: &size,
	})
	return err
}

// PresignGet — временная ссылка для <img>/<a> (cross-origin simple GET, без auth-заголовков).
func (s *Store) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	r, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	url := r.URL
	// подменить internal-endpoint на публичный (если бэкенд ходит в garage:3900, а браузер — в localhost:3900)
	if s.publicBase != "" && s.publicBase != s.endpoint {
		url = strings.Replace(url, s.endpoint, s.publicBase, 1)
	}
	return url, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	return err
}

// Get — поток объекта (для прокси через бэкенд: <img> грузится с того же origin, без redirect).
func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return nil, "", 0, err
	}
	ct := "application/octet-stream"
	if out.ContentType != nil && *out.ContentType != "" {
		ct = *out.ContentType
	}
	var sz int64
	if out.ContentLength != nil {
		sz = *out.ContentLength
	}
	return out.Body, ct, sz, nil
}
