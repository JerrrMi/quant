import { NextResponse } from "next/server";

import { buildAgentsListDTO } from "@/lib/console-seed-agents-accounts";

/** BFF：生产环境将 `NEXT_PUBLIC_API_BASE_URL` 指向 SaaS 网关后由适配层转发同源契约。 */
export async function GET() {
  return NextResponse.json(buildAgentsListDTO());
}
