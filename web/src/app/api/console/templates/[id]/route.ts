import {
  forwardConsoleRequest,
  mirrorJsonResponse,
} from "@/lib/saas-console-proxy";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(_req: Request, ctx: RouteContext) {
  const { id } = await ctx.params;
  const upstream = await forwardConsoleRequest(
    `/v1/console/templates/${encodeURIComponent(id)}`,
  );
  return mirrorJsonResponse(upstream);
}
