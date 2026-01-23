# 快速开始指南

> **重构版本** - API调用架构，无需GPU和本地模型

## 系统要求

- **Docker Engine** 20.10+
- **Docker Compose** 2.0+ (或v1兼容)
- **内存**: 8GB+
- **存储**: 5GB (无需大模型存储)
- **GPU**: 不需要

## 外部服务依赖

在启动前，需要配置以下外部API服务：

### 1. 火山引擎ASR API
- 注册账号：https://console.volcengine.com/speech/
- 获取APP ID和Access Token

### 2. GLM翻译API
- 注册账号：https://open.bigmodel.cn/
- 获取API Key

### 3. 远程TTS服务
- **推荐**: 部署index-tts-vllm到远程GPU服务器
- **替代**: 使用支持OpenAI格式的TTS服务

## 快速启动

### 步骤1：克隆项目
```bash
git clone <repository-url>
cd video-auto-dubbing
```

### 步骤2：配置环境变量
```bash
cp .env.example .env
# 编辑 .env 文件，填入真实的API凭证
```

**必需配置项**：
```bash
# 火山引擎ASR
VOLCENGINE_ASR_APP_KEY=your_app_key
VOLCENGINE_ASR_ACCESS_KEY=your_access_key

# GLM翻译
GLM_API_KEY=your_glm_key

# 远程TTS服务
TTS_SERVICE_URL=https://your-tts-server.com:8000
```

### 步骤3：启动服务
```bash
# 启动基础设施和API服务
docker-compose up -d

# 查看状态
docker-compose ps
```

### 步骤4：验证部署
```bash
# API健康检查
curl http://localhost:8080/health

# 查看前端
open http://localhost
```

## 部署远程TTS服务

### AutoDL部署（推荐）
```bash
# 1. 创建AutoDL GPU实例（RTX 4090）
# 2. SSH连接到实例

# 3. 克隆和安装index-tts-vllm
git clone https://github.com/Ksuriuri/index-tts-vllm.git
cd index-tts-vllm
pip install -r requirements.txt

# 4. 下载模型
modelscope download IndexTeam/IndexTTS-2 --local_dir checkpoints/IndexTTS-2-vLLM

# 5. ⚠️ 重要：替换为本项目优化的服务端文件
# 从本项目根目录复制 api_server_v2.py 到 index-tts-vllm 目录
# 或直接下载：
# wget https://raw.githubusercontent.com/<your-repo>/main/api_server_v2.py

# 6. 启动服务（使用优化版本）
python api_server_v2.py \
  --model_dir checkpoints/IndexTTS-2-vLLM \
  --port 6006 \
  --is_fp16 \
  --gpu_memory_utilization 0.25 \
  --qwenemo_gpu_memory_utilization 0.10

# 7. 配置端口映射（AutoDL控制台）
# 容器端口6006 -> 公网端口（启用HTTPS）
```

### 本地GPU部署
```bash
# 如有本地GPU，可直接部署TTS服务
cd tts_service
pip install -e .
uvicorn app.main:app --host 0.0.0.0 --port 8000
```

## 使用示例

### 上传视频进行配音
```bash
curl -X POST http://localhost:8080/api/v1/tasks \\
  -F "video=@test.mp4" \\
  -F "source_language=zh" \\
  -F "target_language=en"
```

### 查询任务状态
```bash
curl http://localhost:8080/api/v1/tasks/{task_id}
```

### 下载结果
```bash
curl -O http://localhost:8080/api/v1/tasks/{task_id}/result
```

## 故障排查

### 常见问题

1. **API服务无法启动**
   ```bash
   # 检查依赖服务状态
   docker-compose logs db minio rabbitmq
   ```

2. **ASR识别失败**
   - 检查火山引擎API凭证
   - 确认账号余额和权限

3. **翻译失败**
   - 检查GLM API Key是否正确
   - 确认API限流设置

4. **TTS合成失败**
   - 检查远程TTS服务连通性
   - 验证TTS_SERVICE_URL配置

### 日志查看
```bash
# API服务日志
docker-compose logs api

# Worker服务日志
docker-compose logs worker

# 全部日志
docker-compose logs -f
```

## 性能优化

### 水平扩展
```bash
# 扩展多个worker处理并发任务
docker-compose up -d --scale worker=3
```

### TTS并发优化
```bash
# 在.env中调整TTS并发设置
TTS_BATCH_SIZE=30
TTS_MAX_CONCURRENCY=6
```

## 下一步

- 查看[API文档](api-contracts.md)了解详细接口
- 阅读[架构概览](ARCHITECTURE_OVERVIEW.md)了解系统设计
- 参考[部署指南](deployment-guide.md)进行生产环境配置