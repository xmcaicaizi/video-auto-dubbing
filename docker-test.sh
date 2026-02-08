#!/bin/bash
# Docker 部署测试脚本

set -e

echo "========================================="
echo "视频自动配音系统 - Docker 部署测试"
echo "========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 Docker
echo -e "${YELLOW}[1/8] 检查 Docker 环境...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker 未安装${NC}"
    exit 1
fi
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose 未安装${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker 环境正常${NC}"
echo ""

# 检查环境变量文件
echo -e "${YELLOW}[2/8] 检查环境变量配置...${NC}"
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  .env 文件不存在，从模板复制...${NC}"
    if [ -f .env.docker.example ]; then
        cp .env.docker.example .env
        echo -e "${YELLOW}⚠️  请编辑 .env 文件，填写必要的配置（API Key、OSS等）${NC}"
        echo -e "${YELLOW}⚠️  按回车继续测试（使用示例配置）或 Ctrl+C 退出编辑配置${NC}"
        read
    else
        echo -e "${RED}❌ .env.docker.example 不存在${NC}"
        exit 1
    fi
fi
echo -e "${GREEN}✅ 环境变量配置存在${NC}"
echo ""

# 停止已有服务
echo -e "${YELLOW}[3/8] 停止已有服务...${NC}"
docker-compose -f docker-compose.v2.yml down 2>/dev/null || true
echo -e "${GREEN}✅ 已停止旧服务${NC}"
echo ""

# 构建镜像
echo -e "${YELLOW}[4/8] 构建 Docker 镜像...${NC}"
docker-compose -f docker-compose.v2.yml build --no-cache
echo -e "${GREEN}✅ 镜像构建完成${NC}"
echo ""

# 启动服务
echo -e "${YELLOW}[5/8] 启动服务...${NC}"
docker-compose -f docker-compose.v2.yml up -d
echo ""

# 等待服务就绪
echo -e "${YELLOW}[6/8] 等待服务启动...${NC}"
echo "等待数据库启动..."
sleep 10

# 运行数据库迁移
echo -e "${YELLOW}[7/8] 运行数据库迁移...${NC}"
docker-compose -f docker-compose.v2.yml exec -T api alembic upgrade head || {
    echo -e "${YELLOW}⚠️  数据库迁移失败，可能是首次运行，继续...${NC}"
}
echo ""

# 检查服务状态
echo -e "${YELLOW}[8/8] 检查服务状态...${NC}"
echo ""

# 检查容器状态
echo "容器状态:"
docker-compose -f docker-compose.v2.yml ps
echo ""

# 检查数据库
echo -e "${YELLOW}检查数据库连接...${NC}"
if docker-compose -f docker-compose.v2.yml exec -T db psql -U dubbing -d dubbing -c "SELECT 1;" &>/dev/null; then
    echo -e "${GREEN}✅ 数据库连接正常${NC}"
else
    echo -e "${RED}❌ 数据库连接失败${NC}"
fi
echo ""

# 检查 Redis
echo -e "${YELLOW}检查 Redis 连接...${NC}"
if docker-compose -f docker-compose.v2.yml exec -T redis redis-cli ping | grep -q "PONG"; then
    echo -e "${GREEN}✅ Redis 连接正常${NC}"
else
    echo -e "${RED}❌ Redis 连接失败${NC}"
fi
echo ""

# 检查 API
echo -e "${YELLOW}检查后端 API...${NC}"
sleep 5
if curl -s http://localhost:8000/ | grep -q "Video Dubbing API"; then
    echo -e "${GREEN}✅ 后端 API 正常${NC}"
else
    echo -e "${RED}❌ 后端 API 无响应${NC}"
    echo "API 日志:"
    docker-compose -f docker-compose.v2.yml logs --tail=20 api
fi
echo ""

# 检查前端
echo -e "${YELLOW}检查前端服务...${NC}"
sleep 2
if curl -s http://localhost:3000/ | grep -q "视频配音"; then
    echo -e "${GREEN}✅ 前端服务正常${NC}"
else
    echo -e "${RED}❌ 前端服务无响应${NC}"
    echo "前端日志:"
    docker-compose -f docker-compose.v2.yml logs --tail=20 frontend
fi
echo ""

# 检查 Celery Worker
echo -e "${YELLOW}检查 Celery Worker...${NC}"
if docker-compose -f docker-compose.v2.yml exec -T worker celery -A app.workers.celery_app inspect active &>/dev/null; then
    echo -e "${GREEN}✅ Celery Worker 正常${NC}"
    # 显示队列信息
    echo "监听的队列:"
    docker-compose -f docker-compose.v2.yml exec -T worker celery -A app.workers.celery_app inspect active_queues | grep "name" || true
else
    echo -e "${RED}❌ Celery Worker 异常${NC}"
    echo "Worker 日志:"
    docker-compose -f docker-compose.v2.yml logs --tail=20 worker
fi
echo ""

# 总结
echo "========================================="
echo -e "${GREEN}部署测试完成！${NC}"
echo "========================================="
echo ""
echo "访问地址:"
echo "  - 前端: http://localhost:3000"
echo "  - 后端: http://localhost:8000"
echo "  - API 文档: http://localhost:8000/api/v1/docs"
echo ""
echo "查看日志:"
echo "  docker-compose -f docker-compose.v2.yml logs -f"
echo ""
echo "停止服务:"
echo "  docker-compose -f docker-compose.v2.yml down"
echo ""
