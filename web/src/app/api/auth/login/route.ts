import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { z } from "zod";

import { SESSION_COOKIE_NAME } from "@/lib/constants";

const loginSchema = z.object({
  account: z
    .string({ message: "请填写账号" })
    .min(2, { message: "账号至少 2 个字符" }),
  password: z
    .string({ message: "请填写密码" })
    .min(6, { message: "密码至少 6 位" }),
});

export async function POST(request: Request) {
  const json = await request.json().catch(() => null);
  const parsed = loginSchema.safeParse(json);

  if (!parsed.success) {
    return NextResponse.json(
      {
        message: "请检查账号与密码格式",
        issues: parsed.error.flatten(),
      },
      { status: 400 },
    );
  }

  const { account, password } = parsed.data;

  // Demo: predictable failure for UX testing (remove when SaaS auth is wired).
  if (account === "fail" || password === "wrong") {
    return NextResponse.json(
      { message: "账号或密码错误", code: "INVALID_CREDENTIALS" },
      { status: 401 },
    );
  }

  const email = account.includes("@") ? account : `${account}@console.local`;
  const user = {
    id: crypto.randomUUID(),
    email,
    displayName: account.includes("@") ? account.split("@")[0]! : account,
  };

  const cookieStore = await cookies();
  cookieStore.set(SESSION_COOKIE_NAME, JSON.stringify(user), {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24 * 7,
  });

  const payload: { ok: true; accessToken?: string; expiresAt?: number } = {
    ok: true,
  };

  if (process.env.NEXT_PUBLIC_ISSUE_DEV_BEARER === "true") {
    payload.accessToken = `dev.${user.id}`;
    payload.expiresAt = Date.now() + 60 * 60 * 24 * 1000;
  }

  return NextResponse.json(payload);
}
