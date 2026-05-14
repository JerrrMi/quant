import type {
  AgentDetailDTO,
  AgentListRowDTO,
  AgentsListDTO,
} from "@/types/agents";
import type { AccountsHistoryDTO, AccountsOverviewDTO } from "@/types/accounts";

const BASE_TIME = Date.now();

function isoMinusMinutes(m: number): string {
  return new Date(BASE_TIME - m * 60_000).toISOString();
}

export const SEED_AGENT_ROWS: AgentListRowDTO[] = [
  {
    id: "agent-tokyo-1",
    displayName: "Tokyo-Mixed-Alpha",
    isOnline: true,
    lastHeartbeatAt: isoMinusMinutes(0.3),
    environment: "production",
    wsStatus: "connected",
    execMode: "mixed",
    lastError: null,
    instanceCount: 4,
    hasLiquidityWarning: false,
  },
  {
    id: "agent-sg-futures",
    displayName: "SG-Futures-Core",
    isOnline: true,
    lastHeartbeatAt: isoMinusMinutes(2),
    environment: "production",
    wsStatus: "connecting",
    execMode: "futures",
    lastError: null,
    instanceCount: 6,
    hasLiquidityWarning: true,
  },
  {
    id: "agent-local-spot",
    displayName: "Dev-Spot-Sandbox",
    isOnline: false,
    lastHeartbeatAt: isoMinusMinutes(120),
    environment: "local",
    wsStatus: "disconnected",
    execMode: "spot",
    lastError: "WS auth rejected: timestamp drift > 5000ms",
    instanceCount: 0,
    hasLiquidityWarning: false,
  },
];

export function buildAgentsListDTO(): AgentsListDTO {
  return {
    agents: SEED_AGENT_ROWS.map((a) => ({ ...a })),
    generatedAt: new Date().toISOString(),
  };
}

export function buildAgentDetailDTO(id: string): AgentDetailDTO | null {
  const row = SEED_AGENT_ROWS.find((a) => a.id === id);
  if (!row) return null;
  const agent = { ...row };
  const heartbeats = Array.from({ length: 24 }).map((_, i) => ({
    at: isoMinusMinutes(i * 5),
    latencyMs: 18 + (i % 5) * 4,
    ok: i !== 19,
  }));

  return {
    agent,
    connection: {
      saasApiEndpoint: "https://saas.example.internal/v1",
      wsEndpoint: "wss://saas.example.internal/ws/agent",
      agentVersion: "0.9.4-altshort",
      tlsEnabled: true,
    },
    heartbeatHistory: heartbeats,
    boundInstances:
      agent.instanceCount > 0
        ? [
            {
              id: "inst-1001",
              symbolLabel: "1000PEPE/USDT",
              statusLabel: "running",
            },
            {
              id: "inst-1002",
              symbolLabel: "WIF/USDT",
              statusLabel: "paused",
            },
            {
              id: "inst-1003",
              symbolLabel: "BONK/USDT",
              statusLabel: "running",
            },
          ].slice(0, Math.min(3, agent.instanceCount))
        : [],
    recentCommands: [
      {
        id: "cmd-501",
        kind: "PAUSE_INSTANCE",
        summary: "inst-1002 throttle reduce 50%",
        status: "completed",
        at: isoMinusMinutes(12),
      },
      {
        id: "cmd-500",
        kind: "REFRESH_POSITIONS",
        summary: "pull futures positions + balances",
        status: "accepted",
        at: isoMinusMinutes(30),
      },
      {
        id: "cmd-499",
        kind: "START_INSTANCE",
        summary: "inst-1003 bootstrap",
        status: "failed",
        at: isoMinusMinutes(90),
      },
    ],
    recentReports: [
      {
        id: "rep-901",
        kind: "HEARTBEAT",
        summary: "latency 22ms, streams healthy",
        severity: "info",
        at: isoMinusMinutes(0.5),
      },
      {
        id: "rep-888",
        kind: "EXECUTION",
        summary: "partial fill PEPE short +12%",
        severity: "info",
        at: isoMinusMinutes(18),
      },
      {
        id: "rep-880",
        kind: "VENUE",
        summary: "ORDER_BOOK thin depth on WIF",
        severity: "warn",
        at: isoMinusMinutes(40),
      },
    ],
    generatedAt: new Date().toISOString(),
  };
}

export function buildAccountsOverviewDTO(): AccountsOverviewDTO {
  return {
    updatedAt: new Date().toISOString(),
    spot: {
      label: "现货",
      totalEquity: 128_430.52,
      availableBalance: 42_110.0,
      marginUsed: 0,
      unrealizedPnl: 0,
      realizedPnl: 3_204.11,
      netPositionSide: "flat",
      netPositionSize: 0,
      leverage: 1,
      riskRatio: 0.08,
      positions: [],
    },
    futures: {
      label: "U 本位合约",
      totalEquity: 96_220.44,
      availableBalance: 28_400.0,
      marginUsed: 51_020.0,
      unrealizedPnl: -4_120.33,
      realizedPnl: 18_902.55,
      netPositionSide: "short",
      netPositionSize: 185_000,
      leverage: 6.2,
      riskRatio: 0.74,
      positions: [
        {
          symbol: "1000PEPE/USDT",
          side: "short",
          positionSize: 72_000,
          leverage: 8,
          riskRatio: 0.62,
          unrealizedPnl: -910.2,
          marginUsed: 18_200,
        },
        {
          symbol: "WIF/USDT",
          side: "short",
          positionSize: 58_000,
          leverage: 5,
          riskRatio: 0.81,
          unrealizedPnl: -140.5,
          marginUsed: 14_800,
        },
        {
          symbol: "BONK/USDT",
          side: "short",
          positionSize: 55_000,
          leverage: 6,
          riskRatio: 0.93,
          unrealizedPnl: -210.4,
          marginUsed: 12_400,
        },
      ],
    },
  };
}

export function buildAccountsHistoryDTO(
  venue: "spot" | "futures",
  metric: "equity" | "available",
): AccountsHistoryDTO {
  const steps = 48;
  const base =
    venue === "futures"
      ? metric === "equity"
        ? 97200
        : 27500
      : metric === "equity"
        ? 127800
        : 41800;
  const points = Array.from({ length: steps }).map((_, i) => {
    const wave = Math.sin(i / 4.2) * (venue === "futures" ? 2200 : 900);
    const drift = i * (venue === "futures" ? 35 : 12);
    return {
      at: new Date(BASE_TIME - (steps - 1 - i) * 3600_000).toISOString(),
      value: Math.round(base + wave + drift),
    };
  });
  return {
    venue,
    metric,
    points,
    generatedAt: new Date().toISOString(),
  };
}
