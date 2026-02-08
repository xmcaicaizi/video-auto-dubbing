# 本地开发 vs Docker 部署对比

## 📊 配置对比

| 配置项 | 本地开发 | Docker 部署 | 说明 |
|--------|----------|-------------|------|
| **Redis 密码** | 无 | 无 | ✅ 统一配置 |
| **数据库** | 本地 PostgreSQL | Docker PostgreSQL | 端口可能不同 |
| **Python 环境** | uv venv | Docker 镜像 | 隔离环境 |
| **Node 环境** | 本地 Node.js | Docker 镜像 | 隔离环境 |
| **配置文件** | `.env` 相对路径 | 环境变量注入 | Docker 优先使用环境变量 |
| **Celery 队列** | default,media,ai,celery | default,media,ai,celery | ✅ 一致 |
| **日志输出** | 终端/文件 | Docker logs | 通过 docker logs 查看 |

## 🚀 启动命令对比

### 本地开发

```bash
# 后端
cd backend && uv run uvicorn app.main:app --reload --host 0.0.0.0 --port 8000 &

# Celery Worker
cd backend && REDIS_PASSWORD="" uv run celery -A app.workers.celery_app worker \
  --loglevel=info --concurrency=2 --queues=default,media,ai,celery &

# 前端
cd frontend && npm run dev &
```

### Docker 部署

```bash
# 所有服务一键启动
docker-compose -f docker-compose.v2.yml up -d

# 或生产环境
docker-compose -f docker-compose.prod.yml up -d
```

## 🔧 配置文件对比

### 本地开发环境变量 (`.env`)

```bash
# Redis（本地无密码）
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
# REDIS_PASSWORD=  # 注释掉

# 数据库
POSTGRES_DB=dubbing
POSTGRES_USER=dubbing
POSTGRES_PASSWORD=dubbing123
POSTGRES_PORT=5433  # 本地端口可能不同

# 其他配置...
```

### Docker 环境变量 (`.env`)

```bash
# Redis（Docker 内部，无密码）
# 通过 docker-compose 自动配置
# REDIS_HOST=redis
# REDIS_PORT=6379

# 数据库（Docker 内部）
DB_NAME=dubbing
DB_USER=dubbing
DB_PASSWORD=dubbing123
DB_PORT=5432

# 其他配置...
```

## 📁 文件结构对比

### 本地开发

```
video-auto-dubbing/
├── backend/
│   ├── .venv/           # Python 虚拟环境
│   ├── .env -> ../.env  # 符号链接
│   └── ...
├── frontend/
│   ├── node_modules/    # Node 依赖
│   └── ...
├── .env                 # 环境变量
└── celery.log          # Celery 日志
```

### Docker 部署

```
video-auto-dubbing/
├── backend/
│   ├── Dockerfile       # 后端镜像配置
│   └── ...
├── frontend/
│   ├── Dockerfile       # 前端镜像配置
│   └── ...
├── docker-compose.v2.yml     # 开发环境
├── docker-compose.prod.yml   # 生产环境
├── .env                      # 环境变量
├── .dockerignore            # 构建优化
└── DEPLOYMENT.md            # 部署文档
```

## 🔍 访问方式对比

### 本地开发

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端 | http://localhost:3000 | Next.js dev server |
| 后端 | http://localhost:8000 | Uvicorn |
| API 文档 | http://localhost:8000/api/v1/docs | Swagger UI |
| 数据库 | localhost:5433 | 本地 PostgreSQL |
| Redis | localhost:6379 | 本地 Redis |

### Docker 部署

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端 | http://localhost:3000 | Docker 容器 |
| 后端 | http://localhost:8000 | Docker 容器 |
| API 文档 | http://localhost:8000/api/v1/docs | Swagger UI |
| 数据库 | 内部网络 | 不对外暴露 |
| Redis | 内部网络 | 不对外暴露 |

## 🐛 调试对比

### 本地开发

```bash
# 查看后端日志
tail -f backend/logs/app.log

# 查看 Celery 日志
tail -f celery.log

# 查看前端日志
# 终端输出

# 进入 Python REPL
cd backend && uv run python

# 运行测试
cd backend && uv run pytest
```

### Docker 部署

```bash
# 查看后端日志
docker-compose logs -f api

# 查看 Celery 日志
docker-compose logs -f worker

# 查看前端日志
docker-compose logs -f frontend

# 进入容器 Shell
docker-compose exec api bash
docker-compose exec worker python

# 运行测试
docker-compose exec api pytest
```

## 🔄 热重载对比

### 本地开发

- ✅ 后端：`--reload` 自动重载
- ✅ 前端：`npm run dev` 自动重载
- ✅ 修改代码立即生效

### Docker 开发环境 (`docker-compose.v2.yml`)

- ✅ 后端：挂载卷 + `--reload` 自动重载
- ✅ 前端：挂载卷 + dev 模式自动重载
- ✅ 修改代码立即生效

### Docker 生产环境 (`docker-compose.prod.yml`)

- ❌ 不支持热重载
- 需要重新构建镜像：`docker-compose build`

## 💾 数据持久化对比

### 本地开发

- 数据库数据：本地 PostgreSQL data 目录
- Redis 数据：本地 Redis dump.rdb
- 上传文件：OSS（云端）
- 日志：本地文件系统

### Docker 部署

- 数据库数据：Docker volume `postgres_data`
- Redis 数据：Docker volume `redis_data`
- 上传文件：OSS（云端）
- 日志：Docker volume `api_logs`, `worker_logs`

查看卷：
```bash
docker volume ls
docker volume inspect video-auto-dubbing_postgres_data
```

## 🔧 依赖管理对比

### 本地开发

```bash
# Python 依赖
cd backend
uv pip install package-name
uv pip freeze > requirements.txt

# Node 依赖
cd frontend
npm install package-name
npm install
```

### Docker 部署

```bash
# 修改 pyproject.toml 或 package.json
# 重新构建镜像
docker-compose build api
docker-compose build frontend

# 重启服务
docker-compose up -d
```

## 🚨 故障排查对比

### 本地开发

```bash
# 检查进程
ps aux | grep uvicorn
ps aux | grep celery
ps aux | grep next

# 检查端口
lsof -i :8000
lsof -i :3000
lsof -i :6379

# 重启服务
pkill -f uvicorn
pkill -f celery
# 重新启动
```

### Docker 部署

```bash
# 检查容器
docker-compose ps

# 检查日志
docker-compose logs api
docker-compose logs worker

# 检查网络
docker network inspect video-auto-dubbing_dubbing-network

# 重启服务
docker-compose restart api
docker-compose restart worker

# 完全重建
docker-compose down
docker-compose up -d --build
```

## 📊 资源使用对比

### 本地开发

- CPU：直接使用主机资源
- 内存：直接使用主机资源
- 磁盘：直接使用主机磁盘
- 网络：localhost 回环

### Docker 部署

- CPU：可配置限制（`cpus: '2'`）
- 内存：可配置限制（`memory: 2G`）
- 磁盘：Docker 卷
- 网络：Docker 桥接网络

## ✅ 优缺点对比

### 本地开发

**优点：**
- ✅ 启动快速
- ✅ 调试方便
- ✅ 资源占用少
- ✅ 直接使用 IDE 调试器

**缺点：**
- ❌ 环境依赖复杂
- ❌ 不同开发者环境可能不一致
- ❌ 与生产环境差异大

### Docker 部署

**优点：**
- ✅ 环境一致性
- ✅ 一键启动所有服务
- ✅ 隔离性好
- ✅ 接近生产环境
- ✅ 易于部署和扩展

**缺点：**
- ❌ 首次构建较慢
- ❌ 资源占用相对多
- ❌ 调试稍复杂（需进入容器）

## 🎯 推荐使用场景

### 本地开发

- ✅ 日常开发调试
- ✅ 快速原型验证
- ✅ 单元测试
- ✅ IDE 集成开发

### Docker 部署

- ✅ 集成测试
- ✅ 生产部署
- ✅ 多环境部署（开发/测试/生产）
- ✅ 团队协作（统一环境）
- ✅ CI/CD 流程

## 🔄 迁移建议

### 从本地迁移到 Docker

1. 确保本地功能正常
2. 运行配置检查：`./check-config.sh`
3. 测试 Docker 部署：`./docker-test.sh`
4. 验证功能一致性
5. 逐步过渡到 Docker 开发

### 保持两者同步

1. 配置文件统一
   - 使用同一个 `.env` 模板
   - 相同的环境变量命名

2. 依赖同步
   - `pyproject.toml` 与 Dockerfile 一致
   - `package.json` 与 Dockerfile 一致

3. 定期测试
   - 本地测试通过后再 Docker 测试
   - 确保两种环境都能正常运行

---

**总结：** 本地开发和 Docker 部署各有优势，建议日常开发使用本地环境，部署和测试使用 Docker 环境。两种方式的配置已经完全统一，可以无缝切换。
