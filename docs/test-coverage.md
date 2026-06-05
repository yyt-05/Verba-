# Verba 测试覆盖说明

> 更新时间：2026-06-05

---

## 测试运行命令

```bash
# 后端（全部模块）
cd server && go test ./internal/... -v -count=1

# 前端（纯 Dart 单测）
cd client && flutter test
```

---

## 一、后端测试覆盖（Go）

### 1.1 session 包 — `internal/session/manager_test.go`

| 测试用例 | 覆盖点 | 为什么重要 |
|---------|------|------|
| `TestCreateSession` | 创建、查询、删除会话 | 会话生命周期的基础能力 |
| `TestAppendSentence` | 追加句子、索引递增、修正触发条件（seq=6 和 seq=9 各触发一次） | 修正引擎的触发逻辑是否正确 |
| `TestAppendSentenceNoTriggerOnNonMultiple` | seq=7 时不触发修正（7%3≠0） | 防止每次追加都触发修正 |
| `TestGetWindow` | 取最近 12 句（裁剪掉足够前面的句子）、不足 12 句时返回全部 | 滑动窗口的边界正确性 |
| `TestApplyCorrection` | 高版本号覆盖成功、低版本号被拒绝、最终值验证 | 乐观锁版本控制是修正的核心保障 |
| `TestApplyCorrectionWrongIndex` | 不存在的 segmentId 修正失败 | 边界防护 |
| `TestSetStatus` | 状态切换（created→listening→stopped） | 客户端依赖 session 状态做 UI |
| `TestManagerThreadSafety` | 10 个并发 goroutine 同时追加句子，验证最终数量为 10 | 音频流是多线程上传的，并发安全是必备 |

**覆盖的核心逻辑**：
- 会话创建/查询/删除
- 句子追加 + 修正触发条件（`len≥6 && seq%3==0`）
- 滑动窗口裁剪（最近 N 句）
- 修正版本锁（低 revision 拒绝）
- 并发追加安全性

---

### 1.2 audio 包 — `internal/audio/chunk_test.go`

| 测试用例 | 覆盖点 |
|---------|------|
| `TestChunkValidateEmpty` | 空 chunk 合法（静音片段跳过而非报错） |
| `TestChunkValidateTooLarge` | 超 2MB chunk 被拒绝 |
| `TestChunkValidateNormal` | 正常大小 chunk 通过验证 |
| `TestProcessorAccumulateBeforeTrigger` | 缓冲未到 64000 字节不触发，累计刚好达到时触发 |
| `TestProcessorSkipsEmptyChunk` | 空 chunk 不累积到缓冲区 |
| `TestProcessorResetsAfterTrigger` | 触发后缓冲区归零重新累积 |
| `TestProcessorOversizeTriggers` | 一次性超量的 chunk 直接触发 |

**覆盖的核心逻辑**：
- chunk 大小校验（空/正常/超大）
- 缓冲区累积阈值（64000 字节 = 2秒 16kHz PCM16）
- 触发后重置
- 超大 chunk 一次性触发

---

### 1.3 sse 包 — `internal/sse/broker_test.go`

| 测试用例 | 覆盖点 |
|---------|------|
| `TestSubscribeUnsubscribe` | 订阅后取消，发布不阻塞 |
| `TestPublishToSingleSubscriber` | 发布事件→订阅者正确收到，字段完整（ID/Type/SegmentID） |
| `TestPublishToMultipleSubscribers` | 同一 session 多个订阅者都收到事件 |
| `TestPublishSessionIsolation` | session A 的事件不会发送到 session B |
| `TestPublishDropsOnFullChannel` | channel 满时丢弃而非阻塞（慢客户端不拖累其他用户） |
| `TestBuildSubtitleFinal` | 事件 JSON 包含 correct fields（original/translation/segmentId/revision） |
| `TestBuildCorrection` | 修正事件 JSON 包含 oldText/newText/revision |

**覆盖的核心逻辑**：
- 发布/订阅/退订机制
- session 隔离（不会串数据）
- 慢客户端丢弃保护
- 事件构建函数的字段正确性

---

### 1.4 pipeline 包 — `internal/pipeline/corrector_test.go`

| 测试用例 | 覆盖点 |
|---------|------|
| `TestNeedsCorrectionTooFewSentences` | 仅有 4 句时不触发（<6） |
| `TestNeedsCorrectionAtTriggerPoint` | 第 6 句时触发（≥6 && seq%3==0） |
| `TestNeedsCorrectionNotAtNonTriggerPoint` | 第 7 句时不触发（≥6 但 seq%3≠0） |
| `TestBuildCorrectionPrompt` | Prompt 包含原文、译文、segment_index、confidence |
| `TestWindowHashConsistency` | 相同窗口→相同 hash，不同窗口→不同 hash |
| `TestWindowHashEmpty` | 空窗口也能生成 hash |
| `TestCorrectorDefaults` | 默认参数：WindowSize=12, TriggerEvery=3, LookbackCount=6 |

**覆盖的核心逻辑**：
- 触发条件精确性
- LLM prompt 构建
- 乐观锁 hash 生产（变更检测）

---

## 二、前端测试覆盖（Flutter/Dart）

### 2.1 subtitle_entry_test.dart — 数据模型

| 测试用例 | 覆盖点 |
|---------|------|
| 构造 + 默认值 | segmentId/original/translation/revision 赋值正确，isCorrected 默认 false |
| fromJson 解析 | subtitle.final 事件 JSON 反序列化正确 |
| fromJson 缺失字段 | 缺失字段用默认值（空字符串、revision=1） |
| copyWith 不改变未指定字段 | 仅 translation 变，其余不变 |
| copyWith 递增 revision | revision/isCorrected/oldTranslation 全部正确更新 |

### 2.2 session_provider_test.dart — 状态管理

| 测试用例 | 覆盖点 |
|---------|------|
| 初始状态为空 | 新创建的 notifier state 为空列表 |
| 追加字幕 | 单条追加后列表长度为 1 |
| 追加多条保持顺序 | 5 条字幕 segmentId 顺序 0→4 |
| applyCorrection 更新 | 按 segmentId 找到→更新 translation→revision++→isCorrected=true→oldTranslation 保存旧值 |
| applyCorrection 低版本拒绝 | revision 3→2 的修正被忽略 |
| applyCorrection 不存在索引 | segmentId 999 修正无影响 |
| 列表超 200 条裁剪 | 250→200，前 50 条被丢弃，segmentId 从 50 开始 |
| clear 清空 | 5 条→clear→0 条 |

### 2.3 api_client_test.dart — SSE 协议解析

| 测试用例 | 覆盖点 |
|---------|------|
| 单个 subtitle.final 事件 | event/id/data/空行 → 解析为 1 条完整 JSON |
| 多个连续事件 | 3 个事件依次产出，按顺序 |
| 多行 data 合并 | 两行 data 拼成一行 |
| 空事件/注释行忽略 | `:comment` 和空行不产出多余事件 |
| 流尾无空行 | 结尾 data 行 + 空行正常产出 |

---

## 三、测试结果汇总

```
=== Go 后端 ===
✅ internal/audio      — 7 tests passed
✅ internal/pipeline   — 7 tests passed
✅ internal/session    — 8 tests passed
✅ internal/sse        — 7 tests passed
-  internal/config     — (无测试，纯配置读取)

=== Flutter 前端 ===
✅ subtitle_entry_test      — 5 tests passed
✅ session_provider_test    — 8 tests passed
✅ api_client_test          — 6 tests passed

总计: 48 tests, 0 failures
```

---

## 四、未覆盖项（有意为之）

| 模块 | 原因 |
|------|------|
| WASAPI C++ DLL | COM 硬件初始化依赖真实设备，单测无意义，实机验证 |
| ASR/翻译 API 调用 | 目前是 Phase 1 stub，Phase 1 加入集成测试 |
| Flutter Widget | Phase 0 布局还在快速迭代，Widget test 维护成本过高 |
| pipeline.go (HTTP handlers) | 依赖真实 http.Request，用集成测试更合适 |
| config 包 | 纯环境变量读取，逻辑为零，不测 |

---

## 五、将来要加的测试（Phase 1+）

- [ ] ASR 调用 mock 集成测试（指定音频 → 验证返回文本）
- [ ] 翻译调用 mock 集成测试（指定英文 → 验证中文）
- [ ] 修正引擎全链路集成测试（ASR→翻译→修正 pipeline）
- [ ] HTTP handler 集成测试（真实 POST/GET 请求）
- [ ] SSE 断线重连 + Last-Event-ID 续传
- [ ] 成本控制（单 session 预算超限自动停止）
