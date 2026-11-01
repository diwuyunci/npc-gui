# Documentation standard

This repository follows one documentation model for the root README and the docs directory.

## Language pairs

- Each user-facing document has an English source and a Simplified Chinese counterpart in the same directory: `README.md` / `README.zh.md`, `docs/business.md` / `docs/business.zh.md`.
- The English source links to its counterpart immediately after the H1 with `English | [中文](foo.zh.md)`; the Chinese counterpart links back with `[English](foo.md) | 中文`.
- Both sides change in the same edit. The pair keeps the same heading order, list and table structure, and code blocks.
- Relative documentation links in the English side target `.md` files; links in the Chinese side target `.zh.md` files.
- The project has no automated translation-pairing gate; review confirms that both sides state the same facts.

## Agent reading

- Read `README.md` and this file before changing documentation.
- Read the English `.md` source for implementation work; read `.zh.md` only for Chinese wording or translation work.
- Use the owning document listed below for a fact; do not reconstruct a fact from both language sides.
- Treat code and tests as the authority for runtime behavior.

## Document ownership

| Document | Owns |
|---|---|
| `README.md` | Project entry, business overview, build and release entry points |
| `docs/business.md` | Users, scenarios, capabilities, workflows, scope, and value |
| `docs/technical.md` | Stack, architecture, startup, state, data security, system integration, updates, build, release, and cleanup |
| `docs/manual-test.md` | Release acceptance checks |
| `docs/AGENTS.md` | This documentation standard; English only, excluded from pairing |

Each fact has one home. Other documents summarize it briefly and link to the owning document.

## Writing rules

- State current behavior. Do not add development-stage, migration, or implementation-status notes outside their owning document.
- Separate reference material from step-by-step procedures. Use numbered steps only in procedures.
- Use declarative sentences and name actors, files, commands, and observable results directly.
- Keep one physical line per paragraph; use editor soft wrapping. Code blocks, tables, and lists keep their formatting.
- Use relative Markdown links for repository files and verify that every local target exists.
