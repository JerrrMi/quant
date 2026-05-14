import type { SystemDashboardDTO } from "@/types/system";

import type { SystemDashboardAdapter } from "./system.types";

function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchDashboard(): Promise<SystemDashboardDTO> {
  await delay(280);
  const now = new Date();
  return {
    overview: {
      saas: {
        status: "running",
        version: "mock-1.0.0",
        detail: "Mock 数据：接入真实 SaaS 后由 BFF 转发",
      },
      agentsOnline: 1,
      agentsTotal: 3,
      activeStrategyInstances: 0,
      positions: {
        status: "idle",
        openCount: 0,
        summary: "当前无持仓（示例）",
      },
      lastCommand: {
        status: "idle",
        summary: "尚未下发远程命令",
        atLabel: "—",
      },
      lastReport: {
        status: "idle",
        summary: "等待 Agent 回报",
        atLabel: "—",
      },
    },
    connections: [
      {
        channel: "api",
        label: "SaaS API",
        status: "up",
        detail: "Mock · BFF 可达",
      },
      {
        channel: "websocket",
        label: "信令 / 行情 WS",
        status: "degraded",
        detail: "Mock · 未建立长连接",
      },
      {
        channel: "database",
        label: "数据库",
        status: "up",
        detail: "Mock · 控制面库",
      },
      {
        channel: "backtest",
        label: "回测服务",
        status: "unknown",
        detail: "Mock · 可选进程",
      },
    ],
    activities: [
      {
        id: "mock-1",
        kind: "system",
        title: "控制台已加载 Mock 总览",
        description: "设置 NEXT_PUBLIC_USE_MOCK_API=false 可请求 BFF /api/system/dashboard",
        atLabel: now.toLocaleString(),
        severity: "info",
      },
    ],
    generatedAt: now.toISOString(),
  };
}

export const systemMockAdapter: SystemDashboardAdapter = {
  fetchDashboard,
};
