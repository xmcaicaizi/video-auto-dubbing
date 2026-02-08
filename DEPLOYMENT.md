# 视频自动配音系统 - Docker 部署指南

## 📋 前置要求

- Docker Engine 20.10+
- Docker Compose 2.0+
- 服务器内存建议 4GB+
- 阿里云 OSS 存储桶
- 阿里云百炼 DashScope API Key

## 🚀 快速部署

### 1. 克隆项目

```bash
git clone <repository-url>
cd video-auto-dubbing
```

### 2. 配置环境变量

```bash
# 复制环境变量模板
cp .env.docker.example .env

# 编辑环境变量，填写以下关键配置：
# - DASHSCOPE_API_KEY: 阿里云百炼 API Key
# - OSS_ACCESS_KEY_ID: 阿里云 OSS Access Key
# - OSS_ACCESS_KEY_SECRET: 阿里云 OSS Secret Key
# - OSS_BUCKET: OSS 存储桶名称
# - OSS_ENDPOINT: OSS 区域端点
vi .env
```

### 3. 启动服务

```bash
# 构建并启动所有服务
docker-compose -f docker-compose.v2.yml up -d

# 查看服务状态
docker-compose -f docker-compose.v2.yml ps

# 查看日志
docker-compose -f docker-compose.v2.yml logs -f
```

### 4. 数据库初始化

```bash
# 运行数据库迁移
docker-compose -f docker-compose.v2.yml exec api alembic upgrade head
```

### 5. 访问服务

- 前端界面: http://your-server-ip:3000
- 后端 API: http://your-server-ip:8000
- API 文档: http://your-server-ip:8000/api/v1/docs

## 🔧 服务管理

### 启动服务

```bash
docker-compose -f docker-compose.v2.yml up -d
```

### 停止服务

```bash
docker-compose -f docker-compose.v2.yml down
```

### 重启服务

```bash
docker-compose -f docker-compose.v2.yml restart
```

### 查看日志

```bash
# 所有服务
docker-compose -f docker-compose.v2.yml logs -f

# 特定服务
docker-compose -f docker-compose.v2.yml logs -f api
docker-compose -f docker-compose.v2.yml logs -f worker
docker-compose -f docker-compose.v2.yml logs -f frontend
```

### 重新构建

```bash
# 重新构建所有服务
docker-compose -f docker-compose.v2.yml build

# 重新构建特定服务
docker-compose -f docker-compose.v2.yml build api
docker-compose -f docker-compose.v2.yml build worker
```

## 📊 服务架构

```
┌─────────────────────────────────────────────┐
│            NGINX Gateway (80/443)           │
└─────────────────┬───────────────────────────┘
                  │
        ┌─────────┴──────────┐
        │                    │
┌───────▼──────┐    ┌───────▼──────┐
│   Frontend   │    │   Backend    │
│   (Next.js)  │    │  (FastAPI)   │
│   Port 3000  │    │  Port 8000   │
└──────────────┘    └───────┬──────┘
                            │
                    ┌───────┴──────┐
                    │              │
            ┌───────▼──────┐ ┌────▼─────┐
            │ Celery Worker│ │  Redis   │
            │ (Background) │ │ (Broker) │
            └──────┬───────┘ └──────────┘
                   │
            ┌──────▼───────┐
            │  PostgreSQL  │
            │  (Database)  │
            └──────────────┘
```

## 🔍 健康检查

### 检查所有服务状态

```bash
docker-compose -f docker-compose.v2.yml ps
```

### 检查数据库连接

```bash
docker-compose -f docker-compose.v2.yml exec db psql -U dubbing -d dubbing -c "SELECT 1;"
```

### 检查 Redis 连接

```bash
docker-compose -f docker-compose.v2.yml exec redis redis-cli ping
```

### 检查 Celery Worker

```bash
docker-compose -f docker-compose.v2.yml exec worker celery -A app.workers.celery_app inspect active
```

## 🐛 故障排查

### 问题：服务无法启动

```bash
# 检查日志
docker-compose -f docker-compose.v2.yml logs

# 检查容器状态
docker-compose -f docker-compose.v2.yml ps -a
```

### 问题：数据库连接失败

```bash
# 检查数据库是否就绪
docker-compose -f docker-compose.v2.yml exec db pg_isready -U dubbing

# 检查环境变量
docker-compose -f docker-compose.v2.yml exec api env | grep DB_
```

### 问题：Celery 任务不执行

```bash
# 检查 Worker 日志
docker-compose -f docker-compose.v2.yml logs -f worker

# 检查 Redis 队列
docker-compose -f docker-compose.v2.yml exec redis redis-cli LLEN celery

# 检查 Worker 状态
docker-compose -f docker-compose.v2.yml exec worker celery -A app.workers.celery_app inspect stats
```

### 问题：前端无法连接后端

```bash
# 检查网络
docker network inspect video-auto-dubbing_dubbing-network

# 检查 CORS 配置
docker-compose -f docker-compose.v2.yml exec api env | grep CORS
```

## 🔐 生产环境配置建议

### 1. 使用强密码

```bash
# 数据库密码
DB_PASSWORD=$(openssl rand -hex 32)

# 应用密钥
SECRET_KEY=$(openssl rand -hex 32)
```

### 2. 启用 HTTPS

- 配置 SSL 证书（Let's Encrypt 推荐）
- 更新 NGINX 配置
- 修改 CORS_ORIGINS 使用 https

### 3. 限制端口暴露

```yaml
# 仅在开发环境暴露所有端口
# 生产环境只暴露 Gateway (80/443)
```

### 4. 配置日志轮转

```bash
# 使用 Docker 日志驱动
# 配置 max-size 和 max-file
```

### 5. 数据备份

```bash
# 定期备份数据库
docker-compose -f docker-compose.v2.yml exec db pg_dump -U dubbing dubbing > backup.sql

# 备份 Redis 数据
docker-compose -f docker-compose.v2.yml exec redis redis-cli SAVE
```

## 📈 性能优化

### 1. 调整 Worker 并发数

```bash
# 根据 CPU 核心数调整
WORKER_CONCURRENCY=8
```

### 2. 配置资源限制

```yaml
# docker-compose.v2.yml
services:
  worker:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
```

### 3. 使用生产级数据库

- 调整 PostgreSQL 配置（shared_buffers, work_mem 等）
- 启用连接池
- 定期 VACUUM

## 🔄 更新部署

```bash
# 拉取最新代码
git pull

# 重新构建并重启
docker-compose -f docker-compose.v2.yml up -d --build

# 运行数据库迁移
docker-compose -f docker-compose.v2.yml exec api alembic upgrade head
```

## 📝 环境变量说明

| 变量名 | 说明 | 必需 | 默认值 |
|--------|------|------|--------|
| DASHSCOPE_API_KEY | 阿里云百炼 API Key | ✅ | - |
| OSS_ACCESS_KEY_ID | 阿里云 OSS Access Key | ✅ | - |
| OSS_ACCESS_KEY_SECRET | 阿里云 OSS Secret Key | ✅ | - |
| OSS_BUCKET | OSS 存储桶名称 | ✅ | - |
| OSS_ENDPOINT | OSS 区域端点 | ✅ | - |
| DB_PASSWORD | 数据库密码 | ⚠️ | dubbing123 |
| WORKER_CONCURRENCY | Worker 并发数 | ❌ | 4 |
| DEBUG | 调试模式 | ❌ | false |

## 🆘 支持

如遇问题，请查看：
- 项目文档: README.md
- 问题追踪: GitHub Issues
- 日志文件: `docker-compose logs`
