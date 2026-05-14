"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { ArrowLeft } from "lucide-react";

import {
  fetchInstanceDetail,
  instanceAction,
  patchInstance,
} from "@/api/instances";
import { ConsolePage } from "@/components/layout/console-page";
import { DataFreshness } from "@/components/layout/data-freshness";
import { ErrorState } from "@/components/feedback/error-state";
import { LoadingState } from "@/components/feedback/loading-state";
import { useConfirm } from "@/components/feedback/confirm-provider";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { StrategyInstanceDetailDTO } from "@/types/strategies";
import { formatDateTime } from "@/lib/format-trading";
import { cn } from "@/lib/utils";
import { useConsolePoll } from "@/hooks/use-console-poll";

const POLL_MS = 4000;

function runtimeLabel(rt: string): string {
  switch (rt) {
    case "running":
      return "运行中";
    case "paused":
      return "已暂停";
    case "agent_disconnected":
      return "Agent 断连";
    case "idle":
      return "空闲 / 未入场";
    default:
      return rt;
  }
}

export default function StrategyInstanceDetailPage() {
  const params = useParams();
  const id = String(params.id ?? "");
  const confirm = useConfirm();

  const [data, setData] = useState<StrategyInstanceDetailDTO | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [paramsJsonText, setParamsJsonText] = useState("");
  const [paramsSaved, setParamsSaved] = useState<string | null>(null);
  const paramsDirtyRef = useRef(false);

  const refresh = useCallback(async () => {
    try {
      const d = await fetchInstanceDetail(id);
      setData(d);
      setParamsJsonText((prev) => {
        if (paramsDirtyRef.current) return prev;
        return JSON.stringify(d.instance_params_json ?? {}, null, 2);
      });
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "加载失败");
    }
  }, [id]);

  const { lastUpdated, refreshNow } = useConsolePoll(refresh, POLL_MS);

  useEffect(() => {
    const h = setTimeout(() => {
      setParamsSaved(null);
      paramsDirtyRef.current = false;
    }, 0);
    return () => clearTimeout(h);
  }, [id]);

  async function saveParams() {
    if (!data) return;
    try {
      const parsed = JSON.parse(paramsJsonText) as Record<string, unknown>;
      await patchInstance(data.id, { params: parsed });
      paramsDirtyRef.current = false;
      await refresh();
      setParamsSaved("已保存实例参数");
    } catch (e) {
      setParamsSaved(
        e instanceof Error ? e.message : "JSON 无效或保存失败",
      );
    }
  }

  function exportConfig() {
    if (!data) return;
    const blob = new Blob([JSON.stringify(data, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `instance-${data.id}-export.json`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <ConsolePage
      title={data?.display_name ?? "实例详情"}
      description="模板参数与实例参数分区展示；底部为固定的编排动作区，避免误触。"
      meta={
        <DataFreshness
          lastUpdated={lastUpdated}
          onRefresh={refreshNow}
          hint={`每 ${POLL_MS / 1000}s 自动刷新 · 含 Agent 连通与最近命令`}
        />
      }
      actions={
        <Button variant="outline" size="sm" asChild>
          <Link href="/strategies/instances">
            <ArrowLeft className="mr-2 h-4 w-4" />
            返回列表
          </Link>
        </Button>
      }
    >
      {err ? (
        <ErrorState description={err} onRetry={() => void refreshNow()} />
      ) : null}
      {!data && !err ? <LoadingState label="加载实例…" /> : null}

      {data ? (
        <>
          {!data.agent_connected ? (
            <div className="rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm font-medium text-destructive">
              Agent 连接已断开：编排指令可能无法送达执行端。请检查 Agent 进程与其{" "}
              <code className="rounded bg-background px-1">identity.client_id</code>{" "}
              是否与实例 AgentKey 一致。
            </div>
          ) : null}

          <div className="grid gap-4 lg:grid-cols-3">
            <Card className="lg:col-span-2">
              <CardHeader>
                <CardTitle>配置摘要</CardTitle>
                <CardDescription>
                  调度侧 Status：
                  <code className="mx-1 rounded bg-muted px-1">{data.status}</code>
                  · 推导运行态：
                  <Badge variant="outline" className="ml-2">
                    {runtimeLabel(data.derived_runtime)}
                  </Badge>
                </CardDescription>
              </CardHeader>
              <CardContent className="grid gap-3 text-sm md:grid-cols-2">
                <div>
                  <div className="text-muted-foreground">模板</div>
                  <Link
                    href={`/strategies/templates/${data.template_id}`}
                    className="font-medium text-primary hover:underline"
                  >
                    {data.template_name}
                  </Link>
                  <div className="text-xs text-muted-foreground">{data.template_kind}</div>
                </div>
                <div>
                  <div className="text-muted-foreground">标的 / 市场 / 模式</div>
                  <div>
                    <code>{data.symbol}</code> · {data.market_kind} ·{" "}
                    <span className="capitalize">{data.run_mode}</span>
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">AgentKey</div>
                  <Link
                    href={`/agents/${encodeURIComponent(data.agent_key)}`}
                    className="break-all font-mono text-xs text-primary hover:underline"
                  >
                    {data.agent_key}
                  </Link>
                </div>
                <div>
                  <div className="text-muted-foreground">心跳</div>
                  <div className="text-xs">
                    {data.last_heartbeat_at || "暂无"}
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>风险视图</CardTitle>
                <CardDescription>由实例 ParamsJSON 推导的摘要</CardDescription>
              </CardHeader>
              <CardContent className="text-sm leading-relaxed text-muted-foreground">
                {data.risk_status}
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-base">快速跳转</CardTitle>
              <CardDescription>实例、模板、执行端与可观测性串联</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" asChild>
                <Link href={`/strategies/templates/${data.template_id}`}>模板详情</Link>
              </Button>
              <Button variant="outline" size="sm" asChild>
                <Link href={`/agents/${encodeURIComponent(data.agent_key)}`}>
                  Agent 控制台
                </Link>
              </Button>
              <Button variant="outline" size="sm" asChild>
                <Link href={`/commands?instance_id=${data.id}`}>命令流</Link>
              </Button>
              <Button variant="outline" size="sm" asChild>
                <Link
                  href={`/logs?instance_id=${data.id}&agent_key=${encodeURIComponent(data.agent_key)}`}
                >
                  审计日志
                </Link>
              </Button>
              <Button variant="outline" size="sm" asChild>
                <Link href="/accounts">资金视图</Link>
              </Button>
            </CardContent>
          </Card>

          <div className="grid gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>模板参数（只读）</CardTitle>
                <CardDescription>
                  来自 Strategy.ConfigJSON；修改请在模板库记录上进行（当前页不提供编辑以避免误改全局默认值）。
                </CardDescription>
              </CardHeader>
              <CardContent>
                <pre className="max-h-[280px] overflow-auto rounded-md border bg-muted/40 p-3 text-xs">
                  {JSON.stringify(data.template_config_json, null, 2)}
                </pre>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>实例参数（可编辑）</CardTitle>
                <CardDescription>
                  写入 Instance.ParamsJSON；保存后立即生效于风控摘要。
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-2">
                <textarea
                  className={cn(
                    "min-h-[220px] w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-xs",
                  )}
                  value={paramsJsonText}
                  onChange={(e) => {
                    paramsDirtyRef.current = true;
                    setParamsJsonText(e.target.value);
                  }}
                />
                <div className="flex flex-wrap gap-2">
                  <Button type="button" size="sm" variant="secondary" onClick={() => void saveParams()}>
                    保存实例参数
                  </Button>
                  {paramsSaved ? (
                    <span className="text-xs text-muted-foreground">{paramsSaved}</span>
                  ) : null}
                </div>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>运行时间轴</CardTitle>
              <CardDescription>源自审计表 resource=instance</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {data.timeline.length === 0 ? (
                <p className="text-sm text-muted-foreground">暂无记录</p>
              ) : (
                <ul className="space-y-2 text-sm">
                  {data.timeline.map((ev) => (
                    <li
                      key={`${ev.action}-${ev.occurred_at}`}
                      className="rounded-md border border-border/80 px-3 py-2"
                    >
                      <div className="flex flex-wrap justify-between gap-2">
                        <span className="font-medium">{ev.action}</span>
                        <span className="text-xs text-muted-foreground">
                          {new Date(ev.occurred_at).toLocaleString()}
                        </span>
                      </div>
                      {ev.payload ? (
                        <pre className="mt-1 max-h-24 overflow-auto text-[11px] text-muted-foreground">
                          {ev.payload}
                        </pre>
                      ) : null}
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>

          <div className="grid gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>最近命令</CardTitle>
                <CardDescription>
                  <Link
                    href={`/commands?instance_id=${data.id}`}
                    className="text-primary hover:underline"
                  >
                    在命令流页查看全部
                  </Link>
                </CardDescription>
              </CardHeader>
              <CardContent className="overflow-x-auto text-xs">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>状态</TableHead>
                      <TableHead>标的 / 意图</TableHead>
                      <TableHead>下发</TableHead>
                      <TableHead>ack</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.recent_commands.map((c) => (
                      <TableRow key={c.id}>
                        <TableCell>
                          {c.kind} · {c.status}
                          {c.error ? (
                            <div className="text-destructive">{c.error}</div>
                          ) : null}
                        </TableCell>
                        <TableCell className="max-w-[180px]">
                          <div>
                            <code>{c.symbol ?? data.symbol}</code>
                          </div>
                          <div
                            className="truncate text-muted-foreground"
                            title={c.intent ?? c.summary}
                          >
                            {c.intent || c.summary}
                          </div>
                        </TableCell>
                        <TableCell className="whitespace-nowrap text-muted-foreground">
                          {formatDateTime(c.dispatched_at || c.issued_at || c.created_at)}
                        </TableCell>
                        <TableCell className="whitespace-nowrap text-muted-foreground">
                          {c.acked_at ? formatDateTime(c.acked_at) : "—"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>最近回报 / 错误提示</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-xs">
                {data.recent_errors_hint ? (
                  <div className="rounded-md border border-destructive/40 bg-destructive/10 p-2 text-destructive">
                    {data.recent_errors_hint}
                  </div>
                ) : null}
                {data.recent_reports.map((r) => (
                  <div key={r.id} className="rounded border px-2 py-1">
                    <div className="font-medium">{r.type}</div>
                    <div className="text-muted-foreground">{r.summary}</div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>

          <div
            className={cn(
              "sticky bottom-0 z-10 mt-4 rounded-xl border border-border bg-card/95 p-4 shadow-lg backdrop-blur",
            )}
          >
            <div className="mb-3 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              编排动作（固定区域）
            </div>
            <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  onClick={async () => {
                    const isLive = data.run_mode === "live";
                    const ok = await confirm({
                      title: "启动编排",
                      description: isLive
                        ? "当前为实盘模式：启动后可能向交易所发送真实订单，请确认 Agent 在线、风控参数与账户余额。"
                        : "实例将标记为 active 并尝试拉起运行周期；请确认 Agent 与标的可用。",
                      confirmLabel: "启动编排",
                    });
                    if (!ok) return;
                    await instanceAction(data.id, "start");
                    await refresh();
                  }}
                >
                  启动编排
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="secondary"
                  onClick={async () => {
                    const isLive = data.run_mode === "live";
                    const ok = await confirm({
                      title: "停止编排",
                      description: isLive
                        ? "实盘停止：调度器将结束当前运行周期；请务必在交易所核对挂单与持仓。"
                        : "停止运行周期并标记 paused；未成交挂单请在交易所侧自行核对。",
                      confirmLabel: "停止编排",
                      destructive: true,
                    });
                    if (!ok) return;
                    await instanceAction(data.id, "stop");
                    await refresh();
                  }}
                >
                  停止编排
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={async () => {
                    const isLive = data.run_mode === "live";
                    const ok = await confirm({
                      title: isLive ? "暂停实盘编排" : "暂停编排",
                      description: isLive
                        ? "暂停后不再下发新意图；请同时在交易所核对未完成订单。"
                        : "暂停后调度器默认跳过该实例。",
                      confirmLabel: "暂停",
                    });
                    if (!ok) return;
                    await instanceAction(data.id, "pause");
                    await refresh();
                  }}
                >
                  暂停
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={async () => {
                    const isLive = data.run_mode === "live";
                    const ok = await confirm({
                      title: isLive ? "恢复实盘编排" : "恢复编排",
                      description: isLive
                        ? "恢复后将再次向 Agent 下发策略意图，可能影响真实资金。"
                        : "恢复运行周期；请确认市场数据与 Agent 可用。",
                      confirmLabel: "恢复",
                    });
                    if (!ok) return;
                    await instanceAction(data.id, "resume");
                    await refresh();
                  }}
                >
                  恢复
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={async () => {
                    const isLive = data.run_mode === "live";
                    const ok = await confirm({
                      title: "重启编排",
                      description: isLive
                        ? "实盘重启：等价于停止后立即启动，期间可能出现短暂敞口风险，请确认。"
                        : "等价于停止后立即启动；用于排障。",
                      confirmLabel: "重启",
                      destructive: true,
                    });
                    if (!ok) return;
                    await instanceAction(data.id, "restart");
                    await refresh();
                  }}
                >
                  重启编排
                </Button>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="secondary"
                  onClick={async () => {
                    const isLive = data.run_mode === "live";
                    const ok = await confirm({
                      title: "手动运行一轮 Step",
                      description: isLive
                        ? "实盘：本次 Tick 可能生成并下发真实交易意图，请明确知晓风险。"
                        : "触发单次编排 Tick（需要市场数据可读）；可能生成意图命令。",
                      confirmLabel: "运行一轮",
                    });
                    if (!ok) return;
                    await instanceAction(data.id, "run_once");
                    await refresh();
                  }}
                >
                  手动运行一轮
                </Button>
                <Button type="button" size="sm" variant="outline" onClick={exportConfig}>
                  导出配置 JSON
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  onClick={async () => {
                    const ok = await confirm({
                      title: "结束实例（软删除）",
                      description:
                        "该操作将终止运行记录并软删除实例行，历史审计仍可追溯。实盘场景下请在交易所自行核对仓位与挂单；此操作不可从前端撤销。",
                      confirmLabel: "确认结束实例",
                      destructive: true,
                      cancelLabel: "取消",
                    });
                    if (!ok) return;
                    await instanceAction(data.id, "terminate");
                    window.location.href = "/strategies/instances";
                  }}
                >
                  结束实例
                </Button>
              </div>
            </div>
          </div>
        </>
      ) : null}
    </ConsolePage>
  );
}
