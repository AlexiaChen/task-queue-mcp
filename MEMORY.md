# Project Memory

> Local structured memory for this project.
> Optimized for `rg` and `grep`: one `MEMORY_*` field per line, `---` between entries.
> Keep entries short. Prefer append-only; if a memory is replaced, mark the old one `superseded`.

## Search Tips

- `rg -n "MEMORY_TAGS:.*sqlite|MEMORY_TRIGGER:.*sqlite" MEMORY.md`
- `rg -n -C 2 "^MEMORY_ID: M-0001|^MEMORY_SUBJECT: workflow|^MEMORY_OBJECT: MEMORY.md" MEMORY.md`
- `rg -n "MEMORY_STATUS: active" MEMORY.md`

## Entry Template

```markdown
### M-0001: <short title>
MEMORY_ID: M-0001
MEMORY_STATUS: active
MEMORY_SCOPE: project
MEMORY_KIND: decision
MEMORY_DATE: YYYY-MM-DD
MEMORY_CONFIDENCE: 8/10
MEMORY_TAGS: tag1 tag2 tag3
MEMORY_TRIGGER: keyword1 | keyword2 | keyword3
MEMORY_TITLE: <short title>
MEMORY_BODY: <1-2 sentence memory>
MEMORY_ACTION: <what to do when this matches>
MEMORY_EVIDENCE: <path:line, command, or concrete example>
MEMORY_RELATES_TO: <module, feature, work item, or area>
MEMORY_SUPERSEDES: none
MEMORY_SUPERSEDED_BY: none
```

For relation entries, add:

```markdown
MEMORY_SUBJECT: <entity>
MEMORY_PREDICATE: <relationship>
MEMORY_OBJECT: <entity-or-state>
```

---

## Entries

### M-0001: Local memory file replaces external task-board memory dependency
MEMORY_ID: M-0001
MEMORY_STATUS: active
MEMORY_SCOPE: project
MEMORY_KIND: decision
MEMORY_DATE: 2026-05-20
MEMORY_CONFIDENCE: 10/10
MEMORY_TAGS: workflow instructions memory local-file
MEMORY_TRIGGER: copilot instructions | local memory | MEMORY.md | grep
MEMORY_TITLE: Local memory file replaces external task-board memory dependency
MEMORY_BODY: This repository uses MEMORY.md as the local persistent memory surface for reusable workflow context and project knowledge.
MEMORY_ACTION: Search MEMORY.md during pre-flight and append new structured entries during the compound step instead of depending on external task-board memory tools.
MEMORY_EVIDENCE: instructions/copilot-instructions.md
MEMORY_RELATES_TO: workflow
MEMORY_SUPERSEDES: none
MEMORY_SUPERSEDED_BY: none

---

### M-0002: workflow stores_context_in MEMORY.md
MEMORY_ID: M-0002
MEMORY_STATUS: active
MEMORY_SCOPE: project
MEMORY_KIND: relation
MEMORY_DATE: 2026-05-20
MEMORY_CONFIDENCE: 10/10
MEMORY_TAGS: workflow relation memory
MEMORY_TRIGGER: workflow | relation | MEMORY.md | context
MEMORY_TITLE: workflow stores_context_in MEMORY.md
MEMORY_BODY: The local kanban workflow persists reusable context in MEMORY.md so future work can recover both prose memories and lightweight relations with grep-friendly searches.
MEMORY_ACTION: When a task produces durable project knowledge, append either a standard memory entry or a relation entry to MEMORY.md.
MEMORY_EVIDENCE: instructions/copilot-instructions.md
MEMORY_RELATES_TO: workflow
MEMORY_SUPERSEDES: none
MEMORY_SUPERSEDED_BY: none
MEMORY_SUBJECT: workflow
MEMORY_PREDICATE: stores_context_in
MEMORY_OBJECT: MEMORY.md