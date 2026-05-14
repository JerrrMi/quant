import { publicEnv } from "@/lib/env";

import { accountsHttpAdapter } from "./adapters/accounts.http";
import { accountsMockAdapter } from "./adapters/accounts.mock";
import type { AccountsHistoryDTO, AccountsOverviewDTO } from "@/types/accounts";

const adapter = publicEnv.useMockApi ? accountsMockAdapter : accountsHttpAdapter;

export async function fetchAccountsOverview(): Promise<AccountsOverviewDTO> {
  return adapter.fetchOverview();
}

export async function fetchAccountsHistory(params: {
  venue: "spot" | "futures";
  metric: "equity" | "available";
}): Promise<AccountsHistoryDTO> {
  return adapter.fetchHistory(params);
}
