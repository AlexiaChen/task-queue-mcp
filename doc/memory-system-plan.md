# 记忆系统实施计划

> Issue #35: 给 issue kanban mcp 开发一个记忆系统
> 参考: [设计文档](./memory-system-design.md)

---

## 实施策略

遵循 TDD（RED-GREEN-REFACTOR）+ 原子提交。每个阶段产出可测试、可验证的增量。

---

## Phase 1: 数据层（Models + Storage Interface + SQLite 实现）

### 1.1 定义数据模型
- **文件**: `internal/memory/models.go`
- **内容**: Memory 结构体、MemoryCategory 常量、StoreMemoryInput/SearchOptions/ListOptions DTO、MemorySearchResult、错误类型
- **验证**: 编译通过

### 1.2 定义 Storage 接口
- **文件**: `internal/memory/storage.go`
- **内容**: `memory.Storage` 接口（6 个方法）
- **验证**: 编译通过

### 1.3 实现 Mock Storage
- **文件**: `internal/memory/mock_storage.go`
- **内容**: `MockMemoryStorage` 实现 `memory.Storage`，内存 map 存储
- **验证**: 编译通过

### 1.4 SQLite Migration
- **文件**: `internal/storage/sqlite.go` — `runMigrations()` 函数
- **内容**: 添加 memories 表、memories_fts 虚拟表、索引、同步触发器
- **验证**: `make build` 通过，服务启动后检查表存在

### 1.5 SQLite Storage 实现
- **文件**: `internal/storage/sqlite.go` — 新增方法
- **内容**: `SQLiteStorage` 实现 `memory.Storage` 的 6 个方法
  - `StoreMemory`: 归一化 → 哈希 → INSERT OR IGNORE → 去重处理
  - `GetMemory`: 按 ID 查询
  - `SearchMemories`: FTS5 MATCH + BM25 + project_id 过滤
  - `ListMemories`: 分页查询，按 created_at DESC
  - `DeleteMemory`: 验证 project_id 归属后删除
  - `DeleteMemoriesByProject`: 按 project_id 批量删除
- **验证**: 单元测试（Phase 3）

### 1.6 DeleteProject 联动
- **文件**: `internal/storage/sqlite.go` — `DeleteProject()` 方法
- **内容**: 在删除 tasks 之后、删除 queues 之前，添加 `DELETE FROM memories WHERE project_id = ?`
- **验证**: 单元测试

---

## Phase 2: 业务逻辑层（MemoryManager）

### 2.1 实现 MemoryManager
- **文件**: `internal/memory/manager.go`
- **内容**:
  - `NewMemoryManager(storage Storage) *MemoryManager`
  - `Store()`: 验证输入 → 归一化 → 哈希 → 调用 storage → 去重语义
  - `Search()`: 验证 query 非空 → 调用 storage
  - `List()`: 设置默认 limit → 调用 storage
  - `Delete()`: 调用 storage（验证归属在 storage 层）
  - 输入验证: content 非空、content 长度上限（如 50KB）、category 合法、importance 范围
- **验证**: 单元测试

---

## Phase 3: 测试（TDD — RED 先于 GREEN）

### 3.1 Manager 单元测试
- **文件**: `internal/memory/manager_test.go`
- **测试用例**:
  - 存储记忆（正常 + 各参数组合）
  - 内容去重（相同内容返回已有记忆）
  - 输入验证（空 content、超长 content、非法 category、importance 越界）
  - 搜索（正常 + 空 query + category 过滤）
  - 列表（正常 + 分页 + category 过滤）
  - 删除（正常 + 不存在的记忆）
- **验证**: `go test ./internal/memory/...`

### 3.2 SQLite Storage 集成测试
- **文件**: `internal/storage/sqlite_memory_test.go`（如果需要独立文件）
- **测试用例**:
  - CRUD 完整流程
  - FTS5 搜索准确性
  - 去重（并发安全：UNIQUE 约束）
  - DeleteProject 级联删除 memories
  - BM25 排序正确性
- **验证**: `make test`

---

## Phase 4: MCP 工具集成

### 4.1 注册 MCP 工具
- **文件**: `internal/mcp/tools.go`
- **内容**: 在 `registerTools()` 中添加 4 个工具
  - `memory_search` + `memory_list`: readonly（始终注册）
  - `memory_store` + `memory_delete`: admin（`!s.readonly` 时注册）
- **验证**: MCP 客户端可见工具列表

### 4.2 实现 MCP Handler
- **文件**: `internal/mcp/tools.go`
- **内容**: 4 个 handler 函数
  - `handleMemoryStore`: 提取参数 → 调用 manager.Store() → 返回 JSON
  - `handleMemorySearch`: 提取参数 → 调用 manager.Search() → 返回 JSON
  - `handleMemoryList`: 提取参数 → 调用 manager.List() → 返回 JSON
  - `handleMemoryDelete`: 提取参数 → 调用 manager.Delete() → 返回 JSON
- **验证**: MCP 工具调用返回正确结果

### 4.3 MCP Server 初始化
- **文件**: `internal/mcp/server.go`
- **内容**: MCP Server 接收 `memory.MemoryManager` 依赖
- **验证**: 编译通过

---

## Phase 5: REST API 集成

### 5.1 注册 REST 路由
- **文件**: `internal/api/handlers.go`
- **内容**: 在 `RegisterRoutes()` 中添加 4 条路由
  ```
  POST   /api/projects/{id}/memories
  GET    /api/projects/{id}/memories
  GET    /api/projects/{id}/memories/search
  DELETE /api/projects/{id}/memories/{mid}
  ```
- **验证**: curl 测试

### 5.2 实现 REST Handler
- **文件**: `internal/api/handlers.go`
- **内容**: 4 个 handler 函数
  - `StoreMemory`: JSON body → manager.Store() → 201 Created
  - `ListMemories`: query params → manager.List() → 200 OK
  - `SearchMemories`: query params → manager.Search() → 200 OK
  - `DeleteMemory`: path params → manager.Delete() → 204 No Content
- **验证**: curl + e2e 测试

### 5.3 API Handler 初始化
- **文件**: `internal/api/handlers.go`
- **内容**: Handler 接收 `memory.MemoryManager` 依赖
- **验证**: 编译通过

---

## Phase 6: 服务入口整合

### 6.1 Server main.go 整合
- **文件**: `cmd/server/main.go`
- **内容**:
  - 创建 `memory.MemoryManager` (复用 SQLiteStorage)
  - 传入 MCP Server
  - 传入 API Handler
- **验证**: `make build && make run` 启动成功

---

## Phase 7: 端到端测试

### 7.1 E2E 测试
- **文件**: `test/e2e_test.go`（或新增 `test/e2e_memory_test.go`）
- **测试场景**:
  - 创建项目 → 存储记忆 → 搜索记忆 → 列出记忆 → 删除记忆
  - 去重验证
  - 项目隔离验证（不同项目的记忆互不可见）
  - 删除项目后记忆被清理
- **验证**: `make e2e`

---

## Phase 8: 收尾

### 8.1 文档更新
- 更新 `AGENTS.md`: 新增 Memory 相关架构、API、工具说明
- 更新 `README.md`: 新增记忆系统说明（如果有）

### 8.2 全量验证
- `make test` — 所有单元测试通过
- `make build-all` — 所有二进制编译通过
- `make e2e` — 端到端测试通过

---

## 依赖关系

```
Phase 1 (数据层)
    ↓
Phase 2 (业务逻辑) ← Phase 3 (测试，与 Phase 2 TDD 交替进行)
    ↓
Phase 4 (MCP) + Phase 5 (REST)  ← 可并行
    ↓
Phase 6 (整合)
    ↓
Phase 7 (E2E)
    ↓
Phase 8 (收尾)
```

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| FTS5 在 modernc.org/sqlite 中可能不可用 | 启动时验证 FTS5 支持，降级为 LIKE 搜索 |
| FTS5 trigger 与 external content 模式兼容性 | 先写集成测试验证，必要时改用普通 FTS5 表 |
| 内容哈希碰撞 | SHA-256 碰撞概率极低（2^-128），可忽略 |
| 大量记忆导致 FTS 索引膨胀 | 添加 content 长度上限 + 列表分页 |
