package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/abueno/go-auth-jwt/internal/config"
	"github.com/abueno/go-auth-jwt/internal/db"
	_ "github.com/lib/pq"
)

func main() {
	var (
		command         string
		steps           int
		version         int
		migrationsPath  string
		databaseDSN     string
		useEmbedded     bool
	)

	flag.StringVar(&command, "command", "up", "Migration command: up, down, steps, version, force")
	flag.IntVar(&steps, "steps", 0, "Number of migration steps (positive for up, negative for down)")
	flag.IntVar(&version, "version", 0, "Force migration to specific version")
	flag.StringVar(&migrationsPath, "path", "./migrations", "Path to migrations directory")
	flag.StringVar(&databaseDSN, "database", "", "Database connection string (overrides environment)")
	flag.BoolVar(&useEmbedded, "embedded", false, "Use embedded migrations")
	flag.Parse()

	// Get database DSN
	dsn := databaseDSN
	if dsn == "" {
		dsn = os.Getenv("DATABASE_DSN")
	}
	if dsn == "" {
		log.Fatal("DATABASE_DSN is required")
	}

	// Connect to database
	database, err := db.New(&config.DatabaseConfig{
		DSN: dsn,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations based on command
	switch command {
	case "up":
		fmt.Println("Running all pending migrations...")
		if useEmbedded {
			migrator := db.NewMigrator(database.DB, db.MigrationConfig{})
			if err := migrator.Up(); err != nil {
				log.Fatalf("Failed to run migrations: %v", err)
			}
		} else {
			if err := db.RunMigrationsFromPath(database.DB, migrationsPath, db.MigrationConfig{}); err != nil {
				log.Fatalf("Failed to run migrations: %v", err)
			}
		}
		fmt.Println("Migrations completed successfully!")

	case "down":
		fmt.Println("Rolling back last migration...")
		migrator := db.NewMigrator(database.DB, db.MigrationConfig{})
		if err := migrator.Down(); err != nil {
			log.Fatalf("Failed to rollback migration: %v", err)
		}
		fmt.Println("Rollback completed successfully!")

	case "steps":
		if steps == 0 {
			log.Fatal("Steps count is required for steps command")
		}
		fmt.Printf("Running %d migration steps...\n", steps)
		migrator := db.NewMigrator(database.DB, db.MigrationConfig{})
		if err := migrator.Steps(steps); err != nil {
			log.Fatalf("Failed to run migration steps: %v", err)
		}
		fmt.Println("Migration steps completed successfully!")

	case "version":
		migrator := db.NewMigrator(database.DB, db.MigrationConfig{})
		v, dirty, err := migrator.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Current version: %d (dirty: %v)\n", v, dirty)

	case "force":
		if version == 0 {
			log.Fatal("Version is required for force command")
		}
		fmt.Printf("Forcing migration to version %d...\n", version)
		fmt.Println("WARNING: This is a dangerous operation!")
		
		// Add confirmation
		fmt.Print("Are you sure? (yes/no): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Operation cancelled")
			return
		}

		migrator := db.NewMigrator(database.DB, db.MigrationConfig{})
		if err := migrator.Force(version); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		fmt.Printf("Forced to version %d successfully!\n", version)

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}