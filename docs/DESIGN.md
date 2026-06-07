# 方案设计

这是 Verba 方案设计文档的入口。设计文档回答“怎么做”，包括 UI 行为、系统取舍、API 选型、交互细节和实现边界。

## 基本原则

- 悬浮字幕要轻，不要像工具面板一样遮挡视频主体。
- 中文译文是主视觉，英文原文是辅助对照。
- 修正是可信度反馈，只高亮被修正的句子，不闪整个窗口。
- 后端服务商接入要配置化、可替换，避免把模型和供应商写死。

## 文档入口

- [设计文档目录](design-docs/index.md)
- [核心设计信念](design-docs/core-beliefs.md)
- [API 服务商选型](design-docs/api-provider-selection.md)
- [悬浮窗口设计](design-docs/floating-window-design.md)
- [歌词式悬浮 UI 设计](design-docs/lyric-floating-ui-design.md)
- [修正效果设计](design-docs/correction-effect.md)
- [全局 UI 流程](design-docs/global-ui-flow.svg)
- [修正流程 UI](design-docs/correction-flow-ui.svg)
