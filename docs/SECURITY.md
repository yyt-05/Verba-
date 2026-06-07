# 安全性

Verba 会接入外部 AI 服务，也会捕获本机音频，所以安全重点是密钥、隐私和数据流动边界。

## 基本原则

- 不提交 `.env`、API key、token、日志、构建产物或平台 generated 文件。
- 服务商凭据只放在基于环境变量的配置里。
- 除非明确用于本地调试，不记录原始音频、完整转写文本或密钥。
- generated 目录里的 schema、导出文件等也要先审查再提交。

## 检查项

- 检查 staged diff 中是否有 `API_KEY`、`SECRET`、`TOKEN` 或服务商 key。
- 确认 generated 文件是有意提交的。
- 确认 `.env` 和本地可执行文件没有被 stage。
