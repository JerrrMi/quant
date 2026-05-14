import {
  forwardConsoleRequest,
  mirrorJsonResponse,
} from "@/lib/saas-console-proxy";

export async function GET(req: Request) {
  const url = new URL(req.url);
  const qs = url.searchParams.toString();
  const path = qs ? `/v1/console/audit?${qs}` : "/v1/console/audit";
  const upstream = await forwardConsoleRequest(path);
  return mirrorJsonResponse(upstream);
}
