import {
  forwardConsoleRequest,
  mirrorJsonResponse,
} from "@/lib/saas-console-proxy";

type RouteContext = { params: Promise<{ id: string }> };

export async function POST(req: Request, ctx: RouteContext) {
  const { id } = await ctx.params;
  const body = await req.text();
  const upstream = await forwardConsoleRequest(
    `/v1/console/instances/${encodeURIComponent(id)}/actions`,
    {
      method: "POST",
      body,
    },
  );
  return mirrorJsonResponse(upstream);
}
