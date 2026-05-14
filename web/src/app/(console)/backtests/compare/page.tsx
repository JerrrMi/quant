"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { ArrowLeft, GitCompare } from "lucide-react";

import { fetchBacktestJobDetail } from "@/api/backtests";
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
import { formatRatioAsPercent } from "@/lib/format-trading";
import type { BacktestJobDetailResponseDTO, BacktestReportDTO } from "@/types/backtests";

function isReport(x: unknown): x is BacktestReportDTO {
  if (!x || typeof x !== "object") return false;
  const o = x as Record<string, unknown>;
  return typeof o.metrics === "object" && o.metrics !== null;
}

export default function BacktestComparePage() {
  const sp = useSearchParams();
  const leftId = sp.get("left") ?? "";
  const rightId = sp.get("right") ?? "";
  const [left, setLeft] = useState<BacktestJobDetailResponseDTO | null>(null);
  const [right, setRight] = useState<BacktestJobDetailResponseDTO | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!leftId || !rightId) {
      setErr("请在 URL 中提供 left 与 right 两个任务 ID。");
      return;
    }
    try {
      const [a, b] = await Promise.all([
        fetchBacktestJobDetail(leftId),
        fetchBacktestJobDetail(rightId),
      ]);
      setLeft(a);
      setRight(b);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "加载失败");
    }
  }, [leftId, rightId]);

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 0);
    return () => window.clearTimeout(timer);
  }, [load]);

  const lrep = left && isReport(left.report) ? left.report : null;
  const rrep = right && isReport(right.report) ? right.report : null;

  return (
    <ConsolePage
      title="回测对比"
      description="并排比较两次回测运行的核心 KPI；差异列使用浅色高亮。"
      actions={
        <Button variant="outline" size="sm" asChild>
          <Link href="/backtests">
            <ArrowLeft className="mr-2 h-4 w-4" />
            返回
          </Link>
        </Button>
      }
    >
      {err ? (
        <ErrorState description={err} onRetry={() => void load()} />
      ) : !left || !right ? (
        <p className="text-sm text-muted-foreground">加载中…</p>
      ) : (
        <div className="space-y-6">
          <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            <GitCompare className="h-4 w-4" />
            对比任务{" "}
            <Link className="text-primary underline" href={`/backtests/${left.job.id}`}>
              #{left.job.id}
            </Link>
            与{" "}
            <Link className="text-primary underline" href={`/backtests/${right.job.id}`}>
              #{right.job.id}
            </Link>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">任务 A</CardTitle>
                <CardDescription>
                  {left.job.template_name} · {left.job.symbol}
                </CardDescription>
              </CardHeader>
              <CardContent className="text-xs text-muted-foreground">
                状态 {left.job.status}
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-base">任务 B</CardTitle>
                <CardDescription>
                  {right.job.template_name} · {right.job.symbol}
                </CardDescription>
              </CardHeader>
              <CardContent className="text-xs text-muted-foreground">
                状态 {right.job.status}
              </CardContent>
            </Card>
          </div>

          {lrep && rrep ? (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">指标对比</CardTitle>
              </CardHeader>
              <CardContent className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>指标</TableHead>
                      <TableHead className="text-right">A (#{left.job.id})</TableHead>
                      <TableHead className="text-right">B (#{right.job.id})</TableHead>
                      <TableHead className="text-right">差异 (B−A)</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <CmpRow
                      label="总收益"
                      a={lrep.metrics.total_return}
                      b={rrep.metrics.total_return}
                      fmt={(v) => formatRatioAsPercent(v, { digits: 2 })}
                    />
                    <CmpRow
                      label="最大回撤"
                      a={lrep.metrics.max_drawdown_01}
                      b={rrep.metrics.max_drawdown_01}
                      fmt={(v) => formatRatioAsPercent(v, { digits: 2 })}
                    />
                    <CmpRow
                      label="胜率"
                      a={lrep.metrics.win_rate}
                      b={rrep.metrics.win_rate}
                      fmt={(v) => formatRatioAsPercent(v, { digits: 1 })}
                    />
                    <CmpRow
                      label="交易次数（往返）"
                      a={lrep.metrics.num_round_trips}
                      b={rrep.metrics.num_round_trips}
                      fmt={(v) => String(Math.round(v))}
                      integer
                    />
                    <CmpRow
                      label="平均持仓周期（步）"
                      a={lrep.metrics.avg_holding_steps}
                      b={rrep.metrics.avg_holding_steps}
                      fmt={(v) => v.toFixed(2)}
                    />
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          ) : (
            <ErrorState
              title="无法对比结果"
              description="至少一方缺少可解析的报告（请确认任务已结束并成功写出 report）。"
            />
          )}
        </div>
      )}
    </ConsolePage>
  );
}

type CmpRowProps = {
  label: string;
  a: number;
  b: number;
  fmt: (v: number) => string;
  integer?: boolean;
};

function CmpRow({ label, a, b, fmt, integer }: CmpRowProps) {
  const diff = b - a;
  const diffDisplay = integer ? String(Math.round(diff)) : fmt(diff);
  const strong = Math.abs(diff) > 1e-9;
  return (
    <TableRow>
      <TableCell className="font-medium">{label}</TableCell>
      <TableCell className="text-right tabular-nums">{fmt(a)}</TableCell>
      <TableCell className="text-right tabular-nums">{fmt(b)}</TableCell>
      <TableCell
        className={`text-right tabular-nums ${strong ? "bg-muted/80 font-semibold" : ""}`}
      >
        {diffDisplay}
      </TableCell>
    </TableRow>
  );
}
