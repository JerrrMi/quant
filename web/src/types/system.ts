import type { SystemStatusKind } from "@/types/console";

export type ConnectionChannel = "api" | "websocket" | "database" | "backtest";

export type ConnectionHealth = "up" | "degraded" | "down" | "unknown";

export type SystemConnectionDTO = {
  channel: ConnectionChannel;
  label: string;
  status: ConnectionHealth;
  detail?: string;
};

export type SystemOverviewDTO = {
  saas: {
    status: SystemStatusKind;
    version?: string;
    detail?: string;
  };
  agentsOnline: number;
  agentsTotal: number;
  activeStrategyInstances: number;
  positions: {
    status: SystemStatusKind;
    openCount: number;
    summary: string;
  };
  lastCommand: {
    status: SystemStatusKind;
    summary: string;
    atLabel: string;
  };
  lastReport: {
    status: SystemStatusKind;
    summary: string;
    atLabel: string;
  };
};

export type ActivityKind =
  | "login"
  | "instance_start"
  | "command"
  | "report"
  | "backtest_complete"
  | "system";

export type ActivityItemDTO = {
  id: string;
  kind: ActivityKind;
  title: string;
  description?: string;
  atLabel: string;
  severity?: "info" | "warn" | "error";
};

export type SystemDashboardDTO = {
  overview: SystemOverviewDTO;
  connections: SystemConnectionDTO[];
  activities: ActivityItemDTO[];
  generatedAt: string;
};
