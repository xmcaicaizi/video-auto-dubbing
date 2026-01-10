# 部署指南

> 更新日期：2026-01-10｜适用版本：main 分支

本指南面向 Linux/云服务器部署，默认使用 Docker Compose。需要 GPU 推理时请准备 NVIDIA 驱动与 NVIDIA Container Toolkit。

## 前置要求

- Linux x86_64（Ubuntu 20.04+ / Debian 11+ / CentOS 8+）
- Docker Engine 20.10+
- Docker Compose（推荐 v2；如仅有 `docker-compose` v1 也可用）
- 至少 8GB 可用内存（模型推理建议 16GB+）
- 至少 50GB 可用磁盘（模型权重与数据单独预留）
- 可选：NVIDIA GPU + 驱动 + NVIDIA Container Toolkit
- 无 GPU 环境可运行但速度较慢，需移除 `docker-compose.yml` 中 `tts_service` 的 `runtime: nvidia` 与 `deploy.resources`

## 快速开始（Docker Compose）

> 下文示例使用 `docker compose`（v2）。如果你的服务器只有 `docker-compose`（v1），请将命令中的 `docker compose` 替换为 `docker-compose`；或直接使用 `bash scripts/bootstrap.sh`（会自动兼容 v1/v2）。

1. **克隆项目并进入目录**
```bash
git clone <repository-url>
cd vedio
```

2. **配置环境变量**

复制示例并按需修改（务必替换密码与 API Key）：

```bash
cp env.example .env
```

关键配置建议关注：
- `POSTGRES_PASSWORD` / `MINIO_ROOT_PASSWORD` / `RABBITMQ_PASSWORD`
- `MINIO_PUBLIC_ENDPOINT`（公网访问时必填，用于生成可访问的下载链接，例如 `your.public.ip:9000`）
- `GLM_API_KEY` / `GLM_API_URL` / `GLM_MODEL`
- `TTS_PORT`（直连 TTS 服务端口，`env.example` 中为 8000；未设置则按 compose 默认映射 8001:8000）
- `INDEXTTS_MODEL_DIR` / `INDEXTTS_CFG_PATH` / `INDEXTTS_DEVICE`
- `HF_ENDPOINT`（国内可用镜像）/ `HF_HUB_CACHE`

3. **下载 IndexTTS2 模型权重**

推荐直接在容器内下载到持久化卷：

```bash
docker compose run --rm tts_service python -c "from huggingface_hub import snapshot_download; snapshot_download(repo_id='IndexTeam/IndexTTS-2', local_dir='/app/models/IndexTTS-2', local_dir_use_symlinks=False)"
```

如需离线部署，请将模型解压到与 `INDEXTTS_MODEL_DIR` 一致的目录。

4. **（可选）预拉取 Moonshine ASR 模型**

Moonshine ASR 会在首次调用时自动下载模型；也可以提前拉取以减少首个任务的等待时间：

```bash
docker compose run --rm asr_service python -c "import os, moonshine_onnx; moonshine_onnx.MoonshineOnnxModel(model_name=os.environ.get('ASR_MODEL_ID','moonshine/tiny')); print('Moonshine ASR model ready')"
```

5. **启动服务**

```bash
docker compose up -d
```

6. **验证状态**

```bash
docker compose ps
curl http://localhost:8080/health
```

## 服务访问地址（默认端口）

| 服务 | 地址 | 说明 |
|------|------|------|
| API 服务 | http://localhost:8080 | 后端 API |
| ASR Service | http://localhost:8002 | Moonshine ASR |
| TTS 服务 | http://localhost:8000/8001 | 直连 TTS（端口由 `TTS_PORT` 决定） |
| MinIO 控制台 | http://localhost:9001 | 对象存储管理 |
| RabbitMQ 管理 | http://localhost:15672 | 消息队列管理 |
| 网关 | http://localhost:80 | 统一入口 |

## 云服务部署要点（Linux）

- **安全组/防火墙**：仅开放必要端口（通常为 80/443；如需要通过预签名链接直连 MinIO 下载，请开放 9000；管理端口 9001/15672 建议内网或限制访问）。
- **GPU 主机**：选择带 NVIDIA GPU 的实例并安装驱动与 Container Toolkit，确保容器可见 GPU。
- **对象存储访问**：请将 `MINIO_PUBLIC_ENDPOINT` 设置为浏览器可访问的地址（例如 `公网IP:9000` 或域名），用于生成可下载的预签名链接。
- **持久化存储**：使用云硬盘挂载 Docker 数据目录，避免重建容器导致模型与数据丢失。

## 扩展 Worker 实例

```bash
docker compose up -d --scale worker=3
docker compose ps worker
```

## 数据持久化与备份

Docker volumes：
- `postgres_data`: PostgreSQL 数据
- `minio_data`: MinIO 数据
- `rabbitmq_data`: RabbitMQ 数据
- `tts_models`: TTS 模型权重
- `tts_temp`: TTS 临时文件
- `api_logs` / `worker_logs`: 服务日志

备份示例：
```bash
docker compose exec db pg_dump -U dubbing dubbing > backup.sql
docker compose exec minio mc mirror /data /backup
```

## 健康检查与日志

```bash
docker compose logs -f api
docker compose logs -f worker
docker compose logs -f tts_service
```

## 停止与清理

```bash
docker compose stop
docker compose down
```

谨慎执行（会删除数据卷）：

```bash
docker compose down -v
```

## 更新服务

```bash
git pull
docker compose build
docker compose up -d
```

## 故障排查

- **TTS 无法就绪**：检查模型是否下载到 `INDEXTTS_MODEL_DIR`，以及 GPU/驱动是否可用。
- **ASR 启动失败**：确认 ASR 模型下载与 `ASR_DEVICE` 设置。
- **翻译报错**：确认 `GLM_API_KEY` 与网络连通性。
- **音频上传失败**：检查 MinIO 账号与 `MINIO_PUBLIC_ENDPOINT` 配置。

## 性能调优

- 通过 `TTS_MAX_CONCURRENCY`、`TTS_BATCH_SIZE` 调整合成并发。
- 通过 `TRANSLATE_BATCH_SIZE`、`TRANSLATE_MAX_TEXT_LENGTH` 控制翻译批量与长度。
- 结合 `docker compose up -d --scale worker=N` 扩展吞吐。
