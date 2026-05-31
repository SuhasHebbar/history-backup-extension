## Repository layout

This monorepo contains two independent subprojects:

- [extension/](extension/) — Chrome Manifest V3 extension (plain JS, no build step)
- [server/](server/) — Go HTTP server that receives history uploads and stores them in SQLite

Each subproject has its own `AGENTS.md` with detailed guidance for working within it.
