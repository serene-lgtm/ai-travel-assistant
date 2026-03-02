# ChatCompletion 进度查询 API 文档

## 概述

本文档说明了新增的 `GetRequestProgress` API，用于实时查询 `inspirationHandler.ChatCompletion` 请求的处理进展。该 API 可帮助前端在等待响应时显示进度状态，提供更好的用户体验。

## 状态定义

### 四个处理阶段

1. **decoding** - 初始阶段
   - 从进入 `ChatCompletion` 到 `analyzeRequirement` 之前
   - 负责消息验证和解析
   - 检查用户输入是否与旅行相关

2. **analyzing** - 分析阶段
   - 进入 `analyzeRequirement` 之后
   - 通过 LLM 提取并评分需求字段（情感基调、旅行场景、核心焦点）
   - 保存用户消息到数据库

3. **generating** - 生成阶段
   - 进入 `generateInspiration` 或 `generateClarifyQuestion` 时
   - 生成旅行灵感或澄清问题

4. **completed** - 成功完成
   - ChatCompletion 完成并返回结果

5. **failed** - 处理失败
   - 任何阶段发生错误，包含错误信息

## 数据模型

### RequestProcess

```go
type RequestProcess struct {
    ID          string       // 记录唯一ID
    SessionID   string       // 关联的 session ID
    UserID      string       // 用户ID
    Stage       RequestStage // 当前阶段 (decoding/analyzing/generating/completed/failed)
    StartedAt   time.Time    // 开始时间
    CompletedAt time.Time    // 完成时间 (完成时才填充)
    CreatedAt   time.Time    // 记录创建时间
    UpdatedAt   time.Time    // 最后更新时间
    Error       string       // 错误信息 (仅在失败时填充)
}
```

## API 端点

### 查询请求进度

**请求**
```
GET /inspiration/chat/progress?session_id=<session_id>
```

**参数**
- `session_id` (必需) - 字符串，来自 `/inspiration/session/create` 的响应

**响应示例 (成功)**
```json
{
  "id": "650e2c8a7f3c5a1b2d4e9f3a",
  "session_id": "65012345abcdef",
  "user_id": "user123",
  "stage": "generating",
  "started_at": "2024-03-02T10:15:30Z",
  "completed_at": "0001-01-01T00:00:00Z",
  "created_at": "2024-03-02T10:15:30Z",
  "updated_at": "2024-03-02T10:15:35Z",
  "error": ""
}
```

**响应示例 (失败)**
```json
{
  "id": "650e2c8a7f3c5a1b2d4e9f3a",
  "session_id": "65012345abcdef",
  "user_id": "user123",
  "stage": "failed",
  "started_at": "2024-03-02T10:15:30Z",
  "completed_at": "0001-01-01T00:00:00Z",
  "created_at": "2024-03-02T10:15:30Z",
  "updated_at": "2024-03-02T10:15:35Z",
  "error": "当前输入未识别为旅行请求,请更具体描述你的旅行意图"
}
```

**错误响应**
```json
{
  "error": "session_id is required"
}
```

HTTP 状态码：
- 200 OK - 成功获取进度
- 400 Bad Request - 缺少必需参数
- 500 Internal Server Error - 服务器错误

## 使用示例

### 前端轮询实例

```javascript
async function pollChatProgress(sessionId) {
  const maxAttempts = 120; // 最多轮询2分钟
  let attempts = 0;
  
  const pollInterval = setInterval(async () => {
    attempts++;
    
    if (attempts >= maxAttempts) {
      clearInterval(pollInterval);
      console.error('Progress query timeout');
      return;
    }
    
    try {
      const response = await fetch(
        `/inspiration/chat/progress?session_id=${sessionId}`
      );
      const progress = await response.json();
      
      if (!response.ok) {
        console.error('Error:', progress.error);
        return;
      }
      
      // 更新UI显示当前阶段
      updateProgressUI(progress.stage);
      
      // 当完成或失败时停止轮询
      if (progress.stage === 'completed') {
        clearInterval(pollInterval);
        console.log('Request completed successfully');
      } else if (progress.stage === 'failed') {
        clearInterval(pollInterval);
        console.error('Request failed:', progress.error);
      }
      
    } catch (err) {
      console.error('Poll error:', err);
    }
  }, 500); // 每500ms轮询一次
}

function updateProgressUI(stage) {
  const stageLabels = {
    'decoding': '正在解析...',
    'analyzing': '正在分析需求...',
    'generating': '正在生成结果...',
    'completed': '已完成',
    'failed': '处理失败'
  };
  
  document.getElementById('progress').textContent = 
    stageLabels[stage] || stage;
}
```

## 数据库集合

新增 MongoDB 集合 `request_process`，结构如下：

```javascript
{
  "_id": ObjectId,           // 自动生成的ID
  "sid": string,             // session_id
  "uid": string,             // user_id
  "stg": string,             // 阶段状态
  "sat": Date,               // started_at
  "cat": Date,               // completed_at
  "cat_at": Date,            // created_at
  "uat": Date,               // updated_at
  "err": string              // 错误信息 (可选)
}
```

### 索引建议

为提高查询性能，建议创建以下索引：

```javascript
// 按session_id和created_at的复合索引，用于快速查询最新记录
db.request_process.createIndex({
  "sid": 1,
  "cat_at": -1
})

// 单独的session_id索引
db.request_process.createIndex({
  "sid": 1
})
```

## 工作流程示例

```
用户发送消息 (POST /inspiration/chat/completion)
    ↓
服务端创建 RequestProcess 记录 (stage: decoding)
    ↓
通过 GET /inspiration/chat/progress 查询状态
    ↓
进入分析阶段 (stage: analyzing)
    ↓
继续轮询查询
    ↓
进入生成阶段 (stage: generating)
    ↓
生成响应并返回 (stage: completed)
    ↓
前端接收完整响应
```

## 改进点

相比用户初始设计，以下是优化的地方：

1. **增强的状态模型**
   - 添加 `completed` 和 `failed` 状态，更清晰地表示流程结果
   - 记录 `StartedAt` 和 `CompletedAt` 时间，便于计算处理耗时

2. **更完整的跟踪信息**
   - 记录 `UserID` 和 `SessionID`，便于关联查询和日志分析
   - 记录 `Error` 字段，失败时返回具体的错误信息

3. **更细粒度的状态更新**
   - 在关键步骤之前/之后更新状态
   - 在错误发生时立即捕获并记录

4. **数据持久化**
   - 每个请求的完整生命周期都被记录在 MongoDB
   - 便于后续的审计、分析和故障排查

## 实现细节

### 服务层 (Service)

- 在 `ChatCompletion` 开始时创建新的 `RequestProcess` 记录
- 在各个关键阶段更新状态
- 使用 defer 确保最终状态被正确记录
- 异常情况下标记为 `failed` 并保存错误信息

### 数据访问层 (DAO)

- `RequestProcessDao.Create()` - 创建新记录
- `RequestProcessDao.Update()` - 更新现有记录
- `RequestProcessDao.GetBySessionID()` - 获取最新的进度记录

### 处理器层 (Handler)

- `GetRequestProgress()` - REST API 端点，接收 `session_id` 参数

### 路由 (Router)

- `GET /inspiration/chat/progress` - 新增进度查询路由

