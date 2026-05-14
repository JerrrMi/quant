const ACCESS_TOKEN_KEY = "altshort_access_token";
const ACCESS_TOKEN_EXP_KEY = "altshort_access_token_exp";
/** Placeholder only — wire to longer cookie/session TTL when backend supports it. */
const REMEMBER_ME_KEY = "altshort_remember_me_placeholder";

export function getStoredAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  const expRaw = window.localStorage.getItem(ACCESS_TOKEN_EXP_KEY);
  if (expRaw) {
    const exp = Number(expRaw);
    if (!Number.isNaN(exp) && exp > 0 && exp < Date.now()) {
      clearStoredCredentials();
      return null;
    }
  }
  return window.localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function setStoredAccessToken(token: string, expiresAtMs?: number | null) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(ACCESS_TOKEN_KEY, token);
  if (expiresAtMs != null && expiresAtMs > 0) {
    window.localStorage.setItem(ACCESS_TOKEN_EXP_KEY, String(expiresAtMs));
  } else {
    window.localStorage.removeItem(ACCESS_TOKEN_EXP_KEY);
  }
}

export function getRememberMePlaceholder(): boolean {
  if (typeof window === "undefined") return false;
  return window.localStorage.getItem(REMEMBER_ME_KEY) === "1";
}

export function setRememberMePlaceholder(value: boolean) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(REMEMBER_ME_KEY, value ? "1" : "0");
}

/** Clears bearer credentials; httpOnly session cookies are cleared via logout API. */
export function clearStoredCredentials() {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(ACCESS_TOKEN_KEY);
  window.localStorage.removeItem(ACCESS_TOKEN_EXP_KEY);
}

/** Named facade for auth/session persistence (tokens + placeholders). */
export const SessionStore = {
  getAccessToken: getStoredAccessToken,
  setAccessToken: setStoredAccessToken,
  clearCredentials: clearStoredCredentials,
  getRememberMePlaceholder,
  setRememberMePlaceholder,
} as const;
