"use client";

import { Activity, RefreshCw, TrendingUp } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { fetchAccountsHistory, fetchAccountsOverview } from "@/api/accounts";
import { ApiError } from "@/api/errors";
import { ConsolePage } from "@/components/layout/console-page";
import { ErrorState } from "@/components/feedback/error-state";
import { LoadingState } from "@/components/feedback/loading-state";
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
import {
  formatCompactNumber,
  formatDateTime,
  formatRatioAsPercent,
  formatSignedUsd,
  formatUsd,
} from "@/lib/format-trading";
import { cn } from "@/lib/utils";
import type {
  AccountHistoryPointDTO,
  AccountPositionDTO,
  AccountVenueSliceDTO,
  AccountsOverviewDTO,
  PositionSideKind,
} from "@/types/accounts";

const POLL_MS = 20_000;

function riskTone(ratio: number): string {
  if (ratio >= 0.95) return "text-destructive font-semibold";
  if (ratio >= 0.8) return "text-amber-700 dark:text-amber-300 font-semibold";
  return "text-muted-foreground tabular-nums";
}

function sideLabel(side: PositionSideKind): string {
  switch (side) {
    case "long":
      return "多";
    case "short":
      return "空";
    default:
      return "中性";
  }
}

function EquitySparkline({
  points,
  className,
}: {
  points: AccountHistoryPointDTO[];
  className?: string;
}) {
  if (points.length < 2) {
    return (
      <div className={cn("text-xs text-muted-foreground", className)}>
        暂无历史点
      </div>
    );
  }
  const values = points.map((p) => p.value);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const pad = max === min ? 1 : (max - min) * 0.08;
  const lo = min - pad;
  const hi = max + pad;
  const w = 320;
  const h = 72;
  const coords = points.map((p, i) => {
    const x = (i / (points.length - 1)) * w;
    const y = h - ((p.value - lo) / (hi - lo)) * h;
    return `${x},${y}`;
  });
  const polyline = coords.join(" ");

  return (
    <svg
      viewBox={`0 0 ${w} ${h}`}
      className={cn("w-full max-h-24 text-primary", className)}
      preserveAspectRatio="none"
      aria-hidden
    >
      <polyline
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinejoin="round"
        strokeLinecap="round"
        points={polyline}
      />
    </svg>
  );
}

function TimelineStrip({ points }: { points: AccountHistoryPointDTO[] }) {
  const edges = useMemo(() => {
    if (!points.length) return { first: null as string | null, last: null as string | null };
    return {
      first: points[0]?.at ?? null,
      last: points[points.length - 1]?.at ?? null,
    };
  }, [points]);

  return (
    <div className="flex justify-between text-[11px] text-muted-foreground">
      <span>{edges.first ? formatDateTime(edges.first) : "—"}</span>
      <span>{edges.last ? formatDateTime(edges.last) : "—"}</span>
    </div>
  );
}

export function AccountsConsolePage() {
  const [overview, setOverview] = useState<AccountsOverviewDTO | null>(null);
  const [history, setHistory] = useState<AccountHistoryPointDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const [venue, setVenue] = useState<"spot" | "futures">("futures");
  const [symbolFilter, setSymbolFilter] = useState("");

  const loadOverview = useCallback(async () => {
    setError(null);
    try {
      const data = await fetchAccountsOverview();
      setOverview(data);
    } catch (e) {
      setOverview(null);
      setError(e instanceof ApiError ? e : new ApiError("加载失败", 0));
    } finally {
      setLoading(false);
    }
  }, []);

  const loadHistory = useCallback(async () => {
    setHistoryLoading(true);
    try {
      const h = await fetchAccountsHistory({
        venue,
        metric: "equity",
      });
      setHistory(h.points);
    } catch {
      setHistory([]);
    } finally {
      setHistoryLoading(false);
    }
  }, [venue]);

  const refreshAll = useCallback(
    async (manual?: boolean) => {
      if (manual) setRefreshing(true);
      await loadOverview();
      await loadHistory();
      if (manual) setRefreshing(false);
    },
    [loadOverview, loadHistory],
  );

  useEffect(() => {
    let cancelled = false;
    void Promise.resolve().then(() => {
      if (!cancelled) void loadOverview();
    });
    return () => {
      cancelled = true;
    };
  }, [loadOverview]);

  useEffect(() => {
    let cancelled = false;
    void Promise.resolve().then(() => {
      if (!cancelled) void loadHistory();
    });
    return () => {
      cancelled = true;
    };
  }, [loadHistory]);

  useEffect(() => {
    const id = window.setInterval(() => void refreshAll(), POLL_MS);
    return () => window.clearInterval(id);
  }, [refreshAll]);

  const slice: AccountVenueSliceDTO | null = overview
    ? venue === "spot"
      ? overview.spot
      : overview.futures
    : null;

  const filteredPositions = useMemo(() => {
    if (!slice) return [];
    const q = symbolFilter.trim().toLowerCase();
    if (!q) return slice.positions;
    return slice.positions.filter((p) => p.symbol.toLowerCase().includes(q));
  }, [slice, symbolFilter]);

  const permissionDenied = error?.status === 403;

  return (
    <ConsolePage
      title="账户"
      description="现货与合约资金状态 · 权益、保证金、盈亏与风险（统一 API；不含密钥）。"
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
            onClick={() => void refreshAll(true)}
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
        <div className="flex flex-wrap gap-2 rounded-lg border border-border/80 bg-muted/20 p-1">
          <Button
            type="button"
            size="sm"
            variant={venue === "spot" ? "secondary" : "ghost"}
            className="flex-1 sm:flex-none"
            onClick={() => setVenue("spot")}
          >
            现货账户
          </Button>
          <Button
            type="button"
            size="sm"
            variant={venue === "futures" ? "secondary" : "ghost"}
            className="flex-1 sm:flex-none"
            onClick={() => setVenue("futures")}
          >
            合约账户
          </Button>
        </div>

        {loading ? <LoadingState label="加载账户数据…" /> : null}

        {!loading && error ? (
          <ErrorState
            title={permissionDenied ? "权限不足" : "无法加载账户"}
            description={error.message}
            onRetry={() => void refreshAll()}
          />
        ) : null}

        {!loading && !error && slice ? (
          <>
            <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <Activity className="h-3.5 w-3.5" />
              数据更新时间：
              <span className="font-medium text-foreground tabular-nums">
                {formatDateTime(overview?.updatedAt)}
              </span>
            </div>

            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <Card className="border-border/80 bg-card/50">
                <CardHeader className="pb-1">
                  <CardTitle className="text-xs font-medium text-muted-foreground">
                    总权益
                  </CardTitle>
                </CardHeader>
                <CardContent className="text-xl font-semibold tabular-nums">
                  {formatUsd(slice.totalEquity)}
                </CardContent>
              </Card>
              <Card className="border-border/80 bg-card/50">
                <CardHeader className="pb-1">
                  <CardTitle className="text-xs font-medium text-muted-foreground">
                    可用余额
                  </CardTitle>
                </CardHeader>
                <CardContent className="text-xl font-semibold tabular-nums">
                  {formatUsd(slice.availableBalance)}
                </CardContent>
              </Card>
              <Card className="border-border/80 bg-card/50">
                <CardHeader className="pb-1">
                  <CardTitle className="text-xs font-medium text-muted-foreground">
                    保证金占用
                  </CardTitle>
                </CardHeader>
                <CardContent className="text-xl font-semibold tabular-nums">
                  {formatUsd(slice.marginUsed)}
                </CardContent>
              </Card>
              <Card className="border-border/80 bg-card/50">
                <CardHeader className="pb-1">
                  <CardTitle className="text-xs font-medium text-muted-foreground">
                    风险率
                  </CardTitle>
                </CardHeader>
                <CardContent
                  className={cn("text-xl tabular-nums", riskTone(slice.riskRatio))}
                >
                  {formatRatioAsPercent(slice.riskRatio, { digits: 2 })}
                </CardContent>
              </Card>
            </div>

            <div className="grid gap-3 lg:grid-cols-3">
              <Card className="border-border/80 lg:col-span-2">
                <CardHeader className="pb-2">
                  <CardTitle className="text-base">盈亏与敞口</CardTitle>
                  <CardDescription>
                    未实现 / 已实现盈亏；净持仓方向与名义规模
                  </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 sm:grid-cols-2">
                  <div className="rounded-md border border-border/70 bg-muted/15 px-3 py-2">
                    <div className="text-xs text-muted-foreground">未实现盈亏</div>
                    <div
                      className={cn(
                        "text-lg font-semibold tabular-nums",
                        slice.unrealizedPnl < 0 ? "text-destructive" : "text-emerald-700 dark:text-emerald-400",
                      )}
                    >
                      {formatSignedUsd(slice.unrealizedPnl)}
                    </div>
                  </div>
                  <div className="rounded-md border border-border/70 bg-muted/15 px-3 py-2">
                    <div className="text-xs text-muted-foreground">已实现盈亏</div>
                    <div className="text-lg font-semibold tabular-nums text-foreground">
                      {formatSignedUsd(slice.realizedPnl)}
                    </div>
                  </div>
                  <div className="rounded-md border border-border/70 bg-muted/15 px-3 py-2">
                    <div className="text-xs text-muted-foreground">持仓方向</div>
                    <div className="mt-1 flex items-center gap-2">
                      <Badge variant="outline">{sideLabel(slice.netPositionSide)}</Badge>
                    </div>
                  </div>
                  <div className="rounded-md border border-border/70 bg-muted/15 px-3 py-2">
                    <div className="text-xs text-muted-foreground">仓位 / 杠杆</div>
                    <div className="text-lg font-semibold tabular-nums">
                      {formatCompactNumber(slice.netPositionSize)}{" "}
                      <span className="text-sm font-normal text-muted-foreground">
                        · {slice.leverage.toFixed(2)}x
                      </span>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card className="border-border/80">
                <CardHeader className="pb-2">
                  <CardTitle className="flex items-center gap-2 text-base">
                    <TrendingUp className="h-4 w-4" />
                    权益快照趋势
                  </CardTitle>
                  <CardDescription>
                    {venue === "futures" ? "合约" : "现货"} · 按小时采样（BFF 占位）
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-2">
                  {historyLoading ? (
                    <p className="text-xs text-muted-foreground">加载历史…</p>
                  ) : (
                    <>
                      <EquitySparkline points={history} />
                      <TimelineStrip points={history} />
                    </>
                  )}
                </CardContent>
              </Card>
            </div>

            <Card className="border-border/80">
              <CardHeader className="flex flex-col gap-3 space-y-0 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <CardTitle className="text-base">持仓列表</CardTitle>
                  <CardDescription>
                    {slice.positions.length === 0
                      ? "当前无持仓明细"
                      : `共 ${slice.positions.length} 条 · 可按交易对筛选`}
                  </CardDescription>
                </div>
                <Input
                  className="sm:w-56"
                  placeholder="筛选交易对…"
                  value={symbolFilter}
                  onChange={(e) => setSymbolFilter(e.target.value)}
                  aria-label="筛选交易对"
                />
              </CardHeader>
              <CardContent className="px-0 sm:px-6">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>交易对</TableHead>
                      <TableHead>方向</TableHead>
                      <TableHead>仓位</TableHead>
                      <TableHead>杠杆</TableHead>
                      <TableHead>风险率</TableHead>
                      <TableHead>未实现盈亏</TableHead>
                      <TableHead>保证金</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredPositions.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={7} className="text-muted-foreground">
                          {slice.positions.length === 0 ? "无持仓" : "无匹配交易对"}
                        </TableCell>
                      </TableRow>
                    ) : (
                      filteredPositions.map((p: AccountPositionDTO) => (
                        <TableRow key={p.symbol}>
                          <TableCell className="font-medium">{p.symbol}</TableCell>
                          <TableCell>
                            <Badge variant="outline">{sideLabel(p.side)}</Badge>
                          </TableCell>
                          <TableCell className="tabular-nums">
                            {formatCompactNumber(p.positionSize)}
                          </TableCell>
                          <TableCell className="tabular-nums">{p.leverage}x</TableCell>
                          <TableCell className={riskTone(p.riskRatio)}>
                            {formatRatioAsPercent(p.riskRatio, { digits: 2 })}
                          </TableCell>
                          <TableCell
                            className={cn(
                              "tabular-nums",
                              p.unrealizedPnl < 0
                                ? "text-destructive"
                                : "text-emerald-700 dark:text-emerald-400",
                            )}
                          >
                            {formatSignedUsd(p.unrealizedPnl)}
                          </TableCell>
                          <TableCell className="tabular-nums text-muted-foreground">
                            {formatUsd(p.marginUsed)}
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </>
        ) : null}
      </div>
    </ConsolePage>
  );
}
