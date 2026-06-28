package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/pms/backend/internal/config"
)

type Mail struct {
	To      string
	Subject string
	Body    string
}

type Sender interface {
	Send(ctx context.Context, m Mail) error
}

// New: SMTP если задан SMTP_HOST, иначе лог-отправитель (dev — письма в консоль).
func New(cfg config.Config) Sender {
	if cfg.SMTPHost == "" {
		return &logSender{}
	}
	return &smtpSender{host: cfg.SMTPHost, port: cfg.SMTPPort, user: cfg.SMTPUser, pass: cfg.SMTPPass, from: cfg.SMTPFrom}
}

type logSender struct{}

func (l *logSender) Send(_ context.Context, m Mail) error {
	slog.Info("email (dev, not sent — set SMTP_*)", "to", m.To, "subject", m.Subject)
	return nil
}

type smtpSender struct{ host, port, user, pass, from string }

func (s *smtpSender) Send(_ context.Context, m Mail) error {
	addr := s.host + ":" + s.port
	msg := strings.Builder{}
	// заголовок From может быть "Имя <email>", а конверт MAIL FROM — только голый адрес
	fmt.Fprintf(&msg, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", s.from, m.To, m.Subject, m.Body)
	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}
	return smtp.SendMail(addr, auth, envelopeAddr(s.from), []string{m.To}, []byte(msg.String()))
}

// envelopeAddr вытаскивает голый email из "Имя <email>" (или возвращает как есть).
func envelopeAddr(from string) string {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		if j := strings.Index(from[i:], ">"); j > 0 {
			return from[i+1 : i+j]
		}
	}
	return strings.TrimSpace(from)
}
