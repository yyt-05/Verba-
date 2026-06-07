# Verba 项目开发规范

> AI 实时双语字幕助手 | Windows 桌面端 | Flutter + Go
>
> 本文档是项目的全局开发规范。每一位开发者（包括 AI 助手）在开始任何代码改动前，
> 必须先阅读并遵守本规范。

---

## 一、开发铁律（违反即退回）

### 1. 测试先行

```
功能开发 = 先写测试 → 跑测试(红) → 写实现 → 跑测试(绿) → 重构 → 跑测试(绿)
Bug 修复 = 先写复现测试 → 跑测试(红) → 修复 → 跑测试(绿)
```

- **新功能必须在同个 PR/commit 中附带测试**
- 测试文件放在对应源码目录：Go 用 `_test.go`，Flutter 用 `test/` 目录
- 最小覆盖要求：每个公开函数的正常路径 + 关键边界 + 错误路径
- 运行命令：
  ```bash
  # 后端
  cd server && go test ./internal/... -v -count=1

  # 前端
  cd client && flutter test
  ```
- **测试不通过 = 功能没做完。不允许合并。**

### 2. 后端必须打点日志

- 每个 HTTP handler 入口打一条 `[handler]` 日志
- 每个外部 API 调用打请求/响应日志（含耗时）
- 每个错误路径打 `[ERROR]` 级别日志
- 每个 session 生命周期事件打日志（created / started / stopped / timeout）
- 日志格式统一：
  ```
  [模块] 动作 key=value key2=value2
  ```
  示例：
  ```
  [session] created id=sess_123
  [asr] transcribed bytes=64000 duration=782ms
  [pipeline] error stage=translate session=sess_123 err="timeout"
  ```

### 3. 代码必须可复用

- **禁止重复代码**：同一逻辑出现两次 → 提取函数
- **模块边界清晰**：`internal/` 下每个包只做一件事
- **接口隔离**：对外暴露接口而非具体实现
- **配置外置**：API Key、端口、阈值全部走环境变量/config struct，禁止硬编码

### 4. 每次改动范围最小化

- 一个 PR/commit 只做一件事
- 不改和当前任务无关的代码
- 不顺手重构（除非该重构阻塞当前任务）
- 不在同一个 commit 中混入格式调整和逻辑修改

---

## 二、项目架构约束

### 目录结构（不得随意变更）

```
Verba/
├── AGENTS.md                    # Agent 工作规则和变更边界
├── ARCHITECTURE.md              # 项目整体架构设计
├── docs/                        # 项目文档目录
│   ├── DOCUMENTATION_ARCHITECTURE.md  # 文档目录架构说明
│   ├── DESIGN.md                # 方案设计入口和基本原则
│   ├── FRONTEND.md              # 前端形态设计
│   ├── PLANS.md                 # 总体计划及计划入口
│   ├── PRODUCT_SENSE.md         # 需求文档入口及基本原则
│   ├── QUALITY_SCORE.md         # 质量文档入口及基本原则
│   ├── RELIABILITY.md           # 可靠性文档入口及基本原则
│   ├── SECURITY.md              # 安全性文档入口及基本原则
│   ├── design-docs/             # 设计文档：解决“怎么做”
│   │   ├── index.md             # 方案设计目标和目录
│   │   ├── core-beliefs.md      # 核心设计信念
│   │   └── ...
│   ├── exec-plans/              # 任务计划
│   │   ├── active/              # 活动中的任务
│   │   ├── completed/           # 已完成任务
│   │   └── tech-debt-tracker.md # 技术债跟踪
│   ├── generated/               # 技术生成文档
│   ├── product-specs/           # 需求文档：解决“做什么”
│   │   ├── index.md             # 需求目录
│   │   └── ...
│   └── references/              # 外部参考资料和 LLM 友好材料
├── server/                      # Go 后端
│   ├── cmd/verba/main.go        # 入口 + 路由
│   └── internal/
│       ├── config/              # 配置加载
│       ├── session/             # 会话管理
│       ├── audio/               # 音频chunk处理
│       ├── sse/                 # SSE推送层
│       └── pipeline/            # 业务流水线(ASR/翻译/修正)
├── client/                      # Flutter 客户端
│   ├── lib/
│   │   ├── main.dart            # 入口 + window_manager配置
│   │   ├── models/              # 数据模型
│   │   ├── services/            # API/SSE/WASAPI客户端
│   │   ├── providers/           # Riverpod 状态管理
│   │   ├── pages/               # 页面
│   │   └── widgets/             # 可复用组件
│   ├── windows/wasapi/          # WASAPI C++ DLL
│   └── test/                    # 前端测试
```

### 文档架构规则（AI 必读）

AI 在开始涉及需求、设计、计划、架构、质量、安全、可靠性或较大功能改动的任务前，必须先阅读：

1. `AGENTS.md`：确认工作规则和变更边界。
2. `ARCHITECTURE.md`：确认项目整体架构和模块职责。
3. `docs/DOCUMENTATION_ARCHITECTURE.md`：确认文档应该放在哪里、怎么命名、怎么补充。

文档维护规则：

- 需求变化写入 `docs/product-specs/`，并更新 `docs/product-specs/index.md` 或 `docs/PRODUCT_SENSE.md`。
- 方案、接口、流程、UI、架构取舍写入 `docs/design-docs/`，并更新 `docs/design-docs/index.md` 或 `docs/DESIGN.md`。
- 当前任务计划写入 `docs/exec-plans/active/`；完成后移动到 `docs/exec-plans/completed/`，并补完成结果摘要。
- 技术债写入 `docs/exec-plans/tech-debt-tracker.md`，不要混进当前任务计划。
- 自动生成或机械导出的技术文件写入 `docs/generated/`。
- 外部资料、LLM 参考材料、框架/服务商文档快照写入 `docs/references/`。
- 如果本轮改动改变了架构、需求边界、接口行为、测试覆盖、可靠性策略或安全边界，AI 应主动补充对应文档；如果只做很小的代码修复且不改变设计结论，可以在最终回复中说明无需更新文档。
- 新文档文件名使用英文小写短横线，例如 `correction-pipeline-design.md`，避免中文路径在终端里乱码。

### 技术栈锁定（不引入新依赖需评审）

| 层 | 技术 | 备注 |
|------|------|------|
| 前端框架 | Flutter 3.44+ | Windows 桌面端为主 |
| 状态管理 | Riverpod | 已选型，不换 |
| 后端语言 | Go 1.25+ | 标准库为主，不引入框架 |
| ASR | OpenAI Whisper API | Phase 1 接入 |
| 翻译/修正 | OpenAI GPT-4o-mini | Phase 1 接入 |
| 实时推送 | SSE (Server-Sent Events) | 不用 WebSocket |
| 音频捕获 | WASAPI loopback (C++ DLL) | 已实现 polling 模式 |

---

## 三、开发流程（每步操作顺序）

### 新增功能的标准流程

```
Step 1: 确认需求范围和验收标准
        → 对照 PRD 和执行路线图，明确「做到什么程度算完成」

Step 2: 先写测试
        → Go:  创建 xxx_test.go
        → Flutter: 创建 test/xxx_test.dart
        → 测试必须覆盖：正常路径 + 边界条件 + 错误路径

Step 3: 跑测试确认失败
        → go test ./internal/...  (应该是 FAIL)
        → flutter test            (应该是 FAIL)

Step 4: 写实现代码
        → 只写让测试通过的最小代码量
        → 后端加入打点日志
        → 保持可复用性

Step 5: 跑测试确认通过
        → go test ./internal/...  (全部 PASS)
        → flutter test            (全部 PASS)
        → dart analyze lib/       (零 error)

Step 6: 编译验证
        → go build ./cmd/verba/
        → flutter build windows --debug

Step 7: 运行验证
        → 启动后端: go run ./cmd/verba/
        → 启动前端: flutter run -d windows
        → 手动验收功能

Step 8: 更新测试覆盖文档
        → docs/测试覆盖.md 补充新测试的覆盖说明
```

### Bug 修复的标准流程

```
Step 1: 先写复现测试（证明 bug 存在）
Step 2: 跑测试确认失败
Step 3: 修复代码
Step 4: 跑全部测试确认通过
Step 5: 编译 + 运行验证
```

---

## 四、后端日志规范（详细）

### 日志级别

| 级别 | 用途 | 示例 |
|------|------|------|
| `[INFO]` | 正常流程关键节点 | session 创建、ASR 调用、翻译完成 |
| `[WARN]` | 可恢复异常 | API 重试、修正超时、静音检测 |
| `[ERROR]` | 不可恢复错误 | API Key 无效、端口占用、crash |

### 必须打日志的位置

```
服务启动/停止
  → [server] starting on :8080
  → [server] shutting down...

每个 HTTP 请求
  → [http] POST /api/v1/sessions 201 3ms
  → [http] POST /api/v1/sessions/{id}/audio 202 1ms

每个外部 API 调用
  → [asr] request bytes=64000
  → [asr] response text="Hello world" duration=782ms
  → [translate] request text="Hello world"
  → [translate] response text="你好世界" duration=320ms

每个 session 事件
  → [session] created id=sess_123
  → [session] listening id=sess_123
  → [session] stopped id=sess_123 duration=5m32s

每个错误
  → [ERROR] asr timeout session=sess_123 retry=2/3
  → [ERROR] translate api_key_invalid
```

### 日志实现方式

Go 后端统一使用 `log` 标准库（不引入第三方日志框架）：

```go
log.Printf("[asr] transcribed session=%s bytes=%d duration=%dms", sessionID, len(audio), elapsed)
log.Printf("[ERROR] asr failed session=%s err=%v", sessionID, err)
```

---

## 五、测试规范

### Go 测试

- 测试文件与源码同目录：`xxx.go` → `xxx_test.go`
- 使用标准库 `testing`，不引入 testify 等框架
- 每个测试函数命名：`Test<FunctionName>` 或 `Test<Feature>_<Scenario>`
- 表驱动测试优先于多个独立测试函数
- 并发测试必须用 `-race` 标志验证

### Flutter 测试

- 测试文件放在 `test/` 目录
- 纯逻辑测数据模型 / 状态管理 / 协议解析
- Widget 测试仅在 UI 稳定后添加
- 使用 `flutter test` 运行（不依赖设备）

### 测试命名约定

```
Test<被测对象>_<场景>_<预期结果>

示例:
TestCreateSession                              // 正常创建
TestAppendSentence_TooFewSentences_NoTrigger   // 不足6句不触发修正
TestApplyCorrection_LowerRevision_Rejected     // 低版本修正被拒绝
```

---

## 六、Git 规范

### 提交前 Checklist

每次提交前必须完成下面检查，并用实际 diff 验证：

1. 用户本轮明确要求改什么？
2. 我实际改了什么？
3. `git diff --name-only` 里的每个文件是否都属于用户要求范围？
4. 是否出现了“偏移”：改了用户没要求的 UI、样式、布局、文案、重构或清理？
5. 如果有偏移，是否已经先向用户说明并获得确认？
6. 是否包含 `.env`、API key、token、日志、构建产物或 Flutter/平台 generated 文件？
7. 是否有用户已有的无关脏改被误 stage？
8. 是否跑过与改动范围匹配的测试或静态检查？

### Commit 格式

```
<type>: <简短描述>

<详细说明（可选）>

- 测试: <测试文件及覆盖点>
```

type 取值：
- `feat`: 新功能
- `fix`: Bug 修复
- `test`: 纯测试
- `refactor`: 重构（不改变行为）
- `docs`: 文档
- `chore`: 构建/依赖/工具

示例：
```
feat: 实现 WASAPI 音频捕获 + 实时音量表

- 通过 C++ DLL 调用 Windows Core Audio API
- 采用 polling 模式（10ms 间隔），规避事件驱动兼容问题
- Flutter 端显示实时音量条 + 诊断信息

测试: server/internal/session/manager_test.go (8个用例)
      client/test/session_provider_test.dart (8个用例)
```

### 分支策略

- `main` — 稳定分支，只接受 PR
- `dev` — 开发分支
- `feat/<功能名>` — 功能分支
- `fix/<问题描述>` — 修复分支

---

## 七、当前进度与下一步

### 已完成（第0步 WASAPI 部分）

- [x] WASAPI loopback 系统音频捕获（polling 模式，已验证通过）
- [x] Flutter 实时音量表 + 诊断信息显示
- [x] Go 后端编译通过（API stub + session 管理 + 修正引擎骨架）
- [x] 测试框架就绪（后端 29 测，前端 19 测，全部通过）

### 下一步 — 第0步剩余项

| 任务 | 说明 |
|------|------|
| 0.3 前后端联调 | 客户端创建 session → SSE 接收事件 |
| 0.4 音频 chunk 上传 | WASAPI 捕获音频 → 编码 → 上传到后端 |

### 然后进入第1步

| 任务 | 说明 |
|------|------|
| 1.1 Flutter `record` 包 | 麦克风录音（Phase 1 先用麦克风走通链路） |
| 1.2 后端 Whisper ASR | 真实 OpenAI API 调用 |
| 1.3 后端 GPT-4o-mini 翻译 | 英文 → 中文 |
| 1.4 SSE 推送字幕 | subtitle.final 事件 |
| 1.5 客户端字幕列表渲染 | 收到事件 → 追加列表 → 自动滚动 |

---

> **记住：先测试，后实现。先日志，后功能。每步可验证。**
