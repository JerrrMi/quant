"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { fetchConsoleAudit } from "@/api/audit";
import { ConsolePage } from "@/components/layout/console-page";
import { DataFreshness } from "@/components/layout/data-freshness";
import { ErrorState } from "@/components/feedback/error-state";
import { LoadingState } from "@/components/feedback/loading-state";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useConsolePoll } from "@/hooks/use-console-poll";
import { cn } from "@/lib/utils";
import { formatDateTime } from "@/lib/format-trading";
import type { ConsoleAuditEventDTO } from "@/types/audit";

const POLL_MS = 12_000;

type AuditFilters = {
  from: string;
  to: string;
  level: string;
  module: string;
  instanceId: string;
  agentKey: string;
  action: string;
};

const emptyFilters = (): AuditFilters => ({
  from: "",
  to: "",
  level: "",
  module: "",
  instanceId: "",
  agentKey: "",
  action: "",
});

function levelBadge(level: string) {
  const l = level.toLowerCase();
  if (l === "error")
    return (
      <Badge variant="outline" className="border-destructive/50 text-destructive">
        error
      </Badge>
    );
  if (l === "warn")
    return (
      <Badge variant="outline" className="border-amber-500/50 text-amber-800 dark:text-amber-200">
        warn
      </Badge>
    );
  return <Badge variant="outline">info</Badge>;
}

function levelRowClass(level: string) {
  const l = level.toLowerCase();
  if (l === "error") return "border-l-4 border-l-destructive bg-destructive/5";
  if (l === "warn") return "border-l-4 border-l-amber-500 bg-amber-500/10";
  return "border-l-4 border-l-border bg-card";
}

export default function LogsPage() {
  const sp = useSearchParams();

  const [form, setForm] = useState<AuditFilters>(emptyFilters);
  const [query, setQuery] = useState<AuditFilters>(emptyFilters);

  useEffect(() => {
    const inst = sp.get("instance_id") ?? "";
    const agent = sp.get("agent_key") ?? "";
    setForm((f) => ({ ...f, instanceId: inst, agentKey: agent }));
    setQuery((q) => ({ ...q, instanceId: inst, agentKey: agent }));
  }, [sp]);

  const [events, setEvents] = useState<ConsoleAuditEventDTO[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const res = await fetchConsoleAudit({
        from: query.from.trim() || undefined,
        to: query.to.trim() || undefined,
        level: query.level.trim() || undefined,
        module: query.module.trim() || undefined,
        instance_id: query.instanceId.trim() || undefined,
        agent_key: query.agentKey.trim() || undefined,
        action: query.action.trim() || undefined,
        limit: 200,
      });
      setEvents(res.events);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "加载失败");
    }
  }, [query]);

  const { lastUpdated, refreshNow } = useConsolePoll(refresh, POLL_MS);

  function applyFilters() {
    setQuery({ ...form });
  }

  return (
    <ConsolePage
      title="日志与审计"
      description="SaaS 侧审计事件（控制台动作、指令生命周期等）。支持按时间、级别、模块、实例与 Agent 过滤；可与命令流、实例详情对照排查。"
      meta={
        <DataFreshness
          lastUpdated={lastUpdated}
          onRefresh={refreshNow}
          hint={`每 ${POLL_MS / 1000}s 按已应用条件自动刷新`}
        />
      }
      actions={
        <Button variant="outline" size="sm" asChild>
          <Link href="/commands">命令流</Link>
        </Button>
      }
    >
      <div className="grid gap-4 rounded-lg border border-border bg-card p-4 md:grid-cols-2 lg:grid-cols-3">
        <div className="space-y-2">
          <Label htmlFor="log-from">时间起（RFC3339，可选）</Label>
          <Input
            id="log-from"
            placeholder="2026-01-01T00:00:00Z"
            value={form.from}
            onChange={(e) => setForm((f) => ({ ...f, from: e.target.value }))}
            className="font-mono text-xs"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="log-to">时间止（RFC3339，可选）</Label>
          <Input
            id="log-to"
            value={form.to}
            onChange={(e) => setForm((f) => ({ ...f, to: e.target.value }))}
            className="font-mono text-xs"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="log-level">级别</Label>
          <Input
            id="log-level"
            placeholder="error | warn | info"
            value={form.level}
            onChange={(e) => setForm((f) => ({ ...f, level: e.target.value }))}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="log-mod">模块（resource_type）</Label>
          <Input
            id="log-mod"
            placeholder="instance / trade_command"
            value={form.module}
            onChange={(e) => setForm((f) => ({ ...f, module: e.target.value }))}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="log-inst">实例 ID</Label>
          <Input
            id="log-inst"
            value={form.instanceId}
            onChange={(e) =>
              setForm((f) => ({ ...f, instanceId: e.target.value }))
            }
            className="font-mono text-xs"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="log-agent">AgentKey</Label>
          <Input
            id="log-agent"
            value={form.agentKey}
            onChange={(e) =>
              setForm((f) => ({ ...f, agentKey: e.target.value }))
            }
            className="font-mono text-xs"
          />
        </div>
        <div className="space-y-2 md:col-span-2 lg:col-span-3">
          <Label htmlFor="log-action">动作前缀（action LIKE）</Label>
          <Input
            id="log-action"
            placeholder="console.instance"
            value={form.action}
            onChange={(e) =>
              setForm((f) => ({ ...f, action: e.target.value }))
            }
            className="font-mono text-xs"
          />
        </div>
        <div className="flex flex-wrap gap-2 md:col-span-2 lg:col-span-3">
          <Button
            type="button"
            size="sm"
            onClick={() => {
              applyFilters();
            }}
          >
            应用筛选
          </Button>
          <Button type="button" variant="secondary" size="sm" onClick={refreshNow}>
            立即刷新
          </Button>
        </div>
      </div>

      {err ? (
        <ErrorState title="无法加载审计日志" description={err} onRetry={refreshNow} />
      ) : null}

      {events === null && !err ? <LoadingState label="加载审计事件…" /> : null}

      {events && events.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          没有符合条件的事件（缩小过滤范围或先在实例上产生动作）。
        </p>
      ) : null}

      {events && events.length > 0 ? (
        <div className="space-y-2">
          <div className="text-xs text-muted-foreground">
            共 {events.length} 条（单次最多 200 条，后端限制 500）
          </div>
          <ul className="space-y-2">
            {events.map((ev) => (
              <li
                key={`${ev.id}-${ev.occurred_at}`}
                className={cn(
                  "rounded-md border border-border px-3 py-2 text-sm",
                  levelRowClass(ev.level),
                )}
              >
                <div className="flex flex-wrap items-center gap-2">
                  {levelBadge(ev.level)}
                  <span className="font-mono text-[11px] text-muted-foreground">
                    {ev.module}
                  </span>
                  <span className="font-medium">{ev.action}</span>
                  <span className="text-xs text-muted-foreground">
                    {formatDateTime(ev.occurred_at)}
                  </span>
                </div>
                <div className="mt-1 flex flex-wrap gap-2 text-[11px] text-muted-foreground">
                  <span>
                    actor: {ev.actor_type}/{ev.actor_id || "—"}
                  </span>
                  <span>
                    resource: {ev.resource_type}/{ev.resource_id}
                  </span>
                  {ev.resource_type === "instance" ? (
                    <Link
                      className="text-primary hover:underline"
                      href={`/strategies/instances/${ev.resource_id}`}
                    >
                      打开实例
                    </Link>
                  ) : null}
                  {ev.resource_type === "trade_command" ? (
                    <Link className="text-primary hover:underline" href="/commands">
                      命令流
                    </Link>
                  ) : null}
                </div>
                {ev.payload_json ? (
                  <pre className="mt-2 max-h-36 overflow-auto rounded bg-muted/40 p-2 text-[11px] leading-relaxed">
                    {ev.payload_json}
                  </pre>
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </ConsolePage>
  );
}
