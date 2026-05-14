import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { SESSION_COOKIE_NAME } from "@/lib/constants";
import type { AuthSessionDTO } from "@/types/auth";

export async function GET() {
  if (process.env.NEXT_PUBLIC_DEV_MOCK_AUTH === "true") {
    const session: AuthSessionDTO = {
      user: {
        id: "mock-user",
        email: "mock@local.dev",
        displayName: "Mock Operator",
      },
    };
    return NextResponse.json(session);
  }

  const cookieStore = await cookies();
  const raw = cookieStore.get(SESSION_COOKIE_NAME)?.value;

  if (!raw) {
    const session: AuthSessionDTO = { user: null };
    return NextResponse.json(session);
  }

  try {
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null) {
      const session: AuthSessionDTO = { user: null };
      return NextResponse.json(session);
    }

    const payload = parsed as Record<string, unknown>;
    const email = payload.email;

    if (typeof email !== "string" || email.length === 0) {
      const session: AuthSessionDTO = { user: null };
      return NextResponse.json(session);
    }

    const id = payload.id;
    const displayName = payload.displayName;

    const session: AuthSessionDTO = {
      user: {
        id: typeof id === "string" && id.length > 0 ? id : crypto.randomUUID(),
        email,
        displayName:
          typeof displayName === "string" && displayName.length > 0
            ? displayName
            : undefined,
      },
    };
    return NextResponse.json(session);
  } catch {
    const session: AuthSessionDTO = { user: null };
    return NextResponse.json(session);
  }
}
