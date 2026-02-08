# Docker 部署修复说明

## 🔧 修复内容

### 1. Redis 配置统一 ✅

**问题：**
- Docker Compose 中 Redis 设置了密码 `--requirepass`
- 本地开发环境未设置密码
- 导致连接失败

**解决方案：**
- 移除 Redis 密码配置，统一使用无密码模式
- 更新 `docker-compose.v2.yml` 和 `docker-compose.prod.yml`
- 移除所有环境变量中的 `REDIS_PASSWORD`

**文件修改：**
```yaml
# docker-compose.v2.yml
redis:
  command: redis-server --appendonly yes  # 移除 --requirepass
```

### 2. Celery Worker 队列配置 ✅

**问题：**
- Worker 命令未指定监听队列
- 任务路由配置与实际任务名称不匹配
- 任务被发送到 `celery` 默认队列，但 worker 未监听

**解决方案：**
- 更新 Celery worker 启动命令，明确指定监听的队列
- 修正任务路由配置，使用实际的任务名称

**文件修改：**
```yaml
# docker-compose.v2.yml & docker-compose.prod.yml
worker:
  command: celery -A app.workers.celery_app worker --loglevel=info --concurrency=${WORKER_CONCURRENCY:-4} --queues=default,media,ai,celery
```

```python
# backend/app/workers/celery_app.py
celery_app.conf.task_routes = {
    "process_video_pipeline": {"queue": "default"},
    "extract_audio": {"queue": "media"},
    "transcribe_audio": {"queue": "ai"},
    "translate_segments": {"queue": "ai"},
    "synthesize_audio": {"queue": "ai"},
    "mux_video": {"queue": "media"},
    # 保留通配符规则作为后备
    "workers.tasks.*": {"queue": "default"},
    ...
}
```

### 3. 配置文件路径修复 ✅

**问题：**
- `backend/app/config.py` 中硬编码了本地开发路径
- Docker 容器中路径不存在，导致配置加载失败

**解决方案：**
- 使用相对路径 `.env`，适用于本地和 Docker 环境
- Docker 通过环境变量注入配置，不依赖 .env 文件

**文件修改：**
```python
# backend/app/config.py
model_config = SettingsConfigDict(
    env_file=".env",  # 简化为相对路径
    env_file_encoding="utf-8",
    case_sensitive=False,
    extra="ignore",
)
```

### 4. 新增文件

#### `.env.docker.example`
- Docker 部署专用的环境变量模板
- 包含所有必需的配置项
- 无 Redis 密码配置

#### `docker-compose.prod.yml`
- 生产环境专用配置
- 移除开发模式的卷挂载
- 添加资源限制
- 使用生产构建

#### `DEPLOYMENT.md`
- 完整的 Docker 部署指南
- 包含故障排查、性能优化等

#### `docker-test.sh`
- 自动化测试脚本
- 验证 Docker 部署是否正常

## 📋 部署前检查清单

### 必需配置
- [ ] `DASHSCOPE_API_KEY` - 阿里云百炼 API Key
- [ ] `OSS_ACCESS_KEY_ID` - 阿里云 OSS Access Key
- [ ] `OSS_ACCESS_KEY_SECRET` - 阿里云 OSS Secret Key
- [ ] `OSS_BUCKET` - OSS 存储桶名称
- [ ] `OSS_ENDPOINT` - OSS 区域端点
- [ ] `OSS_PUBLIC_DOMAIN` - OSS 公网访问域名

### 可选配置
- [ ] `DB_PASSWORD` - 数据库密码（生产环境建议修改）
- [ ] `WORKER_CONCURRENCY` - Worker 并发数（根据 CPU 调整）
- [ ] `CORS_ORIGINS` - 跨域配置（根据实际域名配置）

## 🚀 快速部署

### 开发环境

```bash
# 1. 复制环境变量
cp .env.docker.example .env

# 2. 编辑配置（填写必需的 API Key 等）
vi .env

# 3. 启动服务
docker-compose -f docker-compose.v2.yml up -d

# 4. 初始化数据库
docker-compose -f docker-compose.v2.yml exec api alembic upgrade head

# 5. 验证
./docker-test.sh
```

### 生产环境

```bash
# 1. 复制环境变量
cp .env.docker.example .env

# 2. 配置生产环境参数
vi .env
# - 修改数据库密码
# - 配置 CORS 为实际域名
# - 调整 Worker 并发数

# 3. 构建并启动
docker-compose -f docker-compose.prod.yml up -d --build

# 4. 初始化数据库
docker-compose -f docker-compose.prod.yml exec api alembic upgrade head

# 5. 配置 HTTPS（推荐使用 Let's Encrypt）
```

## 🔍 验证部署

### 1. 检查服务状态

```bash
docker-compose -f docker-compose.v2.yml ps
```

所有服务应显示 "Up" 状态。

### 2. 检查日志

```bash
# 所有服务
docker-compose -f docker-compose.v2.yml logs -f

# 特定服务
docker-compose -f docker-compose.v2.yml logs -f worker
```

无错误信息。

### 3. 访问测试

- 前端: http://localhost:3000
- 后端: http://localhost:8000
- API 文档: http://localhost:8000/api/v1/docs

### 4. 功能测试

1. 上传视频
2. 检查任务状态（不应卡在"等待处理"）
3. 查看 Worker 日志确认任务执行
4. 等待处理完成
5. 下载结果

## 📊 关键改进对比

| 项目 | 修复前 | 修复后 |
|------|--------|--------|
| Redis 配置 | 有密码，配置不一致 | 无密码，统一配置 |
| Celery 队列 | 未指定，任务丢失 | 明确指定4个队列 |
| 任务路由 | 模式不匹配 | 精确匹配任务名 |
| 配置路径 | 硬编码本地路径 | 相对路径，通用 |
| 部署文档 | 无 | 完整的部署指南 |
| 测试脚本 | 无 | 自动化测试 |

## ⚠️ 注意事项

1. **Redis 密码**
   - 当前配置为无密码，适合内网部署
   - 如需公网暴露，建议配置密码并更新所有配置

2. **数据库密码**
   - 默认密码仅用于开发
   - 生产环境务必修改为强密码

3. **OSS 配置**
   - 确保 OSS 存储桶已创建
   - 配置正确的区域端点
   - 验证 Access Key 权限

4. **CORS 配置**
   - 开发环境使用 localhost
   - 生产环境配置实际域名

5. **资源限制**
   - 生产环境配置了 CPU 和内存限制
   - 根据实际情况调整

## 🐛 常见问题

### Q: 任务卡在"等待处理"

**A:** 检查 Worker 是否监听了正确的队列：

```bash
docker-compose exec worker celery -A app.workers.celery_app inspect active_queues
```

应该显示 `default`, `media`, `ai`, `celery` 四个队列。

### Q: Redis 连接失败

**A:** 检查 Redis 是否正常运行：

```bash
docker-compose exec redis redis-cli ping
```

应返回 `PONG`。

### Q: 数据库连接失败

**A:** 检查数据库环境变量是否一致：

```bash
docker-compose exec api env | grep DB_
docker-compose exec db env | grep POSTGRES_
```

## 📚 相关文档

- [DEPLOYMENT.md](./DEPLOYMENT.md) - 完整部署指南
- [README.md](./README.md) - 项目说明
- [docker-compose.v2.yml](./docker-compose.v2.yml) - 开发环境配置
- [docker-compose.prod.yml](./docker-compose.prod.yml) - 生产环境配置
