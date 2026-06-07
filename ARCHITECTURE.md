# Verba 整体架构

Verba 是一个面向 Windows 桌面的 AI 实时双语字幕助手。客户端负责捕获本机音频、展示悬浮字幕；Go 后端负责会话、音频接收、ASR、翻译、修正和 SSE 推送。

## 系统形态

```text
Windows 系统音频 / 麦克风
  -> Flutter Windows 客户端
  -> Go HTTP API
  -> 音频分片与缓冲
  -> ASR 服务
  -> 翻译服务
  -> 修正流水线
  -> SSE 推送
  -> 悬浮双语字幕
```

## 运行时职责

| 区域 | 职责 |
|---|---|
| `client/` | Flutter Windows 外壳、悬浮窗、音频捕获接线、API 客户端、SSE 状态、字幕渲染 |
| `server/cmd/verba/` | 服务启动、路由注册、配置加载 |
| `server/internal/audio/` | 音频 chunk 校验、缓冲、分片 |
| `server/internal/config/` | 基于环境变量的运行时配置 |
| `server/internal/pipeline/` | ASR、翻译、修正和流水线编排 |
| `server/internal/session/` | 会话生命周期、字幕历史、修正 revision 规则 |
| `server/internal/sse/` | 后端事件到客户端的 fan-out |
| `server/internal/tts/` | 启用 TTS 时的语音合成接入 |

## 文档地图

项目文档按“决策类型”拆分：

| 路径 | 用途 |
|---|---|
| `AGENTS.md` | Agent 工作规则和变更边界 |
| `ARCHITECTURE.md` | 项目整体系统架构 |
| `docs/DOCUMENTATION_ARCHITECTURE.md` | 文档目录架构说明：每类文件放什么、怎么写 |
| `docs/product-specs/` | 需求文档：解决“做什么、为什么做”的问题 |
| `docs/design-docs/` | 设计文档：解决“怎么做”的问题 |
| `docs/exec-plans/` | 执行计划：当前任务、完成任务和技术债 |
| `docs/generated/` | 技术生成文件，例如数据库 schema、接口导出等 |
| `docs/references/` | 外部参考资料和 LLM 友好的长文本资料 |
| `docs/DESIGN.md` | 方案设计入口和基本原则 |
| `docs/FRONTEND.md` | 前端形态和交互原则 |
| `docs/PLANS.md` | 总体计划和计划入口 |
| `docs/PRODUCT_SENSE.md` | 产品需求入口和基本原则 |
| `docs/QUALITY_SCORE.md` | 质量和测试覆盖入口 |
| `docs/RELIABILITY.md` | 可靠性入口和基本原则 |
| `docs/SECURITY.md` | 安全性入口和基本原则 |
