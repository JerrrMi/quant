import { NextResponse } from "next/server";

import { buildAgentDetailDTO } from "@/lib/console-seed-agents-accounts";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(_req: Request, ctx: RouteContext) {
  const { id } = await ctx.params;
  const detail = buildAgentDetailDTO(decodeURIComponent(id));
  if (!detail) {
    return NextResponse.json({ message: "Agent not found" }, { status: 404 });
  }
  return NextResponse.json(detail);
}
