# 部署指南

## 📋 系统要求

### 软件依赖

- **Python**: 3.11+
- **PostgreSQL**: 14+
- **Redis**: 6.2+
- **FFmpeg**: 4.4+
- **系统**: Linux/macOS

### 硬件要求

- **CPU**: 4核+
- **内存**: 8GB+
- **存储**: 100GB+（取决于视频数量）

---

## 🚀 快速开始

### 1. 克隆项目

```bash
git clone <repository-url>
cd video-auto-dubbing/backend
```

### 2. 创建虚拟环境

```bash
python3.11 -m venv venv
source venv/bin/activate  # Linux/macOS
# venv\Scripts\activate   # Windows
```

### 3. 安装依赖

```bash
pip install --upgrade pip
pip install -r requirements.txt
```

### 4. 安装 FFmpeg

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install ffmpeg
```

**macOS:**
```bash
brew install ffmpeg
```

**验证安装:**
```bash
ffmpeg -version
```

### 5. 配置环境变量

```bash
cp .env.example .env
vim .env
```

**必需配置:**
```bash
# 数据库
DB_HOST=localhost
DB_PORT=5432
DB_NAME=dubbing
DB_USER=dubbing
DB_PASSWORD=your-secure-password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# 阿里云 DashScope API
DASHSCOPE_API_KEY=sk-your-api-key

# OSS 存储
OSS_ACCESS_KEY_ID=your-access-key
OSS_ACCESS_KEY_SECRET=your-secret-key
OSS_BUCKET_NAME=your-bucket
OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
```

### 6. 初始化数据库

```bash
# 创建数据库
createdb dubbing

# 运行迁移
alembic upgrade head
```

### 7. 启动服务

**开发环境（推荐使用 tmux）:**
```bash
./dev.sh
```

**或手动启动:**

```bash
# Terminal 1: FastAPI 服务器
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000

# Terminal 2: Celery Worker
./run_worker.sh
```

### 8. 验证部署

访问健康检查接口：
```bash
curl http://localhost:8000/api/v1/monitoring/health
```

预期响应：
```json
{
  "status": "healthy",
  "services": {
    "database": true,
    "redis": true,
    "ffmpeg": true
  },
  "version": "2.0.0"
}
```

---

## 🐳 Docker 部署

### 使用 Docker Compose

```bash
# 构建镜像
docker-compose build

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### Docker Compose 配置示例

```yaml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8000:8000"
    environment:
      - DATABASE_URL=postgresql://dubbing:password@db:5432/dubbing
      - REDIS_URL=redis://redis:6379/0
    depends_on:
      - db
      - redis

  worker:
    build: .
    command: celery -A app.workers.celery_app worker --loglevel=info
    environment:
      - DATABASE_URL=postgresql://dubbing:password@db:5432/dubbing
      - REDIS_URL=redis://redis:6379/0
    depends_on:
      - db
      - redis

  db:
    image: postgres:14
    environment:
      - POSTGRES_DB=dubbing
      - POSTGRES_USER=dubbing
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:6.2
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

---

## 🔧 生产环境配置

### 1. 使用 Gunicorn

```bash
# 安装 Gunicorn
pip install gunicorn

# 启动（4 个 worker 进程）
gunicorn app.main:app \
  --workers 4 \
  --worker-class uvicorn.workers.UvicornWorker \
  --bind 0.0.0.0:8000 \
  --log-level info \
  --access-logfile logs/access.log \
  --error-logfile logs/error.log
```

### 2. 使用 Supervisor 管理进程

**supervisor.conf:**
```ini
[program:dubbing-api]
command=/path/to/venv/bin/gunicorn app.main:app --workers 4 --worker-class uvicorn.workers.UvicornWorker --bind 0.0.0.0:8000
directory=/path/to/backend
user=www-data
autostart=true
autorestart=true
stdout_logfile=/var/log/dubbing/api.log
stderr_logfile=/var/log/dubbing/api_error.log

[program:dubbing-worker]
command=/path/to/venv/bin/celery -A app.workers.celery_app worker --loglevel=info --concurrency=4
directory=/path/to/backend
user=www-data
autostart=true
autorestart=true
stdout_logfile=/var/log/dubbing/worker.log
stderr_logfile=/var/log/dubbing/worker_error.log
```

启动：
```bash
supervisorctl reread
supervisorctl update
supervisorctl start dubbing-api dubbing-worker
```

### 3. Nginx 反向代理

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket 支持（可选）
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # 文件上传大小限制
        client_max_body_size 500M;
    }

    # 静态文件（如果有）
    location /static/ {
        alias /path/to/backend/static/;
    }
}
```

---

## 📊 监控和日志

### 1. 健康检查

```bash
# 系统健康
curl http://localhost:8000/api/v1/monitoring/health

# 系统统计
curl http://localhost:8000/api/v1/monitoring/stats

# Celery 状态
curl http://localhost:8000/api/v1/monitoring/celery/inspect
```

### 2. 日志查看

```bash
# API 日志
tail -f logs/app.log

# Worker 日志
tail -f logs/worker.log

# 数据库日志
sudo tail -f /var/log/postgresql/postgresql-14-main.log
```

### 3. Flower（Celery 监控）

```bash
# 安装 Flower
pip install flower

# 启动
celery -A app.workers.celery_app flower --port=5555

# 访问 http://localhost:5555
```

---

## 🔐 安全建议

1. **使用强密码**: 数据库、Redis、API Key
2. **启用 HTTPS**: 生产环境必须使用 SSL
3. **限制 CORS**: 只允许可信域名
4. **API 认证**: 添加 JWT 或 OAuth2
5. **定期备份**: 数据库和 OSS
6. **更新依赖**: 定期更新安全补丁

---

## 🐛 故障排查

### 问题 1: FFmpeg 未找到

```bash
# 检查安装
which ffmpeg

# Ubuntu
sudo apt install ffmpeg

# macOS
brew install ffmpeg
```

### 问题 2: 数据库连接失败

```bash
# 检查 PostgreSQL 状态
sudo systemctl status postgresql

# 检查连接
psql -h localhost -U dubbing -d dubbing

# 检查配置
cat .env | grep DB_
```

### 问题 3: Celery Worker 不工作

```bash
# 检查 Redis
redis-cli ping

# 检查 Worker 日志
tail -f logs/worker.log

# 手动启动 Worker（调试模式）
celery -A app.workers.celery_app worker --loglevel=debug
```

### 问题 4: OSS 上传失败

```bash
# 检查配置
echo $OSS_ACCESS_KEY_ID
echo $OSS_BUCKET_NAME

# 测试连接
python -c "from app.integrations.oss import get_oss_client; client = get_oss_client(); print(client.list_files())"
```

---

## 📞 技术支持

- **文档**: `/docs` (API 文档)
- **健康检查**: `/api/v1/monitoring/health`
- **日志**: `logs/app.log`

---

## 🔄 更新和迁移

### 更新代码

```bash
git pull origin main
pip install -r requirements.txt
alembic upgrade head
supervisorctl restart dubbing-api dubbing-worker
```

### 数据库迁移

```bash
# 创建迁移
alembic revision --autogenerate -m "Description"

# 查看待执行的迁移
alembic current
alembic history

# 执行迁移
alembic upgrade head

# 回滚
alembic downgrade -1
```

---

## 🎯 性能优化

1. **Worker 并发数**: 根据 CPU 核心数调整 `--concurrency`
2. **数据库连接池**: 调整 `pool_size` 和 `max_overflow`
3. **Redis 持久化**: 生产环境启用 AOF
4. **FFmpeg 优化**: 使用硬件加速（如果支持）
5. **OSS CDN**: 为下载链接配置 CDN

---

祝部署顺利！🚀
