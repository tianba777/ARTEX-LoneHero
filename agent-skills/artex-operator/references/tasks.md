# 任务、监控与接管

## 准备与创建

先核对 `/api/health`、`/api/tasks`、`/api/llm/profiles`、`/api/agents`、`/api/mcp` 和 `/api/skills`。已有 ID 就复用。模型测试会调用模型，不是免费的只读健康探针。

`POST /api/tasks` 没有普通草稿开关：不带 schedule 通常立即启动，`seed_first_intent:false` 也不代表不启动。`POST /api/active` 可能启动引擎，不能用来只读查看。

创建主要字段（必须换成当前用户实际授权的信息）：

```json
{
  "name": "用户委托的任务名称",
  "description": "授权范围、排除范围、操作限制、已有证据和期望交付",
  "goal": "可验证的完成标准",
  "company_ids": [],
  "llm_profile_ids": [],
  "source_task_ids": [],
  "timeout_seconds": 900,
  "worker_count": 1,
  "seed_first_intent": false,
  "coverage_enabled": true,
  "plan_quiet_mode": false,
  "intercept_rules": []
}
```

ID 来自当前实例查询。llm_profile_ids 是有序链，空列表沿用现有路由，不等于无模型。source_task_ids 是字符串数组，只读继承；company_ids 是整数数组，会快照公司的现有范围/资产，不能将整家公司当成仅获授权的一个路径。

使用用户预算；未给出时可说明并用 1 Worker/15 分钟的保守起点，不猜目标或授权。timeout_seconds=0 表示不限时。task scope 是覆盖度分母来源，不等于网络沙箱；描述、约束、适当的任务拦截规则共同表达边界。拦截规则参数从源码 handler/当前前端类型核实，不编造白名单通配语义。

保存返回 ID；创建断线先按名称、时间和目标查重，不直接再次创建。

## 跟进

```bash
python3 agent-skills/artex-operator/scripts/artex.py snapshot 123
python3 agent-skills/artex-operator/scripts/artex.py watch 123 --seconds 60 --interval 15
```

snapshot 最多读 100 条意图及待审批第一页，需要完整历史要分页。“此页没有”不等于全部没有。watch 返回后判断原因并继续任务；窗口结束时任务可能还在运行。

| 操作 | 接口及请求 |
| --- | --- |
| 详情 | `GET /api/tasks/{id}` |
| 模型解析/耗尽 | `GET /api/tasks/{id}/llm/resolution` |
| 暂停/恢复 | `POST /api/tasks/{id}/control`，`{"action":"pause"}` 或 resume |
| Worker | `GET /api/exploration/intents?task={id}&limit=100` |
| 主会话 | `GET /api/exploration/activity/history?task={id}&session=main&limit=50` |
| 规划会话 | 同 history，`session=plan` |
| Worker 会话 | 同 history，`session=intent:{iid}` |
| 主 Agent 消息 | `POST /api/chat?task={id}`，`{"message":"..."}`；202 仅表示接受 |
| 主 Agent 忙闲 | `GET /api/tasks/{id}/chat/status` |
| 新约束 | `POST /api/tasks/{id}/constraints`，`{"kind":"deny","text":"..."}`（或 allow） |
| 新目标 | `POST /api/tasks/{id}/goals`，`{"text":"...","vulnclass":""}`；可能复活终态任务 |
| 意图控制 | `POST /api/tasks/{id}/intents/{iid}/control`；action=pause/resume/cancel，cancel 需 reason，默认 soft |

排队不是失败，不绕过并发上限。分别处理手动暂停、日历、模型耗尽和 Worker 绑定。任务 resume 适用于暂停任务，不能随意重启终态任务。

## 人工问询、审批和恢复

ask_operator 在对应 intent 活动写入 `OPERATOR INPUT REQUESTED` 并暂停。读具体问题/必要附件；真人未提供的一次性口令、验证码或必须由人作的决定，转达并等待，不猜答案。

拿到真实回复后：

```text
POST /api/tasks/{id}/intents/{iid}/messages
{"message":"用户实际回复","request_id":"持久保存的唯一请求ID"}
```

回复会继续执行 Worker，主任务须具备可执行状态。一个逻辑回复复用同一个 request_id，断线先查会话和意图，不用新 ID 重复投递。不要错发到主 Agent，或只 resume 而丢失答案。

审批：`GET /api/intercept/task/{id}?status=pending&page=1&size=100`；详情 `GET /api/intercept/pending/{approvalID}`。按既有授权策略检查实际操作/影响，需要真实授权人决定时转达。决定 `POST /api/intercept/pending/{approvalID}/decide`，body 的 decision 是 **allowed / denied**。不自动允许全部待办，不改全局裁判/拦截配置消除待办。

`POST /api/tasks/{id}/intents/{iid}/rerun` 重新执行 blocked/exhausted/stopped 意图，可能复活/排队任务；rerun-blocked 是批量版本。确认原因已处理且旧操作没有成功后再按委托使用。

## 交付

读 `/api/tasks/{id}/outcome`、`/api/exploration/findings?task={id}`、任务 coverage/coverage-graph 和 `/api/report?task={id}`。GET report 是已有发现的 Markdown 汇总，不保证自动完成最终审计结论；单发现报告生成/复测会调用模型或重新执行，按委托触发并跟进。

done 且无确认发现时报告“完成、无已确认发现”；failed/timeout 列出已取得证据及未完成方向。费用是按当前单价估算，不是账单。经验提取只产生候选，不自动采纳/启用注入。
