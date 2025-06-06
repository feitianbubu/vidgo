# 视频生成 API 设计文档

## 1. 设计背景

* **缺乏统一标准**：当前各视频生成大模型厂商提供的 API 接口格式差异很大，没有统一的调用规范。302.AI 等平台已尝试将不同模型的接口 **统一格式** 进行封装。
* **厂商接口多样**：不同厂商（如可灵、即梦、Vidu 等）的视频生成接口各不相同。有的厂商已开放 API（如 Vidu 正式开放了视频生成 API），而有的接口尚未公开或仅限内部使用，给前端接入带来困难。
* **设计目标**：本接口旨在抽象出通用的调用格式，兼容当前和未来多家视频大模型厂商的能力。前端开发者可统一接入，背后可灵活选择不同模型提供商。
* **参考 OpenAI 风格**：借鉴 OpenAI API 的设计思路，统一使用版本前缀（如 `/v1` 路径）和 JSON 参数格式；并采用 **异步任务机制**：提交视频生成任务后仅返回任务 ID，前端需轮询查询接口以获取最终结果。

## 2. 接口设计

### 接口路径

* **创建任务**（异步/同步请求）：

  ```
  POST /v1/video/generations
  ```
* **查询结果**（轮询或回调）：

  ```
  GET /v1/video/generations/{task_id}
  ```

### 请求参数说明

| 参数名               | 类型              | 是否必需 | 说明                                                                        |
| ----------------- | --------------- | ---- | ------------------------------------------------------------------------- |
| `prompt`          | 字符串             | 可选   | 文本提示词，用于描述生成视频的内容（文本生视频）。**prompt** 和 **image** 至少需提供一个。                  |
| `image`           | 字符串（URL/Base64） | 可选   | 图像输入，可作为视频首帧或灵感图（图生视频）。支持外部URL或 Base64 编码。**prompt** 和 **image** 至少需提供一个。 |
| `style`           | 字符串             | 可选   | 风格名称或模型标识，可指定不同的视频风格（如"动画"、"写实"等），后台可对应不同模型。                              |
| `duration`        | 数值（秒）           | 必需   | 视频时长（秒）。前端可指定生成视频的时长，例如 `5.0` 表示 5 秒。                                     |
| `fps`             | 整数              | 可选   | 帧率，表示每秒多少帧视频画面。例如常用 `24` 或 `30`。若不填写，可使用后端默认值。                            |
| `width`           | 整数              | 必需   | 视频宽度（像素）。常见可选值如 512、768 等。                                                |
| `height`          | 整数              | 必需   | 视频高度（像素）。常见可选值如 512、768 等。                                                |
| `response_format` | 字符串             | 可选   | 生成结果的返回格式：`"url"` 返回视频链接（默认），`"b64_json"` 返回 Base64 编码数据。                 |
| `quality_level`   | 字符串             | 可选   | 画质级别，可选 `"low"`、`"standard"`、`"high"` 等。默认为中等质量。                          |
| `seed`            | 整数              | 可选   | 随机种子，用于结果复现。相同参数和种子会生成相同视频；可不填写，则随机生成。                                    |

### 示例请求 JSON

  * POST: `/v1/video/generations`
  * Content-Type: `application/json`
```json
{
  "prompt": "在山间日出时分，飞鸟展翅的动画场景。",
  "image": "https://example.com/first_frame.png",
  "style": "写实",
  "duration": 5.0,
  "fps": 30,
  "width": 512,
  "height": 512,
  "response_format": "url",
  "quality_level": "standard",
  "seed": 20231234
}
```

### 查询接口响应结构

查询接口（`GET /v1/video/generations/{task_id}`）返回任务的状态和结果信息，其 JSON 结构示例如下：

| 字段名        | 类型  | 说明                                                                                                           |
| ---------- | --- | ------------------------------------------------------------------------------------------------------------ |
| `task_id`  | 字符串 | 任务 ID，与请求时返回的 ID 保持一致。                                                                                       |
| `status`   | 字符串 | 任务状态：`"queued"`（已排队）、`"processing"`（处理中）、`"succeeded"`（完成）或`"failed"`（失败）等。                                  |
| `url`      | 字符串 | 视频资源的 URL 地址。当 `status` 为 `"succeeded"` 时有效。若选择 Base64 返回格式，则此处省略，直接在 `result` 或 `data` 中返回编码内容。             |
| `format`   | 字符串 | 视频文件格式，例如 `"mp4"`、`"gif"` 等。                                                                                 |
| `metadata` | 对象  | 结果元数据，包括实际生成的视频时长、帧率、分辨率、使用种子等信息。例如：<br>`{"duration":5.0,"fps":30,"width":512,"height":512,"seed":20231234}` |

### 示例响应 JSON

以下示例展示查询任务成功后的响应 JSON：

```json
{
  "task_id": "abcd1234efgh",
  "status": "succeeded",
  "url": "https://cdn.example.com/videos/abcd1234efgh.mp4",
  "format": "mp4",
  "metadata": {
    "duration": 5.0,
    "fps": 30,
    "width": 512,
    "height": 512,
    "seed": 20231234
  }
}
```

### 错误结构格式

若请求参数错误或接口调用失败，返回错误信息，其格式可以为：

```json
{
  "error": {
    "code": 400,
    "message": "Invalid prompt or parameters"
  }
}
```

其中 `code` 为错误码（HTTP 状态码或内部定义码），`message` 为错误描述。

## 3. 示例流程图

下面给出一个示例流程图，展示视频生成的基本流程：提交任务 -> 等待生成 -> 查询状态 -> 获取视频链接。

```mermaid
flowchart LR
    A[提交任务] --> B[等待生成]
    B --> C[查询状态]
    C --> D[获取视频链接]
```

上述流程为：前端通过 **POST /v1/video/generations** 提交生成任务，后端返回任务 ID 后进入生成队列；前端不断调用 **GET /v1/video/generations/{task\_id}** 查询状态，当状态变为 `succeeded` 时，从响应中获取 `url` 字段，即可获得生成的视频文件链接。

**参考资料：** 已有示例接口设计往往采用异步模式，仅返回任务 ID，再通过查询获取最终结果；OpenAI 等大厂 API 多以 `/v1/` 前缀和 JSON 参数为规范；302.AI 平台也提出统一多模型视频接口的方案。

## 4. 实现状态

### 已实现功能

- ✅ **视频生成接口**：`POST /v1/video/generations`
- ✅ **任务查询接口**：`GET /v1/video/generations/{task_id}`
- ✅ **可灵(Kling)适配器**：支持可灵视频生成模型
- ✅ **统一适配器架构**：使用 `GetAdaptor(relayInfo.ApiType)` 统一获取适配器
- ✅ **JWT认证**：自动处理可灵API的JWT token生成
- ✅ **前端集成**：渠道选择下拉框已添加"可灵"选项
- ✅ **代理支持**：支持通过代理访问上游API
- ✅ **智能路径识别**：通过URL自动区分生成和查询请求
- ✅ **扣费系统**：完整的预扣费、最终扣费和配额管理
- ✅ **消费记录**：详细的视频生成消费日志和统计

### 使用示例

#### 1. 创建视频生成任务

```bash
curl -X POST "http://localhost:3000/v1/video/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "prompt": "在山间日出时分，飞鸟展翅的动画场景。",
    "duration": 5.0,
    "width": 512,
    "height": 512,
    "fps": 30,
    "model": "kling-v2-master",
    "quality_level": "standard"
  }'
```

**响应示例：**
```json
{
  "task_id": "abcd1234efgh",
  "status": "queued"
}
```

#### 2. 查询任务状态

```bash
curl -X GET "http://localhost:3000/v1/video/generations/abcd1234efgh" \
  -H "Authorization: Bearer sk-your-api-key"
```

**响应示例（处理中）：**
```json
{
  "task_id": "abcd1234efgh",
  "status": "processing"
}
```

**响应示例（完成）：**
```json
{
  "task_id": "abcd1234efgh",
  "status": "succeeded",
  "url": "https://cdn.example.com/videos/abcd1234efgh.mp4",
  "format": "mp4",
  "metadata": {
    "duration": 5.0,
    "fps": 30,
    "width": 512,
    "height": 512,
    "seed": 20231234
  }
}
```

### 配置说明

1. **添加可灵渠道**：
   - 渠道类型：选择"可灵 Kling"（类型值：50）
   - API Key格式：`{access_key},{secret_key}`
   - Base URL：可灵API的基础URL

2. **支持的模型**：
   - `kling-v1`
   - `kling-v1-6`
   - `kling-v2-master`