# 队列消息规范

## RabbitMQ 配置

### Exchange 和 Queue 设计

**Exchange**: `task_exchange` (topic)

**Queues**:
- `task.extract_audio` - 音频提取任务
- `task.asr` - 语音识别任务
- `task.translate` - 翻译任务
- `task.tts` - 语音合成任务
- `task.mux_video` - 视频合成任务
- `task.dlq` - 死信队列

**Routing Keys**:
- `task.extract_audio`
- `task.asr`
- `task.translate`
- `task.tts`
- `task.mux_video`

### 死信队列配置

- 消息被拒绝（reject）且 `requeue=false` 时进入 DLQ
- 消息 TTL 过期后进入 DLQ
- 队列达到最大长度后进入 DLQ

## 通用消息格式

所有任务消息遵循以下通用格式：

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "step": "extract_audio",
  "attempt": 1,
  "trace_id": "trace-123456",
  "created_at": "2024-01-01T12:00:00Z",
  "payload": {}
}
```

**字段说明**:
- `task_id`: 任务 ID（UUID）
- `step`: 步骤名称
- `attempt`: 重试次数（从 1 开始）
- `trace_id`: 追踪 ID，用于日志关联
- `created_at`: 消息创建时间
- `payload`: 步骤特定的负载数据

## 批处理与限流

- Translate 阶段支持批处理，受环境变量 `TRANSLATE_BATCH_SIZE` 控制，批失败会降级为单条重试，单条重试次数由 `TRANSLATE_ITEM_MAX_RETRIES` 控制；文本长度超出 `TRANSLATE_MAX_TEXT_LENGTH` 时会自动分段再翻译。
- TTS 阶段按 `TTS_BATCH_SIZE` 批量拉取 `tts_audio_key` 为空的 segments，按 `TTS_MAX_CONCURRENCY` 控制并发；单段失败按 `TTS_MAX_RETRIES` 与 `TTS_RETRY_DELAY_SECONDS` 重试，仍失败会写入补偿队列 `task.tts_compensation`。
- 幂等：TTS 消费前会查询 `segments.tts_audio_key`，已存在则跳过，避免重复合成阻塞后续 mux。
- 超时：每个步骤可通过 `TIMEOUT_EXTRACT_AUDIO_SECONDS` / `TIMEOUT_ASR_SECONDS` / `TIMEOUT_TTS_SECONDS` / `TIMEOUT_MUX_SECONDS` 设置处理超时，Worker 会在日志中打印当前超时时间。

## 各步骤消息 Schema

### 1. extract_audio

**Routing Key**: `task.extract_audio`

**Payload**:
```json
{
  "source_video_key": "videos/550e8400-e29b-41d4-a716-446655440000/source.mp4",
  "output_audio_key": "audios/550e8400-e29b-41d4-a716-446655440000/source.wav"
}
```

**处理逻辑**:
- 使用 ffmpeg 从视频提取音频
- 保存到 MinIO
- 成功后投递 `task.asr` 消息

### 2. asr

**Routing Key**: `task.asr`

**Payload**:
```json
{
  "audio_key": "audios/550e8400-e29b-41d4-a716-446655440000/source.wav",
  "language": "zh",
  "output_key": "asr/550e8400-e29b-41d4-a716-446655440000/asr.json"
}
```

**处理逻辑**:
- 调用 Moonshine ASR 服务
- 保存识别结果（含时间戳）到 MinIO
- 解析结果并写入 `segments` 表
- 成功后投递 `task.translate` 消息（每个 segment 一条消息，或批量处理）

### 3. translate

**Routing Key**: `task.translate`

**Payload**:
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "segment_ids": ["seg-0", "seg-1"],
  "source_language": "zh",
  "target_language": "en",
  "batch_size": 20
}
```

**处理逻辑**:
- 从数据库按 idx 读取 `mt_text` 为空的 segments（`src_text` 为输入）
- 按 `batch_size`（或 `TRANSLATE_BATCH_SIZE` 环境变量）调用 GLM 翻译 API，批失败降级为单条重试，超长文本按 `TRANSLATE_MAX_TEXT_LENGTH` 拆分重试
- 单条重试次数：`TRANSLATE_ITEM_MAX_RETRIES`
- 更新 segments 表的 `mt_text`
- 所有 segments 翻译完成后，投递单条 `task.tts` 消息，TTS 阶段会自行按批消费

**批量处理优化**:
- 可以一次翻译多个 segments，减少 API 调用
- 如果 segment 数量多，可以分批投递多个 translate 消息

### 4. tts

**Routing Key**: `task.tts`

**Payload**:
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "batch_size": 20,
  "max_concurrency": 4,
  "max_retries": 3,
  "retry_delay_sec": 2.0,
  "speaker_id": "default"
}
```

**处理逻辑**:
- 从数据库批量获取 `tts_audio_key` 为空的 segments，按照 `batch_size` 控制批大小
- 并发数由 `max_concurrency`（或 `TTS_MAX_CONCURRENCY`）限制；单段失败按 `max_retries` / `retry_delay_sec` 重试
- 调用 tts_service 的 `/synthesize` 接口；若目标段已有 `tts_audio_key` 则跳过（幂等）
- 保存生成的音频到 MinIO，更新 segments 表的 `tts_audio_key` 和 `tts_params_json`
- 所有 segments 的 TTS 完成后，投递 `task.mux_video` 消息；批内失败超过重试次数会把 segment 写入 `task.tts_compensation` 队列

**并发控制**:
- 使用 `TTS_MAX_CONCURRENCY` 控制批内并发，结合 `TTS_MAX_RETRIES` 与 `TTS_RETRY_DELAY_SECONDS` 减少长视频场景的积压

### 5. mux_video

**Routing Key**: `task.mux_video`

**Payload**:
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "source_video_key": "videos/550e8400-e29b-41d4-a716-446655440000/source.mp4",
  "tts_audio_key": "tts/550e8400-e29b-41d4-a716-446655440000/dub.wav",
  "output_video_key": "outputs/550e8400-e29b-41d4-a716-446655440000/final.mp4"
}
```

**处理逻辑**:
- 先合并所有 segment 的 TTS 音频（如果还没有合并）
- 使用 ffmpeg 将 TTS 音频替换原视频音频
- 保存最终视频到 MinIO
- 更新 task 状态为 `done`

## 幂等性策略

### 实现方式

1. **数据库检查**:
   - Worker 处理消息前，先查询 `task_steps` 表
   - 如果该步骤状态为 `succeeded`，直接 ack 消息，跳过处理
   - 如果状态为 `running` 且时间戳较新，可能是重复消息，ack 并跳过

2. **唯一约束**:
   - `task_steps` 表有 `UNIQUE(task_id, step, attempt)` 约束
   - 防止重复插入

3. **消息去重**:
   - 使用 `trace_id` 记录处理历史
   - 相同 `trace_id` 的消息视为重复

### 幂等性检查伪代码

```go
func processMessage(msg TaskMessage) error {
    // 检查步骤是否已完成
    step, err := db.GetTaskStep(msg.TaskID, msg.Step)
    if err == nil && step.Status == "succeeded" {
        log.Info("Step already succeeded, skipping", "task_id", msg.TaskID, "step", msg.Step)
        return nil // ack 消息
    }
    
    // 检查是否正在处理（防止并发）
    if step.Status == "running" && time.Since(step.StartedAt) < 5*time.Minute {
        log.Warn("Step is running, may be duplicate message", "task_id", msg.TaskID)
        return nil // ack 消息
    }
    
    // 执行处理逻辑
    return doProcess(msg)
}
```

## 重试策略

### 重试机制

1. **自动重试**:
   - 最大重试次数: 3 次
   - 重试间隔: 指数退避（1s, 2s, 4s）
   - 使用 RabbitMQ 的延迟队列或消息 TTL 实现

2. **重试条件**:
   - 临时错误（网络超时、服务暂时不可用）
   - 可恢复的错误（API 限流、资源暂时不足）

3. **不重试条件**:
   - 永久错误（参数错误、文件不存在）
   - 达到最大重试次数

### 重试实现

**方案 1: 延迟队列（推荐）**

```go
// 失败后投递到延迟队列
func retryMessage(msg TaskMessage, delay time.Duration) error {
    msg.Attempt++
    if msg.Attempt > 3 {
        // 投递到死信队列
        return publishToDLQ(msg)
    }
    
    // 投递到延迟队列
    return publishWithDelay(msg, delay)
}
```

**方案 2: 消息 TTL + 死信队列**

- 设置消息 TTL
- TTL 过期后自动进入死信队列
- 从死信队列消费，检查重试次数，决定是否重新投递

### 死信队列处理

**DLQ 消息格式**:
```json
{
  "original_message": {...},
  "failed_at": "2024-01-01T12:00:00Z",
  "failure_reason": "API timeout",
  "attempt": 3
}
```

**处理方式**:
- 记录到数据库（task.error 字段）
- 发送告警通知
- 提供手动重试接口

### 补偿 / 死信处理（TTS）

- 路由：`task.tts_compensation`，写入当某个 segment 在 TTS 重试后仍失败。
- 典型负载：
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "segment_idx": 3,
  "error": "API timeout",
  "created_at": "2024-01-01T12:00:00Z",
  "routing_key": "task.tts"
}
```
- 运维补偿：可运行 `go run ./worker/cmd/tts_requeue -limit 100` 扫描 `tts_audio_key` 为空的 segment 并重新投递 `task.tts`，也可使用定时任务执行。

## 消息确认机制

### Ack 策略

- **成功**: 处理完成后 `ack` 消息
- **失败（可重试）**: `nack` 且 `requeue=false`，投递到延迟队列
- **失败（不可重试）**: `nack` 且 `requeue=false`，投递到 DLQ

### 预取设置

```go
// 每个 worker 同时处理的消息数
channel.Qos(1, 0, false)
```

这样可以确保：
- 消息按顺序处理（对于同一任务）
- 负载均衡（多个 worker 时）

## 任务依赖关系

### DAG 流程

```
extract_audio -> asr -> translate -> tts -> mux_video
```

### 依赖检查

每个步骤开始前，检查前置步骤是否完成：

```go
func checkDependencies(taskID string, step string) error {
    dependencies := getStepDependencies(step)
    for _, dep := range dependencies {
        step, err := db.GetTaskStep(taskID, dep)
        if err != nil || step.Status != "succeeded" {
            return fmt.Errorf("dependency %s not completed", dep)
        }
    }
    return nil
}
```

### 步骤依赖映射

```go
var stepDependencies = map[string][]string{
    "asr":         {"extract_audio"},
    "translate":   {"asr"},
    "tts":         {"translate"},
    "mux_video":   {"tts"},
}
```
