# API 接口规范

## 基础信息

- **Base URL**: `http://api:8080/api/v1`
- **Content-Type**: `application/json`
- **响应格式**: JSON

## 通用响应格式

### 成功响应
```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 错误响应
```json
{
  "code": 1001,
  "message": "error message",
  "data": null
}
```

### 错误码定义

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 任务不存在 |
| 1003 | 文件上传失败 |
| 1004 | 内部服务错误 |
| 1005 | 任务处理失败 |

## API 接口列表

### 1. 创建任务（上传视频）

**接口**: `POST /tasks`

**请求**:
- Content-Type: `multipart/form-data`
- Body:
  - `video`: 视频文件（必填）
  - `source_language`: 源语言代码（可选，默认 `zh`）
  - `target_language`: 目标语言代码（可选，默认 `en`）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "created",
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

### 2. 查询任务状态

**接口**: `GET /tasks/:task_id`

**路径参数**:
- `task_id`: 任务 ID（UUID）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "running",
    "progress": 60,
    "source_language": "zh",
    "target_language": "en",
    "error": null,
    "created_at": "2024-01-01T12:00:00Z",
    "updated_at": "2024-01-01T12:05:00Z",
    "steps": [
      {
        "step": "extract_audio",
        "status": "succeeded",
        "started_at": "2024-01-01T12:00:10Z",
        "ended_at": "2024-01-01T12:00:15Z"
      },
      {
        "step": "asr",
        "status": "succeeded",
        "started_at": "2024-01-01T12:00:16Z",
        "ended_at": "2024-01-01T12:01:00Z"
      },
      {
        "step": "translate",
        "status": "succeeded",
        "started_at": "2024-01-01T12:01:01Z",
        "ended_at": "2024-01-01T12:02:00Z"
      },
      {
        "step": "tts",
        "status": "running",
        "started_at": "2024-01-01T12:02:01Z",
        "ended_at": null
      },
      {
        "step": "mux_video",
        "status": "pending",
        "started_at": null,
        "ended_at": null
      }
    ]
  }
}
```

### 3. 获取任务结果

**接口**: `GET /tasks/:task_id/result`

**路径参数**:
- `task_id`: 任务 ID（UUID）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "done",
    "output_video_url": "http://minio.example.com:9000/videos/outputs/550e8400-e29b-41d4-a716-446655440000/final.mp4?X-Amz-Algorithm=...",
    "segments": [
      {
        "idx": 0,
        "start_ms": 0,
        "end_ms": 1500,
        "src_text": "你好，世界",
        "mt_text": "Hello, world",
        "tts_audio_url": "http://minio.example.com:9000/videos/tts/550e8400-e29b-41d4-a716-446655440000/segment_0.wav?X-Amz-Algorithm=..."
      }
    ],
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

**错误响应**（任务未完成）:
```json
{
  "code": 1002,
  "message": "task not completed",
  "data": null
}
```

### 4. 获取下载链接

**接口**: `GET /tasks/:task_id/download`

**路径参数**:
- `task_id`: 任务 ID（UUID）

**查询参数**:
- `type`: 下载类型（可选）
  - `video`: 最终视频（默认）
  - `subtitle`: 字幕文件
  - `audio`: TTS 音频

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "download_url": "http://minio.example.com:9000/videos/outputs/550e8400-e29b-41d4-a716-446655440000/final.mp4?X-Amz-Algorithm=...",
    "expires_in": 3600
  }
}
```

### 5. 列出任务列表

**接口**: `GET /tasks`

**查询参数**:
- `status`: 任务状态过滤（可选）
- `page`: 页码（可选，默认 1）
- `page_size`: 每页数量（可选，默认 20）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "tasks": [
      {
        "task_id": "550e8400-e29b-41d4-a716-446655440000",
        "status": "done",
        "progress": 100,
        "created_at": "2024-01-01T12:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 6. 删除任务

**接口**: `DELETE /tasks/:task_id`

**路径参数**:
- `task_id`: 任务 ID（UUID）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

## 语言代码规范

采用 ISO 639-1 标准：

| 代码 | 语言 |
|------|------|
| `zh` | 中文 |
| `en` | 英语 |
| `ja` | 日语 |
| `ko` | 韩语 |
| `es` | 西班牙语 |
| `fr` | 法语 |
| `de` | 德语 |

