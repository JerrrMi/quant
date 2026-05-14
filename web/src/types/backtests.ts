/** 与 SaaS `/v1/console/backtests` JSON 对齐（snake_case）。 */

export type BacktestJobStatus =
  | "pending"
  | "running"
  | "finished"
  | "failed"
  | "cancelled";

export type BacktestProgressDTO = {
  done: number;
  total: number;
  pct_01: number;
};

export type BacktestJobListItemDTO = {
  id: number;
  status: BacktestJobStatus;
  template_id: number;
  template_name: string;
  instance_id: number | null;
  symbol: string;
  market_kind: string;
  window_start: string;
  window_end: string;
  created_at: string;
  updated_at: string;
  started_at: string;
  finished_at: string;
  duration_ms: number | null;
  progress: BacktestProgressDTO | null;
  error_message: string;
};

export type BacktestJobsListDTO = {
  jobs: BacktestJobListItemDTO[];
};

export type BacktestRequestDTO = {
  source_kind: "template" | "instance";
  template_id: number;
  instance_id: number;
  symbol: string;
  market_kind: "spot" | "futures";
  data_provider: string;
  data_path: string;
  window_start: string;
  window_end: string;
  warmup_bars: number;
  bar_stride: number;
  initial_quote: string;
  currency: string;
  maker_bps: number;
  taker_bps: number;
  use_taker_fees: boolean;
  slippage_bps: number;
  funding_bps_per_day: number;
  lppl_enabled: boolean;
  lppl_bubble_metric_01: number;
  lppl_job_id: string;
  failure_rate: number;
  rng_seed: number;
  external_features?: { lppl?: boolean };
};

export type BacktestJobDetailPayloadDTO = {
  id: number;
  status: BacktestJobStatus;
  template_id: number;
  template_name: string;
  instance_id: number | null;
  symbol: string;
  market_kind: string;
  window_start: string;
  window_end: string;
  created_at: string;
  updated_at: string;
  started_at: string;
  finished_at: string;
  duration_ms: number | null;
  progress: BacktestProgressDTO | null;
  error_message: string;
  request: BacktestRequestDTO;
};

export type BacktestEquityPointDTO = {
  unix_ms: number;
  equity: number;
  balance: number;
  net_position: number;
  mark: number;
  step_sequence: number;
  traded_notional_step?: number;
};

export type BacktestCommandStatDTO = {
  command_id: string;
  intent_id: string;
  status: string;
  partial: boolean;
  message?: string;
};

export type BacktestPerformanceMetricsDTO = {
  initial_equity: number;
  final_equity: number;
  total_return: number;
  max_drawdown_01: number;
  win_rate: number;
  num_round_trips: number;
  turnover_ratio: number;
  command_hit_rate: number;
  command_fail_rate: number;
  partial_fill_rate: number;
  avg_holding_steps: number;
  avg_equity: number;
  cumulative_net_fees?: number;
};

export type BacktestReportDTO = {
  metrics: BacktestPerformanceMetricsDTO;
  equity_curve: BacktestEquityPointDTO[];
  command_stats: BacktestCommandStatDTO[];
};

export type BacktestJobDetailResponseDTO = {
  job: BacktestJobDetailPayloadDTO;
  report: BacktestReportDTO | null;
  logs: string[];
};

export type BacktestCreateResponseDTO = {
  id: number;
  status: BacktestJobStatus;
};

export type BacktestRerunResponseDTO = {
  id: number;
  status: BacktestJobStatus;
};
