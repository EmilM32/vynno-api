package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
)

func TestDiscardSendSucceeds(t *testing.T) {
	t.Parallel()
	if err := Discard().Send(context.Background(), Message{
		To: "a@b.example", Subject: "hi", Text: "body-secret",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestLogSendWritesBody(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	if err := (logMailer{}).Send(context.Background(), Message{
		To: "a@b.example", Subject: "Your code", Text: "123456",
	}); err != nil {
		t.Fatal(err)
	}

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log json: %v\n%s", err, buf.String())
	}
	if rec["to"] != "a@b.example" || rec["subject"] != "Your code" || rec["text"] != "123456" {
		t.Fatalf("log record = %#v", rec)
	}
}

func TestRecorderStoresInOrderAndCallsNext(t *testing.T) {
	t.Parallel()
	inner := &Recorder{}
	rec := &Recorder{Next: inner}

	first := Message{To: "one@example.com", Subject: "a", Text: "1"}
	second := Message{To: "two@example.com", Subject: "b", Text: "2"}
	if err := rec.Send(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := rec.Send(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	if len(rec.Messages) != 2 || rec.Messages[0] != first || rec.Messages[1] != second {
		t.Fatalf("recorder = %#v", rec.Messages)
	}
	if len(inner.Messages) != 2 || inner.Messages[0] != first {
		t.Fatalf("next = %#v", inner.Messages)
	}
}

func TestNewModes(t *testing.T) {
	t.Parallel()
	d, err := New(ModeDiscard, SMTP{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.(discardMailer); !ok {
		t.Fatalf("discard type %T", d)
	}
	l, err := New(ModeLog, SMTP{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := l.(logMailer); !ok {
		t.Fatalf("log type %T", l)
	}
	if _, err := New("resend", SMTP{}); err == nil {
		t.Fatal("expected unknown mode error")
	}
}

func TestSMTPSourceDoesNotLogBody(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("smtp.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(src, []byte(`slog.Info("mail sent", "to", msg.To)`)) {
		t.Fatal("smtp success log must record only the recipient")
	}
	if bytes.Contains(src, []byte(`"text"`)) {
		t.Fatal("smtp must not log the mail body or one-time code")
	}
}

func TestNewSMTPRequiresHostAndFrom(t *testing.T) {
	t.Parallel()
	if _, err := New(ModeSMTP, SMTP{From: "vynno@localhost"}); err == nil {
		t.Fatal("expected error without host")
	}
	if _, err := New(ModeSMTP, SMTP{Host: "127.0.0.1"}); err == nil {
		t.Fatal("expected error without from")
	}
	m, err := New(ModeSMTP, SMTP{Host: "127.0.0.1", From: "vynno@localhost", FromName: "Vynno"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.(*smtpMailer); !ok {
		t.Fatalf("smtp type %T", m)
	}
}
