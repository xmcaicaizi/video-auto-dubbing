FROM python:3.11-slim
COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv

WORKDIR /app

# 安装系统依赖
RUN apt-get update && apt-get install -y \
    ffmpeg \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 复制依赖文件和 README
COPY pyproject.toml README.md ./

# 安装 Python 依赖
RUN uv pip install --system -e .

# 复制应用代码
COPY . .

# 创建日志目录
RUN mkdir -p /app/logs

EXPOSE 8000

# 默认命令
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
