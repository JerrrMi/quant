import {
  setStoredAccessToken,
  clearStoredCredentials,
} from "@/lib/session-store";
import type { AuthSessionDTO } from "@/types/auth";

import * as authHttp from "./adapters/auth.http";
import type { LoginInput, LoginResultDTO } from "./adapters/auth.types";

export type { LoginInput } from "./adapters/auth.types";

export async function login(input: LoginInput): Promise<LoginResultDTO> {
  const result = await authHttp.httpLogin(input);
  if (result.accessToken) {
    setStoredAccessToken(result.accessToken, result.expiresAt ?? null);
  }
  return result;
}

export async function logout(): Promise<void> {
  try {
    await authHttp.httpLogout();
  } finally {
    clearStoredCredentials();
  }
}

export async function getSession(): Promise<AuthSessionDTO | null> {
  return authHttp.httpFetchSession();
}
