# secondmem

AI-powered local knowledge management CLI — your second memory.

## What is secondmem?

secondmem is an automated local librarian for your personal knowledge. Pass it anything — a tweet, article, PDF excerpt, or a note from a coding session — and the AI classifies it, summarizes it, and writes it into the right `.md` file in the right folder. Everything stays on your machine as plain, readable markdown.

Built in Go. No cloud. No vector databases. Just files.

## Features

- **Smart Ingestion** — AI classifies content into topic-based markdown files automatically
- **LORE-GRAPH** — SQLite + FTS5 knowledge graph for fast full-text search
- **Natural Language Queries** — Ask questions, get synthesized answers with citations
- **Bidirectional Cross-References** — Related files link to each other automatically
- **Deduplication** — SHA256 exact match + LLM semantic similarity (>70% threshold)
- **Auto-Split / Merge** — Files over 1,116 lines are split by theme; duplicates can be merged
- **7-Step Rebalance** — Maintains hierarchy, dead links, orphans, graph sync, and merge candidates
- **Three LLM Providers** — Ollama (local, default), OpenAI, GitHub Copilot
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

# Query your knowledge base
secondmem ask "What do I know about transformers?"
secondmem ask "how does deduplication work" --cite

# Browse and maintain
secondmem tree
secondmem stats
secondmem validate
secondmem rebalance
```

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

| Provider | Config | Notes |
|----------|--------|-------|
| `ollama` | `ollama.url`, `ollama.model` | Default. Requires `ollama serve` locally |
| `openai` | `openai.api_key`, `openai.model` | GPT-4o default |
| `copilot` | auto-detected | Reads token from `~/.copilot/config.json` |

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
7. LLM identifies related files → bidirectional cross-reference links

**Rebalance (7 steps):**
1. Split files over 1,116 lines into themed sub-files
2. Detect orphaned files (not listed in hierarchy.md)
3. Validate and clean dead links in hierarchy files
4. Validate cross-reference link targets
5. Identify merge candidates (≥70% keyword overlap)
6. Remove stale graph nodes for deleted files
7. Rebuild root hierarchy.md

## License

MIT
