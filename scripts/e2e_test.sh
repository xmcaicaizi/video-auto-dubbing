#!/bin/bash
# E2E 验收脚本 - 测试完整的视频处理流程（真实翻译，无 mock）

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

API_BASE="${API_BASE:-http://localhost:8080/api/v1}"
SOURCE_LANGUAGE="${SOURCE_LANGUAGE:-zh}"
TARGET_LANGUAGE="${TARGET_LANGUAGE:-en}"
TEST_VIDEO="${TEST_VIDEO:-$ROOT_DIR/test_vedio/test_video_10s.mp4}"
GLM_API_KEY="${GLM_API_KEY:-}"

if [ -z "$GLM_API_KEY" ] && [ -f "$ROOT_DIR/.env" ]; then
    GLM_API_KEY="$(grep -E '^GLM_API_KEY=' "$ROOT_DIR/.env" | tail -n1 | cut -d= -f2- | tr -d '\r' | tr -d '"' | tr -d "'")"
fi

if [ -z "$GLM_API_KEY" ] || [ "$GLM_API_KEY" = "your_glm_api_key" ]; then
    echo "错误: GLM_API_KEY 未设置（需要真实翻译，不使用 mock）。"
    echo "请设置环境变量 GLM_API_KEY 或在 .env 中配置真实值。"
    exit 1
fi

echo "=== E2E 测试脚本 ==="
echo "API Base: $API_BASE"
echo "测试视频: $TEST_VIDEO"
echo "源语言: $SOURCE_LANGUAGE -> 目标语言: $TARGET_LANGUAGE"
echo ""

# 检查测试视频是否存在
if [ ! -f "$TEST_VIDEO" ]; then
    if [ -x "$SCRIPT_DIR/prepare_test_video.sh" ]; then
        echo "未找到测试视频，尝试自动生成..."
        "$SCRIPT_DIR/prepare_test_video.sh"
    fi
fi

if [ ! -f "$TEST_VIDEO" ]; then
    echo "错误: 测试视频文件不存在: $TEST_VIDEO"
    echo "请提供测试视频文件，或运行 scripts/prepare_test_video.sh 生成"
    exit 1
fi

# 1. 创建任务
echo "步骤 1: 创建任务..."
TASK_RESPONSE=$(curl -s -X POST "$API_BASE/tasks" \
    -F "video=@$TEST_VIDEO" \
    -F "source_language=$SOURCE_LANGUAGE" \
    -F "target_language=$TARGET_LANGUAGE" \
    -F "glm_api_key=$GLM_API_KEY")

TASK_ID=$(echo "$TASK_RESPONSE" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TASK_ID" ]; then
    echo "错误: 创建任务失败"
    echo "响应: $TASK_RESPONSE"
    exit 1
fi

echo "任务创建成功: $TASK_ID"
echo ""

# 2. 轮询任务状态
echo "步骤 2: 等待任务完成..."
MAX_WAIT=600  # 最多等待10分钟
WAIT_INTERVAL=5
ELAPSED=0

while [ $ELAPSED -lt $MAX_WAIT ]; do
    STATUS_RESPONSE=$(curl -s "$API_BASE/tasks/$TASK_ID")
    STATUS=$(echo "$STATUS_RESPONSE" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    PROGRESS=$(echo "$STATUS_RESPONSE" | grep -o '"progress":[0-9]*' | cut -d':' -f2)
    
    echo "[$ELAPSED秒] 状态: $STATUS, 进度: ${PROGRESS}%"
    
    if [ "$STATUS" = "done" ]; then
        echo "任务完成！"
        break
    fi
    
    if [ "$STATUS" = "failed" ]; then
        ERROR=$(echo "$STATUS_RESPONSE" | grep -o '"error":"[^"]*"' | cut -d'"' -f4)
        echo "任务失败: $ERROR"
        exit 1
    fi
    
    sleep $WAIT_INTERVAL
    ELAPSED=$((ELAPSED + WAIT_INTERVAL))
done

if [ $ELAPSED -ge $MAX_WAIT ]; then
    echo "错误: 任务超时"
    exit 1
fi

echo ""

# 3. 获取任务结果
echo "步骤 3: 获取任务结果..."
RESULT_RESPONSE=$(curl -s "$API_BASE/tasks/$TASK_ID/result")

if echo "$RESULT_RESPONSE" | grep -q '"status":"done"'; then
    echo "任务结果获取成功"
    OUTPUT_URL=$(echo "$RESULT_RESPONSE" | grep -o '"output_video_url":"[^"]*"' | cut -d'"' -f4)
    echo "输出视频 URL: $OUTPUT_URL"
else
    echo "错误: 获取任务结果失败"
    echo "响应: $RESULT_RESPONSE"
    exit 1
fi

echo ""

# 4. 获取下载链接
echo "步骤 4: 获取下载链接..."
DOWNLOAD_RESPONSE=$(curl -s "$API_BASE/tasks/$TASK_ID/download?type=video")
DOWNLOAD_URL=$(echo "$DOWNLOAD_RESPONSE" | grep -o '"download_url":"[^"]*"' | cut -d'"' -f4)

if [ -n "$DOWNLOAD_URL" ]; then
    echo "下载链接: $DOWNLOAD_URL"
    echo ""
    echo "=== E2E 测试通过 ==="
    echo "任务 ID: $TASK_ID"
    echo "可以访问下载链接获取处理后的视频"
else
    echo "错误: 获取下载链接失败"
    echo "响应: $DOWNLOAD_RESPONSE"
    exit 1
fi
