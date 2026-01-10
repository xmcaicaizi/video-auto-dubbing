# ASR 服务规范（Moonshine）

## 1. 简介
ASR 服务通过 Moonshine ONNX 模型提供语音识别能力，返回句级时间戳（segment）。

## 2. 服务地址
- **生产环境**: `http://asr_service:8002`
- **本地开发**: `http://localhost:8002`

## 3. 接口定义

### 3.1 健康检查
`GET /health`

**响应示例**
```json
{
  "status": "healthy",
  "model_loaded": true
}
```

### 3.2 语音识别
`POST /transcribe`

**请求参数**
```json
{
  "audio_url": "http://minio.example.com:9000/videos/<object_key>?X-Amz-Algorithm=...",
  "language": "zh"
}
```

**响应示例**
```json
{
  "language": "zh",
  "duration_ms": 5321,
  "segments": [
    {
      "start_ms": 0,
      "end_ms": 1234,
      "text": "你好"
    }
  ]
}
```

## 4. 环境变量
- `ASR_MODEL_ID`: Moonshine 模型 ID（默认 `moonshine/tiny`）
- `ASR_DEVICE`: 运行设备（默认 `cpu`）
- `ASR_COMPUTE_TYPE`: 推理精度（默认 `float16`）
- `ASR_BACKEND`: 后端模式（`moonshine_onnx`）
- `HF_ENDPOINT`: HuggingFace 访问地址（可选，国内可配置镜像）
- `ASR_HF_HOME`: HuggingFace home（建议指向 `/app/models` 持久化卷）
- `ASR_HF_HUB_CACHE`: HuggingFace hub cache（建议指向 `/app/models` 持久化卷）

## 5. 依赖说明
- 识别依赖 `useful-moonshine-onnx`（从官方仓库的 `moonshine-onnx` 子目录安装）。
