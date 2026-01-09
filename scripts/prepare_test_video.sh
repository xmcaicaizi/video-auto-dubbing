#!/usr/bin/env bash
# Generate a 10-second test video with real speech audio.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

OUTPUT_DIR="$ROOT_DIR/test_vedio"
OUTPUT_FILE="$OUTPUT_DIR/test_video_10s.mp4"
AUDIO_FILE="$ROOT_DIR/tts_service/assets/voice_01.wav"
FFMPEG_IMAGE="${FFMPEG_IMAGE:-jrottenberg/ffmpeg:4.4-alpine}"

if [ -f "$OUTPUT_FILE" ]; then
    echo "测试视频已存在: $OUTPUT_FILE"
    exit 0
fi

if [ ! -f "$AUDIO_FILE" ]; then
    echo "错误: 未找到音频样本: $AUDIO_FILE"
    exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
    echo "错误: 未找到 docker，请先安装 Docker Desktop。"
    exit 1
fi

mkdir -p "$OUTPUT_DIR"

docker run --rm -v "$ROOT_DIR:/workspace" -w /workspace "$FFMPEG_IMAGE" \
    -f lavfi -i color=c=black:s=1280x720:d=10 \
    -i "tts_service/assets/voice_01.wav" \
    -t 10 \
    -c:v libx264 -pix_fmt yuv420p \
    -c:a aac -b:a 128k \
    -shortest \
    "test_vedio/test_video_10s.mp4"

echo "生成完成: $OUTPUT_FILE"
