"use client";

import { useCallback, useEffect, useState } from "react";

import { fetchConsoleCommands } from "@/api/commands";
import { CommandStreamTable } from "@/features/commands/command-stream-table";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { DataFreshness } from "@/components/layout/data-freshness";
import type { ConsoleCommandDTO } from "@/types/commands";

type CommandStreamDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function CommandStreamDrawer({
  open,
  onOpenChange,
}: CommandStreamDrawerProps) {
  const [rows, setRows] = useState<ConsoleCommandDTO[]>([]);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setErr(null);
    try {
      const res = await fetchConsoleCommands({ limit: 40 });
      setRows(res.commands);
      setLastUpdated(new Date());
    } catch (e) {
      setErr(e instanceof Error ? e.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!open) return;
    void load();
    const id = window.setInterval(() => void load(), 3500);
    return () => window.clearInterval(id);
  }, [open, load]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="flex w-full flex-col gap-0 overflow-y-auto sm:max-w-xl md:max-w-2xl">
        <SheetHeader className="border-b pb-4 text-left">
          <SheetTitle>最近命令流</SheetTitle>
          <SheetDescription>
            SaaS 持久化的交易指令；可与实例详情、审计日志交叉排查。
          </SheetDescription>
          <DataFreshness
            lastUpdated={lastUpdated}
            onRefresh={load}
            refreshing={loading}
            hint="侧栏打开时每 3.5s 刷新"
          />
        </SheetHeader>
        <div className="flex-1 py-4">
          {err ? (
            <p className="text-sm text-destructive">{err}</p>
          ) : (
            <CommandStreamTable commands={rows} dense />
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
