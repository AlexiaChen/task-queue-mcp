# 记忆系统设计文档

> Issue #35: 给 issue kanban mcp 开发一个记忆系统

---

## 1. 背景与目标

AI Agent 在长时间对话中，上下文窗口压缩会导致记忆失真。当前 LEARNINGS.md 只是一种
自我学习机制（知识点和边界），不是独立的记忆系统。

**目标**: 为 issue-kanban-mcp 集成一个 **按项目隔离** 的记忆系统，暴露 MCP 接口，
让 AI Agent 能够存储和检索与特定项目相关的记忆。

---

## 2. 参考系统研究总结

### 2.1 MemPalace（Python）— 96.6% LongMemEval

**核心哲学**: *"Store everything verbatim, then make it findable."*

- **五层宫殿结构**: Wing(项目) → Hall(类型) → Room(概念) → Closet(AAAK索引) → Drawer(原文)
- **混合搜索**: 向量相似度(0.6) + BM25(0.4)，96.6% Recall@5
- **时序知识图谱**: subject-predicate-object 三元组，带 valid_from/valid_to 时间窗口
- **AAAK 压缩**: 有损缩写系统（实体→3字母码，情感→30种编码），用于快速扫描
- **重要性排名**: 1-5 星级
- **自描述协议**: MEMORY_PROTOCOL 嵌入 MCP 响应，Agent 自学工作流

### 2.2 Mempal（Rust）— MemPalace 的改良实现

- **单文件 SQLite**: 用 sqlite-vec 替代 ChromaDB，零外部依赖
- **model2vec-rs**: 本地嵌入，不需外部 API
- **7 个精简 MCP 工具**: 相比 Python 版 19 个，更聚焦
- **Route Detection**: 自动推断 wing/room，无需用户指定

### 2.3 GBrain（TypeScript）— Postgres-native 个人知识库

- **混合 RAG**: 向量搜索 + 关键词搜索 + RRF (Reciprocal Rank Fusion) 融合
- **SHA-256 内容去重**: 内容哈希实现幂等性
- **Contract-first**: 单一操作定义文件，CLI/MCP/REST 自动生成
- **FTS5 + BM25**: 全文搜索，带查询意图分类（零成本模式匹配）
- **编译真理保证**: 去重后确保每个 page 至少有一条核心内容

---

## 3. YAGNI 取舍分析

### 3.1 ❌ 向量搜索 — 不做

**原因**:
- 项目约束 `CGO_ENABLED=0`，纯 Go 无法使用 sqlite-vec 或 faiss
- 嵌入需要外部 API（OpenAI/本地模型），引入外部依赖
- GBrain 数据显示：即使嵌入失败，关键词搜索仍可工作（embedding 是 non-fatal 的）
- MemPalace 96.6% 的成绩中，BM25 贡献了 40% 的权重

**替代**: FTS5 + BM25 是成熟的全文搜索方案，对于结构化的项目记忆检索足够有效。

### 3.2 ❌ 时序知识图谱 — v1 不做，为 v2 预留

**什么是时序 KG**:
MemPalace 的时序知识图谱存储 `(subject, predicate, object, valid_from, valid_to)` 三元组，
例如 `("Max", "works_at", "Google", "2020-01", "2024-06")`，支持按时间点查询事实状态。

**为什么 v1 不做**:

| 考量 | 分析 |
|------|------|
| **前置依赖** | KG 需要**实体抽取**（从文本中识别主体/客体），这通常需要 LLM 调用或 NLP 库。纯 Go + 无外部 API 的约束下，实体抽取质量难以保证 |
| **复杂度** | 需要实体归一化、谓词标准化、矛盾检测、图遍历查询。这本身就是一个独立子系统，代码量不低于基础记忆系统 |
| **使用场景** | Agent 处理 issue 时的核心需求是「记住做过什么、决策原因、踩过什么坑」。这些通过 `category=decision/fact/event` + `created_at` 时间排序已经满足 |
| **渐进式设计** | 基础记忆系统 → 验证 Agent 实际使用模式 → 按需加 KG。反过来则过度设计 |

**v2 预留路径**:
```sql
-- 未来可添加
CREATE TABLE memory_triples (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    subject TEXT NOT NULL,
    predicate TEXT NOT NULL,
    object TEXT NOT NULL,
    valid_from DATETIME,
    valid_to DATETIME,
    confidence REAL DEFAULT 1.0,
    source_memory_id INTEGER,  -- 回溯到原始记忆
    FOREIGN KEY (project_id) REFERENCES queues(id),
    FOREIGN KEY (source_memory_id) REFERENCES memories(id)
);
```

当前 `memories` 表已经有 `created_at`（时间线排序）和 `category`（类型过滤），
提供了基础的时序感知能力，只是没有结构化的实体关系图。

### 3.3 ❌ AAAK 压缩层 — 不做

**什么是 AAAK**:
MemPalace 的有损缩写系统，把原始文本压缩为类似：
```
ZID_001: KAI,CLK | pricing,dx | "Clerk > Auth0" | ⭐⭐⭐ | joy,trust | DECISION,TECHNICAL
```
其中实体用 3 字母码、情感用 30 种编码、内容用单行摘要。

**为什么不做**:

| 考量 | 分析 |
|------|------|
| **精度损失** | MemPalace 基准测试：Raw 模式 96.6% vs AAAK 模式 84.2%，**下降 12.4 个百分点**。这不是小数。AAAK 以精度换取 token 效率 |
| **设计目的不同** | AAAK 的设计目标是让 LLM 能快速扫描**数千条**记忆的索引。在我们的场景中，FTS5 搜索返回排序后的 top-N 结果，Agent 只读最相关的几条，不需要扫描全部 |
| **实现成本** | 需要定义实体编码表、情感分类法、标记系统。这些是 MemPalace 特有的设计，与 issue 管理场景契合度低 |
| **summary 字段替代** | 我们的设计中 `summary` 字段提供了轻量级的摘要功能：Agent 可以先读 summary，需要详情再读 content。功能类似但零额外复杂度 |

**总结**: AAAK 是 MemPalace 针对「大规模记忆 + token 预算有限」场景的优化。
我们的场景是「按项目隔离 + FTS5 精确搜索」，token 压力通过搜索排序天然解决。

### 3.4 ❌ 文本分块管道 — 不做

Agent 存储的记忆通常是结构化的段落（决策、事实、事件），不是长篇文档。
分块是 gbrain/mempalace 处理大文件导入时的需求，不适用于 MCP 工具的逐条存储场景。
如果 Agent 需要存储长文本，它自己在调用 `memory_store` 前分块即可。

---

## 4. 精髓提炼 — 我们借鉴了什么

| 精髓 | 来源 | 如何融入 |
|------|------|---------|
| 原文存储，不做有损压缩 | mempalace | `content` 字段存储完整原文 |
| 结构化导航 | mempalace 五层 | 简化为三层：project→category→tags |
| 重要性排名 | mempalace | `importance` 1-5，影响搜索排序 |
| 内容去重 | gbrain | SHA-256 哈希 + `UNIQUE(project_id, content_hash)` |
| 全文搜索 + BM25 | gbrain | SQLite FTS5，按 BM25 排名 + importance + recency |
| 项目隔离 | mempalace wings | `project_id` 外键，所有操作强制 project scope |
| 单文件数据库 | mempal | 复用现有 SQLite，零额外基础设施 |
| 简洁 MCP 工具 | mempal 7 tools | 4 个工具：store/search/list/delete |
| 自描述协议 | mempalace | memory_store 返回说明中嵌入使用建议 |

---

## 5. 架构设计

### 5.1 整体架构

```
                    ┌─────────────────────┐
                    │   AI Agent (MCP)    │
                    └──────┬──────────────┘
                           │
            ┌──────────────┼──────────────────┐
            │              │                  │
            ▼              ▼                  ▼
    ┌───────────┐  ┌──────────────┐  ┌──────────────┐
    │ MCP Tools │  │  REST API    │  │ MCP Resources│
    │ (4 tools) │  │ (4 endpoints)│  │ (optional)   │
    └─────┬─────┘  └──────┬───────┘  └──────┬───────┘
          │               │                 │
          └───────────────┼─────────────────┘
                          │
                          ▼
                ┌──────────────────┐
                │ memory.Manager   │ ← 业务逻辑（验证、去重、搜索排序）
                └────────┬─────────┘
                         │
                         ▼
                ┌──────────────────┐
                │ memory.Storage   │ ← 接口
                └────────┬─────────┘
                         │
                         ▼
                ┌──────────────────┐
                │ SQLiteStorage    │ ← 同时实现 queue.Storage + memory.Storage
                │ (memories 表)    │
                │ (memories_fts)   │
                └──────────────────┘
```

### 5.2 新增文件结构

```
internal/memory/
├── models.go         # Memory 结构体、MemoryCategory、SearchResult
├── storage.go        # memory.Storage 接口定义
├── manager.go        # MemoryManager 业务逻辑
└── mock_storage.go   # 测试用 MockMemoryStorage
```

### 5.3 数据模型

```go
type Memory struct {
    ID          int64
    ProjectID   int64
    Content     string
    Summary     string
    Category    MemoryCategory  // decision|fact|event|preference|advice|general
    Tags        string          // 逗号分隔
    Source      string          // 来源上下文
    Importance  int             // 1-5
    ContentHash string          // SHA-256
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type MemoryCategory string
const (
    CategoryDecision   MemoryCategory = "decision"
    CategoryFact       MemoryCategory = "fact"
    CategoryEvent      MemoryCategory = "event"
    CategoryPreference MemoryCategory = "preference"
    CategoryAdvice     MemoryCategory = "advice"
    CategoryGeneral    MemoryCategory = "general"
)

type MemorySearchResult struct {
    Memory
    Rank float64  // BM25 排名分数
}
```

### 5.4 数据库 Schema

```sql
CREATE TABLE IF NOT EXISTS memories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    summary TEXT DEFAULT '',
    category TEXT NOT NULL DEFAULT 'general'
        CHECK(category IN ('decision','fact','event','preference','advice','general')),
    tags TEXT DEFAULT '',
    source TEXT DEFAULT '',
    importance INTEGER DEFAULT 1 CHECK(importance BETWEEN 1 AND 5),
    content_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (project_id) REFERENCES queues(id)
);

-- DB层去重（防并发重复）
CREATE UNIQUE INDEX IF NOT EXISTS idx_memories_dedup
    ON memories(project_id, content_hash);

CREATE INDEX IF NOT EXISTS idx_memories_project
    ON memories(project_id);
CREATE INDEX IF NOT EXISTS idx_memories_category
    ON memories(project_id, category);

-- FTS5 全文搜索（external content 模式）
CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    content,
    summary,
    tags,
    content='memories',
    content_rowid='id',
    tokenize='unicode61'
);

-- 同步触发器
CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(rowid, content, summary, tags)
    VALUES (new.id, new.content, new.summary, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, content, summary, tags)
    VALUES ('delete', old.id, old.content, old.summary, old.tags);
END;

CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, content, summary, tags)
    VALUES ('delete', old.id, old.content, old.summary, old.tags);
    INSERT INTO memories_fts(rowid, content, summary, tags)
    VALUES (new.id, new.content, new.summary, new.tags);
END;
```

### 5.5 memory.Storage 接口

```go
type Storage interface {
    StoreMemory(ctx context.Context, input StoreMemoryInput) (*Memory, error)
    GetMemory(ctx context.Context, id int64) (*Memory, error)
    SearchMemories(ctx context.Context, projectID int64, query string, opts SearchOptions) ([]MemorySearchResult, error)
    ListMemories(ctx context.Context, projectID int64, opts ListOptions) ([]*Memory, error)
    DeleteMemory(ctx context.Context, projectID int64, memoryID int64) error
    DeleteMemoriesByProject(ctx context.Context, projectID int64) error
}
```

### 5.6 去重逻辑

```go
func normalizeContent(content string) string {
    // 1. Trim whitespace
    // 2. Normalize line endings (\r\n → \n)
    // 3. Collapse multiple blank lines → single blank line
    return normalized
}

func contentHash(projectID int64, content string) string {
    normalized := normalizeContent(content)
    h := sha256.Sum256([]byte(normalized))
    return hex.EncodeToString(h[:])
}

// StoreMemory: INSERT OR IGNORE + SELECT existing on conflict
// 重复存储返回已有记忆，不报错
```

### 5.7 搜索排序

```sql
SELECT m.*, bm25(memories_fts) as rank
FROM memories m
JOIN memories_fts ON m.id = memories_fts.rowid
WHERE memories_fts MATCH ?
  AND m.project_id = ?
ORDER BY
    bm25(memories_fts),        -- BM25 相关性（负数，越小越好）
    m.importance DESC,          -- 重要性降序
    m.updated_at DESC           -- 最近更新优先
LIMIT ?
```

### 5.8 MCP 工具

| 工具 | 权限 | 参数 | 说明 |
|------|------|------|------|
| `memory_store` | admin | project_id*, content*, category?, tags?, source?, importance?, summary? | 存储记忆，重复返回已有 |
| `memory_search` | readonly | project_id*, query*, category?, limit? | FTS5+BM25 搜索 |
| `memory_list` | readonly | project_id*, category?, limit?, offset? | 按时间倒序列出 |
| `memory_delete` | admin | project_id*, memory_id* | 验证归属后删除 |

### 5.9 REST API

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/projects/{id}/memories` | 存储记忆 |
| GET | `/api/projects/{id}/memories` | 列出记忆 |
| GET | `/api/projects/{id}/memories/search?q=` | 搜索记忆 |
| DELETE | `/api/projects/{id}/memories/{mid}` | 删除记忆 |

### 5.10 DeleteProject 联动

在 `SQLiteStorage.DeleteProject()` 中，删除项目时需同步删除该项目的所有记忆
（触发器会自动清理 FTS 索引）。

---

## 6. v2 路线图（未来扩展方向）

| 版本 | 特性 | 前置条件 |
|------|------|---------|
| v2.0 | 时序知识图谱（memory_triples 表） | 验证 v1 使用模式，确认 KG 需求 |
| v2.1 | 向量搜索（如 Go 生态出现纯 Go vector lib） | CGO_ENABLED=0 兼容的向量库 |
| v2.2 | 查询意图分类（借鉴 gbrain 的零成本模式匹配） | 搜索使用数据积累 |
| v2.3 | 跨项目记忆（全局搜索 + 项目过滤） | 多项目使用场景验证 |
