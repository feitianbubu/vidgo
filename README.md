# vidgo

> 🎬 Unified Video Generation SDK for Go  
> 🧠 一站式调用可灵、即梦、Vidu 等主流视频大模型的 Go SDK  
> 📦 统一接口风格（兼容 OpenAI 风格），前端调用逻辑零改动

---

## ✨ 特性

- ✅ **统一 API**：抽象视频生成参数与返回结构，兼容不同模型
- ✅ **异步任务机制**：任务提交后可轮询查询，前后端解耦
- ✅ **多模型支持**：支持接入可灵、即梦、Vidu（持续扩展中）
- ✅ **Go 原生风格**：简洁、易用、开箱即用
- ✅ **灵活配置**：支持自定义模型映射与回调机制
- ✅ **类型安全**：完整的类型定义和验证
- ✅ **错误处理**：统一的错误处理和重试机制
- ✅ **架构清晰**：基于适配器模式，易于扩展新的提供者

---

## 📦 安装

```bash
go get github.com/feitianbubu/vidgo
```

## 🚀 快速开始

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/feitianbubu/vidgo"
)

func main() {
    // 配置可灵提供者
    config := &vidgo.ProviderConfig{
        BaseURL: "https://api.kuaishou.com",
        APIKey:  "your_access_key,your_secret_key", // 格式：access_key,secret_key
        Timeout: 60 * time.Second,
    }

    // 创建客户端
    client, err := vidgo.NewClient(vidgo.ProviderKling, config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    // 创建视频生成请求
    req := &vidgo.GenerationRequest{
        Prompt:       "在山间日出时分，飞鸟展翅的动画场景",
        Duration:     5.0,  // 5秒视频
        Width:        512,  // 宽度
        Height:       512,  // 高度
        FPS:          30,   // 帧率
        Model:        "kling-v2-master",
        QualityLevel: vidgo.QualityLevelStandard,
    }

    ctx := context.Background()

    // 提交生成任务
    resp, err := client.CreateGeneration(ctx, req)
    if err != nil {
        log.Fatalf("Failed to create generation: %v", err)
    }

    fmt.Printf("任务已创建，任务ID: %s\n", resp.TaskID)

    // 轮询任务状态
    result, err := client.WaitForCompletion(ctx, resp.TaskID, 10*time.Second)
    if err != nil {
        log.Fatalf("Failed to wait for completion: %v", err)
    }

    // 输出结果
    if result.Status == vidgo.TaskStatusSucceeded {
        fmt.Printf("视频生成成功！URL: %s\n", result.URL)
    }
}
```

### 图生视频

```go
// 图生视频请求
req := &vidgo.GenerationRequest{
    Image:    "https://example.com/sample.jpg", // 图片URL或Base64
    Prompt:   "让这张图片动起来，添加自然的动画效果",
    Duration: 5.0,
    Width:    512,
    Height:   512,
    Model:    "kling-v2-master",
}

resp, err := client.CreateGeneration(ctx, req)
// ... 处理结果
```

## 🏗️ 架构设计

### 核心组件

1. **Client**: 主客户端，提供统一的API接口
2. **Provider**: 提供者接口，定义视频生成能力
3. **Adapter**: 适配器实现，封装各厂商的具体API
4. **Types**: 统一的数据结构定义

### 目录结构

```
vidgo/
├── go.mod              # Go模块文件
├── README.md           # 项目说明
├── types.go            # 核心数据结构
├── provider.go         # Provider接口定义
├── client.go           # 主客户端实现
├── adapter_wrapper.go  # 适配器包装器
├── errors.go           # 错误定义
├── adapters/           # 适配器实现
│   └── kling.go       # 可灵适配器
└── examples/           # 使用示例
    └── main.go
```

## 🔌 支持的提供者

| 提供者 | 状态 | 模型支持 |
|--------|------|----------|
| 可灵 (Kling) | ✅ 已实现 | kling-v1, kling-v1-6, kling-v2-master |
| 即梦 (Jimeng) | 🚧 计划中 | - |
| Vidu | 🚧 计划中 | - |

## 📝 API 参考

### GenerationRequest

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `Prompt` | string | 可选* | 文本提示词（文本生视频） |
| `Image` | string | 可选* | 图片URL或Base64（图生视频） |
| `Duration` | float64 | 必需 | 视频时长（秒） |
| `Width` | int | 必需 | 视频宽度 |
| `Height` | int | 必需 | 视频高度 |
| `FPS` | int | 可选 | 帧率 |
| `Model` | string | 可选 | 模型名称 |
| `QualityLevel` | QualityLevel | 可选 | 画质级别 |

*注：Prompt 和 Image 至少需要提供一个

### TaskResult

| 字段 | 类型 | 说明 |
|------|------|------|
| `TaskID` | string | 任务ID |
| `Status` | TaskStatus | 任务状态 |
| `URL` | string | 视频链接（完成时） |
| `Format` | string | 视频格式 |
| `Metadata` | *Metadata | 视频元数据 |

## ⚙️ 配置选项

### ProviderConfig

```go
config := &vidgo.ProviderConfig{
    BaseURL:    "https://api.provider.com", // API基础URL
    APIKey:     "your_api_key",             // API密钥
    SecretKey:  "your_secret_key",          // 密钥（如需要）
    Timeout:    30 * time.Second,           // 请求超时
    RetryCount: 3,                          // 重试次数
    Extra:      map[string]string{},        // 额外配置
}
```

### ClientConfig

```go
clientConfig := &vidgo.ClientConfig{
    Timeout:    30 * time.Second,  // API请求超时
    MaxRetries: 3,                 // 最大重试次数
    RetryDelay: time.Second,       // 重试延迟
    Debug:      false,             // 调试模式
}

client, err := vidgo.NewClient(vidgo.ProviderKling, providerConfig, clientConfig)
```

## 🔧 错误处理

SDK提供了完整的错误处理机制：

```go
resp, err := client.CreateGeneration(ctx, req)
if err != nil {
    switch e := err.(type) {
    case *vidgo.APIError:
        fmt.Printf("API错误: %d - %s", e.Code, e.Message)
    case *vidgo.ValidationError:
        fmt.Printf("验证错误: %s", e.Message)
    default:
        fmt.Printf("其他错误: %v", err)
    }
}
```

## 🔄 状态轮询

```go
// 手动轮询
for {
    result, err := client.GetGeneration(ctx, taskID)
    if err != nil {
        break
    }
    
    switch result.Status {
    case vidgo.TaskStatusSucceeded, vidgo.TaskStatusFailed:
        return result, nil
    default:
        time.Sleep(5 * time.Second) // 等待5秒后重试
    }
}

// 自动轮询（推荐）
result, err := client.WaitForCompletion(ctx, taskID, 10*time.Second)
```

## 🚀 扩展新的提供者

实现新的提供者只需要实现 `adapters.Provider` 接口：

```go
// 在 adapters/ 目录下创建新的适配器
type MyProvider struct {
    // 实现字段
}

func (p *MyProvider) Name() string { /* 实现 */ }
func (p *MyProvider) CreateGeneration(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) { /* 实现 */ }
func (p *MyProvider) GetGeneration(ctx context.Context, taskID string) (*TaskResult, error) { /* 实现 */ }
func (p *MyProvider) SupportedModels() []string { /* 实现 */ }
func (p *MyProvider) ValidateRequest(req *GenerationRequest) error { /* 实现 */ }
```

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！
