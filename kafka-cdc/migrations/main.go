package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/db/postgres"
)

func setupDatabase(service string) error {
	envPath := filepath.Join("cmd", service+"-server", ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: No .env file found at %s, using environment variables", envPath)
	} else {
		log.Printf("Loaded .env from %s", envPath)
	}

	cfg := postgres.NewPostgresConfig("")

	postgresURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port,
	)

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer db.Close()

	log.Printf("Creating database '%s' if not exists...", cfg.DBName)
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName))
	if err != nil {

		if isAlreadyExistsError(err) {
			log.Printf("Database '%s' already exists, skipping creation", cfg.DBName)
		} else {
			return fmt.Errorf("failed to create database: %w", err)
		}
	} else {
		log.Printf("Database '%s' created successfully", cfg.DBName)
	}

	log.Println("Applying WAL settings...")
	walSettings := []string{
		"ALTER SYSTEM SET wal_level = 'logical'",
		"ALTER SYSTEM SET max_wal_senders = 4",
		"ALTER SYSTEM SET max_replication_slots = 4",
	}

	for _, setting := range walSettings {
		_, err := db.Exec(setting)
		if err != nil {
			log.Printf("Warning: Failed to apply setting (%s): %v", setting, err)
			log.Println("This is not critical if settings are already configured")
		} else {
			log.Printf("Applied: %s", setting)
		}
	}

	log.Println("WAL settings applied (PostgreSQL restart may be required for changes to take effect)")
	return nil
}

func isAlreadyExistsError(err error) bool {

	return err != nil && (err.Error() == fmt.Sprintf("pq: database \"%s\" already exists", err) ||
		contains(err.Error(), "already exists"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func main() {
	service := flag.String("service", "", "Service name: payment or inventory")
	action := flag.String("action", "up", "Migration action: up, down, or version")
	steps := flag.Int("steps", 0, "Number of migrations to apply (for down)")
	flag.Parse()

	if *service == "" {
		log.Fatal("Please specify -service flag (payment or inventory)")
	}

	log.Println("========================================")
	log.Println("Step 1: Database Setup")
	log.Println("========================================")
	if err := setupDatabase(*service); err != nil {
		log.Fatalf("Database setup failed: %v", err)
	}

	log.Println("======================================")
	log.Println("Step 2: Schema Migrations")
	log.Println("======================================")

	envPath := filepath.Join("cmd", *service+"-server", ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: No .env file found at %s, using environment variables", envPath)
	} else {
		log.Printf("Loaded .env from %s", envPath)
	}

	cfg := postgres.NewPostgresConfig("")

	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName,
	)

	migrationPath := filepath.Join("migrations", *service)

	log.Printf("Running migrations for %s service from %s", *service, migrationPath)

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationPath),
		dbURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	switch *action {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration up failed: %v", err)
		}
		log.Println("Migrations applied successfully")

	case "down":
		if *steps > 0 {
			if err := m.Steps(-*steps); err != nil {
				log.Fatalf("Migration down failed: %v", err)
			}
		} else {
			if err := m.Down(); err != nil {
				log.Fatalf("Migration down failed: %v", err)
			}
		}
		log.Println("Migrations rolled back successfully")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		log.Printf("Current version: %d, Dirty: %v", version, dirty)

	default:
		log.Fatalf("Unknown action: %s (use up, down, or version)", *action)
	}

	log.Println("=================================")
	log.Printf("%s Migration Complete!\n", *service)
	log.Println("=================================")
}
