/** `/v1/console/audit` 审计/日志事件。 */
export type ConsoleAuditEventDTO = {
  id: number;
  actor_type: string;
  actor_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  level: "error" | "warn" | "info" | string;
  module: string;
  payload_json: string;
  occurred_at: string;
};

export type ConsoleAuditListDTO = {
  events: ConsoleAuditEventDTO[];
  server_time: string;
};
