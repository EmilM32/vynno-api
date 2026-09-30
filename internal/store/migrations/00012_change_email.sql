-- +goose Up

-- change_email challenges are keyed by the new address and bound to the account
-- that asked, so another account cannot spend them. Register and reset rows keep
-- user_id NULL.
ALTER TABLE email_challenges
    ADD COLUMN user_id UUID REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE email_challenges DROP CONSTRAINT email_challenges_purpose_check;
ALTER TABLE email_challenges ADD CONSTRAINT email_challenges_purpose_check
    CHECK (purpose IN ('register', 'password_reset', 'change_email'));

-- +goose Down

DELETE FROM email_challenges WHERE purpose = 'change_email';

ALTER TABLE email_challenges DROP CONSTRAINT email_challenges_purpose_check;
ALTER TABLE email_challenges ADD CONSTRAINT email_challenges_purpose_check
    CHECK (purpose IN ('register', 'password_reset'));

ALTER TABLE email_challenges DROP COLUMN user_id;
