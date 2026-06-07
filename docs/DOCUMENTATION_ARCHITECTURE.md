# 文档目录架构

这份文档说明 Verba 的文档应该放在哪里、每类文件解决什么问题，以及新增文档时该怎么命名。

## 总体结构

```text
AGENTS.md
ARCHITECTURE.md
docs/
├── DOCUMENTATION_ARCHITECTURE.md
├── DESIGN.md
├── FRONTEND.md
├── PLANS.md
├── PRODUCT_SENSE.md
├── QUALITY_SCORE.md
├── RELIABILITY.md
├── SECURITY.md
├── design-docs/
├── exec-plans/
├── generated/
├── product-specs/
└── references/
```

## 根目录文件

| 文件 | 作用 | 应该写什么 |
|---|---|---|
| `AGENTS.md` | Agent 工作规则 | 变更边界、提交前检查、禁止事项、用户偏好 |
| `ARCHITECTURE.md` | 项目整体架构 | 系统职责、模块边界、运行时链路、关键技术选型 |

根目录只放全局入口文档。具体需求、方案、计划和参考资料都放进 `docs/`。

## AI 使用规则

AI 进入项目后，应把 `AGENTS.md` 当成最高优先级的项目规则入口。涉及架构、需求、设计、计划或较大功能变更时，还应主动阅读：

- `ARCHITECTURE.md`：理解系统整体架构和模块边界。
- `docs/DOCUMENTATION_ARCHITECTURE.md`：理解文档目录怎么用。
- 对应领域入口：例如需求看 `docs/PRODUCT_SENSE.md`，设计看 `docs/DESIGN.md`，计划看 `docs/PLANS.md`。

AI 主动补充文档的判断标准：

- 改了用户可见需求或功能边界：补 `product-specs/`。
- 改了技术方案、接口协议、状态流、UI 行为或架构取舍：补 `design-docs/`。
- 开始一个有步骤的任务：补 `exec-plans/active/`。
- 完成一个计划：移动到 `exec-plans/completed/` 并写完成摘要。
- 发现暂不处理但确实存在的工程问题：补 `exec-plans/tech-debt-tracker.md`。
- 生成了 schema、接口清单、依赖报告等机械产物：放 `generated/`。
- 收集了外部文档或 LLM 参考资料：放 `references/`。

如果本次只是很小的局部修复，且没有改变需求、设计、接口、测试策略、可靠性或安全边界，可以不补文档，但应在最终回复里说明“不需要更新文档”的原因。

## docs 入口文件

| 文件 | 作用 | 应该写什么 |
|---|---|---|
| `docs/DOCUMENTATION_ARCHITECTURE.md` | 文档架构说明 | 文档目录怎么用、每类文档的写法和迁移规则 |
| `docs/PRODUCT_SENSE.md` | 需求入口 | 产品原则、需求目录、MVP 边界 |
| `docs/DESIGN.md` | 设计入口 | 设计原则、方案目录、核心取舍 |
| `docs/FRONTEND.md` | 前端入口 | 前端形态、交互原则、视觉边界 |
| `docs/PLANS.md` | 计划入口 | 活动计划、已完成计划、技术债 |
| `docs/QUALITY_SCORE.md` | 质量入口 | 测试覆盖、验收方式、质量现状 |
| `docs/RELIABILITY.md` | 可靠性入口 | 重试、降级、状态恢复、SSE 和会话可靠性 |
| `docs/SECURITY.md` | 安全入口 | 密钥、日志、隐私、generated 文件提交边界 |

入口文件不应该写成长篇正文。它们的职责是建立原则和链接，把读者引到具体文档。

## product-specs

`docs/product-specs/` 放需求文档，回答“做什么”和“为什么做”。

适合放在这里：

- 用户画像和使用场景。
- MVP 范围。
- 用户故事和验收标准。
- 需求评审。
- 功能边界和暂缓事项。

不适合放在这里：

- 具体 API 调用方案。
- UI 动画实现细节。
- 数据结构、模块拆分和代码计划。

命名建议：

```text
product-specs/
├── index.md
├── new-user-onboarding.md
├── tts-playback.md
└── requirements-review.md
```

## design-docs

`docs/design-docs/` 放设计文档，回答“怎么做”。

适合放在这里：

- UI/UX 方案。
- API 服务商选型。
- 后端模块设计。
- 状态机、流程图、事件协议。
- 关键技术取舍。

不适合放在这里：

- 纯产品愿景。
- 当前执行 checklist。
- 自动生成的 schema 或接口导出。

命名建议：

```text
design-docs/
├── index.md
├── core-beliefs.md
├── floating-window-design.md
├── correction-pipeline-design.md
└── api-provider-selection.md
```

## exec-plans

`docs/exec-plans/` 放执行计划，回答“这次怎么落地”。

```text
exec-plans/
├── active/
├── completed/
└── tech-debt-tracker.md
```

`active/` 放正在推进的任务计划。每个计划最好包含：

- 背景。
- 明确范围。
- 不做什么。
- 分步骤 checklist。
- 验收标准。
- 需要运行的测试或检查。

`completed/` 放完成后的计划。移动到这里前，应在文件顶部补：

- 完成日期。
- 实际完成内容。
- 未完成或后续事项。

`tech-debt-tracker.md` 只记录已知技术债，不承载具体功能计划。

## generated

`docs/generated/` 放机器生成或技术导出的文档。

适合放在这里：

- 数据库 schema 导出。
- OpenAPI 导出。
- 自动生成的接口清单。
- 依赖分析结果。

写作规则：

- 生成文件要标明生成方式和生成时间。
- 不要把需求讨论写进 generated。
- 提交前确认没有密钥、token、日志或本地路径隐私。

## references

`docs/references/` 放外部参考资料和 LLM 友好的原始资料。

适合放在这里：

- 框架文档摘录。
- 服务商文档快照。
- 设计系统参考。
- 部署平台说明。
- 长文本 prompt 参考材料。

写作规则：

- 文件名尽量说明来源，例如 `openai-audio-api-reference.md`。
- 如果内容来自外部文档，顶部写明来源链接和获取日期。
- 不把 references 当成正式方案，正式结论要沉淀到 `design-docs/` 或 `product-specs/`。

## 新增文档流程

1. 先判断文档回答的问题：
   - 做什么、为什么做：放 `product-specs/`。
   - 怎么做：放 `design-docs/`。
   - 当前怎么执行：放 `exec-plans/active/`。
   - 机器生成：放 `generated/`。
   - 外部参考：放 `references/`。
2. 使用英文小写短横线文件名，避免中文路径在终端里乱码。
3. 在对应 `index.md` 或入口文件里加链接。
4. 如果计划完成，移动到 `exec-plans/completed/`，不要留在 active。
