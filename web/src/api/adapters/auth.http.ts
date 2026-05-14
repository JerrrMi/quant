import { apiFetch } from "@/api/client";
import { API_PATHS } from "@/lib/api-paths";
import type { AuthSessionDTO } from "@/types/auth";

import type { LoginInput, LoginResultDTO } from "./auth.types";

export async function httpLogin(input: LoginInput): Promise<LoginResultDTO> {
  return apiFetch<LoginResultDTO>(API_PATHS.auth.login, {
    method: "POST",
    json: input,
  });
}

export async function httpLogout(): Promise<void> {
  await apiFetch<{ ok: boolean }>(API_PATHS.auth.logout, {
    method: "POST",
  });
}

export async function httpFetchSession(): Promise<AuthSessionDTO | null> {
  try {
    return await apiFetch<AuthSessionDTO>(API_PATHS.auth.session, {
      method: "GET",
    });
  } catch {
    return null;
  }
}
