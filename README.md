# 视频本地化自动配音系统

> 更新日期：2026-01-03｜适用版本：main 分支



面向视频本地化的自动配音系统，通过结合先进的语音识别、机器翻译与可控语音合成技术，实现视频中对白的自动识别、跨语言翻译和目标语种配音。



## 系统架构



本系统采用微服务 + 异步任务队列方式构建，将语音识别、翻译、可控TTS 分布成独立模块，并结合对象存储与数据库统一管理资源，从而实现高并发、易扩展、可靠可部署的视频本地化自动配音平台。



### 核心特性



- ✅ **ASR 语音识别**: 使用 Moonshine 开源模型进行带时间戳的语音识别

- ✅ **机器翻译**: 利用 GLM 模型进行跨语言翻译

- ✅ **可控 TTS**: 通过魔搭(ModelScope)社区的 IndexTTS-2 API 实现受时间轴约束的可控语音合成

- ✅ **音画同步**: 通过时间轴约束解决跨语言配音时音画不同步的问题

- ✅ **微服务架构**: Go 语言后端，模块解耦，易于扩展

- ✅ **异步处理**: RabbitMQ 任务队列，支持高并发和任务重试

- ✅ **容器化部署**: Docker Compose 一键部署，支持横向扩展



## 技术栈



### 后端服务

- **语言**: Go 1.21+

- **框架**: Gin/Echo

- **数据库**: PostgreSQL 15

- **消息队列**: RabbitMQ

- **对象存储**: MinIO (S3 兼容)



### TTS 服务

- **语言**: Python 3.11+

- **框架**: FastAPI

- **依赖管理**: uv

- **模型**: IndexTTS-2（通过 ModelScope API 调用）



### 基础设施

- **容器化**: Docker + Docker Compose

- **网关**: NGINX

- **前端**: (待实现)



## 快速开始（权威入口）

完整、去重的步骤请阅读 [`docs/startup-guide.md`](docs/startup-guide.md)（唯一权威的快速启动指南）。概要流程如下：

1. 安装 Docker Engine 20.10+ 与 Docker Compose 2.0+，预留 8GB 内存 / 50GB 磁盘。
2. 克隆仓库并进入目录：`git clone <repository-url> && cd vedio`。
3. （可选）根据 `env.example` 配置 API Key、模型与存储等环境变量。
4. 启动并验证：`docker compose up -d && docker compose ps`，前端访问 `http://localhost`，API 健康检查 `http://localhost:8080/health`。

- **MinIO 控制台**: http://localhost:9001

- **RabbitMQ 管理**: http://localhost:15672

- **网关**: http://localhost:80



详细部署说明请参考[部署指南](docs/deployment.md)



## 项目结构



```

vedio/

├── api/                    # API 服务（Go）

├── worker/                  # Worker 服务（Go）

├── tts_service/            # TTS 服务（Python + uv）

├── asr_service/            # ASR service (Python + Moonshine)

├── gateway/                # NGINX 网关配置

├── web/                    # 前端（待实现）

├── docs/                   # 文档

│   ├── architecture.md    # 架构设计

│   ├── data-model.md      # 数据模型

│   ├── api-contracts.md   # API 接口规范

│   ├── queue-messages.md  # 队列消息规范

│   ├── tts-service.md     # TTS 服务规范

│   ├── coding-standards.md # 代码规范

│   ├── code-review.md     # 代码审查规范

│   ├── commit-message.md  # 提交信息规范

│   └── deployment.md      # 部署指南

├── .github/workflows/     # GitHub Actions CI 配置

├── Makefile              # 统一开发命令入口

├── .golangci.yml         # Go 静态检查配置

├── .editorconfig         # 编辑器配置

├── CONTRIBUTING.md       # 贡献指南

├── docker-compose.yml     # Docker Compose 配置

├── .env.example           # 环境变量示例

└── README.md             # 本文档

```



## 核心流程



1. **上传视频**: 用户通过 API 上传视频文件

2. **提取音频**: Worker 使用 ffmpeg 从视频提取音轨

3. **语音识别**: 调用 Moonshine ASR 服务，获取带时间戳的识别结果

4. **机器翻译**: 调用 GLM API，将识别文本翻译为目标语言

5. **语音合成**: 调用 TTS 服务，根据时间轴约束合成配音音频

6. **视频合成**: 使用 ffmpeg 将配音音频替换原视频音轨

7. **下载结果**: 用户下载处理完成的视频



详细流程请参考[架构设计文档](docs/architecture.md)



## API 文档



### 创建任务

```bash

POST /api/v1/tasks

Content-Type: multipart/form-data



video: <file>

source_language: zh

target_language: en

```



### 查询任务状态

```bash

GET /api/v1/tasks/:task_id

```



### 获取任务结果

```bash

GET /api/v1/tasks/:task_id/result

```



详细 API 文档请参考[API 接口规范](docs/api-contracts.md)



## 扩展 Worker



系统支持水平扩展 worker 实例：



```bash

# 启动 3 个 worker 实例

docker compose up -d --scale worker=3

```



## 开发指南



### 代码规范



本项目遵循 Google Style Guides 和 Engineering Practices 的最佳实践：



- 📋 [代码规范](docs/coding-standards.md) - Go/Python/Shell 编码规范

- 🔍 [代码审查规范](docs/code-review.md) - 代码审查流程和检查清单

- 📝 [提交信息规范](docs/commit-message.md) - Git 提交信息格式要求

- 🤝 [贡献指南](CONTRIBUTING.md) - 完整的开发工作流和贡献流程



### 代码检查



提交代码前，**必须**运行以下检查：



```bash

# 使用 Makefile（推荐）

make format    # 格式化所有代码

make lint      # 运行所有静态检查

make test      # 运行所有测试

make ci        # 运行完整 CI 检查（format + lint + test）



# 查看所有可用命令

make help

```



**CI 状态：** 所有代码提交都会通过 [GitHub Actions](.github/workflows/ci.yml) 自动检查，包括：

- ✅ Go 代码格式化和静态检查（gofmt, goimports, golangci-lint）

- ✅ Python 代码格式化和静态检查（black, ruff）

- ✅ Shell 脚本检查（shfmt, shellcheck）

- ✅ 所有测试必须通过



### Go 服务开发



```bash

cd api  # 或 worker

go mod download

go run main.go

```



### Python TTS 服务开发



```bash

cd tts_service

uv venv

source .venv/bin/activate  # Windows: .venv\Scripts\activate

uv sync

uv run uvicorn app.main:app --host 0.0.0.0 --port 8000

```



## 文档索引



### 系统文档

- [架构设计](docs/architecture.md) - 系统架构和服务划分

- [数据模型](docs/data-model.md) - 数据库表和对象存储设计

- [API 接口规范](docs/api-contracts.md) - REST API 详细说明

- [队列消息规范](docs/queue-messages.md) - RabbitMQ 消息格式和重试策略

- [ASR 服务规范](docs/asr-service.md) - Moonshine ASR 服务接口和配置

- [TTS 服务规范](docs/tts-service.md) - TTS 服务接口和配置

- [部署指南](docs/deployment.md) - 详细部署说明和故障排查



### 开发规范

- [代码规范](docs/coding-standards.md) - Go/Python/Shell 编码规范和最佳实践

- [代码审查规范](docs/code-review.md) - 代码审查流程和检查清单

- [提交信息规范](docs/commit-message.md) - Git 提交信息格式要求

- [贡献指南](CONTRIBUTING.md) - 完整的开发工作流和贡献流程



## 贡献指南



我们欢迎所有形式的贡献！请先阅读以下文档：



1. 📋 [贡献指南](CONTRIBUTING.md) - 详细的开发工作流

2. 📝 [代码规范](docs/coding-standards.md) - 编码规范和最佳实践

3. 🔍 [代码审查规范](docs/code-review.md) - 代码审查流程

4. 📝 [提交信息规范](docs/commit-message.md) - 提交信息格式



**快速开始：**



1. Fork 本项目

2. 创建特性分支(`git checkout -b feature/AmazingFeature`)

3. 开发代码并运行检查(`make ci`)

4. 提交更改（遵循提交信息规范）

5. 推送到分支并开启 Pull Request



**重要：** 所有代码必须通过 CI 检查才能合并！



## 许可协议



本项目采用 MIT 许可协议。



## 联系方式



如有问题或建议，请提交 Issue 或联系项目维护者。





## IndexTTS2 说明



- TTS 服务使用的 IndexTTS2 模型代码位于 `tts_service/indextts` 目录

- `_vendor/` 目录包含第三方依赖，不应提交到 Git
