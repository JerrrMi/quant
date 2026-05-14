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
} as const;
