"use client";

import { useCallback, useEffect, useState } from "react";

/**
 * 固定间隔轮询（控制台实时联动的默认方案；可与未来 SSE/WS 并存）。
 */
export function useConsolePoll(
  refresh: () => void | Promise<void>,
  intervalMs: number,
  options?: { immediate?: boolean },
) {
  const immediate = options?.immediate !== false;
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const run = useCallback(async () => {
    await refresh();
    setLastUpdated(new Date());
  }, [refresh]);

  useEffect(() => {
    if (immediate) void run();
    const id = window.setInterval(() => void run(), intervalMs);
    return () => window.clearInterval(id);
  }, [run, intervalMs, immediate]);

  return { lastUpdated, refreshNow: run };
}
