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

### L-003: [gotcha] FTS5 rank column is implicit BM25 — no DDL needed (2026-04-15)
- **Issue**: #36 — 重新检查记忆模块
- **Trigger**: fts5, bm25, rank, search, ranking
- **Pattern**: FTS5's `rank` column IS `bm25()` by default — no DDL/function definition needed. Just SELECT rank FROM fts_table. Negative values, lower = more relevant.
- **Evidence**: internal/storage/sqlite.go:524 (SELECT rank), SQLite FTS5 docs: "built-in rank column is equivalent to calling bm25()"
- **Confidence**: 10/10
- **Action**: When implementing FTS5 search, use `rank` directly in ORDER BY. Document this for team clarity since it's non-obvious.

### L-004: [gotcha] Float64 equality in FTS5 test assertions needs epsilon (2026-04-15)
- **Issue**: #36 — 重新检查记忆模块
- **Trigger**: fts5, bm25, float, test, epsilon, rank
- **Pattern**: BM25 scores for similar-length documents can differ at 1e-15 level even when logically identical. Direct `==` comparison misses tiebreaker validation.
- **Evidence**: internal/storage/sqlite_memory_test.go:264 (epsilon comparison), initial test used `==` and silently skipped the check
- **Confidence**: 9/10
- **Action**: Always use epsilon-based comparison (e.g., `math.Abs(a-b) < 1e-6`) when comparing FTS5 rank values in tests.

### L-005: [convention] Update all workflow touchpoints when integrating a new subsystem (2026-04-15)
- **Issue**: #37 — 更新项目
- **Trigger**: workflow, integration, documentation, update, subsystem, loop
- **Pattern**: When integrating a new subsystem into workflow documentation, there are 5+ touchpoints to update (loop diagram, step descriptions, tool reference, compliance checklist, final report template). Missing any one causes confusion.
- **Evidence**: instructions/copilot-instructions.md — had to update opening quote, loop diagram, Step 3a, Step 5c, MCP Tools Reference, Appendix E checklist, and Final Report template
- **Confidence**: 8/10
- **Action**: Create a checklist of all workflow doc touchpoints before starting doc integration work

### L-006: [gotcha] Memory importance CHECK constraint in tests (2026-04-15)
- **Issue**: #38 — 开发时序知识图谱
- **Trigger**: memory, store, test, importance, CHECK
- **Pattern**: When creating memories as test fixtures, the `Importance` field has a CHECK constraint (1-5). Omitting it defaults to 0, which violates the constraint and causes cryptic "CHECK constraint failed" errors.
- **Evidence**: internal/storage/sqlite_triple_test.go — StoreWithSourceMemory test initially failed with CHECK constraint violation
- **Confidence**: 8/10
- **Action**: Always set `Importance: 3` (or valid 1-5 value) when creating memory fixtures in tests
