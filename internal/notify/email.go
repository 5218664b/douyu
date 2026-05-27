package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/5218664b/douyu-streamer/internal/config"
)

const defaultSendTimeout = 15 * time.Second

type Emailer struct {
	cfg           config.NotifyConfig
	mu            sync.Mutex
	lastSentAt    time.Time
	lastSentKey   string
	problemActive bool
}

func NewEmailer(cfg config.NotifyConfig) *Emailer {
	return &Emailer{cfg: cfg}
}

func (e *Emailer) Enabled() bool {
	return e != nil && e.cfg.Enabled
}

func (e *Emailer) NotifyProblem(ctx context.Context, subject, body, dedupeKey string) error {
	if !e.Enabled() {
		return nil
	}

	e.mu.Lock()
	if e.problemActive {
		e.mu.Unlock()
		return nil
	}
	if e.shouldSkipLocked(dedupeKey) {
		e.mu.Unlock()
		return nil
	}
	e.lastSentAt = time.Now()
	e.lastSentKey = dedupeKey
	e.problemActive = true
	e.mu.Unlock()

	return e.send(ctx, subject, body)
}

func (e *Emailer) NotifyEvent(ctx context.Context, subject, body string) error {
	if !e.Enabled() {
		return nil
	}

	return e.send(ctx, subject, body)
}

func (e *Emailer) shouldSkipLocked(dedupeKey string) bool {
	if e.cfg.CooldownSeconds <= 0 {
		return false
	}
	if e.lastSentKey != dedupeKey {
		return false
	}
	return time.Since(e.lastSentAt) < time.Duration(e.cfg.CooldownSeconds)*time.Second
}

func (e *Emailer) send(ctx context.Context, subject, body string) error {
	address := net.JoinHostPort(e.cfg.SMTPHost, e.cfg.SMTPPort)
	message := buildMessage(e.cfg.From, e.cfg.To, subjectLine(e.cfg.SubjectPrefix, subject), body)

	deadline := time.Now().Add(defaultSendTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	dialer := &net.Dialer{Timeout: time.Until(deadline)}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("dial smtp server: %w", err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set smtp deadline: %w", err)
	}

	client, err := smtp.NewClient(conn, e.cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: e.cfg.SMTPHost}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	if e.cfg.Username != "" {
		auth := smtp.PlainAuth("", e.cfg.Username, e.cfg.Password, e.cfg.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(e.cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	for _, recipient := range e.cfg.To {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp rcpt to %s: %w", recipient, err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		writer.Close()
		return fmt.Errorf("write smtp body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp body: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}

	return nil
}

func buildMessage(from string, to []string, subject string, body string) string {
	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", strings.Join(to, ", ")),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		`Content-Type: text/plain; charset="UTF-8"`,
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body + "\r\n"
}

func subjectLine(prefix string, subject string) string {
	prefix = strings.TrimSpace(prefix)
	subject = strings.TrimSpace(subject)
	if prefix == "" {
		return subject
	}
	if subject == "" {
		return prefix
	}
	return prefix + " " + subject
}
