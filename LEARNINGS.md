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

---

### L-007: [convention] readonly/admin tool categorization sync (2025-07-15)
- **Issue**: #39 — 更新MD文件
- **Trigger**: readonly, admin, tool, mcp, mode
- **Pattern**: When moving MCP tools between readonly and admin, at least 5 files need updating: tools.go (code), AGENTS.md, README.md, copilot-instructions.md, and design doc. Missing any one creates inconsistency.
- **Evidence**: Issue #39 required updating all 5 files simultaneously
- **Confidence**: 9/10
- **Action**: When changing a tool's readonly/admin status, check ALL doc files referencing tool categorization

---

### L-008: [gotcha] instructions 文件 memory importance 量级须与服务器约束对齐 (2026-04-24)
- **Issue**: #58 — 修正下instructions.md文件
- **Trigger**: instructions, memory_store, importance, scale, 1-5
- **Pattern**: copilot-instructions.md 中 importance 量级示例若超出服务器 CHECK 约束（1-5），AI 每次调用 memory_store 都会出错并重试，浪费上下文。原写 8-10/5-7/1-4（1-10 scale），而服务器实际限制 1-5。
- **Evidence**: instructions/copilot-instructions.md:623 — 修正为 5=architectural, 3-4=useful, 1-2=minor
- **Confidence**: 9/10
- **Action**: 修改指令文件中 memory_store 相关示例/说明时，始终确认 importance 值在服务器约束范围 1-5 内
