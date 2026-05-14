import { apiFetch } from "@/api/client";
import type { AuthSessionDTO } from "@/types/auth";

/** Backend-shaped auth payloads (adjust when SaaS exposes real endpoints). */
export type LoginApiPayload = {
  email: string;
  password: string;
};

export async function loginRequest(payload: LoginApiPayload): Promise<void> {
  await apiFetch<{ ok: boolean }>("/api/auth/login", {
    method: "POST",
    json: payload,
  });
}

export async function logoutRequest(): Promise<void> {
  await apiFetch<{ ok: boolean }>("/api/auth/logout", {
    method: "POST",
  });
}

export async function fetchSession(): Promise<AuthSessionDTO | null> {
  try {
    return await apiFetch<AuthSessionDTO>("/api/auth/session", {
      method: "GET",
    });
  } catch {
    return null;
  }
}
