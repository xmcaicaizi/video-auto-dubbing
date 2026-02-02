#!/bin/bash
#
# 开发环境启动脚本
# 同时启动 FastAPI 服务器和 Celery Worker
#

set -e

echo "🚀 Starting Development Environment..."

# 检查必要的服务
check_service() {
    local service=$1
    local port=$2

    if ! nc -z localhost $port 2>/dev/null; then
        echo "❌ $service is not running on port $port"
        echo "   Please start it first"
        return 1
    fi
    echo "✅ $service is running"
    return 0
}

echo ""
echo "Checking services..."
check_service "PostgreSQL" 5432 || exit 1
check_service "Redis" 6379 || exit 1

# 设置环境变量
export PYTHONPATH="${PYTHONPATH}:$(pwd)"

# 激活虚拟环境
if [ -d "venv" ]; then
    source venv/bin/activate
elif [ -d ".venv" ]; then
    source .venv/bin/activate
fi

# 创建日志目录
mkdir -p logs

echo ""
echo "📋 Starting components:"
echo "   1. FastAPI Server (http://localhost:8000)"
echo "   2. Celery Worker (4 processes)"
echo ""

# 使用 tmux 分屏启动（如果安装了 tmux）
if command -v tmux &> /dev/null; then
    echo "Using tmux for split view..."

    # 创建新会话
    tmux new-session -d -s dubbing

    # 分割窗口
    tmux split-window -h

    # 左边：FastAPI
    tmux send-keys -t dubbing:0.0 'cd backend && uvicorn app.main:app --reload --host 0.0.0.0 --port 8000' C-m

    # 右边：Celery Worker
    tmux send-keys -t dubbing:0.1 'cd backend && ./run_worker.sh' C-m

    # 附加到会话
    echo ""
    echo "✅ Services started in tmux session 'dubbing'"
    echo ""
    echo "Commands:"
    echo "  - Attach to session: tmux attach -t dubbing"
    echo "  - Detach: Ctrl+B, then D"
    echo "  - Kill session: tmux kill-session -t dubbing"
    echo ""

    tmux attach -t dubbing

else
    # 没有 tmux，使用后台进程
    echo "tmux not found, starting in background..."

    # 启动 FastAPI
    nohup uvicorn app.main:app --reload --host 0.0.0.0 --port 8000 > logs/api.log 2>&1 &
    API_PID=$!
    echo "✅ FastAPI started (PID: $API_PID)"

    # 启动 Celery Worker
    nohup celery -A app.workers.celery_app worker --loglevel=info --concurrency=4 > logs/worker.log 2>&1 &
    WORKER_PID=$!
    echo "✅ Celery Worker started (PID: $WORKER_PID)"

    echo ""
    echo "Logs:"
    echo "  - API: tail -f logs/api.log"
    echo "  - Worker: tail -f logs/worker.log"
    echo ""
    echo "Stop services:"
    echo "  kill $API_PID $WORKER_PID"
    echo ""
fi
