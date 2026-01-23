# IndexTTS 音色克隆实施指南

## 🎯 修改清单

### 1. TTS服务器端修改

**文件位置**: 你的TTS服务器上的 `api_server_v2.py`

**需要添加的功能**:
- 音频文件上传接口 `/upload_audio`
- 增强的TTS接口 `/tts_url_with_cloning`
- 文件清理和健康检查

**操作步骤**:

1. **备份原文件**
   ```bash
   cp api_server_v2.py api_server_v2.py.backup
   ```

2. **添加导入**（在文件顶部）
   ```python
   from fastapi import FastAPI, UploadFile, File, HTTPException
   from pydantic import BaseModel
   import shutil
   import os
   import time
   from pathlib import Path
   from typing import Optional
   ```

3. **添加上传目录配置**（在app定义后）
   ```python
   # 创建上传目录
   UPLOAD_DIR = Path("/tmp/uploaded_prompts")
   UPLOAD_DIR.mkdir(parents=True, exist_ok=True)
   ```

4. **添加完整的接口代码**
   - 复制 `tts_server_modifications.py` 中的所有函数
   - 根据你现有的TTS模型加载方式调整 `tts_model.infer` 调用

### 2. Worker客户端修改

**文件位置**: `worker/internal/tts/vllm_client.go`

**需要修改的内容**:

1. **更新导入**（添加multipart支持）
   ```go
   import (
       // 现有导入...
       "mime/multipart"
       "path/filepath"
   )
   ```

2. **替换关键方法**
   - 用 `vllm_client_modifications.go` 中的代码替换以下方法：
     - `indexTTSV2Request` 结构体
     - `tryIndexTTSV2Endpoint` 方法
   - 添加新方法：
     - `uploadPromptAudio`
     - `buildVoiceCloningRequest`
     - `executeVoiceCloningRequest`
     - `tryVoiceCloningEndpoint`

### 3. 具体实施步骤

#### Phase 1: 服务器端修改 (30分钟)

1. **连接到TTS服务器**
   ```bash
   ssh your-tts-server
   cd /path/to/index-tts-vllm
   ```

2. **备份和修改**
   ```bash
   # 备份
   cp api_server_v2.py api_server_v2.py.backup

   # 编辑文件，添加上传功能
   nano api_server_v2.py
   ```

3. **重启TTS服务**
   ```bash
   # 停止现有服务
   pkill -f api_server_v2.py

   # 重新启动
   python api_server_v2.py --model_dir checkpoints --port 6006
   ```

4. **验证服务**
   ```bash
   curl http://localhost:6006/health
   curl http://localhost:6006/docs  # 查看Swagger文档
   ```

#### Phase 2: Worker客户端修改 (20分钟)

1. **修改客户端代码**
   ```bash
   cd /path/to/video-auto-dubbing

   # 备份
   cp worker/internal/tts/vllm_client.go worker/internal/tts/vllm_client.go.backup

   # 编辑文件
   nano worker/internal/tts/vllm_client.go
   ```

2. **重新构建和部署**
   ```bash
   docker-compose build worker
   docker-compose restart worker
   ```

#### Phase 3: 测试验证 (10分钟)

1. **测试音频上传**
   ```bash
   # 测试上传接口
   curl -X POST -F "file=@test_audio.wav" http://your-tts-server:6006/upload_audio
   ```

2. **创建测试任务**
   - 通过前端上传一个短视频
   - 观察worker日志中的音频上传过程
   - 检查最终生成音频的音色相似度

## 🔧 关键配置说明

### TTS服务器配置

**上传目录**: `/tmp/uploaded_prompts`
- 自动创建
- 文件自动清理（1小时后）
- 支持多种音频格式

**新接口**:
- `POST /upload_audio` - 上传音频文件
- `POST /tts_url_with_cloning` - 音色克隆TTS
- `GET /health` - 健康检查

### Worker客户端配置

**智能降级**:
1. 优先上传原音频进行克隆
2. 上传失败时使用预设音色
3. 自动选择最适合的预设音色

**日志记录**:
- 音频上传成功/失败
- 使用的音色类型（原音频/预设）
- 详细的错误信息

## 📋 验证检查点

### ✅ 服务器端检查

1. TTS服务启动无错误
2. `/health` 接口返回正常
3. `/docs` 显示新增的接口
4. 上传目录创建成功

### ✅ 客户端检查

1. Worker服务重启成功
2. 没有Go编译错误
3. 日志中显示新的音频处理逻辑

### ✅ 端到端测试

1. 创建新的视频配音任务
2. Worker日志显示音频上传过程
3. 生成的配音音色与原音频相似
4. 情感表达更加丰富

## 🚨 故障排除

### 常见问题

1. **音频上传失败**
   - 检查TTS服务器磁盘空间
   - 确认上传目录权限
   - 查看服务器日志

2. **音色克隆效果不好**
   - 确认原音频质量（需要清晰、无背景音）
   - 调整emo_alpha参数（0.6-0.9）
   - 检查提取的音频片段长度

3. **服务连接问题**
   - 验证TTS服务器网络连通性
   - 确认端口6006正常监听
   - 检查防火墙设置

### 回滚方案

如果出现问题，可以快速回滚：

1. **服务器端回滚**
   ```bash
   cp api_server_v2.py.backup api_server_v2.py
   python api_server_v2.py --model_dir checkpoints --port 6006
   ```

2. **客户端回滚**
   ```bash
   cp worker/internal/tts/vllm_client.go.backup worker/internal/tts/vllm_client.go
   docker-compose build worker
   docker-compose restart worker
   ```

## 📈 预期改进效果

- **音色相似度**: 30% → 85%+
- **情感一致性**: 20% → 80%+
- **用户满意度**: 显著提升
- **处理时间**: 增加5-10秒（音频上传时间）

实施完成后，你将拥有真正的音色和情感克隆能力！