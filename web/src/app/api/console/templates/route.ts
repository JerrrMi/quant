import {
  forwardConsoleRequest,
  mirrorJsonResponse,
} from "@/lib/saas-console-proxy";

export async function GET() {
  const upstream = await forwardConsoleRequest("/v1/console/templates");
  return mirrorJsonResponse(upstream);
}
