/** SaaS `/v1/console/templates` 列表项（与后端 JSON 对齐）。 */
export type StrategyTemplateDTO = {
  id: number;
  name: string;
  kind: string;
  markets: string[];
  description: string;
  allow_live: boolean;
  allow_backtest: boolean;
  updated_at: string;
};

export type StrategyTemplatesListDTO = {
  templates: StrategyTemplateDTO[];
};

/** 模板详情含 config_json（模板默认参数快照，非 Step 逻辑）。 */
export type StrategyTemplateDetailDTO = {
  id: number;
  name: string;
  kind: string;
  markets: string[];
  description: string;
  allow_live: boolean;
  allow_backtest: boolean;
  config_json: unknown;
  updated_at: string;
};

export type StrategyInstanceRowDTO = {
  id: number;
  display_name: string;
  template_id: number;
  template_name: string;
  symbol: string;
  market_kind: string;
  run_mode: string;
  status: string;
  derived_runtime: string;
  agent_key: string;
  agent_connected: boolean;
  last_heartbeat_at?: string;
  last_command_summary?: string;
  last_report_summary?: string;
  risk_status: string;
  updated_at: string;
};

export type StrategyInstancesListDTO = {
  instances: StrategyInstanceRowDTO[];
};

export type StrategyInstanceTimelineEntryDTO = {
  action: string;
  occurred_at: string;
  payload: string;
};

export type StrategyInstanceCommandEntryDTO = {
  id: string;
  kind: string;
  status: string;
  created_at: string;
  summary: string;
  error: string;
};

export type StrategyInstanceReportEntryDTO = {
  id: string;
  type: string;
  received_at: string;
  summary: string;
};

export type StrategyInstanceDetailDTO = {
  id: number;
  display_name: string;
  template_id: number;
  template_name: string;
  template_kind: string;
  template_markets: string[];
  template_description: string;
  template_config_json: unknown;
  symbol: string;
  market_kind: string;
  run_mode: string;
  status: string;
  derived_runtime: string;
  agent_key: string;
  agent_connected: boolean;
  last_heartbeat_at: string;
  risk_status: string;
  instance_params_json: unknown;
  recent_commands: StrategyInstanceCommandEntryDTO[];
  recent_reports: StrategyInstanceReportEntryDTO[];
  recent_errors_hint: string;
  timeline: StrategyInstanceTimelineEntryDTO[];
  updated_at: string;
};

export type CreateStrategyInstanceBody = {
  display_name: string;
  strategy_id: number;
  symbol: string;
  market_kind: "spot" | "futures";
  run_mode: "backtest" | "paper" | "live";
  agent_key: string;
  params: Record<string, unknown>;
};

export type CreateStrategyInstanceResponseDTO = {
  id: number;
};

export type InstanceLifecycleActionDTO =
  | "start"
  | "stop"
  | "pause"
  | "resume"
  | "restart"
  | "run_once"
  | "terminate";
