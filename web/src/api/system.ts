import { publicEnv } from "@/lib/env";

import { systemHttpAdapter } from "./adapters/system.http";
import { systemMockAdapter } from "./adapters/system.mock";
import type { SystemDashboardDTO } from "@/types/system";

export async function fetchSystemDashboard(): Promise<SystemDashboardDTO> {
  const adapter = publicEnv.useMockApi ? systemMockAdapter : systemHttpAdapter;
  return adapter.fetchDashboard();
}
