-- +goose Up

CREATE TABLE email_challenges (
    email TEXT NOT NULL,
    purpose TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempt_count INT NOT NULL DEFAULT 0,
    sent_at TIMESTAMPTZ NOT NULL,
    send_count INT NOT NULL DEFAULT 1,
    send_window_start TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (email, purpose),
    CONSTRAINT email_challenges_purpose_check CHECK (purpose IN ('register', 'password_reset'))
);

-- +goose Down

DROP TABLE IF EXISTS email_challenges;
