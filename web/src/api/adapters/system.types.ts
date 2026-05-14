import type { SystemDashboardDTO } from "@/types/system";

export type SystemDashboardAdapter = {
  fetchDashboard: () => Promise<SystemDashboardDTO>;
};
