# Lean TTS Service

> **精简版TTS服务** - 用于远程服务器部署，体积减少95%

## 概述

这是原TTS服务的精简版本，移除了庞大的IndexTTS2推理代码（1.9MB），改为使用：
- 外部`index-tts-vllm`包（推荐）
- HTTP API调用模式
- 轻量级依赖

## 与原版对比

| 对比项 | 原版 tts_service | 精简版 tts_service_lean |
|--------|------------------|------------------------|
| 大小 | 2.0MB | ~0.1MB |
| 文件数 | 140个Python文件 | ~15个文件 |
| 依赖复杂度 | 重型ML依赖 | 轻量化依赖 |
| 推理方式 | 内置IndexTTS2 | 外部index-tts-vllm |

## 部署方式

### 方式一：使用 index-tts-vllm (推荐)

```bash
# 1. 克隆项目
git clone https://github.com/Ksuriuri/index-tts-vllm.git
cd index-tts-vllm

# 2. 安装依赖
pip install -r requirements.txt

# 3. 下载模型
modelscope download IndexTeam/IndexTTS-2 --local_dir checkpoints/IndexTTS-2

# 4. 启动服务
python api_server_v2.py --model_dir checkpoints/IndexTTS-2 --port 8000
```

### 方式二：Docker部署精简版

```bash
# 从项目根目录
cd tts_service_lean

# 构建镜像
docker build -t tts-service-lean .

# 启动服务
docker run -d -p 8000:8000 \\
  -e TTS_REMOTE_URL=http://your-index-tts-vllm-server:8000 \\
  tts-service-lean
```

## 配置说明

### 环境变量

```bash
# 远程TTS服务地址（如果使用HTTP API模式）
TTS_REMOTE_URL=http://localhost:8000

# API密钥（可选）
TTS_API_KEY=your_api_key

# 服务端口
TTS_PORT=8000
TTS_HOST=0.0.0.0
```

## API接口

与原版完全兼容，支持以下端点：

- `POST /synthesize` - 主要合成接口
- `GET /health` - 健康检查
- `GET /models` - 可用模型列表

### 请求示例

```bash
curl -X POST http://localhost:8000/synthesize \\
  -H "Content-Type: application/json" \\
  -d '{
    "text": "你好世界",
    "speaker_id": "default",
    "language": "zh",
    "target_duration_ms": 2000
  }'
```

## 自动端点发现

精简版会自动尝试以下API端点：
1. `/synthesize` - 原生接口
2. `/tts` - 简化接口
3. `/api/synthesize` - API版本
4. `/audio/speech` - OpenAI兼容接口
5. `/v1/audio/speech` - OpenAI v1接口

## 优势

1. **轻量化**：体积减少95%，适合容器部署
2. **灵活性**：支持多种后端（本地/远程/HTTP）
3. **兼容性**：API接口完全兼容原版
4. **维护性**：依赖简单，更容易维护和升级

## 迁移指南

从原版tts_service迁移：

1. 停用原版容器
2. 部署index-tts-vllm服务
3. 启动精简版TTS服务
4. 更新配置指向新服务

## 故障排查

### 常见问题

1. **连接失败**：检查TTS_REMOTE_URL配置
2. **端点不存在**：确认远程服务支持的API格式
3. **模型未加载**：检查index-tts-vllm服务状态

### 日志查看

```bash
# Docker日志
docker logs tts-service-lean

# 直接运行日志
uvicorn app.main:app --host 0.0.0.0 --port 8000 --log-level debug
```