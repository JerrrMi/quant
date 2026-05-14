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
} as const;
