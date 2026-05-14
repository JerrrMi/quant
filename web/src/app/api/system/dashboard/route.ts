import { NextResponse } from "next/server";

import type { SystemDashboardDTO } from "@/types/system";

function timeLabel(iso: string) {
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

/** BFF aggregation placeholder — replace with SaaS/health checks when ready. */
export async function GET() {
  const now = new Date();
  const iso = now.toISOString();

  const payload: SystemDashboardDTO = {
    overview: {
      saas: {
        status: "running",
        version: process.env.NEXT_PUBLIC_APP_NAME ?? "AltShort Console",
        detail: "SaaS 控制面进程需通过 ops 监测；此处为网关可达性占位",
      },
      agentsOnline: 0,
      agentsTotal: 0,
      activeStrategyInstances: 0,
      positions: {
        status: "idle",
        openCount: 0,
        summary: "暂无持仓汇总，连接 Agent 并启动实例后可在此查看",
      },
      lastCommand: {
        status: "idle",
        summary: "尚未记录命令",
        atLabel: "—",
      },
      lastReport: {
        status: "idle",
        summary: "暂无 Agent 回报",
        atLabel: "—",
      },
    },
    connections: [
      {
        channel: "api",
        label: "SaaS API",
        status: "up",
        detail: "当前请求经由 Next BFF；生产环境请指向真实网关",
      },
      {
        channel: "websocket",
        label: "WebSocket（信令 / 推送）",
        status: "unknown",
        detail: (() => {
          const ws = process.env.NEXT_PUBLIC_WS_URL ?? "";
          return ws
            ? `已配置 NEXT_PUBLIC_WS_URL：${ws}`
            : "未配置 NEXT_PUBLIC_WS_URL，无法自动探测 WS";
        })(),
      },
      {
        channel: "database",
        label: "数据库",
        status: "unknown",
        detail: "库状态需由 SaaS 暴露健康检查；此处仅为占位",
      },
      {
        channel: "backtest",
        label: "回测服务",
        status: "unknown",
        detail: "独立回测进程未在控制台 BFF 中探测",
      },
    ],
    activities: [
      {
        id: "seed-1",
        kind: "system",
        title: "控制台已就绪",
        description: "下一步：启动 cmd/agent、在「策略实例」创建运行单元，或运行回测 CLI",
        atLabel: timeLabel(iso),
        severity: "info",
      },
    ],
    generatedAt: iso,
  };

  return NextResponse.json(payload);
}
