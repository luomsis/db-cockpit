# agentcluster-mock

apiserver ↔ agentcluster **exec 契约的参考实现**（Python FastAPI），兼作系列 2（Agent 集群工程师）的脚手架与联调工具。契约详规见 `docs/交互时序与生命周期.md` §5.1，SSE 金样见 `contracts/api/exec-turns-*.sse`。

## 运行

```bash
cd agentcluster-mock
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 8000
# 环境变量：HEARTBEAT_SECONDS（心跳间隔，默认 15；联调看门狗时可调小）
#           TOKEN_TICK_SECONDS（token 节奏，默认 0.024）
```

apiserver 侧切换事件源（Go 始终终结 SSE，仅切换执行来源）：

```bash
AGENT_MODE=upstream AGENT_UPSTREAM_URL=http://localhost:8000 ./apiserver
# 等价于 AGENT_EXEC_URL=http://localhost:8000/internal/exec/turns
```

## 直测

```bash
curl -N -X POST http://localhost:8000/internal/exec/turns \
  -H 'Content-Type: application/json' \
  -d '{"turn_id":"turn_t1","session_id":"sess_s1","user_msg":"诊断 trade_tenant","config_version":"cv_1"}'
```

## 场景与注入指令

关键词选场景（与 apiserver builtin 场景脚本对齐）：问候 / 「诊断…」/「告警」/「QPS…」/ 兜底。

在 `user_msg` 中加入以下指令可触发异常路径（对应 docs §3.2 错误码表，用于验收 apiserver 的兜底行为）：

| 指令 | 行为 | apiserver 侧预期 |
|---|---|---|
| `[slow]` | 每事件后延迟 200ms | 正常流式（慢） |
| `[break]` | 发 2 个 token 后直接结束（无终态） | `unexpected_end`，已产出截断保留 |
| `[error]` | 发业务错误事件（含 fallback_text） | `failed`，error_code=`budget_exceeded` |
| `[hang]` | 只靠心跳存活 2 分钟后 done | 正常长任务（心跳保活） |
| `[silent]` | 完全静默 10 分钟 | 看门狗 `upstream_idle`（90s） |

## 契约要点（实现即文档）

- 事件为 `data: {json}\n\n`，**不带序号**（seq 由 apiserver 分配）；心跳为 `: ping` 注释行；
- 每轮必发 `done` 或 `error`，唯一例外是被取消；
- **断连即取消**：客户端断开时 Starlette 取消生成协程，无需显式 cancel 端点；
- 本 mock 不连 PG（场景回放无需上下文）；真实 agentcluster 按 docs §8 读写分域直连 PG。
