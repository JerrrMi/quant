import { NextResponse } from "next/server";

import type { AgentControlActionDTO } from "@/types/agents";

type RouteContext = { params: Promise<{ id: string }> };

const ACTIONS: AgentControlActionDTO[] = [
  "start",
  "stop",
  "reconnect",
  "refresh",
];

export async function POST(req: Request, ctx: RouteContext) {
  const { id } = await ctx.params;
  const agentId = decodeURIComponent(id);
  let body: unknown;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ message: "Invalid JSON body" }, { status: 400 });
  }
  const action =
    typeof body === "object" && body !== null && "action" in body
      ? (body as { action: unknown }).action
      : undefined;
  if (typeof action !== "string" || !ACTIONS.includes(action as AgentControlActionDTO)) {
    return NextResponse.json(
      { message: `action must be one of: ${ACTIONS.join(", ")}` },
      { status: 400 },
    );
  }

  const ok =
    agentId !== "agent-local-spot" ||
    action === "refresh" ||
    action === "reconnect";

  return NextResponse.json({
    ok,
    message: ok
      ? `已接受指令「${action}」· ${agentId}（BFF 占位：请接入 SaaS 真源）`
      : `Agent 离线或不可用，无法接受「${action}」`,
    appliedAt: new Date().toISOString(),
  });
}
