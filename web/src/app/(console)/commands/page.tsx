"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { fetchConsoleCommands } from "@/api/commands";
import { CommandStreamTable } from "@/features/commands/command-stream-table";
import { ConsolePage } from "@/components/layout/console-page";
import { DataFreshness } from "@/components/layout/data-freshness";
import { ErrorState } from "@/components/feedback/error-state";
import { LoadingState } from "@/components/feedback/loading-state";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useConsolePoll } from "@/hooks/use-console-poll";
import type { ConsoleCommandDTO } from "@/types/commands";

const POLL_MS = 4000;

export default function CommandsStreamPage() {
  const sp = useSearchParams();
  const fromUrl = sp.get("instance_id") ?? "";

  const [filterInput, setFilterInput] = useState(fromUrl);
  const [activeFilter, setActiveFilter] = useState(fromUrl);

  useEffect(() => {
    const v = sp.get("instance_id") ?? "";
    setFilterInput(v);
    setActiveFilter(v);
  }, [sp]);

  const [rows, setRows] = useState<ConsoleCommandDTO[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const queryInstance = activeFilter.trim() || undefined;

  const refresh = useCallback(async () => {
    try {
      const res = await fetchConsoleCommands({
        limit: 100,
        instance_id: queryInstance,
      });
      setRows(res.commands);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "加载失败");
    }
  }, [queryInstance]);

  const { lastUpdated, refreshNow } = useConsolePoll(refresh, POLL_MS);

  function applyFilter() {
    setActiveFilter(filterInput.trim());
  }

  return (
    <ConsolePage
      title="命令流"
      description="跨实例的指令时间线：编排落库指令与执行端 ack/状态（ack 需 Agent 回写接入后逐步完备）。"
      meta={
        <DataFreshness
          lastUpdated={lastUpdated}
          onRefresh={refreshNow}
          hint={`每 ${POLL_MS / 1000}s 自动刷新`}
        />
      }
      actions={
        <Button variant="outline" size="sm" asChild>
          <Link href="/dashboard">返回总览</Link>
        </Button>
      }
    >
      <div className="flex flex-wrap items-end gap-4 rounded-lg border border-border bg-card p-4">
        <div className="space-y-2">
          <Label htmlFor="cmd-inst">实例 ID（可选）</Label>
          <Input
            id="cmd-inst"
            placeholder="例如 12"
            className="w-48 font-mono text-sm"
            value={filterInput}
            onChange={(e) => setFilterInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") applyFilter();
            }}
          />
        </div>
        <Button type="button" size="sm" onClick={() => applyFilter()}>
          应用筛选
        </Button>
        {queryInstance ? (
          <Button variant="link" size="sm" className="px-0" asChild>
            <Link href={`/strategies/instances/${queryInstance}`}>
              打开实例 #{queryInstance}
            </Link>
          </Button>
        ) : null}
      </div>

      {err ? (
        <ErrorState title="无法加载命令流" description={err} onRetry={refreshNow} />
      ) : null}
      {rows === null && !err ? <LoadingState label="加载命令…" /> : null}
      {rows ? <CommandStreamTable commands={rows} /> : null}
    </ConsolePage>
  );
}
