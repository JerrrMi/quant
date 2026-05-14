/** `/v1/console/commands` 指令视图（与控制台 JSON 对齐）。 */
export type ConsoleCommandDTO = {
  command_id: string;
  instance_id: number;
  symbol: string;
  intent: string;
  kind: string;
  status: string;
  issued_at: string;
  dispatched_at: string;
  acked_at: string;
  /** 记录状态最近迁移时间（WS 回报落库接入前作占位） */
  report_at: string;
  error: string;
  summary: string;
  id: string;
  created_at: string;
};

export type ConsoleCommandsListDTO = {
  commands: ConsoleCommandDTO[];
  server_time: string;
};
