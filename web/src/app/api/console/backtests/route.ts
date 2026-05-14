import {
  forwardConsoleRequest,
  mirrorJsonResponse,
} from "@/lib/saas-console-proxy";

export async function GET() {
  const upstream = await forwardConsoleRequest("/v1/console/backtests");
  return mirrorJsonResponse(upstream);
}

export async function POST(req: Request) {
  const body = await req.text();
  const upstream = await forwardConsoleRequest("/v1/console/backtests", {
    method: "POST",
    body,
  });
  return mirrorJsonResponse(upstream);
}
