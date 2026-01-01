# 代码规范

本文档定义了本项目的代码规范，遵循 Google Style Guides 和 Engineering Practices 的最佳实践。

## 参考资源

- [Google Style Guides](https://google.github.io/styleguide/)
- [Google Engineering Practices](https://google.github.io/eng-practices/)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Google Python Style Guide](https://google.github.io/styleguide/pyguide.html)

## 通用原则

### 1. 代码可读性优先

- 代码应该易于理解，优先考虑可读性而非简洁性
- 使用有意义的变量名和函数名
- 避免过度优化，除非性能瓶颈已确认
- 保持函数简短，单一职责

### 2. 一致性

- 遵循项目统一的命名约定
- 保持代码风格一致
- 使用统一的错误处理模式
- 保持日志格式一致

### 3. 可维护性

- 编写清晰的注释和文档
- 避免重复代码（DRY 原则）
- 模块化设计，低耦合高内聚
- 编写可测试的代码

## Go 语言规范

### 命名约定

遵循 [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) 的命名规范：

- **包名**：小写，简短，避免下划线和混合大小写
  ```go
  package api
  package worker
  ```

- **导出函数/类型**：首字母大写，使用驼峰命名
  ```go
  func CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error)
  type TaskService struct {}
  ```

- **未导出函数/类型**：首字母小写，使用驼峰命名
  ```go
  func validateRequest(req *CreateTaskRequest) error
  type taskRepository struct {}
  ```

- **常量**：使用驼峰命名，导出常量首字母大写
  ```go
  const MaxRetries = 3
  const defaultTimeout = 30 * time.Second
  ```

- **接口名**：通常以 `-er` 结尾，或使用描述性名称
  ```go
  type TaskRepository interface {
      Create(ctx context.Context, task *Task) error
  }
  ```

### 代码格式

- 使用 `gofmt` 自动格式化代码
- 使用 `goimports` 管理导入
- 导入顺序：标准库、第三方库、项目内部包（每组用空行分隔）

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "vedio/api/internal/models"
    "vedio/api/internal/services"
)
```

### 错误处理

- **永远检查错误**，不要忽略 `error` 返回值
- 使用 `fmt.Errorf` 和 `%w` 包装错误，保留错误链
- 在错误消息中包含上下文信息

```go
// 好的做法
if err := processTask(ctx, task); err != nil {
    return fmt.Errorf("failed to process task %s: %w", task.ID, err)
}

// 避免
if err != nil {
    return err  // 缺少上下文
}
```

### 日志规范

- 使用结构化日志（推荐 `log/slog` 或 `zerolog`）
- 日志级别：
  - `DEBUG`: 详细的调试信息
  - `INFO`: 一般信息，如请求处理、状态变更
  - `WARN`: 警告信息，如重试、降级
  - `ERROR`: 错误信息，需要关注但不影响服务
  - `FATAL`: 致命错误，服务无法继续运行

```go
logger.Info("task created",
    "task_id", task.ID,
    "user_id", userID,
    "source_lang", req.SourceLanguage,
)

logger.Error("failed to process task",
    "error", err,
    "task_id", task.ID,
)
```

### 注释规范

- 导出的函数、类型、常量必须有文档注释
- 文档注释以函数/类型名开头，使用完整句子
- 使用 `//` 进行行内注释，解释"为什么"而非"是什么"

```go
// CreateTask creates a new video dubbing task.
// It validates the request, stores the video file, and returns the task ID.
func CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
    // Validate request to prevent invalid submissions
    if err := validateRequest(req); err != nil {
        return nil, err
    }
    // ...
}
```

### 测试规范

- 测试文件以 `_test.go` 结尾
- 测试函数以 `Test` 开头
- 使用表驱动测试（table-driven tests）处理多个测试用例
- 使用子测试（`t.Run`）组织相关测试

```go
func TestValidateRequest(t *testing.T) {
    tests := []struct {
        name    string
        req     *CreateTaskRequest
        wantErr bool
    }{
        {
            name: "valid request",
            req: &CreateTaskRequest{
                SourceLanguage: "zh",
                TargetLanguage: "en",
            },
            wantErr: false,
        },
        {
            name: "missing source language",
            req: &CreateTaskRequest{
                TargetLanguage: "en",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateRequest(tt.req)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateRequest() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Python 语言规范

### 命名约定

遵循 [PEP 8](https://pep8.org/) 和 Google Python Style Guide：

- **模块名**：小写，使用下划线分隔
  ```python
  # tts_service/app/models.py
  # tts_service/app/services/tts_engine.py
  ```

- **类名**：使用驼峰命名（CapWords）
  ```python
  class TTSEngine:
      pass

  class AudioProcessor:
      pass
  ```

- **函数/变量名**：小写，使用下划线分隔
  ```python
  def synthesize_audio(text: str, timestamps: List[float]) -> bytes:
      pass

  max_retries = 3
  ```

- **常量**：全大写，使用下划线分隔
  ```python
  MAX_AUDIO_LENGTH = 600.0
  DEFAULT_SAMPLE_RATE = 22050
  ```

- **私有成员**：单下划线前缀（约定，非强制）
  ```python
  class TTSEngine:
      def __init__(self):
          self._model = None  # 内部使用
  ```

### 代码格式

- 使用 `black` 自动格式化（行宽 100）
- 使用 `ruff` 进行静态检查
- 行宽：100 字符（已在 `pyproject.toml` 配置）

### 类型注解

- 所有函数参数和返回值必须使用类型注解
- 使用 `typing` 模块的类型（`List`, `Dict`, `Optional`, `Union` 等）
- Python 3.10+ 可使用内置类型（`list`, `dict` 等）

```python
from typing import List, Optional
from pydantic import BaseModel

def process_audio(
    audio_data: bytes,
    sample_rate: int = 22050,
    timestamps: Optional[List[float]] = None,
) -> bytes:
    """Process audio data with optional timestamp constraints."""
    pass

class SynthesisRequest(BaseModel):
    text: str
    timestamps: List[float]
    speaker_id: Optional[str] = None
```

### 文档字符串（Docstrings）

遵循 Google Style 的 docstring 格式：

```python
def synthesize_audio(
    text: str,
    timestamps: List[float],
    speaker_id: Optional[str] = None,
) -> bytes:
    """Synthesize audio from text with timestamp constraints.

    Args:
        text: The text to synthesize.
        timestamps: List of timestamps in seconds for each segment.
        speaker_id: Optional speaker ID for voice cloning. If None, uses default voice.

    Returns:
        Synthesized audio data as bytes (WAV format).

    Raises:
        ValueError: If text is empty or timestamps are invalid.
        TTSException: If synthesis fails.

    Example:
        >>> audio = synthesize_audio("Hello world", [0.0, 1.0, 2.0])
        >>> len(audio) > 0
        True
    """
    pass
```

### 错误处理

- 使用具体的异常类型，避免使用裸露的 `Exception`
- 自定义异常继承自合适的基类
- 在异常消息中包含上下文信息

```python
class TTSException(Exception):
    """Base exception for TTS service errors."""
    pass

class SynthesisError(TTSException):
    """Raised when audio synthesis fails."""
    pass

def synthesize_audio(text: str) -> bytes:
    if not text:
        raise ValueError("text cannot be empty")
    
    try:
        result = _do_synthesis(text)
    except ModelLoadError as e:
        raise SynthesisError(f"failed to synthesize '{text}': {e}") from e
```

### 日志规范

- 使用 `logging` 模块，配置结构化日志
- 使用适当的日志级别
- 在日志中包含上下文信息（使用 `extra` 参数）

```python
import logging

logger = logging.getLogger(__name__)

logger.info(
    "Audio synthesis started",
    extra={
        "text_length": len(text),
        "speaker_id": speaker_id,
        "request_id": request_id,
    },
)

logger.error(
    "Synthesis failed",
    extra={"error": str(e), "text": text},
    exc_info=True,
)
```

### 测试规范

- 测试文件以 `test_` 开头或 `_test.py` 结尾
- 测试函数以 `test_` 开头
- 使用 `pytest` 框架
- 使用 fixtures 共享测试数据

```python
import pytest
from tts_service.app.services.tts_engine import TTSEngine

@pytest.fixture
def tts_engine():
    return TTSEngine()

def test_synthesize_audio(tts_engine):
    text = "Hello world"
    timestamps = [0.0, 1.0, 2.0]
    
    audio = tts_engine.synthesize(text, timestamps)
    
    assert len(audio) > 0
    assert isinstance(audio, bytes)
```

## Shell 脚本规范

### 命名约定

- 脚本文件使用 `.sh` 扩展名
- 文件名使用小写，下划线分隔
  ```bash
  deploy.sh
  setup_environment.sh
  ```

### 代码格式

- 使用 `shfmt` 格式化
- 使用 `shellcheck` 进行静态检查
- 使用 2 空格缩进（非制表符）

### 最佳实践

- 始终在脚本开头设置 `set -euo pipefail`
  ```bash
  #!/bin/bash
  set -euo pipefail
  ```

- 使用引号保护变量
  ```bash
  # 好的做法
  echo "Processing file: $filename"
  
  # 避免
  echo Processing file: $filename
  ```

- 使用函数组织代码
  ```bash
  deploy_service() {
      local service_name=$1
      echo "Deploying $service_name..."
      # ...
  }
  ```

## 配置管理

### 环境变量

- 使用 `.env.example` 提供配置模板
- 敏感信息（API 密钥、密码）必须通过环境变量传递，**绝不硬编码**
- 使用配置验证库（Go: `viper`, Python: `pydantic-settings`）

### 配置文件

- 配置文件使用 YAML 或 TOML 格式
- 提供默认配置，允许环境变量覆盖
- 配置文件不应包含敏感信息

## 安全规范

### 密钥管理

- **禁止**在代码中硬编码密钥、密码、API token
- 使用环境变量或密钥管理服务（如 AWS Secrets Manager）
- 在 `.gitignore` 中排除 `.env` 文件

### 依赖安全

- 定期更新依赖，修复安全漏洞
- 使用依赖扫描工具（如 `snyk`, `dependabot`）
- 锁定依赖版本（Go: `go.sum`, Python: `uv.lock`）

### 输入验证

- 所有用户输入必须验证
- 使用参数化查询防止 SQL 注入
- 验证文件类型和大小
- 限制请求大小和频率

### 日志安全

- **禁止**在日志中记录敏感信息：
  - 密码、API 密钥、token
  - 完整的信用卡号、身份证号
  - 完整的用户输入（可能包含 PII）
- 对敏感信息进行脱敏处理

```go
// 好的做法
logger.Info("user authenticated", "user_id", userID)

// 避免
logger.Info("user authenticated", "password", password)  // 危险！
```

## 性能规范

### Go

- 避免不必要的内存分配
- 使用 `sync.Pool` 复用对象
- 合理使用 goroutine，避免泄露
- 使用 `context` 管理超时和取消

### Python

- 避免在循环中进行重复计算
- 使用生成器处理大文件
- 合理使用缓存（`functools.lru_cache`）
- 避免不必要的数据库查询（使用批量操作）

## 代码审查检查清单

提交代码前，请确认：

- [ ] 代码已通过格式化工具（`gofmt`/`black`）
- [ ] 静态检查通过（`golangci-lint`/`ruff`）
- [ ] 所有测试通过
- [ ] 新增代码包含测试
- [ ] 文档注释完整（导出函数/类型）
- [ ] 无硬编码的密钥或敏感信息
- [ ] 错误处理完整
- [ ] 日志级别适当，无敏感信息泄露
- [ ] 提交信息清晰（参考 [提交信息规范](commit-message.md)）

## 工具配置

### Go

- `gofmt`: 自动格式化
- `goimports`: 管理导入
- `golangci-lint`: 静态检查（配置见 `.golangci.yml`）

### Python

- `black`: 代码格式化（配置见 `tts_service/pyproject.toml`）
- `ruff`: 静态检查（配置见 `tts_service/pyproject.toml`）
- `mypy`: 类型检查（可选，严格模式推荐）

### Shell

- `shfmt`: 代码格式化
- `shellcheck`: 静态检查

运行这些工具的命令见 [CONTRIBUTING.md](../CONTRIBUTING.md)。

