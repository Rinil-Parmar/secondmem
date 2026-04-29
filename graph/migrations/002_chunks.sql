CREATE TABLE IF NOT EXISTS chunks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id     INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    content     TEXT    NOT NULL,
    embedding   BLOB    NOT NULL,
    UNIQUE(node_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_chunks_node_id ON chunks(node_id);
