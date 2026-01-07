// LifeLogger LL-Journal
// https://api.lifelogger.life
// company: Tellurian Corp (https://www.telluriancorp.com)
// created in: December 2025

package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations runs all database migrations in order
func RunMigrations(db *sql.DB) error {
	ctx := context.Background()

	// Find migrations directory
	migrationsDir, err := findMigrationsDir()
	if err != nil {
		return fmt.Errorf("failed to find migrations directory: %w", err)
	}

	// Ensure migrations table exists
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Get all SQL files and sort them
	migrationFiles, err := getMigrationFiles(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	// Run each migration
	for _, migrationFile := range migrationFiles {
		migrationName := filepath.Base(migrationFile)

		// Check if migration has already been run
		alreadyRun, err := isMigrationRun(ctx, db, migrationName)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if alreadyRun {
			fmt.Printf("Migration %s already applied, skipping\n", migrationName)
			continue
		}

		fmt.Printf("Running migration: %s\n", migrationName)

		// Read SQL file
		sql, err := os.ReadFile(migrationFile)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migrationFile, err)
		}

		// Execute migration in a transaction
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Execute SQL statements
		statements := splitSQL(string(sql))
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.HasPrefix(stmt, "--") {
				continue
			}
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to execute migration statement: %w\nStatement: %s", err, stmt)
			}
		}

		// Record migration as run
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (migration_name, applied_at) VALUES ($1, NOW())",
			migrationName); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration: %w", err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration: %w", err)
		}

		fmt.Printf("Migration %s completed successfully\n", migrationName)
	}

	return nil
}

// findMigrationsDir finds the migrations directory
func findMigrationsDir() (string, error) {
	possiblePaths := []string{
		"migrations",
		"./migrations",
		"../migrations",
		"LL-journal/migrations",
	}

	// Try current working directory
	if cwd, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths,
			filepath.Join(cwd, "migrations"),
			filepath.Join(cwd, "LL-journal", "migrations"),
		)
	}

	// Try executable directory
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		possiblePaths = append(possiblePaths,
			filepath.Join(execDir, "migrations"),
			filepath.Join(execDir, "LL-journal", "migrations"),
		)
	}

	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path, nil
		}
	}

	return "", fmt.Errorf("migrations directory not found. Tried: %v", possiblePaths)
}

// getMigrationFiles gets all SQL files from the migrations directory
func getMigrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist
func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			migration_name VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

// isMigrationRun checks if a migration has already been run
func isMigrationRun(ctx context.Context, db *sql.DB, migrationName string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE migration_name = $1)",
		migrationName).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}

// splitSQL splits SQL into individual statements
func splitSQL(sql string) []string {
	// Remove comments and split by semicolon
	lines := strings.Split(sql, "\n")
	var statements []string
	var currentStmt strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		currentStmt.WriteString(line)
		currentStmt.WriteString("\n")
		// If line ends with semicolon, it's the end of a statement
		if strings.HasSuffix(strings.TrimSpace(line), ";") {
			stmt := strings.TrimSpace(currentStmt.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			currentStmt.Reset()
		}
	}

	// Add any remaining statement
	if currentStmt.Len() > 0 {
		stmt := strings.TrimSpace(currentStmt.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}

	return statements
}
