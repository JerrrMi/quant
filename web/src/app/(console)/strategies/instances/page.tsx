"use client";

import Link from "next/link";
import { useState, useCallback } from "react";
import { Boxes, Plus } from "lucide-react";

import { fetchInstances, instanceAction } from "@/api/instances";
import { ConsolePage } from "@/components/layout/console-page";
import { DataFreshness } from "@/components/layout/data-freshness";
import { EmptyState } from "@/components/feedback/empty-state";
import { ErrorState } from "@/components/feedback/error-state";
import { LoadingState } from "@/components/feedback/loading-state";
import { useConfirm } from "@/components/feedback/confirm-provider";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { StrategyInstanceRowDTO } from "@/types/strategies";
import { cn } from "@/lib/utils";
import { useConsolePoll } from "@/hooks/use-console-poll";

const POLL_MS = 4000;

function runtimeBadgeClass(rt: string): string {
  switch (rt) {
    case "running":
      return "border-sky-500/40 bg-sky-500/10 text-sky-800 dark:text-sky-200";
    case "paused":
      return "border-amber-500/40 bg-amber-500/10 text-amber-900 dark:text-amber-100";
    case "agent_disconnected":
      return "border-destructive/40 bg-destructive/10 text-destructive";
    case "idle":
    default:
      return "border-border bg-muted text-muted-foreground";
  }
}

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

export default function StrategyInstancesPage() {
  const confirm = useConfirm();
  const [rows, setRows] = useState<StrategyInstanceRowDTO[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await fetchInstances();
      setRows(data.instances);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "加载失败");
    }
  }, []);

  const { lastUpdated, refreshNow } = useConsolePoll(refresh, POLL_MS);

  async function doAction(
    id: number,
    action: "start" | "stop" | "pause" | "resume" | "restart",
  ) {
    await instanceAction(id, action);
    await refresh();
  }

  return (
    <ConsolePage
      title="策略实例"
      description="实例绑定模板、标的与 Agent；编排动作可能影响实盘资金，请谨慎操作。"
      meta={
        <DataFreshness
          lastUpdated={lastUpdated}
          onRefresh={refreshNow}
          hint={`每 ${POLL_MS / 1000}s 自动刷新 · 心跳与 Agent 连通性联动`}
        />
      }
      actions={
        <Button size="sm" asChild>
          <Link href="/strategies/instances/new">
            <Plus className="mr-2 h-4 w-4" />
            新建实例
          </Link>
        </Button>
      }
    >
      {err ? (
        <ErrorState
          title="无法加载实例"
          description={`${err} · 请确认 SaaS 进程已启动且 SAAS_CONSOLE_ORIGIN 配置正确。`}
          onRetry={() => void refresh()}
        />
      ) : null}

      {rows === null && !err ? <LoadingState label="加载实例列表…" /> : null}

      {rows && rows.length === 0 ? (
        <EmptyState
          icon={Boxes}
          title="暂无策略实例"
          description="通过向导创建实例后会在此列出；模板来自后端目录，可随时扩展。"
        />
      ) : null}

      {rows && rows.length > 0 ? (
        <div className="rounded-lg border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>实例</TableHead>
                <TableHead>模板</TableHead>
                <TableHead>标的</TableHead>
                <TableHead>市场</TableHead>
                <TableHead>运行态</TableHead>
                <TableHead>Agent</TableHead>
                <TableHead>最近命令</TableHead>
                <TableHead>最近回报</TableHead>
                <TableHead>风险</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow
                  key={row.id}
                  className={cn(
                    row.derived_runtime === "agent_disconnected" &&
                      "bg-destructive/5",
                  )}
                >
                  <TableCell>
                    <Link
                      href={`/strategies/instances/${row.id}`}
                      className="font-medium text-primary hover:underline"
                    >
                      {row.display_name}
                    </Link>
                    <div className="text-[11px] text-muted-foreground">
                      #{row.id} · {row.run_mode}
                    </div>
                  </TableCell>
                  <TableCell className="max-w-[140px] truncate text-sm">
                    {row.template_name}
                  </TableCell>
                  <TableCell>
                    <code className="text-xs">{row.symbol}</code>
                  </TableCell>
                  <TableCell>{row.market_kind}</TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <Badge
                        variant="outline"
                        className={cn(
                          "w-fit rounded-full px-2 py-0 text-[10px] font-semibold uppercase",
                          runtimeBadgeClass(row.derived_runtime),
                        )}
                      >
                        {runtimeLabel(row.derived_runtime)}
                      </Badge>
                      {!row.agent_connected ? (
                        <span className="text-[11px] font-medium text-destructive">
                          Agent 未连接 · 检查密钥端 WS
                        </span>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell className="max-w-[120px] truncate text-xs">
                    <Link
                      href={`/agents/${encodeURIComponent(row.agent_key)}`}
                      className="text-primary hover:underline"
                    >
                      {row.agent_key}
                    </Link>
                  </TableCell>
                  <TableCell className="max-w-[180px] truncate text-[11px] text-muted-foreground">
                    {row.last_command_summary || "—"}
                  </TableCell>
                  <TableCell className="max-w-[180px] truncate text-[11px] text-muted-foreground">
                    {row.last_report_summary || "—"}
                  </TableCell>
                  <TableCell className="max-w-[160px] text-[11px] leading-snug text-muted-foreground">
                    {row.risk_status}
                  </TableCell>
                  <TableCell className="text-right">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="outline" size="sm">
                          控制
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-52">
                        <DropdownMenuItem asChild>
                          <Link href={`/strategies/instances/${row.id}`}>
                            查看详情
                          </Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem asChild>
                          <Link href={`/commands?instance_id=${row.id}`}>
                            命令流（本实例）
                          </Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem asChild>
                          <Link
                            href={`/logs?instance_id=${row.id}&agent_key=${encodeURIComponent(row.agent_key)}`}
                          >
                            审计日志
                          </Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={async () => {
                            const ok = await confirm({
                              title: "启动编排",
                              description:
                                row.run_mode === "live"
                                  ? "实盘模式：启动后将连接执行端并可能产生真实订单，请再次确认风险敞口与 Agent 在线。"
                                  : "实例将进入 active 状态并尝试恢复运行周期（取决于调度器与市场数据）。确认启动？",
                              confirmLabel: "启动编排",
                            });
                            if (!ok) return;
                            await doAction(row.id, "start");
                          }}
                        >
                          启动编排
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={async () => {
                            const ok = await confirm({
                              title:
                                row.run_mode === "live" ? "暂停实盘编排" : "暂停编排",
                              description:
                                row.run_mode === "live"
                                  ? "实盘实例暂停后，调度器停止下发新意图；请同时在交易所核对未完成挂单与持仓。"
                                  : "暂停后调度器默认跳过该实例；未完成风控指令仍需人工核对。",
                              confirmLabel: "暂停",
                            });
                            if (!ok) return;
                            await doAction(row.id, "pause");
                          }}
                        >
                          暂停编排
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={async () => {
                            const ok = await confirm({
                              title: "停止编排",
                              description:
                                "停止后将关闭运行周期记录（stopped）；恢复需再次启动。",
                              confirmLabel: "停止编排",
                              destructive: true,
                            });
                            if (!ok) return;
                            await doAction(row.id, "stop");
                          }}
                        >
                          停止编排
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={async () => {
                            const ok = await confirm({
                              title:
                                row.run_mode === "live"
                                  ? "恢复实盘编排"
                                  : "恢复编排",
                              description:
                                row.run_mode === "live"
                                  ? "恢复后调度器将继续向已连接的 Agent 下发策略意图，可能影响交易所挂单与持仓，请确认。"
                                  : "恢复运行周期；请确认市场数据与 Agent 可用。",
                              confirmLabel: "恢复编排",
                            });
                            if (!ok) return;
                            await doAction(row.id, "resume");
                          }}
                        >
                          恢复编排
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={async () => {
                            const ok = await confirm({
                              title: "重启编排",
                              description:
                                "先停止当前运行周期再立即重新拉起；用于排障或切换参数后的快速复位。",
                              confirmLabel: "重启",
                              destructive: true,
                            });
                            if (!ok) return;
                            await doAction(row.id, "restart");
                          }}
                        >
                          重启编排
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}
    </ConsolePage>
  );
}
