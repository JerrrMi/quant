"use client";

import { Activity } from "lucide-react";

import { StatusTag } from "@/components/status-tag";
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
import type { SystemStatusKind } from "@/types/console";

const demoAgents: {
  id: string;
  label: string;
  venue: string;
  status: SystemStatusKind;
}[] = [
  {
    id: "agent-asia",
    label: "Asia · Spot Agent",
    venue: "Binance",
    status: "online",
  },
  {
    id: "agent-eu",
    label: "EU · Futures Agent",
    venue: "Binance",
    status: "idle",
  },
  {
    id: "agent-risk",
    label: "Risk Sentinel",
    venue: "Internal",
    status: "running",
  },
];

export function DashboardOverview() {
  return (
    <div className="grid gap-4 xl:grid-cols-3">
      <Card className="border-border/80 xl:col-span-2">
        <CardHeader className="flex flex-col gap-3 space-y-0 sm:flex-row sm:items-start sm:justify-between">
          <div className="space-y-1">
            <CardTitle className="text-base">Agents</CardTitle>
            <CardDescription>
              占位示例表格 · 列表数据后续通过{" "}
              <code className="rounded bg-muted px-1 py-0.5 text-[11px]">
                src/api/adapters
              </code>{" "}
              映射视图模型
            </CardDescription>
          </div>
          <Button type="button" variant="outline" size="sm">
            刷新
          </Button>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Agent</TableHead>
                <TableHead>交易所</TableHead>
                <TableHead className="w-[140px]">状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {demoAgents.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="font-medium">{row.label}</TableCell>
                  <TableCell>{row.venue}</TableCell>
                  <TableCell>
                    <StatusTag status={row.status} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card className="border-border/80">
        <CardHeader className="space-y-1">
          <CardTitle className="flex items-center gap-2 text-base">
            <Activity className="h-4 w-4 text-muted-foreground" aria-hidden />
            运行脉搏
          </CardTitle>
          <CardDescription>
            WebSocket / SSE 订阅示例可在{" "}
            <code className="rounded bg-muted px-1 py-0.5 text-[11px]">
              src/api/subscriptions.ts
            </code>{" "}
            接线。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-muted-foreground">
          <div className="rounded-md border border-dashed border-border bg-muted/40 px-3 py-2">
            <div className="text-xs font-semibold text-foreground">策略负载</div>
            <p className="mt-1 text-xs leading-relaxed">
              下一步接入 Instances / Backtests 指标卡片与 Sparkline。
            </p>
          </div>
          <div className="rounded-md border border-dashed border-border bg-muted/40 px-3 py-2">
            <div className="text-xs font-semibold text-foreground">风控阈值</div>
            <p className="mt-1 text-xs leading-relaxed">
              组合保证金、回撤与熔断占位；数值字段统一走视图模型映射。
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
