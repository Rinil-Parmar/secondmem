-- LORE-GRAPH v0.1 Initial Schema

CREATE TABLE IF NOT EXISTS nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT UNIQUE NOT NULL,
    directory TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    keywords TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '',
    node_type TEXT NOT NULL DEFAULT 'file',
    line_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    target_id INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    edge_type TEXT NOT NULL DEFAULT 'related',
    weight REAL NOT NULL DEFAULT 1.0,
    UNIQUE(source_id, target_id)
);

CREATE VIRTUAL TABLE IF NOT EXISTS nodes_fts USING fts5(
    title, summary, keywords, content=nodes, content_rowid=id
);

-- Triggers to keep FTS in sync
CREATE TRIGGER IF NOT EXISTS nodes_ai AFTER INSERT ON nodes BEGIN
    INSERT INTO nodes_fts(rowid, title, summary, keywords) VALUES (new.id, new.title, new.summary, new.keywords);
END;

CREATE TRIGGER IF NOT EXISTS nodes_ad AFTER DELETE ON nodes BEGIN
    INSERT INTO nodes_fts(nodes_fts, rowid, title, summary, keywords) VALUES('delete', old.id, old.title, old.summary, old.keywords);
END;

CREATE TRIGGER IF NOT EXISTS nodes_au AFTER UPDATE ON nodes BEGIN
    INSERT INTO nodes_fts(nodes_fts, rowid, title, summary, keywords) VALUES('delete', old.id, old.title, old.summary, old.keywords);
    INSERT INTO nodes_fts(rowid, title, summary, keywords) VALUES (new.id, new.title, new.summary, new.keywords);
END;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
