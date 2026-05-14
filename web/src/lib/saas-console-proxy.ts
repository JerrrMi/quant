import { NextResponse } from "next/server";

/**
 * SaaS 控制台 REST 基址（仅服务端可见）。
 * 浏览器通过同源 `/api/console/*` BFF 转发。
 */
export function saasConsoleOrigin(): string {
  const raw =
    process.env.SAAS_CONSOLE_ORIGIN ??
    process.env.ALTSHORT_SAAS_CONSOLE_ORIGIN ??
    "http://127.0.0.1:8080";
  return raw.replace(/\/$/, "");
}

/** 将请求转发至 SaaS `/v1/console/*`。 */
export async function forwardConsoleRequest(path: string, init?: RequestInit): Promise<Response> {
  const url = `${saasConsoleOrigin()}${path.startsWith("/") ? path : `/${path}`}`;
  const headers = new Headers(init?.headers);
  if (init?.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  return fetch(url, {
    ...init,
    headers,
    cache: "no-store",
  });
}

export async function mirrorJsonResponse(upstream: Response): Promise<NextResponse> {
  const body = await upstream.text();
  const ct = upstream.headers.get("Content-Type") ?? "application/json; charset=utf-8";
  return new NextResponse(body, {
    status: upstream.status,
    headers: { "Content-Type": ct },
  });
}
