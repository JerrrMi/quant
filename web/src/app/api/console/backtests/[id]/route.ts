import {
  forwardConsoleRequest,
  mirrorJsonResponse,
} from "@/lib/saas-console-proxy";

type RouteCtx = { params: Promise<{ id: string }> };

export async function GET(_req: Request, ctx: RouteCtx) {
  const { id } = await ctx.params;
  const upstream = await forwardConsoleRequest(
    `/v1/console/backtests/${encodeURIComponent(id)}`,
  );
  return mirrorJsonResponse(upstream);
}
