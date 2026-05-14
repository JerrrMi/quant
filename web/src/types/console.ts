export type ConsoleEnvKind =
  | "development"
  | "staging"
  | "production"
  | "local";

/** Aggregated UI-facing connection indicator for header badges/tooltips. */
export type ConsoleConnectionStatus =
  | "connected"
  | "connecting"
  | "disconnected";

export type StrategyTemplateView = {
  id: string;
  name: string;
  description?: string;
  updatedAtLabel: string;
};

export type StrategyInstanceView = {
  id: string;
  templateId: string;
  symbolLabel: string;
  statusLabel: string;
};

export type AgentSummaryView = {
  id: string;
  label: string;
  venueLabel: string;
  status: SystemStatusKind;
};

export type BacktestRunView = {
  id: string;
  strategyLabel: string;
  createdAtLabel: string;
  status: SystemStatusKind;
};

export type AccountSummaryView = {
  id: string;
  label: string;
  venueLabel: string;
  equityLabel: string;
};

export type LogEntryView = {
  id: string;
  level: "info" | "warn" | "error";
  message: string;
  timestampLabel: string;
};

/** Unified operational status vocabulary for badges across screens. */
export type SystemStatusKind =
  | "online"
  | "offline"
  | "running"
  | "paused"
  | "failed"
  | "idle";
