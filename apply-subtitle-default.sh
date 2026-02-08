#!/bin/bash
# 应用字幕默认值修改

echo "========================================="
echo "应用字幕默认值修改（EXTERNAL -> BURN）"
echo "========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 检查是否在项目根目录
if [ ! -f "docker-compose.v2.yml" ]; then
    echo -e "${RED}错误: 请在项目根目录运行此脚本${NC}"
    exit 1
fi

echo -e "${YELLOW}1. 检查修改...${NC}"
if grep -q "SubtitleMode.BURN" backend/app/models/task.py; then
    echo -e "${GREEN}✓ 代码修改已应用${NC}"
else
    echo -e "${RED}✗ 代码修改未应用${NC}"
    exit 1
fi
echo ""

echo -e "${YELLOW}2. 应用到本地环境...${NC}"

# 检查后端是否运行
if pgrep -f "uvicorn app.main" > /dev/null; then
    echo "重启后端服务..."
    pkill -f "uvicorn app.main"
    sleep 2
    cd backend && nohup uv run uvicorn app.main:app --reload --host 0.0.0.0 --port 8000 > /dev/null 2>&1 &
    echo -e "${GREEN}✓ 后端已重启${NC}"
else
    echo -e "${YELLOW}⚠ 后端未运行，跳过重启${NC}"
fi

# 运行数据库迁移
echo "运行数据库迁移..."
if cd backend && uv run alembic upgrade head 2>/dev/null; then
    echo -e "${GREEN}✓ 数据库迁移成功${NC}"
else
    echo -e "${YELLOW}⚠ 数据库迁移失败（可能已应用）${NC}"
fi
cd ..
echo ""

echo -e "${YELLOW}3. Docker 环境说明...${NC}"
echo "如使用 Docker，请运行:"
echo "  docker-compose -f docker-compose.v2.yml down"
echo "  docker-compose -f docker-compose.v2.yml up -d --build"
echo "  docker-compose -f docker-compose.v2.yml exec api alembic upgrade head"
echo ""

echo "========================================="
echo -e "${GREEN}应用完成！${NC}"
echo "========================================="
echo ""
echo "验证方法:"
echo "  1. 访问 http://localhost:3000/tasks/new"
echo "  2. 检查字幕模式默认值是否为 '烧录字幕（推荐）'"
echo ""
