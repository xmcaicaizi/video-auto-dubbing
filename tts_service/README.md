# TTS Service

基于 IndexTTS2 的语音合成服务，支持时间轴约束的可控语音合成。

## 快速开始

### 使用 uv 管理依赖

```bash
# 创建虚拟环境
uv venv

# 激活虚拟环境
# Linux/Mac:
source .venv/bin/activate
# Windows:
.venv\Scripts\activate

# 安装依赖
uv sync

# 启动服务
uv run uvicorn app.main:app --host 0.0.0.0 --port 8000
```

### 环境配置

复制 `.env.example` 为 `.env` 并修改配置：

```bash
cp .env.example .env
```

### 模型权重

IndexTTS2 使用本地推理模型权重。请提前从 HuggingFace 下载到 `INDEXTTS_MODEL_DIR`：

```bash
python -c "from huggingface_hub import snapshot_download; snapshot_download(repo_id='IndexTeam/IndexTTS-2', local_dir='models/IndexTTS-2', local_dir_use_symlinks=False)"
```

### API 文档

启动服务后访问：
- Swagger UI: http://localhost:8000/docs
- ReDoc: http://localhost:8000/redoc

## 开发

### 添加依赖

```bash
uv add package-name
```

### 更新依赖

```bash
uv lock --upgrade
uv sync
```

### 代码格式化

```bash
uv run black .
uv run ruff check .
```

## 部署

### Docker

```bash
docker build -t tts-service .
docker run -p 8000:8000 tts-service
```

### 生产环境

使用 `uv sync --frozen` 确保依赖版本锁定。

