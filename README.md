# assetagent

Local-first personal wealth platform. Milestone 1 delivers a reliable transaction ingestion pipeline: import Sparkasse CSV exports, persist transactions in PostgreSQL, and query them programmatically.

## Prerequisites

- Go 1.23+
- Docker and Docker Compose (for local PostgreSQL; coming in a later commit)

## Quick start

```bash
make build
./bin/assetagent --version
```

## Status

Early development. See `tmp/ROADMAP.md` for the full phase plan.
