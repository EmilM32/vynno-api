-- +goose Up

ALTER TABLE users RENAME COLUMN username TO email;
ALTER TABLE users RENAME CONSTRAINT users_username_key TO users_email_key;

UPDATE users SET email = lower(email) WHERE email IS NOT NULL;
UPDATE users
SET email = email || '@vynno.local'
WHERE email IS NOT NULL AND position('@' in email) = 0;

UPDATE users SET email = 'unknown@vynno.local' WHERE email IS NULL OR email = '';

ALTER TABLE users ALTER COLUMN email SET NOT NULL;

ALTER TABLE profiles DROP COLUMN handle;

-- +goose Down

ALTER TABLE profiles ADD COLUMN handle TEXT NOT NULL DEFAULT '';

UPDATE profiles p
SET handle = '@' || split_part(u.email, '@', 1)
FROM users u
WHERE u.id = p.user_id;

ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users RENAME CONSTRAINT users_email_key TO users_username_key;
ALTER TABLE users RENAME COLUMN email TO username;
