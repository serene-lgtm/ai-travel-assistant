# Inspiration API

前端对接当前后端（Gin）可用的灵感对话接口说明。

## 1. 创建 Session

`POST /inspiration/session/create`

请求体：

```json
{
  "user_id": "65fe6c86b1c2c1a4bf1b6d93"
}
```

返回：

```json
{
  "session_id": "65ff1a2bb1c2c1a4bf1b6abc"
}
```

说明：
- `user_id` 必填
- 后端会把该 `user_id` 写入 `InspirationSession.user_id`
- 后续新建 `inspiration`（同一个 session 下的一个需求条目）时，也会同步写入 `inspiration.user_id`

## 2. 发送用户消息（对话）

`POST /inspiration/chat/completion`

请求体：

```jsonc
{
  "session_id": "65ff1a2bb1c2c1a4bf1b6abc", // 必填
  "role": "user",                           // 必填，前端固定传 user
  "kind": "user_input",                     // 可选：user_input | user_choice
  "content": "我想在秋天的京都寻找文学灵感",    // 必填（当前后端仍要求非空）
  "selected_option": "",                    // kind=user_choice 时传用户点击的选项文本
  "start_new_inspiration": false            // 可选；true 时先在 session.inspirations 里 append 一个新的
}
```

字段说明：
- `start_new_inspiration=true`
  - 用于“在同一个 session 中开始一个新的灵感需求”
  - 后端会先 `append` 一个新的 `inspiration`
  - 然后本次输入会写入这个新的 `inspiration`
- `selected_option`
  - 当 `kind=user_choice` 时传用户选择的按钮文案（建议与后端返回的 option 文案完全一致）

## 3. 对话响应（助手消息）

后端返回两类消息：
- `clarify_question`：继续澄清
- `assistant_reply`：生成最终灵感文本

示例（澄清问题）：

```json
{
  "session_id": "65ff1a2bb1c2c1a4bf1b6abc",
  "role": "assistant",
  "kind": "clarify_question",
  "content": "为了描绘情感基调，以下哪一个更接近你的期待？",
  "options": [
    { "content": "克制而温柔的安静", "selected": false },
    { "content": "略带浪漫的文学感", "selected": false }
  ],
  "target_field": "mood"
}
```

示例（最终生成）：

```json
{
  "session_id": "65ff1a2bb1c2c1a4bf1b6abc",
  "role": "assistant",
  "kind": "assistant_reply",
  "content": "第1幕 · 清晨的哲学之道...\n..."
}
```

字段说明：
- `target_field` 仅在 `clarify_question` 时出现，值可能为 `mood | scene | focus`
- `options` 仅在 `clarify_question` 时出现

## 4. 获取 Session 详情（含 inspirations）

`GET /inspiration/session/get?id=<session_id>`

返回为 `InspirationSession`（当前后端直接返回 model），核心字段示例：

```json
{
  "id": "65ff1a2bb1c2c1a4bf1b6abc",
  "user_id": "65fe6c86b1c2c1a4bf1b6d93",
  "max_token": 32000,
  "messages": ["..."],
  "requirement": [
    {
      "id": "1",
      "user_id": "65fe6c86b1c2c1a4bf1b6d93",
      "mood": {
        "content": "温柔但克制的安静",
        "score": 4,
        "selected_option": "克制而温柔的安静"
      },
      "scene": {
        "content": "秋天京都，清晨步道",
        "score": 4
      },
      "focus": {
        "content": "散步、观察、记录",
        "score": 3
      },
      "output": "第1幕 · 清晨的哲学之道...\n..."
    }
  ],
  "status": "completed",
  "created_at": "2026-02-25T00:00:00Z"
}
```

注意：
- 字段名在 JSON 中当前仍是 `requirement`，但其内容已经是 `[]inspiration`
- `requirement[].output` 在成功生成后会被后端写入

## 5. 推荐前端流程

1. 创建 session：`POST /inspiration/session/create`（带 `user_id`）
2. 发送首轮自由输入：`POST /inspiration/chat/completion`
3. 若返回 `clarify_question`：
   - 展示 `content + options`
   - 用户点击后，继续调用 `/inspiration/chat/completion`
   - `kind="user_choice"`，`selected_option` 传所选项
4. 若返回 `assistant_reply`：
   - 展示生成文案
   - 若用户要在同一个 session 开新需求：
     - 下一次请求传 `start_new_inspiration=true`

## 6. 前端注意事项

- `role` 前端固定传 `"user"`
- 当前后端对 `content` 有非空校验；即使 `user_choice` 也建议传空格以外的短文本（如 `content: "用户选择了选项"`）更稳妥
- `selected_option` 建议直接使用后端返回的选项文案，避免不一致
- 同一个 session 可以包含多个 `inspiration`（保存在 `requirement` 数组中）

---

session created
user input ---------> sessiion askingMood
ai response, asking mood ---------> session askingMood (optional)
user choice, replying mood ---------> mood fulfilled, session askingScene (optional)
ai response, asking scene ---------> session askingScene (optional)
user choice, replying scene ---------> scene fulfilled, session askingFocus (optional)
ai response, asking focus ---------> session askingFocus (optional)
user choice, replying focus ---------> focus fulfilled, session completed
ai response, generating inspiration, asking if to end current session or to start a new requirement ---------> session completed
if user chooses to end current one, FE ends current session
if user chooses to start a new one, FE sends `start_new_inspiration=true` ---------> session startOver, append new inspiration, then start a new round of input/analyze

