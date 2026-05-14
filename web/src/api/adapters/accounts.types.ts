import type { AccountsHistoryDTO, AccountsOverviewDTO } from "@/types/accounts";

export type AccountsApiAdapter = {
  fetchOverview(): Promise<AccountsOverviewDTO>;
  fetchHistory(params: {
    venue: "spot" | "futures";
    metric: "equity" | "available";
  }): Promise<AccountsHistoryDTO>;
};
