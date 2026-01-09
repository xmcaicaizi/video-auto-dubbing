# 快速启动指南

> 更新日期：2026-01-03｜适用版本：main 分支

本文档采用 UTF-8 编码保存，以避免中文显示乱码。

本指南是视频本地化自动配音系统的唯一权威快速开始文档，包含从环境准备到常见问题的完整步骤。建议新旧使用者均以此为准。若需更多上下文，请参阅 `docs/README.md` 提供的文档索引。

## 1. 前置要求

- **Docker Engine** 20.10+、**Docker Compose** 2.0+
- 至少 **8GB 内存**、**50GB 磁盘** 可用空间
- 终端可访问互联网（用于拉取镜像与依赖）

## 2. 获取代码

```bash
git clone <repository-url>
cd vedio
```

## 3. 配置环境变量（可选，但推荐）

> 推荐在前端“设置”页面填写密钥；下述环境变量供容器启动前兜底使用。

- `GLM_API_KEY`: 智谱 GLM 翻译 API Key
- `GLM_API_URL`: GLM API 地址（默认 `https://open.bigmodel.cn/api/paas/v4/chat/completions`）
- `GLM_MODEL`: GLM 模型名（默认 `glm-4.5`）
- `GLM_RPS`: GLM 每秒请求数上限（默认 5）
- `ASR_SERVICE_URL`: Moonshine ASR 服务地址（默认 `http://localhost:8002`）
- `ASR_MODEL_ID`: Moonshine 模型 ID（默认 `moonshine-base`）
- `ASR_DEVICE`: 运行设备（默认 `cuda`）
- `ASR_COMPUTE_TYPE`: 推理精度（默认 `float16`）
- `ASR_BACKEND`: ASR 后端（默认 `moonshine_onnx`）
- `TTS_BACKEND`: TTS 后端（默认 `index_tts2`，本地推理）
- `INDEXTTS_MODEL_DIR`: IndexTTS2 模型目录（默认 `/app/models/IndexTTS-2`）
- `INDEXTTS_CFG_PATH`: IndexTTS2 配置文件（默认 `/app/models/IndexTTS-2/config.yaml`）
- `HF_ENDPOINT`: HuggingFace 访问地址（可选，默认 `https://hf-mirror.com`）
- `HF_HUB_CACHE`: HuggingFace 缓存目录（默认 `/app/models/IndexTTS-2/hf_cache`）
- `MINIO_PUBLIC_ENDPOINT`: MinIO 外部可访问地址（ASR 服务不在同一网络时使用）

若环境支持 dotfile，可将 `env.example` 复制为 `.env` 并按需修改；否则请在宿主机或 CI 中设置为系统环境变量。

## 4. 下载 IndexTTS-2 模型权重（必需）

IndexTTS2 使用本地推理，需要提前拉取模型权重到 TTS 容器的模型目录。推荐通过容器直接下载到持久化卷：

```bash
docker compose run --rm tts_service python -c "from huggingface_hub import snapshot_download; snapshot_download(repo_id='IndexTeam/IndexTTS-2', local_dir='/app/models/IndexTTS-2', local_dir_use_symlinks=False)"
```

如需走镜像站点，确保 `.env` 中配置 `HF_ENDPOINT`，或在命令前设置环境变量。

## 5. 启动与验证

```bash
# 启动全部服务
docker compose up -d

# 查看服务状态
docker compose ps

# 查看日志（可选）
docker compose logs -f
```

### 服务访问入口

| 服务 | 地址 | 说明 |
| --- | --- | --- |
| 前端页面 | http://localhost | 通过 NGINX 访问 UI |
| API 服务 | http://localhost:8080 | 后端 REST API |
| ASR 服务 | http://localhost:8002 | Moonshine ASR |
| TTS 服务 | http://localhost:8001 | 语音合成（默认 8001，避免与 8000 冲突） |
| MinIO 控制台 | http://localhost:9001 | 对象存储管理 |
| RabbitMQ 管理台 | http://localhost:15672 | 消息队列管理 |
| PostgreSQL | localhost:5432 | 数据库 |

### 健康检查与烟囱测试

```bash
# 检查容器状态
docker compose ps

# 查看所有服务日志
docker compose logs -f

# API 与 TTS 健康检查
curl http://localhost:8080/health
curl http://localhost:8001/health
```

## 6. 常见问题

1. **Docker Engine 未运行**
   - 现象：`error during connect: open //./pipe/docker_engine: The system cannot find the file specified`
   - 处理：启动 Docker Desktop；在 Windows 上以管理员权限执行 `Start-Service com.docker.service`，再运行 `docker version` 确认 Server 信息可见。

2. **端口冲突**
   - 现象：`Bind for 0.0.0.0:8000 failed: port is already allocated`
   - 处理：使用 `docker ps --format "table {{.Names}}\t{{.Ports}}"` 查占用；调整 `docker-compose.yml` 中端口映射，或停止占用端口的容器。TTS 默认监听 8001 以避开 8000。

3. **构建时报缺少 go.sum**
   - 现象：`failed to calculate checksum: "/go.sum": not found`
   - 处理：仓库 Dockerfile 已自动执行 `go mod tidy`。如需本地生成，可分别在 `api` 与 `worker` 目录运行 `go mod tidy`。

4. **Go 代码未用 import 报错**
   - 现象：`"time" imported and not used`
   - 处理：删除未使用的 import，可用 `go vet ./...` 定位。

5. **TTS 启动找不到 uvicorn**
   - 现象：`exec: "uvicorn": executable file not found in $PATH`
   - 处理：镜像已固定使用 `/app/.venv/bin/uvicorn`，请确认未覆盖 `/app` 的 bind mount。

6. **TTS 健康检查找不到 curl**
   - 现象：`exec: "curl": executable file not found in $PATH`
   - 处理：健康检查已改用 Python `urllib`，如仍异常请检查 `docker-compose.yml` 的 healthcheck 配置。

7. **构建或拉取失败（网络/apt 源）**
   - 处理：Dockerfile 已切换 apt 源为 HTTPS；若仍失败，请检查网络或配置代理。

## 7. 停止、重启与重建

```bash
# 停止但保留数据
docker compose stop

# 停止并删除容器（保留数据卷）
docker compose down

# 停止并删除所有资源（含数据卷）
docker compose down -v

# 重启全部或指定服务
docker compose restart
# docker compose restart api
# docker compose restart worker

# 重新构建并启动
docker compose up -d --build
# 或仅重建指定服务
docker compose up -d --build api
```

## 8. 开发模式提示

- Go（API/Worker）：修改代码后可执行 `docker compose up -d --build api` 或 `worker` 触发重建；如使用 bind mount，请避免覆盖镜像内的构建产物。
- Python（TTS）：修改代码后 `docker compose up -d --build tts_service`；谨慎挂载 `.venv` 以防依赖失效。

## 9. 相关文档

- [文档索引](./README.md) — 指向架构、接口与开发规范
- [系统架构设计](architecture.md) — 服务边界与任务流程
- [部署指南](deployment.md)
- [API 合同](api-contracts.md)
- [ASR 服务说明](asr-service.md)
- [TTS 服务说明](tts-service.md)
