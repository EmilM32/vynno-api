-- +goose Up

-- A token issued for the local read-only MCP (cmd/mcp token) is scope 'mcp'.
-- The HTTP API accepts only 'session' tokens, so a leaked MCP token cannot
-- write, change credentials, or sign in to the SPA.
ALTER TABLE auth_tokens
    ADD COLUMN scope TEXT NOT NULL DEFAULT 'session'
        CONSTRAINT auth_tokens_scope_check CHECK (scope IN ('session', 'mcp'));

-- +goose Down

DELETE FROM auth_tokens WHERE scope <> 'session';
ALTER TABLE auth_tokens DROP COLUMN scope;
