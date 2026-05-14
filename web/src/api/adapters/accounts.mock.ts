import type { AccountsHistoryDTO, AccountsOverviewDTO } from "@/types/accounts";

import {
  buildAccountsHistoryDTO,
  buildAccountsOverviewDTO,
} from "@/lib/console-seed-agents-accounts";

import type { AccountsApiAdapter } from "./accounts.types";

function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchOverview(): Promise<AccountsOverviewDTO> {
  await delay(220);
  return buildAccountsOverviewDTO();
}

async function fetchHistory(params: {
  venue: "spot" | "futures";
  metric: "equity" | "available";
}): Promise<AccountsHistoryDTO> {
  await delay(200);
  return buildAccountsHistoryDTO(params.venue, params.metric);
}

export const accountsMockAdapter: AccountsApiAdapter = {
  fetchOverview,
  fetchHistory,
};
