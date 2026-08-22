"""agentcluster-mock：apiserver ↔ agentcluster exec 契约的参考实现（FastAPI）。

契约：docs/交互时序与生命周期.md §5.1（金样 contracts/api/exec-turns-*.sse）
- POST /internal/exec/turns  请求 {turn_id, session_id, user_msg, config_version, kind?, resume_of?}
- 响应 text/event-stream：data 行为六类事件信封（不带序号，seq 由 apiserver 分配）；
  `: ping` 注释行为心跳（默认 15s，HEARTBEAT_SECONDS 可调）
- 断连即取消：客户端断开时 Starlette 会取消本协程，生成器停止（无需显式 cancel 端点）
- 每轮必发 done 或 error，唯一例外是被取消
- 本 mock 不连 PG（场景回放无需上下文）；真实 agentcluster 直连 PG 读写分域见表 docs §8

联调注入指令（写在 user_msg 里，见 README.md）：
[silent] 全静默触发看门狗 / [hang] 仅心跳长任务 / [break] 无终态断流 /
[error] 业务错误 / [slow] 慢速 token
"""

import asyncio
import json
import os
import re

from fastapi import FastAPI, Request
from fastapi.responses import StreamingResponse

app = FastAPI(title="agentcluster-mock", version="0.1.0")

HEARTBEAT = float(os.getenv("HEARTBEAT_SECONDS", "15"))
TOKEN_TICK = float(os.getenv("TOKEN_TICK_SECONDS", "0.024"))

RE_GREETING = re.compile(r"你好|hi|hello|帮助|你能")
RE_ALERT = re.compile(r"告警")
RE_DIAG = re.compile(r"诊断|变慢|排查|根因|为什么.*(慢|卡)|分析.*实例")
RE_DATAQA = re.compile(r"QPS|qps|指标|趋势|多少|统计|查询")


def sse(event: dict) -> str:
    return f"data: {json.dumps(event, ensure_ascii=False)}\n\n"


def ping() -> str:
    return ": ping\n\n"


def thought(step: int, tool: str, status: str, agent: str = "diag_subagent") -> str:
    return sse({"type": "thought", "step": step, "tool_name": tool, "status": status, "agent": agent})


def card(card_id: str, card_type: str, title: str, payload: dict | None = None, fallback: str = "") -> str:
    return sse({
        "type": "card", "mode": "create",
        "card": {
            "card_id": card_id, "card_type": card_type, "protocol_version": "1.0",
            "title": title, "status": "final", "context": {},
            "payload": payload or {}, "fallback_text": fallback,
        },
    })


async def say(text: str, tick: float):
    for i in range(0, len(text), 3):
        yield sse({"type": "token", "text_delta": text[i:i + 3]})
        await asyncio.sleep(tick)


async def tool(tool_name: str, ms: int):
    yield thought(1, tool_name, "running")
    await asyncio.sleep(ms / 1000)
    yield thought(1, tool_name, "success")


# ---------------- 场景（与 apiserver builtin 场景脚本对齐） ----------------

async def scenario_greeting(msg: str):
    async for chunk in say("你好！我是 DB Cockpit 智能运维助手（mock agentcluster）。"
                           "试着说「诊断 trade_tenant」看完整事件流，或在消息里加 [error]/[break]/[silent] 触发异常场景。", 0.02):
        yield chunk
    yield sse({"type": "done", "usage": {"prompt_tokens": 120, "completion_tokens": 60}})


async def scenario_dataqa(msg: str):
    async for chunk in tool("builtin_get_metrics", 600):
        yield chunk
    yield card("card_qps", "metric_chart", "全局 QPS 趋势（近 24h）",
               {"series": [{"name": "qps", "unit": "", "points": 48}]}, "QPS 均值 18.5k")
    async for chunk in say("近 24h 全局 QPS 均值约 18.5k，14:00 起流量高峰约 +35%。", 0.024):
        yield chunk
    yield sse({"type": "done", "usage": {"prompt_tokens": 300, "completion_tokens": 80}})


async def scenario_alerts(msg: str):
    async for chunk in tool("builtin_list_alerts", 500):
        yield chunk
    yield card("card_alerts", "data_table", "活跃告警",
               {"columns": ["级别", "名称", "描述"], "rows": [["P1", "cpu_high", "CPU 超阈值"]]}, "共 12 条活跃告警")
    async for chunk in say("当前共 12 条活跃告警（P1 1 / P2 6 / P3 5）。最高优先级：cpu_high — trade_tenant CPU 持续超 92%。", 0.024):
        yield chunk
    yield sse({"type": "done", "usage": {"prompt_tokens": 260, "completion_tokens": 70}})


async def scenario_diagnosis(msg: str):
    async for chunk in say("好的，我将对 prod-ob-core-01 的 trade_tenant 租户发起诊断，先采集证据：\n\n", 0.024):
        yield chunk
    async for chunk in tool("builtin_get_metrics", 700):
        yield chunk
    yield card("card_cpu", "metric_chart", "CPU 使用率（近 24h）",
               {"series": [{"name": "cpu", "unit": "%", "points": 96}]}, "CPU 14:00 后升至 92%")
    async for chunk in tool("builtin_session_snapshot", 700):
        yield chunk
    yield card("card_sessions", "data_table", "会话快照",
               {"columns": ["会话", "用户", "状态", "耗时"], "rows": [["88231", "app_rw", "活跃", "412s"]]}, "活跃会话 37")
    # 异步任务：进度卡 + progress 事件（真实链路中 progress 由 apiserver 任务总线直发）
    task_id = f"task_{int(asyncio.get_event_loop().time() * 1000):x}"
    yield sse({"type": "card", "mode": "create", "card": {
        "card_id": "card_task", "card_type": "task_progress", "protocol_version": "1.0",
        "title": "多指标长时窗扫描", "status": "streaming", "context": {},
        "payload": {"task_id": task_id, "progress": 0, "stage": "启动"}, "fallback_text": "深度扫描进行中",
    }})
    for p, stage in [(15, "多指标长时窗扫描…"), (45, "异常区间关联分析…"), (70, "根因假设生成…"), (100, "完成")]:
        await asyncio.sleep(0.6)
        yield sse({"type": "progress", "task_id": task_id, "progress": float(p), "stage": stage})
    yield sse({"type": "card", "mode": "update", "card": {
        "card_id": "card_report", "card_type": "diagnosis_report", "protocol_version": "1.0",
        "title": "诊断报告", "status": "final", "context": {},
        "payload": {"conclusion": "trade_order.status 缺索引引发全表扫描（置信度 92%）"},
        "fallback_text": "根因：缺索引；建议：加索引",
    }})
    async for chunk in say("\n\n诊断报告已生成：根因是 trade_order.status 缺索引引发全表扫描，锁等待为次生影响。可追问。", 0.024):
        yield chunk
    yield sse({"type": "done", "usage": {"prompt_tokens": 2100, "completion_tokens": 380}})


async def scenario_fallback(msg: str):
    async for chunk in say("已收到问题（mock 场景回放）。支持的演示：问候 / 「诊断」/「告警」/「QPS 趋势」，"
                           "或加 [slow] [break] [error] [hang] [silent] 指令触发异常路径。", 0.02):
        yield chunk
    yield sse({"type": "done", "usage": {"prompt_tokens": 150, "completion_tokens": 50}})


# ---------------- 主生成器：注入指令 + 场景分发 + 心跳包裹 ----------------

async def gen(msg: str):
    # 异常注入（docs §3.2 错误码触发表）
    if "[silent]" in msg:  # 全静默：触发 apiserver 看门狗 upstream_idle
        await asyncio.sleep(600)
        return
    if "[hang]" in msg:    # 长任务：仅心跳存活（心跳由 with_heartbeat 注入），2 分钟后 done
        await asyncio.sleep(120)
        yield sse({"type": "done", "usage": {"prompt_tokens": 1, "completion_tokens": 1}})
        return
    if "[break]" in msg:   # 中途断流（无终态）：apiserver 兜底 unexpected_end
        yield sse({"type": "token", "text_delta": "突然"})
        yield sse({"type": "token", "text_delta": "中断"})
        return
    if "[error]" in msg:   # 业务错误：携带 fallback_text
        yield sse({"type": "token", "text_delta": "已获部分证据，"})
        yield sse({"type": "error", "code": "budget_exceeded", "message": "轮次步数预算超限",
                   "fallback_text": "已基于已获证据给出初步结论：CPU 飙升与会话堆积相关，建议缩小时间窗重试。"})
        return

    tick = 0.2 if "[slow]" in msg else TOKEN_TICK

    if RE_GREETING.search(msg):
        scenario = scenario_greeting(msg)
    elif RE_DIAG.search(msg):
        scenario = scenario_diagnosis(msg)
    elif RE_ALERT.search(msg):
        scenario = scenario_alerts(msg)
    elif RE_DATAQA.search(msg):
        scenario = scenario_dataqa(msg)
    else:
        scenario = scenario_fallback(msg)

    # 慢速指令：每个事件后追加延迟（模拟上游拥堵）
    if "[slow]" in msg:
        async for chunk in scenario:
            yield chunk
            await asyncio.sleep(0.2)
    else:
        async for chunk in scenario:
            yield chunk


_DONE = object()


async def with_heartbeat(inner):
    """空闲超过 HEARTBEAT 秒时插入 `: ping` 注释行（apiserver 吸收不转发）。"""
    queue: asyncio.Queue = asyncio.Queue()

    async def producer():
        try:
            async for chunk in inner:
                await queue.put(chunk)
        except asyncio.CancelledError:
            raise
        finally:
            await queue.put(_DONE)

    task = asyncio.create_task(producer())
    try:
        while True:
            try:
                item = await asyncio.wait_for(queue.get(), timeout=HEARTBEAT)
            except asyncio.TimeoutError:
                yield ping()
                continue
            if item is _DONE:
                break
            yield item
    finally:
        task.cancel()


@app.get("/healthz")
async def healthz():
    return {"ok": True, "heartbeat_seconds": HEARTBEAT}


@app.post("/internal/exec/turns")
async def exec_turns(request: Request):
    body = await request.json()
    msg = body.get("user_msg", "")
    return StreamingResponse(
        with_heartbeat(gen(msg)),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no", "Connection": "keep-alive"},
    )
