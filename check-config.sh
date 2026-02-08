#!/bin/bash
# 配置检查脚本

set -e

echo "========================================="
echo "Docker 部署配置检查"
echo "========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

errors=0
warnings=0

# 检查 Docker 配置文件
echo -e "${YELLOW}检查 Docker 配置文件...${NC}"
echo ""

# 1. 检查 Redis 密码配置
echo -e "${YELLOW}[1] 检查 Redis 配置...${NC}"
if grep -q "requirepass" docker-compose.v2.yml; then
    echo -e "${RED}❌ docker-compose.v2.yml 中仍包含 Redis 密码配置${NC}"
    errors=$((errors+1))
else
    echo -e "${GREEN}✅ Redis 无密码配置（正确）${NC}"
fi

if grep -q "requirepass" docker-compose.prod.yml 2>/dev/null; then
    echo -e "${RED}❌ docker-compose.prod.yml 中仍包含 Redis 密码配置${NC}"
    errors=$((errors+1))
else
    echo -e "${GREEN}✅ 生产环境 Redis 无密码配置（正确）${NC}"
fi
echo ""

# 2. 检查 Celery 队列配置
echo -e "${YELLOW}[2] 检查 Celery Worker 队列配置...${NC}"
if grep -q "queues=default,media,ai,celery" docker-compose.v2.yml; then
    echo -e "${GREEN}✅ Celery worker 队列配置正确${NC}"
else
    echo -e "${RED}❌ Celery worker 缺少队列配置${NC}"
    errors=$((errors+1))
fi

if grep -q "queues=default,media,ai,celery" docker-compose.prod.yml 2>/dev/null; then
    echo -e "${GREEN}✅ 生产环境 Celery worker 队列配置正确${NC}"
else
    echo -e "${YELLOW}⚠️  生产环境 Celery worker 缺少队列配置${NC}"
    warnings=$((warnings+1))
fi
echo ""

# 3. 检查环境变量 REDIS_PASSWORD
echo -e "${YELLOW}[3] 检查环境变量文件...${NC}"
if grep -q "^REDIS_PASSWORD=" docker-compose.v2.yml; then
    echo -e "${RED}❌ docker-compose.v2.yml 中设置了 REDIS_PASSWORD${NC}"
    errors=$((errors+1))
else
    echo -e "${GREEN}✅ docker-compose.v2.yml 中无 REDIS_PASSWORD${NC}"
fi

if [ -f .env ]; then
    if grep -q "^REDIS_PASSWORD=" .env; then
        echo -e "${YELLOW}⚠️  .env 文件中设置了 REDIS_PASSWORD，建议注释${NC}"
        warnings=$((warnings+1))
    else
        echo -e "${GREEN}✅ .env 文件中无 REDIS_PASSWORD${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  .env 文件不存在，将使用默认配置${NC}"
    warnings=$((warnings+1))
fi
echo ""

# 4. 检查配置文件路径
echo -e "${YELLOW}[4] 检查 backend/app/config.py...${NC}"
if grep -q "/Users/micago/Desktop/" backend/app/config.py; then
    echo -e "${RED}❌ config.py 中仍有硬编码路径${NC}"
    errors=$((errors+1))
else
    echo -e "${GREEN}✅ config.py 配置正确${NC}"
fi
echo ""

# 5. 检查任务路由配置
echo -e "${YELLOW}[5] 检查任务路由配置...${NC}"
if grep -q '"process_video_pipeline".*"queue"' backend/app/workers/celery_app.py; then
    echo -e "${GREEN}✅ 任务路由配置包含 process_video_pipeline${NC}"
else
    echo -e "${RED}❌ 任务路由配置缺少 process_video_pipeline${NC}"
    errors=$((errors+1))
fi
echo ""

# 6. 检查必需文件
echo -e "${YELLOW}[6] 检查必需文件...${NC}"
required_files=(
    "backend/Dockerfile"
    "frontend/Dockerfile"
    "docker-compose.v2.yml"
    "docker-compose.prod.yml"
    ".env.docker.example"
    "DEPLOYMENT.md"
    "docker-test.sh"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✅ $file${NC}"
    else
        echo -e "${RED}❌ $file 不存在${NC}"
        errors=$((errors+1))
    fi
done
echo ""

# 7. 检查 .dockerignore
echo -e "${YELLOW}[7] 检查 .dockerignore...${NC}"
if [ -f .dockerignore ]; then
    echo -e "${GREEN}✅ .dockerignore 存在${NC}"
else
    echo -e "${YELLOW}⚠️  .dockerignore 不存在，构建可能较慢${NC}"
    warnings=$((warnings+1))
fi
echo ""

# 总结
echo "========================================="
if [ $errors -eq 0 ]; then
    echo -e "${GREEN}✅ 配置检查通过！${NC}"
    echo ""
    echo "可以开始部署："
    echo "  ./docker-test.sh"
else
    echo -e "${RED}❌ 发现 $errors 个错误${NC}"
    echo ""
    echo "请修复上述错误后再部署"
fi

if [ $warnings -gt 0 ]; then
    echo -e "${YELLOW}⚠️  有 $warnings 个警告${NC}"
fi
echo "========================================="

exit $errors
