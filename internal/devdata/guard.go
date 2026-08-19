package devdata

import (
	"fmt"
	"net/url"
	"strings"
)

// DevDatabase is the only database seed/reset may touch.
const DevDatabase = "vynno_dev"

// DatabaseName returns the PostgreSQL database name from a URL.
func DatabaseName(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("DATABASE_URL: %w", err)
	}
	name := strings.Trim(u.Path, "/")
	if name == "" || strings.Contains(name, "/") {
		return "", fmt.Errorf("DATABASE_URL has no database name")
	}
	return name, nil
}

// RequireDevDatabase errors unless raw points at vynno_dev.
func RequireDevDatabase(raw string) error {
	name, err := DatabaseName(raw)
	if err != nil {
		return err
	}
	if name != DevDatabase {
		return fmt.Errorf("refusing database %q; seed/reset only run against %s", name, DevDatabase)
	}
	return nil
}
