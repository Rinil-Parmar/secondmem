package graph

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// Node represents a knowledge file in the graph.
type Node struct {
	ID        int64
	FilePath  string
	Directory string
	Title     string
	Summary   string
	Keywords  string
	Tags      string
	NodeType  string
	LineCount int
}

// Edge represents a connection between two nodes.
type Edge struct {
	ID       int64
	SourceID int64
	TargetID int64
	EdgeType string
	Weight   float64
}

// Graph manages the SQLite LORE-GRAPH database.
type Graph struct {
	db *sql.DB
}

// Open creates or opens a graph database at the given path.
func Open(dbPath string) (*Graph, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	g := &Graph{db: db}
	if err := g.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return g, nil
}

// Close closes the database connection.
func (g *Graph) Close() error {
	return g.db.Close()
}

// UpsertNode inserts or updates a node in the graph.
func (g *Graph) UpsertNode(node Node) (int64, error) {
	result, err := g.db.Exec(`
		INSERT INTO nodes (file_path, directory, title, summary, keywords, tags, node_type, line_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(file_path) DO UPDATE SET
			directory = excluded.directory,
			title = excluded.title,
			summary = excluded.summary,
			keywords = excluded.keywords,
			tags = excluded.tags,
			node_type = excluded.node_type,
			line_count = excluded.line_count,
			updated_at = CURRENT_TIMESTAMP`,
		node.FilePath, node.Directory, node.Title, node.Summary,
		node.Keywords, node.Tags, node.NodeType, node.LineCount,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert node: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		// If upsert updated an existing row, look up the ID
		row := g.db.QueryRow("SELECT id FROM nodes WHERE file_path = ?", node.FilePath)
		if err := row.Scan(&id); err != nil {
			return 0, fmt.Errorf("failed to get node ID: %w", err)
		}
	}
	return id, nil
}

// DeleteNode removes a node and its edges from the graph.
func (g *Graph) DeleteNode(filePath string) error {
	tx, err := g.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var nodeID int64
	err = tx.QueryRow("SELECT id FROM nodes WHERE file_path = ?", filePath).Scan(&nodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	_, err = tx.Exec("DELETE FROM edges WHERE source_id = ? OR target_id = ?", nodeID, nodeID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM nodes WHERE id = ?", nodeID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// AddEdge creates a relationship between two nodes.
func (g *Graph) AddEdge(sourceID, targetID int64, edgeType string, weight float64) error {
	_, err := g.db.Exec(`
		INSERT INTO edges (source_id, target_id, edge_type, weight)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(source_id, target_id) DO UPDATE SET
			edge_type = excluded.edge_type,
			weight = excluded.weight`,
		sourceID, targetID, edgeType, weight,
	)
	return err
}

// Search performs a full-text search and returns matching nodes.
func (g *Graph) Search(query string, limit int) ([]Node, error) {
	if limit <= 0 {
		limit = 5
	}

	// Escape and prepare FTS query — use OR for broader matching
	tokens := strings.Fields(query)
	ftsQuery := strings.Join(tokens, " OR ")

	rows, err := g.db.Query(`
		SELECT n.id, n.file_path, n.directory, n.title, n.summary, n.keywords, n.tags, n.node_type, n.line_count
		FROM nodes_fts fts
		JOIN nodes n ON n.id = fts.rowid
		WHERE nodes_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("FTS search failed: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.ID, &n.FilePath, &n.Directory, &n.Title, &n.Summary, &n.Keywords, &n.Tags, &n.NodeType, &n.LineCount); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// GetRelated returns nodes connected to the given node via edges.
func (g *Graph) GetRelated(nodeID int64) ([]Node, error) {
	rows, err := g.db.Query(`
		SELECT n.id, n.file_path, n.directory, n.title, n.summary, n.keywords, n.tags, n.node_type, n.line_count
		FROM nodes n
		JOIN edges e ON (e.target_id = n.id AND e.source_id = ?) OR (e.source_id = n.id AND e.target_id = ?)
		WHERE n.id != ?`, nodeID, nodeID, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.ID, &n.FilePath, &n.Directory, &n.Title, &n.Summary, &n.Keywords, &n.Tags, &n.NodeType, &n.LineCount); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// Stats returns basic graph statistics.
func (g *Graph) Stats() (nodeCount, edgeCount, ftsCount int, err error) {
	g.db.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&nodeCount)
	g.db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&edgeCount)
	g.db.QueryRow("SELECT COUNT(*) FROM nodes_fts").Scan(&ftsCount)
	return
}

// GetNodeByPath returns a node by its file path.
func (g *Graph) GetNodeByPath(filePath string) (*Node, error) {
	var n Node
	err := g.db.QueryRow(`
		SELECT id, file_path, directory, title, summary, keywords, tags, node_type, line_count
		FROM nodes WHERE file_path = ?`, filePath).
		Scan(&n.ID, &n.FilePath, &n.Directory, &n.Title, &n.Summary, &n.Keywords, &n.Tags, &n.NodeType, &n.LineCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

// AllNodes returns all nodes in the graph.
func (g *Graph) AllNodes() ([]Node, error) {
	rows, err := g.db.Query(`
		SELECT id, file_path, directory, title, summary, keywords, tags, node_type, line_count
		FROM nodes ORDER BY directory, title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.ID, &n.FilePath, &n.Directory, &n.Title, &n.Summary, &n.Keywords, &n.Tags, &n.NodeType, &n.LineCount); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}
