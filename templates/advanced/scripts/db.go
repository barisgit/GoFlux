package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string
	SSLMode  string
}

func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	testFlag := contains(os.Args, "--test")
	resetFlag := contains(os.Args, "--reset")

	switch command {
	case "start":
		if err := startDatabase(testFlag, resetFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "stop":
		if err := stopDatabase(testFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "migrate":
		if err := runMigrations(testFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "seed":
		if err := seedDatabase(testFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "generate":
		if err := generateCode(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "setup":
		if err := setupDatabase(testFlag, resetFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := checkDatabaseStatus(testFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`
Database Management Script

Usage: go run scripts/db.go <command> [flags]

Commands:
  start     Start the PostgreSQL database
  stop      Stop the PostgreSQL database
  migrate   Run database migrations
  seed      Seed database with sample data
  generate  Generate Go code from SQL queries
  setup     Complete setup (start + migrate + generate + seed)
  status    Check database status

Flags:
  --test    Use test database instead of development database
  --reset   Reset database (removes all data)

Examples:
  go run scripts/db.go setup           # Setup development database
  go run scripts/db.go setup --test    # Setup test database
  go run scripts/db.go start           # Start development database
  go run scripts/db.go migrate --test  # Run migrations on test database`)
}

func getDatabaseConfig(testFlag bool) DatabaseConfig {
	if testFlag {
		return DatabaseConfig{
			Host:     getEnvOrDefault("DB_TEST_HOST", "localhost"),
			Port:     getEnvOrDefault("DB_TEST_PORT", "5433"),
			Database: getEnvOrDefault("DB_TEST_NAME", "goflux_test"),
			Username: getEnvOrDefault("DB_TEST_USER", "goflux_user"),
			Password: getEnvOrDefault("DB_TEST_PASSWORD", "goflux_pass"),
			SSLMode:  getEnvOrDefault("DB_TEST_SSLMODE", "disable"),
		}
	}

	return DatabaseConfig{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		Database: getEnvOrDefault("DB_NAME", "goflux_dev"),
		Username: getEnvOrDefault("DB_USER", "goflux_user"),
		Password: getEnvOrDefault("DB_PASSWORD", "goflux_pass"),
		SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func startDatabase(testFlag, resetFlag bool) error {
	serviceName := "postgres"
	if testFlag {
		serviceName = "postgres_test"
	}

	fmt.Printf("🐘 Starting %s database...\n", getDbType(testFlag))

	if isContainerRunning(serviceName) {
		fmt.Printf("✅ Database container is already running\n")
		return nil
	}

	if resetFlag {
		fmt.Println("🔄 Resetting database...")
		if err := executeCommand("docker-compose", "down", serviceName); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	if err := executeCommand("docker-compose", "up", "-d", serviceName); err != nil {
		return fmt.Errorf("failed to start database: %w", err)
	}

	// Wait for database to be ready
	config := getDatabaseConfig(testFlag)
	if err := waitForDatabase(config, 30); err != nil {
		return fmt.Errorf("database not ready: %w", err)
	}

	fmt.Println("✅ Database started successfully!")
	return nil
}

func stopDatabase(testFlag bool) error {
	serviceName := "postgres"
	if testFlag {
		serviceName = "postgres_test"
	}

	fmt.Printf("🛑 Stopping %s database...\n", getDbType(testFlag))

	if err := executeCommand("docker-compose", "stop", serviceName); err != nil {
		return fmt.Errorf("failed to stop database: %w", err)
	}

	fmt.Println("✅ Database stopped successfully!")
	return nil
}

func runMigrations(testFlag bool) error {
	fmt.Printf("📦 Running migrations for %s database...\n", getDbType(testFlag))

	// Install goose if not present
	if !isCommandAvailable("goose") {
		fmt.Println("📥 Installing goose...")
		if err := executeCommand("go", "install", "github.com/pressly/goose/v3/cmd/goose@latest"); err != nil {
			return fmt.Errorf("failed to install goose: %w", err)
		}
	}

	config := getDatabaseConfig(testFlag)
	migrationsDir := "sql/migrations"

	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found: %s", migrationsDir)
	}

	args := []string{
		"-dir", migrationsDir,
		"postgres", config.ConnectionString(),
		"up",
	}

	if err := executeCommand("goose", args...); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Println("✅ Migrations completed successfully!")
	return nil
}

func seedDatabase(testFlag bool) error {
	fmt.Printf("🌱 Seeding %s database...\n", getDbType(testFlag))

	seedFile := "sql/seed.sql"
	if _, err := os.Stat(seedFile); os.IsNotExist(err) {
		fmt.Println("⚠️  No seed file found, skipping seeding")
		return nil
	}

	config := getDatabaseConfig(testFlag)

	// Use psql to run the seed file
	cmd := exec.Command("psql", config.ConnectionString(), "-f", seedFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to seed database: %w", err)
	}

	fmt.Println("✅ Database seeded successfully!")
	return nil
}

func generateCode() error {
	fmt.Println("🔧 Generating Go code from SQL...")

	// Install sqlc if not present
	if !isCommandAvailable("sqlc") {
		fmt.Println("📥 Installing sqlc...")
		if err := executeCommand("go", "install", "github.com/sqlc-dev/sqlc/cmd/sqlc@latest"); err != nil {
			return fmt.Errorf("failed to install sqlc: %w", err)
		}
	}

	// Check if sqlc.yaml exists
	if _, err := os.Stat("sqlc.yaml"); os.IsNotExist(err) {
		return fmt.Errorf("sqlc.yaml configuration file not found")
	}

	if err := executeCommand("sqlc", "generate"); err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	fmt.Println("✅ Code generation completed successfully!")
	return nil
}

func setupDatabase(testFlag, resetFlag bool) error {
	fmt.Println("🚀 Setting up database...")

	if err := startDatabase(testFlag, resetFlag); err != nil {
		return err
	}

	if err := runMigrations(testFlag); err != nil {
		return err
	}

	if err := generateCode(); err != nil {
		return err
	}

	if err := seedDatabase(testFlag); err != nil {
		return err
	}

	fmt.Println("✅ Database setup complete!")
	return nil
}

func checkDatabaseStatus(testFlag bool) error {
	fmt.Printf("🔍 Checking %s database status...\n", getDbType(testFlag))

	config := getDatabaseConfig(testFlag)

	// Check container status
	serviceName := "postgres"
	if testFlag {
		serviceName = "postgres_test"
	}

	if isContainerRunning(serviceName) {
		fmt.Printf("✅ Container: %s is running\n", serviceName)
	} else {
		fmt.Printf("❌ Container: %s is not running\n", serviceName)
		return nil
	}

	// Check database connection
	conn, err := pgx.Connect(context.Background(), config.ConnectionString())
	if err != nil {
		fmt.Printf("❌ Connection: Failed to connect - %v\n", err)
		return nil
	}
	defer conn.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		fmt.Printf("❌ Connection: Database not responding - %v\n", err)
		return nil
	}

	fmt.Printf("✅ Connection: Database is responding\n")
	fmt.Printf("📊 Config: %s:%s/%s\n", config.Host, config.Port, config.Database)

	return nil
}

func isContainerRunning(serviceName string) bool {
	cmd := exec.Command("docker-compose", "ps", "-q", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func waitForDatabase(config DatabaseConfig, timeoutSeconds int) error {
	fmt.Printf("⏳ Waiting for database at %s:%s...\n", config.Host, config.Port)

	start := time.Now()
	timeout := time.Duration(timeoutSeconds) * time.Second

	for time.Since(start) < timeout {
		conn, err := pgx.Connect(context.Background(), config.ConnectionString())
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err = conn.Ping(ctx)
			cancel()
			conn.Close(context.Background())

			if err == nil {
				fmt.Println("✅ Database is ready!")
				return nil
			}
		}

		fmt.Print(".")
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("database not ready after %d seconds", timeoutSeconds)
}

func executeCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getDbType(testFlag bool) string {
	if testFlag {
		return "test"
	}
	return "development"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
