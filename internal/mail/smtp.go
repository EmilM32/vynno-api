package mail

import (
	"context"
	"fmt"
	"log/slog"

	gomail "github.com/wneessen/go-mail"
)

type smtpMailer struct {
	client *gomail.Client
	from   string
	name   string
}

func newSMTP(cfg SMTP) (Mailer, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("SMTP from is required")
	}
	port := cfg.Port
	if port == 0 {
		port = 1025
	}
	opts := []gomail.Option{
		gomail.WithPort(port),
		gomail.WithTLSPolicy(tlsPolicy(cfg.StartTLS)),
	}
	if cfg.Username != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
			gomail.WithUsername(cfg.Username),
			gomail.WithPassword(cfg.Password),
		)
	}
	client, err := gomail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, err
	}
	return &smtpMailer{client: client, from: cfg.From, name: cfg.FromName}, nil
}

func tlsPolicy(startTLS bool) gomail.TLSPolicy {
	if startTLS {
		return gomail.TLSMandatory
	}
	return gomail.NoTLS
}

func (s *smtpMailer) Send(ctx context.Context, msg Message) error {
	m := gomail.NewMsg()
	if s.name != "" {
		if err := m.FromFormat(s.name, s.from); err != nil {
			return err
		}
	} else if err := m.From(s.from); err != nil {
		return err
	}
	if err := m.To(msg.To); err != nil {
		return err
	}
	m.Subject(msg.Subject)
	m.SetBodyString(gomail.TypeTextPlain, msg.Text)
	if err := s.client.DialAndSendWithContext(ctx, m); err != nil {
		return err
	}
	slog.Info("mail sent", "to", msg.To)
	return nil
}
