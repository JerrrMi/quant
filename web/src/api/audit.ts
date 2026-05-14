import { apiFetch } from "@/api/client";
import { API_PATHS } from "@/lib/api-paths";
import type { ConsoleAuditListDTO } from "@/types/audit";

export type ConsoleAuditQuery = {
  from?: string;
  to?: string;
  level?: string;
  module?: string;
  instance_id?: string | number;
  agent_key?: string;
  actor_id?: string;
  action?: string;
  resource_id?: string;
  limit?: number;
};

export async function fetchConsoleAudit(
  q?: ConsoleAuditQuery,
): Promise<ConsoleAuditListDTO> {
  const sp = new URLSearchParams();
  if (q?.from) sp.set("from", q.from);
  if (q?.to) sp.set("to", q.to);
  if (q?.level) sp.set("level", q.level);
  if (q?.module) sp.set("module", q.module);
  if (q?.instance_id != null) sp.set("instance_id", String(q.instance_id));
  if (q?.agent_key) sp.set("agent_key", q.agent_key);
  if (q?.actor_id) sp.set("actor_id", q.actor_id);
  if (q?.action) sp.set("action", q.action);
  if (q?.resource_id) sp.set("resource_id", q.resource_id);
  if (q?.limit != null) sp.set("limit", String(q.limit));
  const suffix = sp.toString() ? `?${sp.toString()}` : "";
  return apiFetch<ConsoleAuditListDTO>(`${API_PATHS.console.audit}${suffix}`);
}
