"use client";

import {
  AlertTriangle,
  ArrowDownAZ,
  Droplets,
  Filter,
  RefreshCw,
  Search,
  Wifi,
} from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";

import { fetchAgentsList } from "@/api/agents";
import { ApiError } from "@/api/errors";
import { ConsolePage } from "@/components/layout/console-page";
import { EmptyState } from "@/components/feedback/empty-state";
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
import { Input } from "@/components/ui/input";
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
  AgentExecModeKind,
  AgentListRowDTO,
  AgentWsStatusKind,
} from "@/types/agents";

const POLL_MS = 15_000;

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

type SortKey = "name" | "heartbeat" | "instances";

export function AgentsConsolePage() {
  const [rows, setRows] = useState<AgentListRowDTO[]>([]);
  const [generatedAt, setGeneratedAt] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const [search, setSearch] = useState("");
  const [envFilter, setEnvFilter] = useState<ConsoleEnvKind | "all">("all");
  const [onlineOnly, setOnlineOnly] = useState(false);
  const [wsFilter, setWsFilter] = useState<AgentWsStatusKind | "all">("all");
  const [sortKey, setSortKey] = useState<SortKey>("heartbeat");

  const load = useCallback(async (manual?: boolean) => {
    if (manual) setRefreshing(true);
    setError(null);
    try {
      const data = await fetchAgentsList();
      setRows(data.agents);
      setGeneratedAt(data.generatedAt);
    } catch (e) {
      setError(e instanceof ApiError ? e : new ApiError("加载失败", 0));
    } finally {
      setLoading(false);
      if (manual) setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    void Promise.resolve().then(() => {
      if (!cancelled) void load();
    });
    const id = window.setInterval(() => {
      if (!cancelled) void load();
    }, POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [load]);

  const filtered = useMemo(() => {
    let list = [...rows];
    const q = search.trim().toLowerCase();
    if (q) {
      list = list.filter(
        (r) =>
          r.displayName.toLowerCase().includes(q) ||
          r.id.toLowerCase().includes(q),
      );
    }
    if (envFilter !== "all") {
      list = list.filter((r) => r.environment === envFilter);
    }
    if (onlineOnly) {
      list = list.filter((r) => r.isOnline);
    }
    if (wsFilter !== "all") {
      list = list.filter((r) => r.wsStatus === wsFilter);
    }
    list.sort((a, b) => {
      switch (sortKey) {
        case "name":
          return a.displayName.localeCompare(b.displayName);
        case "instances":
          return b.instanceCount - a.instanceCount;
        case "heartbeat":
        default: {
          const ta = a.lastHeartbeatAt
            ? new Date(a.lastHeartbeatAt).getTime()
            : 0;
          const tb = b.lastHeartbeatAt
            ? new Date(b.lastHeartbeatAt).getTime()
            : 0;
          return tb - ta;
        }
      }
    });
    return list;
  }, [rows, search, envFilter, onlineOnly, wsFilter, sortKey]);

  const permissionDenied = error?.status === 403;

  return (
    <ConsolePage
      title="Agents"
      description="执行节点运行状态 · 心跳、WS、实例绑定与最近异常（数据来自统一 API 层）。"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          {publicEnv.useMockApi ? (
            <Badge variant="outline" className="text-[11px]">
              Mock API
            </Badge>
          ) : null}
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="gap-1.5"
            disabled={refreshing || loading}
            onClick={() => void load(true)}
          >
            <RefreshCw
              className={cn("h-3.5 w-3.5", refreshing ? "animate-spin" : "")}
            />
            刷新
          </Button>
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        <Card className="border-border/80 bg-card/40">
          <CardHeader className="pb-3">
            <CardTitle className="text-base">筛选与排序</CardTitle>
            <CardDescription>
              搜索名称或 ID；可按环境与 WS 状态过滤。快照时间：
              {generatedAt ? formatDateTime(generatedAt) : "—"}
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-end">
            <div className="relative min-w-[200px] flex-1">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                className="pl-9"
                placeholder="搜索名称或 Agent ID…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                aria-label="搜索 Agent"
              />
            </div>
            <div className="flex flex-wrap gap-2">
              <div className="flex items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1">
                <Filter className="h-3.5 w-3.5 text-muted-foreground" />
                <select
                  className="bg-transparent text-sm outline-none"
                  value={envFilter}
                  onChange={(e) =>
                    setEnvFilter(e.target.value as ConsoleEnvKind | "all")
                  }
                  aria-label="环境"
                >
                  <option value="all">全部环境</option>
                  <option value="production">生产</option>
                  <option value="staging">预发</option>
                  <option value="development">开发</option>
                  <option value="local">本地</option>
                </select>
              </div>
              <div className="flex items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1">
                <Wifi className="h-3.5 w-3.5 text-muted-foreground" />
                <select
                  className="bg-transparent text-sm outline-none"
                  value={wsFilter}
                  onChange={(e) =>
                    setWsFilter(e.target.value as AgentWsStatusKind | "all")
                  }
                  aria-label="WebSocket"
                >
                  <option value="all">全部 WS</option>
                  <option value="connected">已连接</option>
                  <option value="connecting">连接中</option>
                  <option value="disconnected">断开</option>
                  <option value="error">异常</option>
                </select>
              </div>
              <Button
                type="button"
                variant={onlineOnly ? "secondary" : "outline"}
                size="sm"
                onClick={() => setOnlineOnly((v) => !v)}
              >
                仅在线
              </Button>
              <div className="flex items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1">
                <ArrowDownAZ className="h-3.5 w-3.5 text-muted-foreground" />
                <select
                  className="bg-transparent text-sm outline-none"
                  value={sortKey}
                  onChange={(e) => setSortKey(e.target.value as SortKey)}
                  aria-label="排序"
                >
                  <option value="heartbeat">最近心跳</option>
                  <option value="instances">实例数</option>
                  <option value="name">名称</option>
                </select>
              </div>
            </div>
          </CardContent>
        </Card>

        {loading ? <LoadingState label="加载 Agent 列表…" /> : null}

        {!loading && error ? (
          <ErrorState
            title={permissionDenied ? "权限不足" : "无法加载 Agents"}
            description={error.message}
            onRetry={() => void load()}
          />
        ) : null}

        {!loading && !error && filtered.length === 0 ? (
          <EmptyState
            title={rows.length === 0 ? "暂无 Agent 记录" : "无匹配结果"}
            description={
              rows.length === 0
                ? "后端接入后将显示在线/离线、心跳与 WS 状态。"
                : "调整筛选条件或清空搜索。"
            }
          />
        ) : null}

        {!loading && !error && filtered.length > 0 ? (
          <Card className="border-border/80">
            <CardHeader className="pb-2">
              <CardTitle className="text-base">Agent 列表</CardTitle>
              <CardDescription>
                离线、WS 异常与低流动性会以标签高亮；点击行进入详情。
              </CardDescription>
            </CardHeader>
            <CardContent className="px-0 sm:px-6">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Agent</TableHead>
                    <TableHead>运行</TableHead>
                    <TableHead>最近心跳</TableHead>
                    <TableHead>环境</TableHead>
                    <TableHead>WS</TableHead>
                    <TableHead>模式</TableHead>
                    <TableHead>实例</TableHead>
                    <TableHead>最近错误</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filtered.map((r) => (
                    <TableRow key={r.id} className="cursor-pointer">
                      <TableCell className="font-medium">
                        <Link
                          href={`/agents/${encodeURIComponent(r.id)}`}
                          className="block hover:underline"
                        >
                          <div>{r.displayName}</div>
                          <div className="text-xs font-normal text-muted-foreground">
                            {r.id}
                          </div>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          <StatusTag
                            status={r.isOnline ? "online" : "offline"}
                            label={r.isOnline ? "在线" : "离线"}
                          />
                          {r.hasLiquidityWarning ? (
                            <Badge
                              variant="outline"
                              className="gap-0.5 border-amber-500/50 bg-amber-500/15 text-[10px] text-amber-950 dark:text-amber-100"
                            >
                              <Droplets className="h-3 w-3" />
                              流动性
                            </Badge>
                          ) : null}
                        </div>
                      </TableCell>
                      <TableCell className="tabular-nums text-muted-foreground">
                        <div>{formatRelativeShort(r.lastHeartbeatAt)}</div>
                        <div className="text-[11px]">
                          {formatDateTime(r.lastHeartbeatAt)}
                        </div>
                      </TableCell>
                      <TableCell>{envLabels[r.environment]}</TableCell>
                      <TableCell>
                        <StatusTag
                          status={wsStatusToSystem(r.wsStatus)}
                          label={wsLabels[r.wsStatus]}
                        />
                      </TableCell>
                      <TableCell>{execLabels[r.execMode]}</TableCell>
                      <TableCell className="tabular-nums">{r.instanceCount}</TableCell>
                      <TableCell className="max-w-[220px]">
                        {r.lastError ? (
                          <span className="flex items-start gap-1 text-xs text-destructive">
                            <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                            <span className="line-clamp-2">{r.lastError}</span>
                          </span>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        ) : null}
      </div>
    </ConsolePage>
  );
}
