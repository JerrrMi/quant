import { apiFetch } from "@/api/client";
import { API_PATHS } from "@/lib/api-paths";
import type { AccountsHistoryDTO, AccountsOverviewDTO } from "@/types/accounts";

import type { AccountsApiAdapter } from "./accounts.types";

async function fetchOverview(): Promise<AccountsOverviewDTO> {
  return apiFetch<AccountsOverviewDTO>(API_PATHS.accounts.overview, {
    method: "GET",
  });
}

async function fetchHistory(params: {
  venue: "spot" | "futures";
  metric: "equity" | "available";
}): Promise<AccountsHistoryDTO> {
  const qs = new URLSearchParams({
    venue: params.venue,
    metric: params.metric,
  });
  return apiFetch<AccountsHistoryDTO>(
    `${API_PATHS.accounts.history}?${qs.toString()}`,
    { method: "GET" },
  );
}

export const accountsHttpAdapter: AccountsApiAdapter = {
  fetchOverview,
  fetchHistory,
};
