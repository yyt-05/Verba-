# 前端形态

前端目标是一个 Windows 桌面 Flutter 应用，核心界面是轻量、可拖动、长期覆盖在视频上方的悬浮字幕层。

## 形态原则

- 打开 App 后先出现小型悬浮入口，不直接展示大面板。
- 监听状态更接近桌面歌词，而不是完整工具窗口。
- 控制按钮必须服从字幕可读性，不能喧宾夺主。
- 字幕布局、颜色、动画、窗口行为属于视觉体验范围，只有用户明确要求时才改。

## 关键参考

- [悬浮窗口设计](design-docs/floating-window-design.md)
- [歌词式悬浮 UI 设计](design-docs/lyric-floating-ui-design.md)
- [修正效果设计](design-docs/correction-effect.md)
