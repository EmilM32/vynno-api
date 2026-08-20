package devdata

import (
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
)

// DefaultSeedPassword is used for Maya and Rio when SEED_PASSWORD is unset.
const DefaultSeedPassword = "local-dev-password"

// Options control reset/seed generation. Passwords are plaintext; Apply hashes them.
type Options struct {
	Now               time.Time
	BootstrapUsername string
	BootstrapPassword string
	SeedPassword      string
}

// Dataset is a complete playground (or the bootstrap-only reset).
type Dataset struct {
	Accounts []Account
}

// Account is one isolated login plus its projects and sessions.
type Account struct {
	ID            uuid.UUID
	Username      string
	Password      string
	Blurb         string
	Profile       domain.Profile
	Projects      []domain.Project
	ActivityTypes []domain.ActivityType
	Sessions      []domain.Session
}

type persona struct {
	id          uuid.UUID
	username    string // empty → bootstrap username
	displayName string
	blurb       string
	live        bool
	daysBack    int
	skipWeekday float64
	weekendProb float64
	minPerDay   int
	maxPerDay   int
	projects    []projectSpec
}

type projectSpec struct {
	name           string
	code           string
	color          string
	archived       bool
	progress       *int
	ticketPrefix   string
	onlyBeforeDays int
	fixedID        string
	notes          []string
	activities     []string
}

var tagBank = []string{"review", "pair", "hotfix", "spike", "oncall"}

func seedPersonas() []persona {
	p42 := 42
	p68 := 68
	p15 := 15
	p80 := 80
	p35 := 35
	return []persona{
		{
			id:          store.DefaultUserID(),
			displayName: "Alex Dev",
			blurb:       "power user, live session",
			live:        true,
			daysBack:    70,
			skipWeekday: 0.06,
			weekendProb: 0.22,
			minPerDay:   5,
			maxPerDay:   6,
			projects:    alexProjects(p42, p68, p15),
		},
		{
			id:          uuid.MustParse("00000000-0000-4000-8000-000000000002"),
			username:    "maya",
			displayName: "Maya Chen",
			blurb:       "contractor, idle, one archived project",
			daysBack:    42,
			skipWeekday: 0.10,
			weekendProb: 0.12,
			minPerDay:   2,
			maxPerDay:   4,
			projects:    mayaProjects(p80),
		},
		{
			id:          uuid.MustParse("00000000-0000-4000-8000-000000000003"),
			username:    "rio",
			displayName: "Rio Alvarez",
			blurb:       "new-ish account, short history",
			daysBack:    10,
			skipWeekday: 0.05,
			weekendProb: 0.08,
			minPerDay:   2,
			maxPerDay:   3,
			projects:    rioProjects(p35),
		},
	}
}

func alexProjects(p42, p68, p15 int) []projectSpec {
	return []projectSpec{
		{
			name: "Identity", code: "AUTH", color: "#3b82f6", progress: &p68,
			ticketPrefix: "AUTH", fixedID: store.DefaultProject().ID,
			activities: []string{"coding", "debugging", "research", "docs"},
			notes: []string{
				"OIDC callback on Safari",
				"Refresh token rotation",
				"Session cookie SameSite review",
				"Fix CORS preflight on /me",
				"Password hashing cost bump",
				"Remember-me expiry edge cases",
				"Username normalize on login",
				"Bearer token for curl smoke tests",
			},
		},
		{
			name: "API Core", code: "API", color: "#8b5cf6", progress: &p42,
			ticketPrefix: "API",
			activities:   []string{"coding", "debugging", "docs", "deep_work"},
			notes: []string{
				"List sessions newest-first",
				"Project code uniqueness",
				"Pause accounting on stop",
				"sqlc ListProjects includeArchived",
				"Goose migration for avatars",
				"Readyz Postgres ping",
				"Error envelope for invalid_json",
				"Hard-delete last active project",
			},
		},
		{
			name: "Marketing Site", code: "WEB", color: "#10b981",
			ticketPrefix: "WEB",
			activities:   []string{"coding", "docs", "meeting", "other"},
			notes: []string{
				"Hero copy for waitlist",
				"Pricing table layout",
				"OG image for blog post",
				"Fix mobile nav overflow",
				"Analytics event names",
				"FAQ draft with legal",
			},
		},
		{
			name: "Mobile", code: "IOS", color: "#f59e0b",
			ticketPrefix: "IOS",
			activities:   []string{"deep_work", "coding", "research", "debugging"},
			notes: []string{
				"Timer background task",
				"Keychain session restore",
				"Haptics on stop",
				"Dynamic island live activity",
				"Offline queue for pause",
			},
		},
		{
			name: "Support", code: "SUP", color: "#ef4444",
			ticketPrefix: "SUP",
			activities:   []string{"meeting", "maintenance", "other", "debugging"},
			notes: []string{
				"Reproduce cookie loss on Firefox",
				"Reply to timezone DST report",
				"Reset a stuck live session",
				"Triage empty note reports",
				"Pair on onboarding email",
			},
		},
		{
			name: "Research", color: "#06b6d4",
			activities: []string{"research", "docs", "deep_work"},
			notes: []string{
				"Cursor pagination sketches",
				"Insights aggregate options",
				"Manual time-entry shapes",
				"Compare activity-type enums",
				"Notes on local-prod backups",
			},
		},
		{
			name: "Old Billing", code: "BILL", color: "#64748b", archived: true,
			ticketPrefix: "BILL", onlyBeforeDays: 45,
			activities: []string{"maintenance", "coding", "meeting"},
			notes: []string{
				"Invoice PDF layout",
				"Stripe webhook retries",
				"Prune unused plan codes",
				"Archive the billing service",
			},
		},
	}
}

func mayaProjects(p80 int) []projectSpec {
	return []projectSpec{
		{
			name: "Northwind", code: "NWND", color: "#ec4899", progress: &p80,
			ticketPrefix: "NW",
			activities:   []string{"coding", "meeting", "docs"},
			notes: []string{
				"Catalog filter by region",
				"Weekly status with buyer",
				"Fix CSV export encoding",
				"Dashboard empty state",
			},
		},
		{
			name: "Helios", code: "HEL", color: "#14b8a6",
			ticketPrefix: "HEL",
			activities:   []string{"deep_work", "coding", "debugging"},
			notes: []string{
				"Ingest lag after deploy",
				"Shard key review",
				"On-call handoff notes",
				"Query plan for daily rollup",
			},
		},
		{
			name: "Internal", code: "INT", color: "#a855f7",
			activities: []string{"docs", "meeting", "other"},
			notes: []string{
				"Contractor timesheet",
				"Access request for staging",
				"Write the engagement recap",
			},
		},
		{
			name: "Legacy Shop", code: "SHOP", color: "#64748b", archived: true,
			ticketPrefix: "SHOP", onlyBeforeDays: 21,
			activities: []string{"maintenance", "debugging"},
			notes: []string{
				"Patch jQuery checkout",
				"Disable unused promo codes",
			},
		},
	}
}

func rioProjects(p35 int) []projectSpec {
	return []projectSpec{
		{
			name: "Portfolio", code: "PORT", color: "#3b82f6", progress: &p35,
			ticketPrefix: "PORT",
			activities:   []string{"coding", "docs", "other"},
			notes: []string{
				"Case study layout",
				"Compress hero video",
				"Write the about page",
				"Fix dark-mode contrast",
			},
		},
		{
			name: "Learning", code: "LEARN", color: "#8b5cf6",
			activities: []string{"research", "coding", "docs"},
			notes: []string{
				"Go generics workbook",
				"Read the postgres MVCC post",
				"SQLC first queries",
			},
		},
	}
}
