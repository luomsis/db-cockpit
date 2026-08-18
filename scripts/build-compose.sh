#!/usr/bin/env bash
# ============ 构建 compose 镜像（自动注入 git 版本与构建时间） ============
# 用法：scripts/build-compose.sh [--no-cache]
# 等价于：
#   GIT_SHA=$(git rev-parse --short HEAD) BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) docker compose build
set -euo pipefail
cd "$(dirname "$0")/.."

GIT_SHA="${GIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_TIME="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

echo "[build] GIT_SHA=${GIT_SHA} BUILD_TIME=${BUILD_TIME}"
export GIT_SHA BUILD_TIME

docker compose build "$@"
echo "[build] done —— 版本信息已注入 frontend(/version.json) 与 apiserver(/api/version)"
