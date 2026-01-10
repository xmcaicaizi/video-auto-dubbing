#!/usr/bin/env bash
# One-click Docker bootstrap: env setup + model download + compose up.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="$ROOT_DIR/.env"
ENV_EXAMPLE="$ROOT_DIR/env.example"

if ! command -v docker >/dev/null 2>&1; then
    echo "错误: 未找到 docker，请先安装 Docker Engine 和 Docker Compose。"
    exit 1
fi

# Compose command compatibility:
# - Prefer Docker Compose v2 plugin: `docker compose ...`
# - Fallback to legacy v1 binary: `docker-compose ...`
if docker compose version >/dev/null 2>&1; then
    COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE=(docker-compose)
else
    echo "错误: 未找到 Docker Compose（尝试安装 docker compose 插件或 docker-compose）。"
    exit 1
fi

if [ ! -f "$ENV_FILE" ] && [ -f "$ENV_EXAMPLE" ]; then
    cp "$ENV_EXAMPLE" "$ENV_FILE"
    echo "已复制 env.example -> .env"
fi

GLM_KEY="${GLM_API_KEY:-}"
if [ -z "$GLM_KEY" ] && [ -f "$ENV_FILE" ]; then
    GLM_KEY="$(grep -E '^GLM_API_KEY=' "$ENV_FILE" | tail -n1 | cut -d= -f2- | tr -d '\r' | tr -d '"' | tr -d "'")"
fi
if [ -z "$GLM_KEY" ] || [ "$GLM_KEY" = "your_glm_api_key" ]; then
    echo "提示: GLM_API_KEY 未配置，翻译步骤会失败。请在 .env 中设置真实值。"
fi

echo "检查 IndexTTS2 模型权重..."
"${COMPOSE[@]}" run --rm tts_service python - <<'PY'
from pathlib import Path
import os
from huggingface_hub import snapshot_download

model_dir = Path(os.environ.get("INDEXTTS_MODEL_DIR", "/app/models/IndexTTS-2"))
cfg = Path(os.environ.get("INDEXTTS_CFG_PATH", model_dir / "config.yaml"))
if cfg.exists():
    print(f"IndexTTS2 已就绪: {cfg}")
else:
    print("正在下载 IndexTTS2 模型权重...")
    snapshot_download(
        repo_id="IndexTeam/IndexTTS-2",
        local_dir=str(model_dir),
        local_dir_use_symlinks=False,
    )
    print("IndexTTS2 模型下载完成")
PY

echo "检查 Moonshine ASR 模型..."
"${COMPOSE[@]}" run --rm asr_service python - <<'PY'
import os

import moonshine_onnx

model_id = os.environ.get("ASR_MODEL_ID", "moonshine/tiny")

# Instantiate model to trigger HuggingFace download into cache/volume.
moonshine_onnx.MoonshineOnnxModel(model_name=model_id)
print(f"Moonshine ASR 模型已就绪: {model_id}")
PY

echo "启动服务..."
"${COMPOSE[@]}" up -d --build
"${COMPOSE[@]}" ps
