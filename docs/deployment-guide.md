# 部署指南

> **API调用架构版本** - 轻量级部署，无需GPU

## 部署架构概览

```
本地环境                          远程环境
┌─────────────────────┐         ┌──────────────────────────┐
│ Docker Compose      │  HTTP   │ GPU服务器（可选）          │
│ ├─ PostgreSQL       │ ────────▶ ├─ index-tts-vllm服务     │
│ ├─ MinIO           │         │ └─ 其他TTS服务            │
│ ├─ RabbitMQ        │         └──────────────────────────┘
│ ├─ API服务         │
│ ├─ Worker服务      │         ┌──────────────────────────┐
│ └─ NGINX网关       │  HTTPS  │ 云服务API               │
└─────────────────────┘ ────────▶ ├─ 火山引擎ASR           │
                                │ ├─ GLM翻译API           │
                                │ └─ 其他第三方API         │
                                └──────────────────────────┘
```

## 环境分类

### 开发环境
- **特点**: 本地开发，快速迭代
- **资源**: 最小化配置
- **TTS**: 使用远程服务或Mock

### 测试环境
- **特点**: CI/CD集成，自动化测试
- **资源**: 中等配置，稳定性优先
- **TTS**: Mock服务或共享远程服务

### 生产环境
- **特点**: 高可用，性能优化
- **资源**: 多实例，负载均衡
- **TTS**: 专用远程GPU实例

## 基础设施配置

### 最小配置
```yaml
# 适用于开发/演示环境
resources:
  cpu: 2 cores
  memory: 8GB
  storage: 10GB
  network: 10Mbps

services:
  - api: 1 instance
  - worker: 1 instance
  - db/minio/rabbitmq: default
```

### 推荐配置
```yaml
# 适用于生产环境
resources:
  cpu: 8 cores
  memory: 32GB
  storage: 100GB
  network: 100Mbps

services:
  - api: 2 instances (load balanced)
  - worker: 4-8 instances
  - db: master-slave setup
  - minio: cluster mode
```

## Docker部署

### 单机部署

1. **环境准备**
```bash
# 安装Docker和Compose
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# 验证安装
docker --version
docker-compose --version
```

2. **配置文件**
```bash
# 复制配置模板
cp .env.example .env

# 编辑配置（必需项）
vim .env
```

3. **启动服务**
```bash
# 一键启动
docker-compose up -d

# 验证部署
docker-compose ps
curl http://localhost:8080/health
```

### 多机部署（Docker Swarm）

```bash
# 初始化Swarm集群
docker swarm init

# 部署服务栈
docker stack deploy -c docker-compose.yml video-dubbing

# 扩展服务
docker service scale video-dubbing_worker=4
```

## Kubernetes部署

### 使用Helm Chart

```bash
# 添加Helm仓库
helm repo add video-dubbing ./helm

# 安装
helm install video-dubbing ./helm/video-dubbing \\
  --set api.replicas=2 \\
  --set worker.replicas=4 \\
  --set external.tts.url=https://your-tts-server.com
```

### 手动部署

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - namespace.yaml
  - configmap.yaml
  - secret.yaml
  - postgres.yaml
  - minio.yaml
  - rabbitmq.yaml
  - api.yaml
  - worker.yaml
  - gateway.yaml

images:
  - name: video-dubbing-api
    newTag: v1.0.0
  - name: video-dubbing-worker
    newTag: v1.0.0
```

## 远程TTS服务部署

### AutoDL部署

**1. 创建实例**
- GPU: RTX 4090 24GB
- 镜像: PyTorch 2.1 + CUDA 12.1
- 存储: 50GB+

**2. 部署脚本**
```bash
#!/bin/bash
# autodl_deploy.sh

# 安装依赖
cd /root/autodl-tmp
git clone https://github.com/Ksuriuri/index-tts-vllm.git
cd index-tts-vllm

pip install -r requirements.txt

# ⚠️ 重要：替换为优化版 api_server_v2.py
# 从本项目仓库下载或复制优化版文件
# wget https://raw.githubusercontent.com/<your-repo>/main/api_server_v2.py -O api_server_v2.py

# 下载模型
pip install modelscope
modelscope download IndexTeam/IndexTTS-2 --local_dir checkpoints/IndexTTS-2-vLLM

# 创建systemd服务
sudo tee /etc/systemd/system/tts.service > /dev/null <<EOF
[Unit]
Description=IndexTTS VLLM Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/root/autodl-tmp/index-tts-vllm
Environment=CUDA_VISIBLE_DEVICES=0
ExecStart=/root/miniconda3/bin/python api_server_v2.py \\
  --model_dir checkpoints/IndexTTS-2-vLLM \\
  --port 6006 \\
  --is_fp16 \\
  --gpu_memory_utilization 0.25 \\
  --qwenemo_gpu_memory_utilization 0.10
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl enable tts
sudo systemctl start tts
```

**3. 配置端口映射**
- AutoDL控制台 → 自定义服务 → 添加端口映射
- 容器端口: 6006 → 公网端口（启用HTTPS）

### AWS/阿里云部署

```bash
# EC2/ECS实例配置
instance_type: p3.2xlarge  # 或同等GPU实例
gpu: V100 16GB (最小)
storage: 100GB SSD

# 安全组规则
inbound:
  - port: 8000, source: your-app-server-sg
  - port: 22, source: admin-ip

# 使用Docker部署
docker run -d --gpus all \\
  -p 8000:8000 \\
  -v /data/models:/app/models \\
  index-tts-vllm:latest
```

## 监控与日志

### 监控配置

```yaml
# docker-compose.monitoring.yml
version: '3.8'
services:
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin

  loki:
    image: grafana/loki
    ports:
      - "3100:3100"
```

### 日志收集

```yaml
# 添加到主docker-compose.yml
services:
  api:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "10"

  worker:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "10"
```

## 备份与恢复

### 数据备份

```bash
#!/bin/bash
# backup.sh

BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/$BACKUP_DATE"

mkdir -p $BACKUP_DIR

# 数据库备份
docker-compose exec -T db pg_dump -U dubbing dubbing > $BACKUP_DIR/database.sql

# MinIO数据备份
docker-compose exec -T minio mc mirror /data $BACKUP_DIR/minio --overwrite

# 配置备份
cp .env $BACKUP_DIR/
cp docker-compose.yml $BACKUP_DIR/

echo "备份完成: $BACKUP_DIR"
```

### 数据恢复

```bash
#!/bin/bash
# restore.sh

BACKUP_DIR=$1
if [ -z "$BACKUP_DIR" ]; then
  echo "用法: $0 <backup_directory>"
  exit 1
fi

# 停止服务
docker-compose down

# 恢复数据库
docker-compose up -d db
sleep 10
docker-compose exec -T db psql -U dubbing -d dubbing < $BACKUP_DIR/database.sql

# 恢复MinIO数据
docker-compose up -d minio
sleep 10
docker-compose exec -T minio mc mirror $BACKUP_DIR/minio /data --overwrite

# 重启所有服务
docker-compose up -d

echo "恢复完成"
```

## 性能优化

### 数据库优化

```sql
-- postgresql.conf 调优
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB
maintenance_work_mem = 64MB
max_connections = 200

-- 索引优化
CREATE INDEX CONCURRENTLY idx_tasks_status ON tasks(status);
CREATE INDEX CONCURRENTLY idx_segments_task_id ON segments(task_id);
```

### 缓存策略

```bash
# Redis缓存（可选）
services:
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

# 应用层缓存
# - 翻译结果缓存
# - TTS结果缓存
# - API响应缓存
```

## 安全配置

### SSL/TLS配置

```bash
# Let's Encrypt证书
certbot --nginx -d your-domain.com

# nginx配置
server {
    listen 443 ssl http2;
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    location / {
        proxy_pass http://api:8080;
    }
}
```

### 访问控制

```bash
# API Key认证
services:
  api:
    environment:
      - API_KEY_REQUIRED=true
      - API_KEYS=key1,key2,key3

# IP白名单
# nginx配置
allow 192.168.1.0/24;
allow 10.0.0.0/8;
deny all;
```

## 故障排查

### 常见问题

1. **服务无法启动**
```bash
# 检查端口占用
netstat -tulpn | grep :8080

# 检查磁盘空间
df -h

# 检查内存使用
free -h
```

2. **API调用失败**
```bash
# 检查网络连通性
curl -v https://external-api.com/health

# 检查DNS解析
nslookup external-api.com

# 检查证书
openssl s_client -connect external-api.com:443
```

3. **性能问题**
```bash
# 查看系统负载
top
htop

# 检查数据库性能
docker-compose exec db psql -U dubbing -c "SELECT * FROM pg_stat_activity;"

# 查看慢查询
docker-compose logs api | grep "slow"
```

### 健康检查脚本

```bash
#!/bin/bash
# health_check.sh

echo "=== 服务健康检查 ==="

# 检查容器状态
echo "1. 容器状态:"
docker-compose ps

echo -e "\\n2. API健康检查:"
curl -s http://localhost:8080/health | jq .

echo -e "\\n3. 数据库连接:"
docker-compose exec -T db pg_isready -U dubbing

echo -e "\\n4. 外部API连通性:"
curl -s --max-time 5 https://open.bigmodel.cn/api/paas/v4/health || echo "GLM API不可达"

echo -e "\\n5. 磁盘空间:"
df -h | grep -E "(Filesystem|/dev/)"

echo -e "\\n=== 检查完成 ==="
```

## 扩展指南

### 添加新的TTS服务

```go
// worker/internal/tts/client.go
func NewClient(cfg config.TTSConfig, logger *zap.Logger) Client {
    switch cfg.Backend {
    case "vllm":
        return NewVLLMClient(cfg, logger)
    case "your_new_tts":
        return NewYourTTSClient(cfg, logger) // 新增
    default:
        return NewVLLMClient(cfg, logger)
    }
}
```

### 添加新的ASR服务

```go
// worker/internal/asr/client.go
func NewClient(cfg config.ASRConfig, logger *zap.Logger) Client {
    switch cfg.Provider {
    case "volcengine":
        return NewVolcengineClient(cfg, logger)
    case "your_new_asr":
        return NewYourASRClient(cfg, logger) // 新增
    default:
        return NewVolcengineClient(cfg, logger)
    }
}
```

---

更多部署问题请参考[故障排查指南](troubleshooting.md)或联系技术支持。