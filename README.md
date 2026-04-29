# secondmem

AI-powered local knowledge management CLI — your second memory.

## What is secondmem?

secondmem is an automated local librarian for your personal knowledge. Pass it anything — a tweet, article, PDF excerpt, or a note from a coding session — and the AI classifies it, summarizes it, and writes it into the right `.md` file in the right folder. Everything stays on your machine as plain, readable markdown.

Built in Go. No cloud. No external vector databases. Just files and your LLM provider.

## Features

- **Smart Ingestion** — AI classifies content into topic-based markdown files automatically
- **RAG Semantic Search** — Embeddings stored in SQLite; `ask` retrieves by meaning, not just keywords
- **LORE-GRAPH** — SQLite + FTS5 knowledge graph; keyword fallback when no chunks matched
- **Natural Language Queries** — Ask questions, get synthesized answers with citations
- **Bidirectional Cross-References** — Related files link to each other automatically
- **Deduplication** — SHA256 exact match + LLM semantic similarity (>70% threshold)
- **Auto-Split / Merge** — Files over 1,116 lines are split by theme; duplicates can be merged
- **7-Step Rebalance** — Maintains hierarchy, dead links, orphans, graph sync, and merge candidates
- **Three LLM Providers** — Ollama (local, default), OpenAI, GitHub Copilot — all support embeddings
- **Local First** — Plain markdown on your filesystem, Git-friendly

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
# Initialize your knowledge base (~/.secondmem/)
secondmem init

# Default provider is Ollama (local). Switch if needed:
secondmem config set model.provider copilot   # GitHub Copilot CLI token auto-detected
secondmem config set model.provider openai
secondmem config set openai.api_key sk-...

# Ingest knowledge
secondmem ingest "Transformers use self-attention to process sequences in parallel, unlike RNNs."
secondmem ingest path/to/article.txt

# Query your knowledge base — natural language works
secondmem ask "what skills should I learn for a co-op?"
secondmem ask "how does deduplication work" --cite
secondmem ask "transformers vs RNNs"

# Browse and maintain
secondmem tree
secondmem stats
secondmem validate
secondmem rebalance
```

## How It Works — Real Example

### 1. Ingest anything

```bash
$ secondmem ingest "RAG (Retrieval Augmented Generation) lets LLMs answer questions using your own documents by embedding chunks into a vector store and retrieving the most semantically similar ones at query time."

Classifying content...
  Topic:    ai-ml
  File:     rag-retrieval-augmented-generation.md
  Keywords: RAG, embeddings, vector store, semantic search, LLM
  Written:  ~/.secondmem/knowledge/ai-ml/rag-retrieval-augmented-generation.md
  Embedded 1 chunk(s)
  Graph updated

Ingestion complete!
```

Works with files and directories too:

```bash
secondmem ingest ./notes/system-design.md
secondmem ingest ./career-playbook/
```

### 2. See your knowledge tree

```bash
$ secondmem tree

/home/user/.secondmem/knowledge
├── ai-ml
│   ├── co-op-skills-roadmap-2026.md
│   ├── model-context-protocol-overview.md
│   ├── rag-retrieval-augmented-generation.md
│   └── transformers-self-attention-parallel.md
├── engineering
│   └── claude-code-session-notes.md
├── personal
│   └── career-playbook.md
└── productivity
    └── secondmem-cli-commands.md
```

### 3. Ask questions in natural language

```bash
$ secondmem ask "how does RAG work?"

RAG works by splitting your documents into chunks, embedding each chunk into a
vector using an embedding model, and storing those vectors. At query time, your
question is also embedded and compared against all stored vectors using cosine
similarity. The most semantically similar chunks are retrieved and sent to the
LLM as context, which then generates a grounded answer.

$ secondmem ask "what should I learn for co-op?" --cite

Based on your notes, the top skills for co-op roles are:
1. Python + ML basics — foundation for all AI work
2. LLM APIs + Prompt Engineering — most co-op AI roles are LLM integration
3. RAG pipelines — ~80% of enterprise AI projects use RAG
...

Sources:
  - ai-ml/co-op-skills-roadmap-2026.md
  - ai-ml/rag-retrieval-augmented-generation.md
```

Semantic search means it finds relevant notes even when your question uses different words than what you ingested. Ask about "speeding up training" and it finds your notes on "learning rate scheduling."

---

## Commands

| Command | Description |
|---------|-------------|
| `secondmem init [--path]` | Initialize knowledge base and config |
| `secondmem ingest <text\|file>` | Ingest content (text string or file path) |
| `secondmem ask "question" [--cite]` | Query your knowledge base |
| `secondmem tree [--depth N]` | Display knowledge structure as ASCII tree |
| `secondmem stats` | Topic count, file count, line count, graph stats |
| `secondmem validate` | Check hierarchy links and orphaned files |
| `secondmem rebalance [--dry-run]` | 7-step knowledge base maintenance |
| `secondmem graph stats` | LORE-GRAPH node/edge/FTS counts |
| `secondmem graph search <query>` | Full-text search the graph |
| `secondmem graph rebuild` | Rebuild graph by scanning all .md files |
| `secondmem graph validate` | Check graph nodes against filesystem |
| `secondmem config show` | Print current configuration |
| `secondmem config set <key> <value>` | Update a config value |

## Providers

| Provider | Config | LLM model | Embed model |
|----------|--------|-----------|------------|
| `ollama` | `ollama.url`, `ollama.model` | `llama3.2` (default) | `nomic-embed-text` |
| `openai` | `openai.api_key`, `openai.model` | `gpt-4o` | `text-embedding-3-small` |
| `copilot` | auto-detected from `~/.copilot/config.json` | `gpt-4o-mini` | `text-embedding-3-small` |

All three providers handle both chat completions and embeddings automatically — no separate embedding config needed.

```bash
secondmem config set model.provider ollama
secondmem config set ollama.model llama3.2

secondmem config set model.provider copilot   # no key needed if Copilot CLI is installed

secondmem config set model.provider openai
secondmem config set openai.api_key sk-...
secondmem config set openai.model gpt-4o-mini
```

## Using with AI CLI Tools

secondmem works as a persistent memory layer alongside any AI coding CLI.

### Claude Code
```bash
# Save knowledge from a session
secondmem ingest "React useEffect cleanup runs before next effect, not just on unmount"

# Pull context before a question
secondmem ask "React hooks gotchas"
```

Add to your project's `CLAUDE.md` to give Claude automatic access:
```markdown
## Knowledge base
Run `secondmem ask "<topic>"` to retrieve relevant notes before answering.
```

### GitHub Copilot CLI
```bash
# Enrich a copilot suggestion with your stored knowledge
context=$(secondmem ask "kubernetes rollback strategies")
gh copilot suggest "roll back a failed deployment -- background: $context"
```

Shell helper:
```bash
sm-suggest() {
  context=$(secondmem ask "$*" 2>/dev/null)
  gh copilot suggest "$* -- context: $context"
}
```

### OpenCode CLI
```bash
# Pipe secondmem context into an opencode session
secondmem ask "docker networking" | opencode --system-prompt -
```

### General pattern
```bash
# Ingest from clipboard, file, or stdin
secondmem ingest "$(pbpaste)"
secondmem ingest ./notes.md

# Before any AI session on a topic, pull your notes
secondmem ask "authentication patterns"
```

## Architecture

```
~/.secondmem/
├── config.toml           # Provider config, API keys
├── skill.md              # Agent constitution (guides AI behavior)
├── secondmem.db          # SQLite LORE-GRAPH (FTS5)
├── logs/
└── knowledge/
    ├── hierarchy.md       # Root table of contents
    ├── ai-ml/
    │   ├── hierarchy.md
    │   └── transformers-self-attention.md
    └── engineering/
        ├── hierarchy.md
        └── ...
```

**Ingestion pipeline:**
1. SHA256 exact dedup check
2. LLM classifies content → `{directory, filename, summary, keywords}`
3. LLM semantic dedup check (FTS candidates + similarity score)
4. Write/append to `.md` file with hash embedded
5. Update `hierarchy.md` for directory and root
6. Upsert graph node, create topic edges
7. **Chunk content → embed each chunk → store vectors in SQLite** (RAG index)
8. LLM identifies related files → bidirectional cross-reference links

**Query pipeline:**
1. Embed question → cosine similarity search across all chunk vectors
2. Top-6 semantically matching chunks used as context
3. Fallback: LLM rewrites question into FTS keywords → FTS5 prefix search → edge expansion
4. LLM synthesizes answer (with optional `--cite` source paths)
5. Last fallback: hierarchy scan if graph returns no results

**Rebalance (7 steps):**
1. Split files over 1,116 lines into themed sub-files
2. Detect orphaned files (not listed in hierarchy.md)
3. Validate and clean dead links in hierarchy files
4. Validate cross-reference link targets
5. Identify merge candidates (≥70% keyword overlap)
6. Remove stale graph nodes for deleted files
7. Rebuild root hierarchy.md

## Further Reading

The RAG implementation in secondmem is documented in the [career-playbook](https://github.com/Rinil-Parmar/career-playbook) knowledge base:

- [RAG deep dive](https://github.com/Rinil-Parmar/career-playbook/blob/main/ai/rag.md) — chunking, embeddings, vector stores, cosine similarity, evaluation
- [secondmem RAG extension plan](https://github.com/Rinil-Parmar/career-playbook/blob/main/ai/secondmem-rag.md) — full architecture, Go implementation details, build phases

## License

MIT
