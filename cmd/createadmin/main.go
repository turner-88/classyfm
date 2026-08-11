// Command createadmin bootstraps the first admin/editor user for the admin panel,
// since there is no public signup flow.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/classyfm/classyfm/internal/config"
	"github.com/classyfm/classyfm/internal/db"
	"github.com/classyfm/classyfm/internal/db/sqlc"
)

func main() {
	email := flag.String("email", "", "admin email (required)")
	password := flag.String("password", "", "admin password (required)")
	name := flag.String("name", "Admin", "display name")
	role := flag.String("role", "superadmin", "role: superadmin|admin")
	envFile := flag.String("env", ".env", "path to a .env file to load DATABASE_DSN etc. from (skipped if absent)")
	flag.Parse()

	// Run by hand rather than under systemd/Makefile, so load .env ourselves; an
	// already-set DATABASE_DSN still wins. See config.LoadEnvFile.
	if err := config.LoadEnvFile(*envFile); err != nil {
		log.Fatalf("load %s: %v", *envFile, err)
	}

	*email = strings.TrimSpace(strings.ToLower(*email))
	if *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "usage: createadmin -email you@example.com -password secret [-name \"Full Name\"] [-role superadmin]")
		os.Exit(2)
	}
	if *role != "superadmin" && *role != "admin" {
		log.Fatalf("invalid role %q: must be superadmin or admin", *role)
	}

	cfg := config.Load()
	if cfg.DatabaseDSN == "" {
		log.Fatal("DATABASE_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("connect to mysql: %v", err)
	}
	defer pool.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	q := sqlc.New(pool)
	res, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        *email,
		PasswordHash: string(hash),
		Name:         *name,
		Role:         sqlc.UsersRole(*role),
		IsActive:     true,
	})
	if err != nil {
		log.Fatalf("create user: %v", err)
	}
	id, _ := res.LastInsertId()
	fmt.Printf("created user #%d: %s (%s)\n", id, *email, *role)
}
