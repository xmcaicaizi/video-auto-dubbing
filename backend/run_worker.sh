#!/bin/bash
#
# Celery Worker 启动脚本
#

set -e

echo "🚀 Starting Celery Worker..."

# 设置环境变量
export PYTHONPATH="${PYTHONPATH}:$(pwd)"

# 激活虚拟环境（如果存在）
if [ -d "venv" ]; then
    source venv/bin/activate
elif [ -d ".venv" ]; then
    source .venv/bin/activate
fi

# 启动 Celery Worker
celery -A app.workers.celery_app worker \
    --loglevel=info \
    --concurrency=4 \
    --max-tasks-per-child=100 \
    --time-limit=3600 \
    --soft-time-limit=3300 \
    -n worker@%h

# 说明:
# --loglevel=info: 日志级别
# --concurrency=4: 并发工作进程数
# --max-tasks-per-child=100: 每个进程最多处理100个任务后重启
# --time-limit=3600: 任务硬超时（1小时）
# --soft-time-limit=3300: 任务软超时（55分钟）
# -n worker@%h: Worker 名称（%h = hostname）
