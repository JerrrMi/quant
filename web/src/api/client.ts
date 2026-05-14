import { parseApiError } from "./errors";
import { publicEnv } from "@/lib/env";
import { API_PATHS } from "@/lib/api-paths";
import { getStoredAccessToken, clearStoredCredentials } from "@/lib/session-store";

export type FetchOptions = RequestInit & {
  json?: unknown;
};

function joinURL(base: string, path: string): string {
  if (path.startsWith("http://") || path.startsWith("https://")) return path;
  const trimmedBase = base.replace(/\/$/, "");
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  return `${trimmedBase}${normalizedPath}`;
}

export async function apiFetch<T>(
  path: string,
  options: FetchOptions = {},
): Promise<T> {
  const { json, headers, ...init } = options;
  const url = joinURL(publicEnv.apiBaseURL, path);
  const token = getStoredAccessToken();

  const response = await fetch(url, {
    ...init,
    credentials: "include",
    headers: {
      ...(json !== undefined ? { "Content-Type": "application/json" } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(headers as Record<string, string> | undefined),
    },
    body: json !== undefined ? JSON.stringify(json) : init.body,
  });

  if (
    response.status === 401 &&
    typeof window !== "undefined" &&
    path !== API_PATHS.auth.login
  ) {
    clearStoredCredentials();
    window.dispatchEvent(new CustomEvent("altshort:session-expired"));
  }

  if (!response.ok) {
    throw await parseApiError(response);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}
