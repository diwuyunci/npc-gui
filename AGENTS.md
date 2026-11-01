# AGENTS.md

NPC GUI is a Wails v2 desktop client for NPS that manages npc reverse-tunnel processes.

## Documentation

Read [README.md](README.md) for the project entry and [docs/AGENTS.md](docs/AGENTS.md) for the documentation standard.

- The English `.md` file is the canonical documentation source.
- The `.zh.md` file is its Simplified Chinese counterpart.
- Read one side of a pair for implementation work; use the other side only for translation or language-specific changes.
- Read the owning document before changing a shared fact; [docs/AGENTS.md](docs/AGENTS.md) lists document ownership.
- Code and tests define current runtime behavior. Documentation describes that behavior.

## Commands

~~~bash
# Backend tests
go test ./internal/...
go vet ./internal/...

# Frontend checks
cd frontend
pnpm typecheck
pnpm test
pnpm build
cd ..

# Desktop build
wails build -clean
~~~

The root Go package embeds `frontend/dist`. Run `go test ./...` only after the frontend has been built.

## Constraints

- Do not commit real vkey values, credentials, or user-specific server addresses.
- Keep English and Chinese documentation pairs aligned when changing documentation.
- Keep runtime facts in code and tests, not in historical notes.
