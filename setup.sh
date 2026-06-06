#!/bin/bash
# Verba 开发环境初始化
set -e

echo "==> 配置 Git hooks..."
git config core.hooksPath .githooks
echo "  OK — pre-push hook 已激活"

echo ""
echo "==> 安装完成"
echo "现在每次 git push 前会自动跑 CI 检查"
