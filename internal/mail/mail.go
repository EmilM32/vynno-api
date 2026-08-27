package mail

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

const (
	ModeSMTP    = "smtp"
	ModeLog     = "log"
	ModeDiscard = "discard"
)

// Message is a plain-text outbound mail. One-time codes belong in Text, not JSON.
type Message struct {
	To      string
	Subject string
	Text    string
}

// Mailer is the outbound mail port. Domain does not import this package.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// SMTP is the transport settings for ModeSMTP.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	StartTLS bool
	From     string
	FromName string
}

// New returns the implementation for mode. Unknown mode is an error.
func New(mode string, smtp SMTP) (Mailer, error) {
	switch mode {
	case ModeSMTP:
		return newSMTP(smtp)
	case ModeLog:
		return logMailer{}, nil
	case ModeDiscard:
		return Discard(), nil
	default:
		return nil, fmt.Errorf("unknown mail mode %q", mode)
	}
}

// Discard is a Mailer that accepts every message and sends nothing.
func Discard() Mailer {
	return discardMailer{}
}

type discardMailer struct{}

func (discardMailer) Send(context.Context, Message) error { return nil }

type logMailer struct{}

func (logMailer) Send(_ context.Context, msg Message) error {
	slog.Info("mail", "to", msg.To, "subject", msg.Subject, "text", msg.Text)
	return nil
}

// Recorder is a test double that stores each Send. Next, if set, is called after record.
type Recorder struct {
	mu       sync.Mutex
	Messages []Message
	Next     Mailer
}

func (r *Recorder) Send(ctx context.Context, msg Message) error {
	r.mu.Lock()
	r.Messages = append(r.Messages, msg)
	r.mu.Unlock()
	if r.Next != nil {
		return r.Next.Send(ctx, msg)
	}
	return nil
}
