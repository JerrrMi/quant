import { apiFetch } from "@/api/client";
import { API_PATHS } from "@/lib/api-paths";
import type { SystemDashboardDTO } from "@/types/system";

import type { SystemDashboardAdapter } from "./system.types";

async function fetchDashboard(): Promise<SystemDashboardDTO> {
  return apiFetch<SystemDashboardDTO>(API_PATHS.system.dashboard, {
    method: "GET",
  });
}

export const systemHttpAdapter: SystemDashboardAdapter = {
  fetchDashboard,
};
