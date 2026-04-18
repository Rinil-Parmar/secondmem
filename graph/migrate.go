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

		// Execute the full migration as one batch.
		// We use ExecContext-compatible approach: split on ";\n" at top level
		// but handle triggers (BEGIN...END) as single statements.
		stmts := splitSQL(string(content))
		for _, stmt := range stmts {
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

// splitSQL splits SQL content into individual statements, correctly handling
// triggers that contain semicolons inside BEGIN...END blocks.
func splitSQL(content string) []string {
	var stmts []string
	var current strings.Builder
	inTrigger := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(trimmed, "--") || trimmed == "" {
			if inTrigger {
				current.WriteString(line)
				current.WriteString("\n")
			}
			continue
		}

		upper := strings.ToUpper(trimmed)

		// Detect trigger start
		if strings.Contains(upper, "CREATE TRIGGER") {
			inTrigger = true
		}

		current.WriteString(line)
		current.WriteString("\n")

		if inTrigger {
			// Trigger ends with "END;"
			if strings.HasSuffix(upper, "END;") {
				stmts = append(stmts, strings.TrimSpace(current.String()))
				current.Reset()
				inTrigger = false
			}
		} else if strings.HasSuffix(trimmed, ";") {
			stmts = append(stmts, strings.TrimSpace(current.String()))
			current.Reset()
		}
	}

	// Catch any remaining statement
	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		stmts = append(stmts, remaining)
	}

	return stmts
}
