# 控制台操作与维护

先查[接口目录](routes.md)，写入前核对对应 handler 或前端 API 的 payload。不要把该目录当成参数 schema。普通任务优先沿用当前配置；下面这些操作只在用户委托涉及它们时使用。

| 能力 | 主要接口 / 关键语义 |
| --- | --- |
| 全局设置 | `GET/PUT /api/settings`；先读快照、只改所需字段，保留代理/限速/并发等用户配置 |
| 模型 | `GET/POST /api/llm/profiles`；`POST /api/llm/test` 会调用模型。编辑时不发送占位 API Key 覆盖原密钥 |
| 任务模型链 | `PUT /api/tasks/{id}/llm`，body 有 llm_profile_ids 与可选 active_llm_profile_id；先读 resolution，保留严格 Worker 绑定 |
| Agent | `/api/agents`、`/api/agents/{key}`、`/api/agents/{key}/config`、`/api/agents/{key}/visibility`；提示词变更先读现有文本，不覆盖用户定制 |
| Skills | `GET/POST /api/skills`，`POST /api/skills/upload`（multipart file），`PUT /api/skills/{name}/meta`；名称与权限/可见性分开配置 |
| MCP | `GET/POST /api/mcp`，`POST /api/mcp/{id}/refresh`，`GET /api/mcp/{id}/tools`；保存/刷新会发起连接，本地路径是 ARTEX 服务端路径 |
| 自定义工具 | `/api/tools`、`/api/tools/custom`；注册不代表依赖已安装，测试端点会实际执行，创建后核对 Agent 绑定 |
| 资产/公司 | `/api/assets`、`/api/companies`、`/api/companies/{id}/scope`；导入公司范围可能影响多个任务，不为一个任务随意扩大公司定义 |
| 任务资产 | `/api/tasks/{id}/assets`、`/asset-refs`、`/scope`、`/coverage`、`/coverage-graph`；scope 不是网络隔离策略 |
| 拓扑 | `POST /api/tasks/{id}/topology-mapping` 与 GET 同路径 `/status`；会运行拓扑解释和报告，应跟进 conversation_id |
| 发现/复测 | `/api/exploration/findings/{id}`、`/retests`、`/generate-report`；复测/深入会重新执行并可能复活任务，不是只读验证 |
| 合并发现 | `/api/exploration/finding-cases` 及其子接口；先读来源和证据，不能仅为好看而合并不同权限边界的发现 |
| 流量/重放 | `/api/traffic`、`/api/traffic/exchange`、`/api/repeater/...`；查看只读，重放会向目标发请求，精确路径及 schema 见目录/源码 |
| 工作区 | `/api/workspace/list`、`/read`、`/download`、`/upload`；用 API 返回路径，不拼接穿越目录路径；证据文件保存到私有位置 |
| 任务归档 | `/api/tasks/{id}/archive`、`/api/task-archives/{id}/restore` 等；返回 job ID 不代表完成，查询 `/api/task-archives/{id}` 的 state/error 跟进；删除/恢复属于显式维护操作 |
| 计划任务 | `/api/schedules`、`/{id}/pause`、`/resume`、`/run-now`；新建/恢复会带来后续执行，不能为方便绕开用户手动暂停和日历 |
| 通知 | `/api/notify/channels` 和 `/test`、deliveries/retry；测试也会外发消息，需有这项委托 |
| 历史经验 | `/api/lonehero/settings`、`/api/lonehero/lessons`、`/api/tasks/{id}/lessons/extract`；默认 shadow，采纳后还需 inject，保持任务/公司隔离 |
| 费用 | `GET /api/tokens/usage`；`PUT /api/lonehero/prices/{pid}`，body input/output/cache_read/cache_write，单位 EUR/百万 Token |
| 程序更新 | `/api/update/check`、`/settings`、`/apply`、`/rollback`；apply/rollback 会重启，必须属于委托且已有备份/恢复路径 |

## 日历

计划请求含 name、enabled、timezone_mode、schedule_type、run_date/end_date 或 weekdays、start_time/end_time、start_immediately、task_ids。具体时区枚举、星期表示及日期有效性按 `server/schedules.go`、`db/schedules.go` 核实；不猜测时区或把一次任务变成周期任务。

## 旁路问答与主会话

主会话 POST `/api/chat?task=...` 可能接受后后台处理。读取 chat/status 和 activity/history，避免上一条仍忙时重复发消息。旁路问答使用以下父对象路径追加 `/side-questions`：

- `/api/conversations/{id}`
- `/api/tasks/{id}/chat`
- `/api/tasks/{id}/intents/{iid}`
- `/api/intercept/history/{id}`

GET/POST/DELETE 按 `server/side_questions.go`；创建需 question 和 client_request_id，事件为 `/api/side-questions/{requestID}/events`。旁路分析不等于批准操作或推进原任务；取消/删除同样要在委托内。

## 更新和恢复

健康/统计 API 不证明真实目标操作成功。程序更新以本项目私有 Release 为来源；Docker 完整工具链更新需用户在部署端拉取镜像，控制台二进制更新不更换镜像工具链。不要为完成任务而自动发布、部署或改网关。

备份包括 PostgreSQL、data、skills、配置和签名密钥，交付记录不含这些秘密。修改配置前保存原值；失败先核对实际状态，避免对已成功的变更再次提交。
