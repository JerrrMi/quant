"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { ArrowLeft, Loader2, Pause, RefreshCw, Square } from "lucide-react";
import { toast } from "sonner";

import {
  backtestJobAction,
  fetchBacktestJobDetail,
} from "@/api/backtests";
import { useConfirm } from "@/components/feedback/confirm-provider";
import { ErrorState } from "@/components/feedback/error-state";
import { ConsolePage } from "@/components/layout/console-page";
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
import { BacktestEquityChart } from "@/features/backtests/backtest-equity-chart";
import { formatDateTime, formatRatioAsPercent } from "@/lib/format-trading";
import type {
  BacktestJobDetailResponseDTO,
  BacktestReportDTO,
} from "@/types/backtests";

function isReport(x: unknown): x is BacktestReportDTO {
  if (!x || typeof x !== "object") return false;
  const o = x as Record<string, unknown>;
  return (
    typeof o.metrics === "object" &&
    o.metrics !== null &&
    Array.isArray(o.equity_curve) &&
    Array.isArray(o.command_stats)
  );
}

export default function BacktestDetailPage() {
  const params = useParams();
  const id = String(params.id ?? "");
  const confirm = useConfirm();
  const [data, setData] = useState<BacktestJobDetailResponseDTO | null>(null);
  const [loadErr, setLoadErr] = useState<string | null>(null);
  const [reportErr, setReportErr] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!id) return;
    try {
      const res = await fetchBacktestJobDetail(id);
      setData(res);
      setLoadErr(null);
      if (res.report != null && !isReport(res.report)) {
        setReportErr("结果 JSON 与预期结构不一致，无法渲染图表与部分指标。");
      } else {
        setReportErr(null);
      }
    } catch (e) {
      setLoadErr(e instanceof Error ? e.message : "加载失败");
    }
  }, [id]);

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 0);
    return () => window.clearTimeout(timer);
  }, [load]);

  const running = data?.job.status === "running";

  useEffect(() => {
    if (!running) return;
    const t = window.setInterval(() => void load(), 2500);
    return () => window.clearInterval(t);
  }, [running, load]);

  async function doPause() {
    if (!data) return;
    const ok = await confirm({
      title: "暂停此回测？",
      description: "运行中的回放将被取消，已产生的日志与进度保留。",
      destructive: true,
      confirmLabel: "暂停",
      cancelLabel: "取消",
    });
    if (!ok) return;
    try {
      await backtestJobAction(data.job.id, "pause");
      toast.message("已请求暂停");
      await load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "失败");
    }
  }

  async function doTerminate() {
    if (!data) return;
    const ok = await confirm({
      title: "终止此回测任务？",
      description:
        "将请求服务端取消运行中的回放并保留已写入的日志。与实盘不同，但会占用计算资源；若任务卡死可安全终止后重试。",
      destructive: true,
      confirmLabel: "终止任务",
      cancelLabel: "取消",
    });
    if (!ok) return;
    try {
      await backtestJobAction(data.job.id, "terminate");
      toast.message("已请求终止");
      await load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "失败");
    }
  }

  async function doRerun() {
    if (!data) return;
    const ok = await confirm({
      title: "复制参数并重新运行？",
      confirmLabel: "重新运行",
    });
    if (!ok) return;
    try {
      const res = await backtestJobAction(data.job.id, "rerun");
      if ("id" in res && typeof res.id === "number") {
        toast.success(`新任务 #${res.id}`);
        window.location.href = `/backtests/${res.id}`;
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "失败");
    }
  }

  const report = data && isReport(data.report) ? data.report : null;

  return (
    <ConsolePage
      title={id ? `回测 #${id}` : "回测详情"}
      description="摘要、进度、日志与交易明细与后端 report 字段对齐。"
      actions={
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href="/backtests">
              <ArrowLeft className="mr-2 h-4 w-4" />
              返回列表
            </Link>
          </Button>
          {data ? (
            <>
              <Button variant="outline" size="sm" asChild>
                <Link href={`/strategies/templates/${data.job.template_id}`}>
                  模板
                </Link>
              </Button>
              <Button variant="outline" size="sm" asChild>
                <Link
                  href={`/backtests/compare?left=${encodeURIComponent(String(data.job.id))}&right=`}
                  title="在地址栏补全 right= 另一任务 ID"
                >
                  结果对比
                </Link>
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={data.job.status !== "running"}
                onClick={() => void doPause()}
              >
                <Pause className="mr-2 h-4 w-4" />
                暂停
              </Button>
              <Button
                size="sm"
                variant="destructive"
                disabled={data.job.status !== "running"}
                onClick={() => void doTerminate()}
              >
                <Square className="mr-2 h-4 w-4" />
                终止
              </Button>
              <Button size="sm" onClick={() => void doRerun()}>
                <RefreshCw className="mr-2 h-4 w-4" />
                重新运行
              </Button>
            </>
          ) : null}
        </div>
      }
    >
      {loadErr ? (
        <ErrorState description={loadErr} onRetry={() => void load()} />
      ) : !data ? (
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          加载中…
        </div>
      ) : (
        <div className="space-y-6">
          {running ? (
            <div className="flex items-center gap-2 rounded-lg border border-sky-500/30 bg-sky-500/5 px-4 py-3 text-sm font-medium text-sky-900 dark:text-sky-100">
              <Loader2 className="h-4 w-4 animate-spin" />
              正在运行 — 进度与日志将自动刷新
              {data.job.progress ? (
                <span className="ml-2 text-xs font-normal text-muted-foreground">
                  {data.job.progress.done}/{data.job.progress.total}（
                  {(data.job.progress.pct_01 * 100).toFixed(0)}%）
                </span>
              ) : null}
            </div>
          ) : null}

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">任务摘要</CardTitle>
                <CardDescription>数据库中的任务元数据</CardDescription>
              </CardHeader>
              <CardContent className="grid gap-2 text-sm">
                <div className="flex justify-between gap-2">
                  <span className="text-muted-foreground">状态</span>
                  <span className="font-medium">{data.job.status}</span>
                </div>
                <div className="flex justify-between gap-2">
                  <span className="text-muted-foreground">模板</span>
                  <Link
                    className="truncate text-right font-medium text-primary hover:underline"
                    href={`/strategies/templates/${data.job.template_id}`}
                  >
                    {data.job.template_name} (#{data.job.template_id})
                  </Link>
                </div>
                {data.job.instance_id != null ? (
                  <div className="flex justify-between gap-2">
                    <span className="text-muted-foreground">实例</span>
                    <Link
                      className="text-primary underline-offset-2 hover:underline"
                      href={`/strategies/instances/${data.job.instance_id}`}
                    >
                      #{data.job.instance_id}
                    </Link>
                  </div>
                ) : null}
                <div className="flex justify-between gap-2">
                  <span className="text-muted-foreground">标的 / 市场</span>
                  <span>
                    {data.job.symbol} · {data.job.market_kind}
                  </span>
                </div>
                <div className="flex justify-between gap-2">
                  <span className="text-muted-foreground">区间</span>
                  <span className="text-right text-xs">
                    {data.job.window_start || "—"} →{" "}
                    {data.job.window_end || "—"}
                  </span>
                </div>
                <div className="flex justify-between gap-2">
                  <span className="text-muted-foreground">创建</span>
                  <span>{formatDateTime(data.job.created_at)}</span>
                </div>
                <div className="flex justify-between gap-2">
                  <span className="text-muted-foreground">更新</span>
                  <span>{formatDateTime(data.job.updated_at)}</span>
                </div>
                {data.job.error_message ? (
                  <div className="rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">
                    {data.job.error_message}
                  </div>
                ) : null}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-base">请求参数（快照）</CardTitle>
                <CardDescription>与发起任务时的表单一致</CardDescription>
              </CardHeader>
              <CardContent>
                <pre className="max-h-56 overflow-auto rounded-md border bg-muted/40 p-3 text-[11px] leading-relaxed">
                  {JSON.stringify(data.job.request, null, 2)}
                </pre>
              </CardContent>
            </Card>
          </div>

          {reportErr ? (
            <ErrorState title="结果解析失败" description={reportErr} />
          ) : report ? (
            <>
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">绩效摘要</CardTitle>
                  <CardDescription>
                    字段名对应{" "}
                    <code className="text-xs">PerformanceMetrics</code>
                  </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  <Metric label="总收益（比例）" value={formatRatioAsPercent(report.metrics.total_return, { digits: 2 })} />
                  <Metric label="最大回撤" value={formatRatioAsPercent(report.metrics.max_drawdown_01, { digits: 2 })} />
                  <Metric label="胜率" value={formatRatioAsPercent(report.metrics.win_rate, { digits: 1 })} />
                  <Metric label="换手率" value={report.metrics.turnover_ratio.toFixed(3)} />
                  <Metric label="命令命中率" value={formatRatioAsPercent(report.metrics.command_hit_rate, { digits: 1 })} />
                  <Metric label="命令失败率" value={formatRatioAsPercent(report.metrics.command_fail_rate, { digits: 1 })} />
                  <Metric label="部分成交率" value={formatRatioAsPercent(report.metrics.partial_fill_rate, { digits: 1 })} />
                  <Metric label="往返次数" value={String(report.metrics.num_round_trips)} />
                  <Metric label="平均持仓（步）" value={report.metrics.avg_holding_steps.toFixed(1)} />
                  <Metric label="初末权益" value={`${report.metrics.initial_equity.toFixed(2)} → ${report.metrics.final_equity.toFixed(2)}`} />
                  {report.metrics.cumulative_net_fees != null ? (
                    <Metric label="累计费用" value={report.metrics.cumulative_net_fees.toFixed(4)} />
                  ) : null}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">收益曲线</CardTitle>
                </CardHeader>
                <CardContent>
                  <BacktestEquityChart points={report.equity_curve} />
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">命令 / 仿真成交</CardTitle>
                  <CardDescription>command_stats</CardDescription>
                </CardHeader>
                <CardContent className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Intent</TableHead>
                        <TableHead>Command</TableHead>
                        <TableHead>状态</TableHead>
                        <TableHead>部分</TableHead>
                        <TableHead>信息</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {report.command_stats.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={5} className="text-center text-muted-foreground">
                            无命令记录
                          </TableCell>
                        </TableRow>
                      ) : (
                        report.command_stats.map((r) => (
                          <TableRow key={`${r.command_id}-${r.intent_id}`}>
                            <TableCell className="font-mono text-xs">
                              {r.intent_id}
                            </TableCell>
                            <TableCell className="font-mono text-xs">
                              {r.command_id}
                            </TableCell>
                            <TableCell className="text-xs">{r.status}</TableCell>
                            <TableCell className="text-xs">
                              {r.partial ? "是" : "否"}
                            </TableCell>
                            <TableCell className="max-w-[200px] truncate text-xs">
                              {r.message ?? "—"}
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </>
          ) : data.job.status === "finished" ? (
            <ErrorState title="无报告数据" description="任务已完成但未写入 report。" />
          ) : null}

          <Card>
            <CardHeader>
              <CardTitle className="text-base">日志</CardTitle>
            </CardHeader>
            <CardContent>
              <pre className="max-h-80 overflow-auto rounded-md border bg-muted/30 p-3 text-[11px] whitespace-pre-wrap">
                {data.logs.length === 0
                  ? "暂无日志"
                  : data.logs.join("\n")}
              </pre>
            </CardContent>
          </Card>
        </div>
      )}
    </ConsolePage>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border bg-card px-3 py-2">
      <div className="text-[11px] text-muted-foreground">{label}</div>
      <div className="text-sm font-semibold tabular-nums">{value}</div>
    </div>
  );
}
