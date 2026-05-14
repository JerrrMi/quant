/** Relative paths resolved against `NEXT_PUBLIC_API_BASE_URL` in `apiFetch`. */
export const API_PATHS = {
  auth: {
    login: "/api/auth/login",
    logout: "/api/auth/logout",
    session: "/api/auth/session",
  },
  system: {
    dashboard: "/api/system/dashboard",
  },
  agents: {
    list: "/api/agents",
    detail: (id: string) => `/api/agents/${encodeURIComponent(id)}`,
    control: (id: string) =>
      `/api/agents/${encodeURIComponent(id)}/control`,
  },
  accounts: {
    overview: "/api/accounts",
    history: "/api/accounts/history",
  },
  console: {
    templates: "/api/console/templates",
    template: (id: string | number) =>
      `/api/console/templates/${encodeURIComponent(String(id))}`,
    instances: "/api/console/instances",
    instance: (id: string | number) =>
      `/api/console/instances/${encodeURIComponent(String(id))}`,
    instanceActions: (id: string | number) =>
      `/api/console/instances/${encodeURIComponent(String(id))}/actions`,
    backtests: "/api/console/backtests",
    backtest: (id: string | number) =>
      `/api/console/backtests/${encodeURIComponent(String(id))}`,
    backtestActions: (id: string | number) =>
      `/api/console/backtests/${encodeURIComponent(String(id))}/actions`,
  },
} as const;
