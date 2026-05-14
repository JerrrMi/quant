"use client";

import { useMemo } from "react";

import { cn } from "@/lib/utils";
import type { BacktestEquityPointDTO } from "@/types/backtests";

type Props = {
  points: BacktestEquityPointDTO[];
  className?: string;
  height?: number;
};

/** 轻量 SVG 权益曲线：单色描边 + 浅填充，避免多色系噪音。 */
export function BacktestEquityChart({
  points,
  className,
  height = 180,
}: Props) {
  const { pathD, areaD, minY, maxY, viewW } = useMemo(() => {
    if (points.length === 0) {
      return { pathD: "", areaD: "", minY: 0, maxY: 1, viewW: 320 };
    }
    const pad = 8;
    const w = 640;
    const h = height;
    const eq = points.map((p) => p.equity);
    const min = Math.min(...eq);
    const max = Math.max(...eq);
    const span = Math.max(max - min, 1e-9);
    const n = points.length;
    const xAt = (i: number) => pad + (i / Math.max(n - 1, 1)) * (w - pad * 2);
    const yAt = (v: number) =>
      pad + (1 - (v - min) / span) * (h - pad * 2);

    let d = "";
    let ad = "";
    for (let i = 0; i < n; i++) {
      const x = xAt(i);
      const y = yAt(points[i].equity);
      d += i === 0 ? `M ${x.toFixed(1)} ${y.toFixed(1)}` : ` L ${x.toFixed(1)} ${y.toFixed(1)}`;
    }
    const y0 = yAt(min);
    ad = `${d} L ${xAt(n - 1).toFixed(1)} ${y0.toFixed(1)} L ${xAt(0).toFixed(1)} ${y0.toFixed(1)} Z`;
    return { pathD: d, areaD: ad, minY: min, maxY: max, viewW: w };
  }, [points, height]);

  if (points.length === 0) {
    return (
      <div
        className={cn(
          "flex h-[180px] items-center justify-center rounded-md border border-dashed text-sm text-muted-foreground",
          className,
        )}
      >
        暂无曲线数据
      </div>
    );
  }

  return (
    <div className={cn("w-full", className)}>
      <svg
        viewBox={`0 0 ${viewW} ${height}`}
        className="h-auto max-h-[220px] w-full text-primary"
        preserveAspectRatio="none"
        role="img"
        aria-label="Equity curve"
      >
        <defs>
          <linearGradient id="eqFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="currentColor" stopOpacity="0.12" />
            <stop offset="100%" stopColor="currentColor" stopOpacity="0" />
          </linearGradient>
        </defs>
        <path d={areaD} fill="url(#eqFill)" stroke="none" />
        <path
          d={pathD}
          fill="none"
          stroke="currentColor"
          strokeWidth={1.5}
          vectorEffect="non-scaling-stroke"
        />
      </svg>
      <div className="mt-1 flex justify-between text-[11px] text-muted-foreground">
        <span>最低 {minY.toFixed(2)}</span>
        <span>最高 {maxY.toFixed(2)}</span>
      </div>
    </div>
  );
}
