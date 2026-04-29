package graph

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

// Chunk is a piece of a document with its embedding vector.
type Chunk struct {
	ID         int64
	NodeID     int64
	ChunkIndex int
	Content    string
	FilePath   string
}

// SaveChunk stores a text chunk and its embedding vector for a given node.
// Replaces any existing chunk at the same (node_id, chunk_index).
func (g *Graph) SaveChunk(nodeID int64, index int, content string, embedding []float32) error {
	blob := float32ToBlob(embedding)
	_, err := g.db.Exec(`
		INSERT INTO chunks (node_id, chunk_index, content, embedding)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(node_id, chunk_index) DO UPDATE SET
			content   = excluded.content,
			embedding = excluded.embedding`,
		nodeID, index, content, blob,
	)
	if err != nil {
		return fmt.Errorf("save chunk: %w", err)
	}
	return nil
}

// DeleteChunksByNode removes all chunks for a node (used before re-indexing).
func (g *Graph) DeleteChunksByNode(nodeID int64) error {
	_, err := g.db.Exec("DELETE FROM chunks WHERE node_id = ?", nodeID)
	return err
}

// SearchByVector loads all chunk embeddings and returns the top-K by cosine similarity.
// For a personal knowledge base this is fast enough in-memory.
func (g *Graph) SearchByVector(query []float32, topK int) ([]Chunk, error) {
	rows, err := g.db.Query(`
		SELECT c.id, c.node_id, c.chunk_index, c.content, c.embedding, n.file_path
		FROM chunks c
		JOIN nodes n ON n.id = c.node_id`)
	if err != nil {
		return nil, fmt.Errorf("vector search query: %w", err)
	}
	defer rows.Close()

	type scored struct {
		chunk Chunk
		score float64
	}
	var results []scored

	for rows.Next() {
		var c Chunk
		var blob []byte
		if err := rows.Scan(&c.ID, &c.NodeID, &c.ChunkIndex, &c.Content, &blob, &c.FilePath); err != nil {
			return nil, err
		}
		vec := blobToFloat32(blob)
		score := cosine(query, vec)
		results = append(results, scored{c, score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	out := make([]Chunk, 0, topK)
	for i := 0; i < topK && i < len(results); i++ {
		out = append(out, results[i].chunk)
	}
	return out, nil
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func float32ToBlob(vec []float32) []byte {
	blob := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(blob[i*4:], math.Float32bits(v))
	}
	return blob
}

func blobToFloat32(blob []byte) []float32 {
	vec := make([]float32, len(blob)/4)
	for i := range vec {
		bits := binary.LittleEndian.Uint32(blob[i*4:])
		vec[i] = math.Float32frombits(bits)
	}
	return vec
}
