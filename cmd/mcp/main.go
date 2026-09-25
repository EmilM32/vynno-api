// Command mcp is a local stdio MCP server. It reads one account's focus history.
// The account is the Vynno session token in VYNNO_MCP_TOKEN (the same secret the
// API accepts as Authorization: Bearer or the vynno_session cookie).
//
//	go run ./cmd/mcp token
//
// checks VYNNO_MCP_EMAIL and VYNNO_MCP_PASSWORD, inserts a session row, and prints
// the raw token once on stdout. Put that line in .env as VYNNO_MCP_TOKEN.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/EmilM32/vynno-api/internal/agentread"
	"github.com/EmilM32/vynno-api/internal/config"
	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "token":
			os.Exit(runToken())
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
			os.Exit(2)
		}
	}
	os.Exit(runServer())
}

func runServer() int {
	if _, err := config.DatabaseURL(); err != nil {
		slog.Error(err.Error())
		return 1
	}
	token, err := sessionToken()
	if err != nil {
		slog.Error(err.Error())
		return 1
	}
	db, pg, svc, err := open()
	if err != nil {
		slog.Error(err.Error())
		return 1
	}
	defer func() { _ = db.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := svc.ResolveToken(ctx, token); err != nil {
		slog.Error(authFailure(err))
		return 1
	}

	tools := &agentread.Tools{
		Read: pg,
		Resolve: func(ctx context.Context) (uuid.UUID, error) {
			return svc.ResolveToken(ctx, token)
		},
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "vynno", Version: "v1.0.0"}, nil)
	agentread.Register(server, tools)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		slog.Error("mcp server stopped", "err", err)
		return 1
	}
	return 0
}

func runToken() int {
	db, _, svc, err := open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	defer func() { _ = db.Close() }()

	email := strings.TrimSpace(os.Getenv("VYNno_MCP_EMAIL"))
	password := os.Getenv("VYNno_MCP_PASSWORD")
	if email == "" || password == "" {
		fmt.Fprintln(os.Stderr, "VYNno_MCP_EMAIL and VYNNO_MCP_PASSWORD are required")
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := svc.Login(ctx, service.LoginInput{Email: email, Password: password})
	if err != nil {
		if de, ok := domain.AsError(err); ok && de.Code == domain.CodeInvalidCredentials {
			fmt.Fprintln(os.Stderr, "invalid email or password")
			return 1
		}
		fmt.Fprintln(os.Stderr, "could not issue a token")
		return 1
	}
	fmt.Fprintln(os.Stderr, "issued a 30-day session token; put it in .env as VYNNO_MCP_TOKEN. Password reset revokes it.")
	if _, err := fmt.Fprintln(os.Stdout, res.Token); err != nil {
		return 1
	}
	return 0
}

func open() (*sql.DB, *store.Postgres, *service.Service, error) {
	databaseURL, err := config.DatabaseURL()
	if err != nil {
		return nil, nil, nil, err
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, nil, nil, errors.New("could not open postgres")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, nil, nil, errors.New("postgres is not reachable; start it and check DATABASE_URL")
	}
	pg := store.NewPostgres(db)
	return db, pg, service.New(pg, nil), nil
}

func sessionToken() (string, error) {
	token := strings.TrimSpace(os.Getenv("VYNno_MCP_TOKEN"))
	if token == "" || strings.Contains(token, "${") {
		return "", errors.New("VYNno_MCP_TOKEN is required in .env or the environment. Set VYNNO_MCP_EMAIL and VYNNO_MCP_PASSWORD, run `go run ./cmd/mcp token`, and paste the printed value")
	}
	return token, nil
}

func authFailure(err error) string {
	if de, ok := domain.AsError(err); ok && de.Code == domain.CodeUnauthorized {
		return "VYNno_MCP_TOKEN was rejected (missing, expired, or revoked)"
	}
	return "could not check VYNNO_MCP_TOKEN: " + err.Error()
}
