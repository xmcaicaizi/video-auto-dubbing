#!/bin/bash
# TTS服务器功能测试脚本

echo "=== IndexTTS 服务器功能测试 ==="
echo "请在你的TTS服务器上运行这些命令进行测试"
echo ""

# 设置服务器地址（请根据实际情况修改）
TTS_SERVER="http://localhost:6006"

echo "1. 测试健康检查接口"
echo "curl -s $TTS_SERVER/health | jq ."
echo ""

echo "2. 测试API信息接口"
echo "curl -s $TTS_SERVER/ | jq ."
echo ""

echo "3. 测试音频上传功能"
echo "# 首先创建一个测试音频文件（如果没有的话）"
echo "# 或者使用现有的音频文件"
echo "curl -X POST -F 'file=@/path/to/test_audio.wav' $TTS_SERVER/upload_audio"
echo ""

echo "4. 测试原有的TTS接口"
echo 'curl -X POST "$TTS_SERVER/tts_url" \'
echo '  -H "Content-Type: application/json" \'
echo '  -d "{'
echo '    \"text\": \"Hello, this is a test.\","'
echo '    \"spk_audio_path\": \"/root/index-tts-vllm/examples/voice_01.wav\""'
echo '  }" --output test_original.wav'
echo ""

echo "5. 测试音色克隆接口"
echo 'curl -X POST "$TTS_SERVER/tts_url_with_cloning" \'
echo '  -H "Content-Type: application/json" \'
echo '  -d "{'
echo '    \"text\": \"Hello, this is a voice cloning test.\","'
echo '    \"spk_audio_path\": \"/tmp/uploaded_prompts/prompt_123456789.wav\","'
echo '    \"emo_alpha\": 0.8'
echo '  }" --output test_cloning.wav'
echo ""

echo "=== 预期结果 ==="
echo "1. 健康检查应该返回 status: healthy"
echo "2. API信息应该显示新增的接口"
echo "3. 音频上传应该返回 server_path 和 status: success"
echo "4. 原有TTS接口应该正常生成音频文件"
echo "5. 音色克隆接口应该生成音色相似的音频文件"