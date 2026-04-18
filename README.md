# secondmem

AI-powered local knowledge management CLI — your second memory.

## What is secondmem?

secondmem is an automated, local librarian for your personal knowledge. When you read a great tweet, PDF, or blog post, you pass it to the CLI. The AI reads the text, figures out what it's about, summarizes it, and writes it into a `.md` file in the correct folder. Everything stays on your local machine — no cloud, no complex vector databases, just clean, readable markdown files.

## Features

- **Smart Ingestion** — AI classifies and organizes content into topic-based markdown files
- **Knowledge Graph** — SQLite + FTS5 powered LORE-GRAPH for fast searching
- **Natural Language Queries** — Ask questions and get answers from your stored knowledge
- **Auto-Rebalancing** — Keeps files organized with a 1,116-line limit per file
- **Cross-References** — Bidirectional links between related knowledge entries
- **Deduplication** — SHA256 + semantic similarity prevents duplicate content
- **Local First** — All data stored as plain markdown on your filesystem

## Installation

```bash
go install github.com/Rinil-Parmar/secondmem@latest
```

Or build from source:

```bash
git clone https://github.com/Rinil-Parmar/secondmem.git
cd secondmem
go build -o secondmem .
```

## Quick Start

```bash
# Initialize your knowledge base
secondmem init

# Set your OpenAI API key
secondmem config set openai.api_key sk-...

# Ingest some knowledge
secondmem ingest "Transformers use self-attention mechanisms to process sequences in parallel, unlike RNNs which process sequentially."

# Ingest from a file
secondmem ingest path/to/article.txt

# Ask a question
secondmem ask "What do I know about transformers?"

# View your knowledge tree
secondmem tree

# Get stats
secondmem stats
```

## Commands

| Command | Description |
|---------|-------------|
| `secondmem init` | Initialize the knowledge base |
| `secondmem ingest <text\|file>` | Ingest content into the knowledge base |
| `secondmem ask "question"` | Query your knowledge base |
| `secondmem rebalance` | Run maintenance on the knowledge base |
| `secondmem tree` | Display knowledge structure |
| `secondmem stats` | Show knowledge base statistics |
| `secondmem validate` | Check integrity of the knowledge base |
| `secondmem graph` | Manage the LORE-GRAPH |
| `secondmem config` | View and update configuration |

## Architecture

- **Knowledge Files** — Plain `.md` files in `~/.secondmem/knowledge/`
- **LORE-GRAPH** — SQLite database with FTS5 for fast full-text search
- **skill.md** — Agent constitution that guides AI behavior
- **hierarchy.md** — Auto-generated table of contents per directory

## License

MIT
