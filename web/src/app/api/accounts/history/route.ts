import { NextResponse } from "next/server";

import { buildAccountsHistoryDTO } from "@/lib/console-seed-agents-accounts";

export async function GET(req: Request) {
  const url = new URL(req.url);
  const venueRaw = url.searchParams.get("venue") ?? "futures";
  const metricRaw = url.searchParams.get("metric") ?? "equity";

  const venue = venueRaw === "spot" ? "spot" : "futures";
  const metric = metricRaw === "available" ? "available" : "equity";

  return NextResponse.json(buildAccountsHistoryDTO(venue, metric));
}
