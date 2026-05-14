"use client";

import {
  AlertTriangle,
  ArrowLeft,
  Droplets,
  Link2,
  RefreshCw,
} from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";

import { fetchAgentDetail, postAgentControl } from "@/api/agents";
import { ApiError } from "@/api/errors";
import { useConfirm } from "@/components/feedback/confirm-provider";
import { ConsolePage } from "@/components/layout/console-page";
import { ErrorState } from "@/components/feedback/error-state";
import { LoadingState } from "@/components/feedback/loading-state";
import { StatusTag } from "@/components/status-tag";
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
import { publicEnv } from "@/lib/env";
import { formatDateTime, formatRelativeShort } from "@/lib/format-trading";
import { cn } from "@/lib/utils";
import type { ConsoleEnvKind, SystemStatusKind } from "@/types/console";
import type {
  AgentControlActionDTO,
  AgentDetailDTO,
  AgentExecModeKind,
  AgentWsStatusKind,
} from "@/types/agents";

const execLabels: Record<AgentExecModeKind, string> = {
  spot: "现货",
  futures: "合约",
  mixed: "混合",
};

const envLabels: Record<ConsoleEnvKind, string> = {
  development: "开发",
  staging: "预发",
  production: "生产",
  local: "本地",
};

function wsStatusToSystem(ws: AgentWsStatusKind): SystemStatusKind {
  switch (ws) {
    case "connected":
      return "online";
    case "connecting":
      return "paused";
    case "error":
      return "failed";
    default:
      return "offline";
  }
}

const wsLabels: Record<AgentWsStatusKind, string> = {
  connected: "WS 已连接",
  connecting: "WS 连接中",
  disconnected: "WS 断开",
  error: "WS 异常",
};

function commandStatusBadge(status: string) {
  switch (status) {
    case "failed":
      return (
        <Badge variant="outline" className="border-destructive/50 text-destructive">
          失败
        </Badge>
      );
    case "pending":
      return <Badge variant="outline">等待</Badge>;
    case "accepted":
      return (
        <Badge variant="outline" className="border-sky-500/40 text-sky-700 dark:text-sky-300">
          已接受
        </Badge>
      );
    case "completed":
      return (
        <Badge variant="outline" className="border-emerald-500/40 text-emerald-800 dark:text-emerald-200">
          完成
        </Badge>
      );
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

function reportSeverityRow(sev: string) {
  if (sev === "error")
    return "border-l-4 border-l-destructive bg-destructive/5";
  if (sev === "warn")
    return "border-l-4 border-l-amber-500 bg-amber-500/10";
  return "border-l-4 border-l-border bg-muted/20";
}

export function AgentDetailConsolePage() {
  const params = useParams<{ id: string }>();
  const rawId = params?.id ?? "";
  const agentId = decodeURIComponent(rawId);
  const confirm = useConfirm();

  const [detail, setDetail] = useState<AgentDetailDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [actionKey, setActionKey] = useState<string | null>(null);

  const load = useCallback(async (manual?: boolean) => {
    if (!agentId) return;
    if (manual) setRefreshing(true);
    setError(null);
    try {
      const d = await fetchAgentDetail(agentId);
      setDetail(d);
    } catch (e) {
      setDetail(null);
      setError(e instanceof ApiError ? e : new ApiError("加载失败", 0));
    } finally {
      setLoading(false);
      if (manual) setRefreshing(false);
    }
  }, [agentId]);

  useEffect(() => {
    let cancelled = false;
    void Promise.resolve().then(() => {
      if (!cancelled) void load();
    });
    return () => {
      cancelled = true;
    };
  }, [load]);

  const runControl = async (action: AgentControlActionDTO) => {
    const labels: Record<AgentControlActionDTO, string> = {
      start: "启动 Agent 进程并将实例纳入编排？",
      stop: "停止 Agent（可能影响挂单与持仓同步）？",
      reconnect: "强制断开并重连 SaaS WebSocket？",
      refresh: "刷新账户与持仓快照（拉取交易所状态）？",
    };
    const ok = await confirm({
      title: `确认执行：${action}`,
      description: labels[action],
      confirmLabel: "执行",
      destructive: action === "stop",
    });
    if (!ok) return;

    const busyKey = action;
    setActionKey(busyKey);
    try {
      const res = await postAgentControl(agentId, action);
      if (res.ok) {
        toast.success(res.message);
        await load();
      } else {
        toast.error(res.message);
      }
    } catch (e) {
      const msg =
        e instanceof ApiError ? e.message : "请求失败";
      toast.error(msg);
    } finally {
      setActionKey(null);
    }
  };

  const permissionDenied = error?.status === 403;

  if (!agentId) {
    return (
      <ConsolePage title="Agent 详情" description="无效的链接。">
        <ErrorState title="缺少 Agent ID" />
      </ConsolePage>
    );
  }

  if (loading) {
    return (
      <ConsolePage title="Agent 详情" description={agentId}>
        <LoadingState label="加载详情…" />
      </ConsolePage>
    );
  }

  if (error || !detail) {
    return (
      <ConsolePage title="Agent 详情" description={agentId}>
        <ErrorState
          title={
            permissionDenied
              ? "权限不足"
              : error?.status === 404
                ? "未找到 Agent"
                : "加载失败"
          }
          description={error?.message ?? "未知错误"}
          onRetry={() => void load()}
        />
        <Button type="button" variant="outline" size="sm" className="mt-4 gap-1" asChild>
          <Link href="/agents">
            <ArrowLeft className="h-3.5 w-3.5" />
            返回列表
          </Link>
        </Button>
      </ConsolePage>
    );
  }

  const { agent, connection, heartbeatHistory, boundInstances, recentCommands, recentReports } =
    detail;

  return (
    <ConsolePage
      title={agent.displayName}
      description={`${agent.id} · ${envLabels[agent.environment]} · ${execLabels[agent.execMode]}`}
      actions={
        <div className="flex flex-wrap items-center gap-2">
          {publicEnv.useMockApi ? (
            <Badge variant="outline" className="text-[11px]">
              Mock API
            </Badge>
          ) : null}
          <Button type="button" variant="outline" size="sm" asChild>
            <Link href="/agents" className="gap-1">
              <ArrowLeft className="h-3.5 w-3.5" />
              列表
            </Link>
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={refreshing}
            className="gap-1"
            onClick={() => void load(true)}
          >
            <RefreshCw className={cn("h-3.5 w-3.5", refreshing ? "animate-spin" : "")} />
            刷新
          </Button>
          <Button
            type="button"
            size="sm"
            disabled={!!actionKey}
            onClick={() => void runControl("start")}
          >
            {actionKey === "start" ? "执行中…" : "Start"}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="destructive"
            disabled={!!actionKey}
            onClick={() => void runControl("stop")}
          >
            {actionKey === "stop" ? "执行中…" : "Stop"}
          </Button>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            disabled={!!actionKey}
            onClick={() => void runControl("reconnect")}
          >
            {actionKey === "reconnect" ? "执行中…" : "Reconnect"}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!!actionKey}
            onClick={() => void runControl("refresh")}
          >
            {actionKey === "refresh" ? "执行中…" : "Refresh"}
          </Button>
        </div>
      }
    >
      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="border-border/80 lg:col-span-2">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">运行状态</CardTitle>
            <CardDescription>在线、心跳与 WS；详情快照 {formatDateTime(detail.generatedAt)}</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-3">
            <div className="flex flex-wrap items-center gap-2">
              <StatusTag
                status={agent.isOnline ? "online" : "offline"}
                label={agent.isOnline ? "在线" : "离线"}
              />
              <StatusTag
                status={wsStatusToSystem(agent.wsStatus)}
                label={wsLabels[agent.wsStatus]}
              />
              {agent.hasLiquidityWarning ? (
                <Badge
                  variant="outline"
                  className="gap-0.5 border-amber-500/50 bg-amber-500/15 text-amber-950 dark:text-amber-100"
                >
                  <Droplets className="h-3 w-3" />
                  低流动性
                </Badge>
              ) : null}
            </div>
            <div className="min-w-[200px] flex-1 rounded-md border border-border/80 bg-muted/20 px-3 py-2 text-sm">
              <div className="text-muted-foreground">最近心跳</div>
              <div className="font-medium tabular-nums">
                {formatRelativeShort(agent.lastHeartbeatAt)}{" "}
                <span className="text-xs font-normal text-muted-foreground">
                  ({formatDateTime(agent.lastHeartbeatAt)})
                </span>
              </div>
            </div>
            <div className="min-w-[200px] flex-1 rounded-md border border-border/80 bg-muted/20 px-3 py-2 text-sm">
              <div className="text-muted-foreground">绑定实例</div>
              <div className="font-semibold tabular-nums">{agent.instanceCount}</div>
            </div>
            {agent.lastError ? (
              <div className="flex w-full items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                <span>{agent.lastError}</span>
              </div>
            ) : null}
          </CardContent>
        </Card>

        <Card className="border-border/80">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-base">
              <Link2 className="h-4 w-4" />
              连接信息
            </CardTitle>
            <CardDescription>SaaS / WS / 版本（不含密钥）</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div>
              <div className="text-xs text-muted-foreground">API</div>
              <div className="break-all font-mono text-xs">{connection.saasApiEndpoint}</div>
            </div>
            <div>
              <div className="text-xs text-muted-foreground">WebSocket</div>
              <div className="break-all font-mono text-xs">{connection.wsEndpoint}</div>
            </div>
            <div className="flex flex-wrap gap-2 pt-1">
              <Badge variant="outline">v{connection.agentVersion}</Badge>
              <Badge variant="outline">{connection.tlsEnabled ? "TLS" : "明文"}</Badge>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="border-border/80">
        <CardHeader className="pb-2">
          <CardTitle className="text-base">心跳历史</CardTitle>
          <CardDescription>最近采样；失败点用于快速定位抖动</CardDescription>
        </CardHeader>
        <CardContent className="px-0 sm:px-6">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>时间</TableHead>
                <TableHead>延迟</TableHead>
                <TableHead>结果</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {heartbeatHistory.map((h, idx) => (
                <TableRow key={`${h.at}-${idx}`}>
                  <TableCell className="tabular-nums text-muted-foreground">
                    {formatDateTime(h.at)}
                  </TableCell>
                  <TableCell className="tabular-nums">
                    {h.latencyMs != null ? `${h.latencyMs} ms` : "—"}
                  </TableCell>
                  <TableCell>
                    {h.ok ? (
                      <Badge variant="outline" className="border-emerald-500/40 text-emerald-800 dark:text-emerald-200">
                        OK
                      </Badge>
                    ) : (
                      <Badge variant="outline" className="border-destructive/50 text-destructive">
                        FAIL
                      </Badge>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="border-border/80">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">当前绑定实例</CardTitle>
          </CardHeader>
          <CardContent className="px-0 sm:px-6">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>标的</TableHead>
                  <TableHead>状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {boundInstances.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={3} className="text-muted-foreground">
                      暂无绑定实例
                    </TableCell>
                  </TableRow>
                ) : (
                  boundInstances.map((b) => (
                    <TableRow key={b.id}>
                      <TableCell className="font-mono text-xs">{b.id}</TableCell>
                      <TableCell>{b.symbolLabel}</TableCell>
                      <TableCell>{b.statusLabel}</TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card className="border-border/80">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">最近执行命令</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {recentCommands.map((c) => (
              <div
                key={c.id}
                className="rounded-md border border-border/70 bg-muted/15 px-3 py-2 text-sm"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-[11px] text-muted-foreground">{c.kind}</span>
                  {commandStatusBadge(c.status)}
                  <span className="text-[11px] text-muted-foreground">
                    {formatDateTime(c.at)}
                  </span>
                </div>
                <div className="mt-1">{c.summary}</div>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>

      <Card className="border-border/80">
        <CardHeader className="pb-2">
          <CardTitle className="text-base">最近回报</CardTitle>
          <CardDescription>Agent → SaaS 推送摘要</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {recentReports.map((r) => (
            <div
              key={r.id}
              className={cn("rounded-md px-3 py-2 text-sm", reportSeverityRow(r.severity))}
            >
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-mono text-[11px]">{r.kind}</span>
                <span className="text-[11px] text-muted-foreground">{formatDateTime(r.at)}</span>
              </div>
              <div className="mt-1">{r.summary}</div>
            </div>
          ))}
        </CardContent>
      </Card>
    </ConsolePage>
  );
}
