import { apiFetch } from "@/api/client";
import { API_PATHS } from "@/lib/api-paths";
import type { ConsoleCommandsListDTO } from "@/types/commands";

export type ConsoleCommandsQuery = {
  limit?: number;
  instance_id?: string | number;
};

export async function fetchConsoleCommands(
  q?: ConsoleCommandsQuery,
): Promise<ConsoleCommandsListDTO> {
  const sp = new URLSearchParams();
  if (q?.limit != null) sp.set("limit", String(q.limit));
  if (q?.instance_id != null) sp.set("instance_id", String(q.instance_id));
  const suffix = sp.toString() ? `?${sp.toString()}` : "";
  return apiFetch<ConsoleCommandsListDTO>(`${API_PATHS.console.commands}${suffix}`);
}
