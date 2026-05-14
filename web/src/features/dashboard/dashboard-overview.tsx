"use client";

import {
  Activity,
  AlertTriangle,
  Cpu,
  Database,
  LayoutDashboard,
  Radio,
  RefreshCw,
  ScrollText,
  Server,
  Waypoints,
} from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { ApiError } from "@/api/errors";
import { fetchSystemDashboard } from "@/api/system";
import { EmptyState } from "@/components/feedback/empty-state";
import { StatusTag } from "@/components/status-tag";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { publicEnv } from "@/lib/env";
import { cn } from "@/lib/utils";
import type { SystemStatusKind } from "@/types/console";
import type {
  ActivityKind,
  ConnectionHealth,
  SystemDashboardDTO,
} from "@/types/system";

const POLL_MS = 12_000;

const statusLabelZh: Record<SystemStatusKind, string> = {
  online: "在线",
  offline: "离线",
  running: "运行中",
  paused: "降级/暂停",
  failed: "失败",
  idle: "空闲",
};

function mapHealthToSystemStatus(health: ConnectionHealth): SystemStatusKind {
  switch (health) {
    case "up":
      return "online";
    case "degraded":
      return "paused";
    case "down":
      return "offline";
    default:
      return "idle";
  }
}

const activityKindLabel: Record<ActivityKind, string> = {
  login: "登录",
  instance_start: "实例",
  command: "命令",
  report: "回报",
  backtest_complete: "回测",
  system: "系统",
};

const channelIcon = {
  api: Server,
  websocket: Radio,
  database: Database,
  backtest: Cpu,
} as const;

export function DashboardOverview() {
  const [data, setData] = useState<SystemDashboardDTO | null>(null);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [lastOkAt, setLastOkAt] = useState<Date | null>(null);
  const [booting, setBooting] = useState(true);
  const [manualRefreshing, setManualRefreshing] = useState(false);

  const load = useCallback(async (opts?: { manual?: boolean }) => {
    if (opts?.manual) setManualRefreshing(true);
    setFetchError(null);
    try {
      const next = await fetchSystemDashboard();
      setData(next);
      setLastOkAt(new Date());
    } catch (e) {
      const msg =
        e instanceof ApiError
          ? e.message
          : "数据刷新失败：已保留上一份快照，请检查网络与 BFF。";
      setFetchError(msg);
    } finally {
      setBooting(false);
      if (opts?.manual) setManualRefreshing(false);
    }
  }, []);

  useEffect(() => {
    void Promise.resolve().then(() => load());
    const id = window.setInterval(() => void load(), POLL_MS);
    return () => window.clearInterval(id);
  }, [load]);

  const overview = data?.overview;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 rounded-lg border border-border/80 bg-card/60 px-4 py-3 backdrop-blur-sm sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
          <LayoutDashboard className="h-4 w-4 shrink-0 text-foreground" />
          <span>
            连接与依赖状态条 · API / WS / DB / 回测{" "}
            <span className="text-xs opacity-80">
              （轮询 {POLL_MS / 1000}s；失败不阻断页面）
            </span>
          </span>
          {publicEnv.useMockApi ? (
            <span className="rounded-full border border-amber-500/40 bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-900 dark:text-amber-100">
              Mock 数据（NEXT_PUBLIC_USE_MOCK_API）
            </span>
          ) : null}
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <p className="text-xs text-muted-foreground">
            数据更新时间：
            {lastOkAt
              ? lastOkAt.toLocaleString()
              : booting
                ? "加载中…"
                : "暂无"}
          </p>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="gap-1.5"
            disabled={manualRefreshing}
            onClick={() => void load({ manual: true })}
          >
            <RefreshCw
              className={cn(
                "h-3.5 w-3.5",
                manualRefreshing ? "animate-spin" : "",
              )}
            />
            刷新数据
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap gap-2 border-b border-dashed border-border pb-3">
        {(data?.connections ?? []).map((c) => {
          const Icon = channelIcon[c.channel];
          const status = mapHealthToSystemStatus(c.status);
          return (
            <div
              key={c.channel}
              className="flex min-w-[220px] flex-1 items-center gap-2 rounded-md border border-border/70 bg-muted/30 px-3 py-2"
            >
              <Icon className="h-4 w-4 text-muted-foreground" />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 text-xs font-medium text-foreground">
                  <span className="truncate">{c.label}</span>
                  <StatusTag status={status} className="shrink-0" />
                </div>
                {c.detail ? (
                  <p className="mt-0.5 truncate text-[11px] text-muted-foreground">
                    {c.detail}
                  </p>
                ) : null}
              </div>
            </div>
          );
        })}
        {!data?.connections?.length && !booting ? (
          <p className="text-sm text-muted-foreground">
            暂无连接项数据，请确认接口或开启 Mock。
          </p>
        ) : null}
      </div>

      {fetchError ? (
        <div
          role="alert"
          className="flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-950 dark:text-amber-50"
        >
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
          <div>
            <div className="font-medium">本次刷新遇到问题</div>
            <p className="mt-1 text-xs opacity-90">{fetchError}</p>
          </div>
        </div>
      ) : null}

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Card className="border-border/80">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">SaaS 控制面</CardTitle>
            <CardDescription>
              编排、账户与策略模板所在服务；需与 Agent 进程版本匹配。
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm text-muted-foreground">运行状态</span>
              {overview ? (
                <StatusTag status={overview.saas.status} />
              ) : (
                <span className="text-xs text-muted-foreground">等待数据…</span>
              )}
            </div>
            {overview?.saas.version ? (
              <p className="text-xs text-muted-foreground">
                标识：{overview.saas.version}
              </p>
            ) : null}
            {overview?.saas.detail ? (
              <p className="text-xs leading-relaxed text-muted-foreground">
                {overview.saas.detail}
              </p>
            ) : null}
          </CardContent>
        </Card>

        <Card className="border-border/80">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Agent 在线</CardTitle>
            <CardDescription>已连接控制面的执行进程数量（示例字段）。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <p className="text-3xl font-semibold tabular-nums">
              {overview ? overview.agentsOnline : "—"}{" "}
              <span className="text-base font-normal text-muted-foreground">
                / {overview?.agentsTotal ?? "—"} 总计
              </span>
            </p>
            <p className="text-xs text-muted-foreground">
              若为 0：请在本机或服务器启动{" "}
              <code className="rounded bg-muted px-1 py-0.5 text-[11px]">
                go run ./cmd/agent
              </code>{" "}
              并检查与 SaaS 的连通。
            </p>
            <Button type="button" variant="outline" size="sm" asChild>
              <Link href="/agents">查看 Agents</Link>
            </Button>
          </CardContent>
        </Card>

        <Card className="border-border/80">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">活跃策略实例</CardTitle>
            <CardDescription>当前处于运行/暂停等活跃态的实例数。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <p className="text-3xl font-semibold tabular-nums">
              {overview?.activeStrategyInstances ?? "—"}
            </p>
            {overview && overview.activeStrategyInstances === 0 ? (
              <EmptyState
                className="border-none bg-transparent p-0 shadow-none"
                title="尚未启动实例"
                description="从模板创建实例并关联账户后，此处会显示活跃数量。"
                action={
                  <div className="flex flex-wrap gap-2">
                    <Button type="button" size="sm" asChild>
                      <Link href="/strategies/templates">去创建模板</Link>
                    </Button>
                    <Button type="button" size="sm" variant="outline" asChild>
                      <Link href="/strategies/instances">管理实例</Link>
                    </Button>
                  </div>
                }
              />
            ) : null}
            {overview && overview.activeStrategyInstances > 0 ? (
              <Button type="button" variant="outline" size="sm" asChild>
                <Link href="/strategies/instances">实例列表</Link>
              </Button>
            ) : null}
            {!overview && !booting ? (
              <p className="text-xs text-muted-foreground">
                暂无法读取实例统计，请检查上方错误提示或稍后重试。
              </p>
            ) : null}
          </CardContent>
        </Card>

        <Card className="border-border/80">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">当前持仓</CardTitle>
            <CardDescription>汇总占位：来自执行端的仓位与名义敞口。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm text-muted-foreground">状态</span>
              {overview ? (
                <StatusTag status={overview.positions.status} />
              ) : (
                <span className="text-xs text-muted-foreground">等待数据…</span>
              )}
            </div>
            <p className="text-sm">{overview?.positions.summary ?? "—"}</p>
            <p className="text-xs tabular-nums text-muted-foreground">
              开仓数：{overview?.positions.openCount ?? "—"}
            </p>
          </CardContent>
        </Card>

        <Card className="border-border/80">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">最近一次命令</CardTitle>
            <CardDescription>下发给 Agent 的最新控制指令摘要。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <StatusTag
                status={overview?.lastCommand.status ?? "idle"}
                label={
                  statusLabelZh[overview?.lastCommand.status ?? "idle"] ?? "空闲"
                }
              />
              <span className="text-xs text-muted-foreground">
                {overview?.lastCommand.atLabel ?? "—"}
              </span>
            </div>
            <p className="text-sm">{overview?.lastCommand.summary ?? "—"}</p>
          </CardContent>
        </Card>

        <Card className="border-border/80">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">最近一次回报</CardTitle>
            <CardDescription>Agent 上报的执行/持仓/错误类摘要。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <StatusTag
                status={overview?.lastReport.status ?? "idle"}
                label={
                  statusLabelZh[overview?.lastReport.status ?? "idle"] ?? "空闲"
                }
              />
              <span className="text-xs text-muted-foreground">
                {overview?.lastReport.atLabel ?? "—"}
              </span>
            </div>
            <p className="text-sm">{overview?.lastReport.summary ?? "—"}</p>
          </CardContent>
        </Card>
      </div>

      <Card className="border-border/80">
        <CardHeader className="flex flex-col gap-2 space-y-0 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-base">
              <Activity className="h-4 w-4 text-muted-foreground" />
              最近活动
            </CardTitle>
            <CardDescription>
              登录、实例启动、命令、回报与回测完成等事件流（数据来自 BFF / Mock）。
            </CardDescription>
          </div>
          <Button type="button" variant="ghost" size="sm" asChild>
            <Link href="/logs" className="gap-1">
              <ScrollText className="h-3.5 w-3.5" />
              查看日志
            </Link>
          </Button>
        </CardHeader>
        <CardContent>
          {data?.activities?.length ? (
            <ul className="space-y-3">
              {data.activities.map((item) => (
                <li
                  key={item.id}
                  className="flex gap-3 rounded-md border border-border/60 bg-muted/20 px-3 py-2"
                >
                  <Waypoints className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                  <div className="min-w-0 flex-1 space-y-1">
                    <div className="flex flex-wrap items-center gap-2 text-sm font-medium">
                      <span className="rounded bg-muted px-1.5 py-0 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                        {activityKindLabel[item.kind]}
                      </span>
                      <span>{item.title}</span>
                    </div>
                    {item.description ? (
                      <p className="text-xs text-muted-foreground">
                        {item.description}
                      </p>
                    ) : null}
                    <p className="text-[11px] text-muted-foreground">
                      {item.atLabel}
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState
              title="还没有活动记录"
              description="启动 Agent、创建策略实例或运行回测后，事件会出现在这里。"
              action={
                <div className="flex flex-wrap gap-2">
                  <Button type="button" size="sm" asChild>
                    <Link href="/strategies/templates">创建策略模板</Link>
                  </Button>
                  <Button type="button" size="sm" variant="outline" asChild>
                    <Link href="/agents">连接 Agent</Link>
                  </Button>
                </div>
              }
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
