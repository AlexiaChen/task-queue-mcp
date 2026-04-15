# Project Learnings

> Append-only knowledge base maintained during issue processing.
> The agent reads this before starting each issue to avoid repeating mistakes.
> Human edits welcome — add, annotate, or mark as [OBSOLETE].

---

### L-001: [gotcha] E2E TestMain os.Exit(0) blocks all tests (2026-04-15)
- **Issue**: #35 — 给issue kanban mcp开发一个记忆系统
- **Trigger**: e2e, integration, TestMain, test skip
- **Pattern**: TestMain with os.Exit(0) when env var is unset kills ALL tests in that package — including standalone integration tests that don't need a server
- **Evidence**: test/e2e/e2e_test.go:28-30, had to move TestIntegration_MemoryWorkflow to internal/memory/
- **Confidence**: 9/10
- **Action**: Put standalone integration tests in their own package (e.g. internal/X/integration_test.go), not in test/e2e/

### L-002: [convention] FTS5 external content triggers must match column order (2026-04-15)
- **Issue**: #35 — 给issue kanban mcp开发一个记忆系统
- **Trigger**: FTS5, full-text search, sqlite, trigger
- **Pattern**: FTS5 external content mode requires INSERT/DELETE triggers that exactly mirror the FTS virtual table columns in order. Mismatch causes silent data corruption.
- **Evidence**: internal/storage/sqlite.go migrations (memories_fts triggers)
- **Confidence**: 8/10
- **Action**: When adding FTS5 tables, verify trigger INSERT column list matches CREATE VIRTUAL TABLE column list exactly
