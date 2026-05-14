import type { ConsoleEnvKind } from "@/types/console";

/** Agent ↔ SaaS WebSocket 会话状态（展示层）。 */
export type AgentWsStatusKind =
  | "connected"
  | "connecting"
  | "disconnected"
  | "error";

/** Agent 执行端模式。 */
export type AgentExecModeKind = "spot" | "futures" | "mixed";

export type AgentListRowDTO = {
  id: string;
  displayName: string;
  /** false 表示离线 */
  isOnline: boolean;
  lastHeartbeatAt: string | null;
  environment: ConsoleEnvKind;
  wsStatus: AgentWsStatusKind;
  execMode: AgentExecModeKind;
  lastError: string | null;
  instanceCount: number;
  /** 低流动性等亚健康提示 */
  hasLiquidityWarning?: boolean;
};

export type AgentsListDTO = {
  agents: AgentListRowDTO[];
  generatedAt: string;
};

export type AgentHeartbeatPointDTO = {
  at: string;
  latencyMs: number | null;
  ok: boolean;
};

export type AgentBoundInstanceDTO = {
  id: string;
  symbolLabel: string;
  statusLabel: string;
};

export type AgentCommandEntryDTO = {
  id: string;
  kind: string;
  summary: string;
  status: "pending" | "accepted" | "failed" | "completed";
  at: string;
};

export type AgentReportEntryDTO = {
  id: string;
  kind: string;
  summary: string;
  severity: "info" | "warn" | "error";
  at: string;
};

export type AgentConnectionInfoDTO = {
  saasApiEndpoint: string;
  wsEndpoint: string;
  agentVersion: string;
  tlsEnabled: boolean;
};

export type AgentDetailDTO = {
  agent: AgentListRowDTO;
  connection: AgentConnectionInfoDTO;
  heartbeatHistory: AgentHeartbeatPointDTO[];
  boundInstances: AgentBoundInstanceDTO[];
  recentCommands: AgentCommandEntryDTO[];
  recentReports: AgentReportEntryDTO[];
  generatedAt: string;
};

export type AgentControlActionDTO =
  | "start"
  | "stop"
  | "reconnect"
  | "refresh";

export type AgentControlResultDTO = {
  ok: boolean;
  message: string;
  appliedAt: string;
};
