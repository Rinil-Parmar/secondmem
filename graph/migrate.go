package graph

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func (g *Graph) migrate() error {
	// Ensure schema_migrations table exists
	_, err := g.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read all migration files
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	// Sort by filename
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := strings.TrimSuffix(entry.Name(), ".sql")

		// Check if already applied
		var count int
		g.db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
		if count > 0 {
			continue
		}

		// Read and execute migration
		content, err := migrationFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		// Execute each statement separately
		statements := strings.Split(string(content), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := g.db.Exec(stmt); err != nil {
				return fmt.Errorf("migration %s failed: %w\nStatement: %s", entry.Name(), err, stmt)
			}
		}

		// Record migration
		_, err = g.db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}
