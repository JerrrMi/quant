import { NextResponse } from "next/server";

import { buildAccountsOverviewDTO } from "@/lib/console-seed-agents-accounts";

export async function GET() {
  return NextResponse.json(buildAccountsOverviewDTO());
}
