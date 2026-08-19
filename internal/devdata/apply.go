package devdata

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/EmilM32/vynno-api/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// Keep in sync with internal/store/migrations. Do not truncate goose_db_version.
const wipeSQL = `
TRUNCATE TABLE
	auth_tokens,
	avatars,
	sessions,
	projects,
	profiles,
	users
RESTART IDENTITY CASCADE;
`

// Wipe removes application rows and leaves goose_db_version intact.
func Wipe(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, wipeSQL); err != nil {
		return fmt.Errorf("wipe: %w", err)
	}
	return nil
}

// Apply inserts the dataset through Store. Passwords are bcrypt-hashed here.
func Apply(ctx context.Context, s store.Store, ds Dataset) error {
	for _, acc := range ds.Accounts {
		hash, err := bcrypt.GenerateFromPassword([]byte(acc.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash %s: %w", acc.Username, err)
		}
		if err := s.CreateAccount(ctx, store.Account{
			ID:           acc.ID,
			Username:     acc.Username,
			PasswordHash: string(hash),
		}); err != nil {
			return fmt.Errorf("account %s: %w", acc.Username, err)
		}
		if err := s.CreateProfile(ctx, acc.ID, acc.Profile); err != nil {
			return fmt.Errorf("profile %s: %w", acc.Username, err)
		}
		for _, p := range acc.Projects {
			if _, err := s.CreateProject(ctx, acc.ID, p); err != nil {
				return fmt.Errorf("project %s/%s: %w", acc.Username, p.Name, err)
			}
		}
		for _, sess := range acc.Sessions {
			if _, err := s.CreateSession(ctx, acc.ID, sess); err != nil {
				return fmt.Errorf("session %s/%s: %w", acc.Username, sess.ID, err)
			}
		}
	}
	return nil
}
