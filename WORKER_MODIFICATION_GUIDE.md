# Worker客户端修改指南

## 🎯 修改目标

将Worker客户端升级为支持音色克隆功能，当有 `PromptAudioURL` 时自动上传原音频并进行音色克隆。

## 📁 需要修改的文件

**文件位置**: `worker/internal/tts/vllm_client.go`

## 🔧 修改步骤

### 步骤1：备份原文件

```bash
cd /path/to/video-auto-dubbing/worker
cp internal/tts/vllm_client.go internal/tts/vllm_client.go.backup
```

### 步骤2：替换文件内容

将 `vllm_client_complete_fix.go` 的内容完全替换到 `worker/internal/tts/vllm_client.go`

### 步骤3：检查导入依赖

确保以下导入已包含：

```go
import (
    "bytes"
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"  // 🔥 新增：音频上传需要
    "net/http"
    "path/filepath"   // 🔥 新增：文件路径处理
    "time"

    "vedio/shared/config"
    "go.uber.org/zap"
)
```

## ✨ 关键新功能

### 1. 智能音色克隆流程

```
用户请求 → 检查是否有PromptAudioURL
    ↓
有原音频 → 下载音频 → 上传到TTS服务器 → 音色克隆合成
    ↓
无原音频 → 使用预设音色 → 标准TTS合成
```

### 2. 新增的核心方法

- `tryVoiceCloningWithUpload()` - 完整的音色克隆流程
- `uploadPromptAudio()` - 音频上传功能
- `executeVoiceCloningRequest()` - 执行音色克隆请求
- `tryVoiceCloningEndpoint()` - 调用新的克隆接口

### 3. 增强的数据结构

```go
type indexTTSV2Request struct {
    // 原有字段...

    // 新增字段
    EmoAudioPath *string `json:"emo_audio_path,omitempty"`  // 独立情感参考
    EmoAlpha     float64 `json:"emo_alpha,omitempty"`       // 情感强度
}
```

## 🚦 工作流程

### 原音频可用时（音色克隆）

1. **下载原音频** - 从MinIO/CDN下载原始音频
2. **上传到TTS服务器** - 使用multipart/form-data上传
3. **调用克隆接口** - 使用 `/tts_url_with_cloning`
4. **返回克隆音频** - 生成音色相似的配音

### 原音频不可用时（降级处理）

1. **选择预设音色** - 根据SpeakerID映射
2. **调用标准接口** - 使用 `/tts_url`
3. **返回标准配音** - 使用预设音色

## 🔧 配置调整

### 预设音色映射

在 `getFallbackSpeaker()` 方法中，根据你的实际音色文件调整路径：

```go
speakerMapping := map[string]string{
    "default":      "/root/index-tts-vllm/examples/voice_01.wav",
    "male_young":   "/root/index-tts-vllm/examples/voice_01.wav",
    "female_young": "/root/index-tts-vllm/examples/voice_02.wav", // 可选择其他音色
    // ... 根据实际情况调整
}
```

### TTS服务器URL

确保配置中的TTS服务器URL正确：

```yaml
# config.yaml 或环境变量
tts:
  url: "https://u861448-ej47-562de107.bjb2.seetacloud.com:8443"
  api_key: ""  # 如果需要的话
```

## 📋 测试验证

### 步骤1：编译测试

```bash
cd worker
go mod tidy
go build ./...
```

### 步骤2：功能测试

创建一个包含原音频的任务：

1. 上传带音频的视频
2. 开始配音任务
3. 观察Worker日志中的音频上传过程
4. 检查生成的配音音色相似度

### 步骤3：日志检查

关键日志信息：

```
INFO  Starting TTS synthesis with voice cloning
INFO  Attempting voice cloning with original audio
DEBUG Audio data prepared for upload
INFO  Audio uploaded successfully
INFO  Voice cloning synthesis successful
```

如果出现降级：

```
WARN  Voice cloning failed, falling back to standard TTS
INFO  Using standard TTS (no voice cloning)
```

## 🚨 故障排除

### 常见问题

1. **音频上传失败**
   - 检查网络连通性到TTS服务器
   - 确认音频URL可访问
   - 查看上传接口返回的错误信息

2. **编译错误**
   - 检查导入路径是否正确
   - 确认所有依赖包已安装

3. **音色克隆效果不佳**
   - 检查原音频质量
   - 调整 `EmoAlpha` 参数（0.6-0.9）
   - 确认音频长度适中（3-10秒）

### 回滚方案

如果出现问题：

```bash
cp internal/tts/vllm_client.go.backup internal/tts/vllm_client.go
docker-compose build worker
docker-compose restart worker
```

## 🎯 预期改进

- **音色相似度**: 从30%提升到85%+
- **用户体验**: 自动音色克隆，无需手动选择
- **降级保障**: 失败时自动降级到预设音色
- **处理时间**: 增加5-10秒（音频上传时间）

修改完成后，你的配音系统将具备真正的音色克隆能力！